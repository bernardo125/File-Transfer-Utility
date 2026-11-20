package transport

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"net"
	"testing"
	"time"

	"github.com/quic-go/quic-go"
)

type ident struct {
	pub  ed25519.PublicKey
	priv ed25519.PrivateKey
}

func newIdent(t *testing.T) ident {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return ident{pub, priv}
}

func newSock(t *testing.T) (*net.UDPConn, *net.UDPAddr) {
	t.Helper()
	c, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close() })
	return c, c.LocalAddr().(*net.UDPAddr)
}

type dialRes struct {
	c   *quic.Conn
	err error
}

// A peer that does not hold the private key matching the public key in the token must never
// obtain a usable session. The token is the only identity the product has (no CA, no signaling
// server), so this is the product's authentication guarantee.
func TestListenerRejectsDialerWithUnexpectedKey(t *testing.T) {
	listener, impostor, expected := newIdent(t), newIdent(t), newIdent(t)
	lSock, lAddr := newSock(t)
	dSock, _ := newSock(t)

	accepted := make(chan dialRes, 1)
	go func() {
		c, err := quicDoListen(lSock, &listener.priv, &expected.pub, 2*time.Second)
		accepted <- dialRes{c, err}
	}()
	dialed := make(chan dialRes, 1)
	go func() {
		c, err := quicDoDial(dSock, lAddr, &impostor.priv, &listener.pub, 2*time.Second)
		dialed <- dialRes{c, err}
	}()

	got := <-accepted
	if got.err == nil {
		got.c.CloseWithError(0, "")
		t.Fatalf("listener accepted a dialer whose key is not the one in the token")
	}
	if d := <-dialed; d.c != nil {
		d.c.CloseWithError(0, "")
	}
}

func TestDialerRejectsListenerWithUnexpectedKey(t *testing.T) {
	impostor, dialer, expected := newIdent(t), newIdent(t), newIdent(t)
	lSock, lAddr := newSock(t)
	dSock, _ := newSock(t)

	go func() {
		if c, err := quicDoListen(lSock, &impostor.priv, &dialer.pub, 2*time.Second); err == nil {
			c.CloseWithError(0, "")
		}
	}()
	c, err := quicDoDial(dSock, lAddr, &dialer.priv, &expected.pub, 2*time.Second)
	if err == nil {
		c.CloseWithError(0, "")
		t.Fatalf("dialer connected to a listener that does not own the token's key")
	}
}

func TestMutualAuthenticationSucceedsWithCorrectKeys(t *testing.T) {
	l, d := newIdent(t), newIdent(t)
	lSock, lAddr := newSock(t)
	dSock, _ := newSock(t)

	accepted := make(chan dialRes, 1)
	go func() {
		c, err := quicDoListen(lSock, &l.priv, &d.pub, 5*time.Second)
		accepted <- dialRes{c, err}
	}()
	dc, err := quicDoDial(dSock, lAddr, &d.priv, &l.pub, 5*time.Second)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { dc.CloseWithError(0, "") })
	lc := <-accepted
	if lc.err != nil {
		t.Fatalf("listen: %v", lc.err)
	}
	t.Cleanup(func() { lc.c.CloseWithError(0, "") })
}

func TestDialToUnavailablePeerFailsWithinTimeout(t *testing.T) {
	me, other := newIdent(t), newIdent(t)
	sock, _ := newSock(t)
	deadSock, deadAddr := newSock(t) // bound, but nobody speaks QUIC on it
	_ = deadSock

	start := time.Now()
	c, err := quicDoDial(sock, deadAddr, &me.priv, &other.pub, 300*time.Millisecond)
	if err == nil {
		c.CloseWithError(0, "")
		t.Fatal("dial to a peer that is not running succeeded")
	}
	if el := time.Since(start); el > 3*time.Second {
		t.Fatalf("dial did not honour its timeout: took %v", el)
	}
}

// Detects (F-04): a spoofed/wrong first candidate address ending the connection attempt even
// though the genuine address is also known.
func TestDialerFallsBackToNextCandidate(t *testing.T) {
	old := dialAttemptTimeout
	dialAttemptTimeout = 500 * time.Millisecond
	t.Cleanup(func() { dialAttemptTimeout = old })

	var d, l ident
	for {
		d, l = newIdent(t), newIdent(t)
		if bytes.Compare(d.pub, l.pub) < 0 { // d dials, l listens
			break
		}
	}
	lSock, lAddr := newSock(t)
	dSock, _ := newSock(t)
	_, deadAddr := newSock(t) // bound, but nobody answers: models a forged first candidate

	accepted := make(chan dialRes, 1)
	go func() {
		c, err := quicDoListen(lSock, &l.priv, &d.pub, 10*time.Second)
		accepted <- dialRes{c, err}
	}()
	c, err := QuicUpgradeCandidates(context.Background(), dSock, []*net.UDPAddr{deadAddr, lAddr}, &d.priv, &l.pub)
	if err != nil {
		t.Fatalf("dialer did not fall back to the genuine candidate: %v", err)
	}
	t.Cleanup(func() { c.CloseWithError(0, "") })
	if r := <-accepted; r.err != nil {
		t.Fatalf("listener: %v", r.err)
	} else {
		t.Cleanup(func() { r.c.CloseWithError(0, "") })
	}
}

func TestQuicUpgradeCandidatesHonoursCancellation(t *testing.T) {
	var d, l ident
	for {
		d, l = newIdent(t), newIdent(t)
		if bytes.Compare(d.pub, l.pub) < 0 {
			break
		}
	}
	dSock, _ := newSock(t)
	_, dead1 := newSock(t)
	_, dead2 := newSock(t)
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	start := time.Now()
	if c, err := QuicUpgradeCandidates(ctx, dSock, []*net.UDPAddr{dead1, dead2}, &d.priv, &l.pub); err == nil {
		c.CloseWithError(0, "")
		t.Fatal("connected to nothing")
	}
	if time.Since(start) > 3*time.Second {
		t.Fatalf("cancellation ignored: %v", time.Since(start))
	}
}
