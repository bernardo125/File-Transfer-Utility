package connectivity

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"net"
	"os"
	"testing"
	"time"
)

// fast config so tests do not wait for the 60s production timeout
func testConfig(timeout time.Duration) HolePunchConfig {
	return HolePunchConfig{Timeout: timeout, ProbeInterval: 10 * time.Millisecond, ListenInterval: 10 * time.Millisecond, MaxAttempts: 1000}
}

type node struct {
	conn *net.UDPConn
	pub  ed25519.PublicKey
	priv ed25519.PrivateKey
}

func (n *node) addr() *net.UDPAddr {
	return &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: n.conn.LocalAddr().(*net.UDPAddr).Port}
}

func newNode(t *testing.T) *node {
	t.Helper()
	c, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close() })
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return &node{c, pub, priv}
}

func wait(t *testing.T, ch <-chan HolePunchResult) HolePunchResult {
	t.Helper()
	select {
	case r := <-ch:
		return r
	case <-time.After(10 * time.Second):
		t.Fatal("hole punch produced no verdict (hang)")
		return HolePunchResult{}
	}
}

func punch(cfg HolePunchConfig, me, peer *node, local, remote *net.UDPAddr) <-chan HolePunchResult {
	return udpHolePunch(cfg, me.conn, local, remote, nil, &me.priv, &peer.pub)
}

// Detects: the two peers failing to find each other through the real UDP path; the verdict
// address must be the peer's actual address (it is what QUIC dials next).
func TestHolePunchTwoPeersFindEachOther(t *testing.T) {
	a, b := newNode(t), newNode(t)
	cfg := testConfig(5 * time.Second)
	ra := punch(cfg, a, b, b.addr(), b.addr())
	rb := punch(cfg, b, a, a.addr(), a.addr())

	for name, c := range map[string]struct {
		ch   <-chan HolePunchResult
		want *net.UDPAddr
	}{"a": {ra, b.addr()}, "b": {rb, a.addr()}} {
		res := wait(t, c.ch)
		if res.Err != nil {
			t.Fatalf("peer %s: %v", name, res.Err)
		}
		if res.Addr.Port != c.want.Port {
			t.Fatalf("peer %s: verdict addr %v, want port %d", name, res.Addr, c.want.Port)
		}
	}
}

// Detects: a stale/unreachable candidate in the token blocking connection when another
// candidate (e.g. the LAN one) is valid.
func TestHolePunchSucceedsWithStaleCandidate(t *testing.T) {
	a, b := newNode(t), newNode(t)
	dead := newNode(t)
	deadAddr := dead.addr()
	dead.conn.Close() // nothing listens there any more

	cfg := testConfig(5 * time.Second)
	ra := punch(cfg, a, b, deadAddr, b.addr())
	rb := punch(cfg, b, a, a.addr(), deadAddr)
	if r := wait(t, ra); r.Err != nil {
		t.Fatalf("a: %v", r.Err)
	}
	if r := wait(t, rb); r.Err != nil {
		t.Fatalf("b: %v", r.Err)
	}
}

// Detects: accepting a PUNCH from someone who does not hold the expected key (spoofing), and
// failing to report a timeout when the peer never shows up.
func TestHolePunchIgnoresWrongKeyAndTimesOut(t *testing.T) {
	a, b, intruder := newNode(t), newNode(t), newNode(t)
	cfg := testConfig(300 * time.Millisecond)

	ra := punch(cfg, a, b, b.addr(), b.addr()) // a expects b ...
	// ... but only the intruder (different key) probes a
	rc := punch(cfg, intruder, a, a.addr(), a.addr())
	_ = rc

	res := wait(t, ra)
	if res.Err == nil || !os.IsTimeout(res.Err) {
		t.Fatalf("expected timeout error, got addr=%v err=%v", res.Addr, res.Err)
	}
	if res.Addr != nil {
		t.Fatalf("timed out punch must not carry an address, got %v", res.Addr)
	}
}

// Detects: junk or truncated datagrams being mistaken for a handshake, or disturbing a punch in
// progress. (Short PUNCHes were previously compared against stale buffer bytes rather than
// rejected; they cannot be distinguished from "ignored" by black-box behaviour, so this test
// guards the externally visible guarantee: no verdict from junk, legitimate peer still wins.)
func TestHolePunchSurvivesMalformedDatagrams(t *testing.T) {
	a, b := newNode(t), newNode(t)
	cfg := testConfig(5 * time.Second)
	ra := punch(cfg, a, b, b.addr(), b.addr())

	inject, err := net.DialUDP("udp4", nil, a.addr())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { inject.Close() })
	for _, pkt := range [][]byte{
		{PUNCH, 1, 2, 3, 4, 5},                     // n=6: was a panic
		append([]byte{PUNCH}, make([]byte, 31)...), // n=32: one byte short of a key
		make([]byte, 33),                           // right size, wrong type byte
		append([]byte{PUNCH}, make([]byte, 32)...), // right shape, all-zero key
	} {
		if _, err := inject.Write(pkt); err != nil {
			t.Fatal(err)
		}
	}
	select {
	case r := <-ra:
		t.Fatalf("junk datagrams produced a verdict: %+v", r)
	default:
	}

	// the legitimate peer still gets through afterwards
	rb := punch(cfg, b, a, a.addr(), a.addr())
	if r := wait(t, ra); r.Err != nil {
		t.Fatalf("a: %v", r.Err)
	}
	if r := wait(t, rb); r.Err != nil {
		t.Fatalf("b: %v", r.Err)
	}
}

// Detects: listenPeer spinning forever (or hanging) when the socket is closed under it.
func TestHolePunchReportsClosedSocket(t *testing.T) {
	a, b := newNode(t), newNode(t)
	a.conn.Close()
	res := wait(t, punch(testConfig(5*time.Second), a, b, b.addr(), b.addr()))
	if res.Err == nil {
		t.Fatal("expected an error for a closed socket")
	}
}

// Detects: probe goroutines outliving the punch (they keep hammering the peer after success).
func TestHolePunchStopsProbingAfterVerdict(t *testing.T) {
	a, b := newNode(t), newNode(t)
	cfg := testConfig(5 * time.Second)
	cfg.MaxAttempts = 100000 // would probe for ~17 minutes if never stopped

	ra := punch(cfg, a, b, b.addr(), b.addr())
	rb := punch(cfg, b, a, a.addr(), a.addr())
	wait(t, ra)
	wait(t, rb)

	// After the verdicts, drain what is already queued towards b, then require silence for a
	// window many probe intervals long. A live probe loop would deliver ~10 packets per 100ms.
	b.conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	buf := make([]byte, 64)
	for {
		if _, _, err := b.conn.ReadFromUDP(buf); err != nil {
			break
		}
	}
	b.conn.SetReadDeadline(time.Now().Add(250 * time.Millisecond))
	if n, _, err := b.conn.ReadFromUDP(buf); err == nil {
		t.Fatalf("still receiving probes (%d bytes) after hole punch finished", n)
	}
}

// Detects: connecting a peer to itself (pasting your own token).
func TestHolePunchRejectsOwnToken(t *testing.T) {
	a := newNode(t)
	res := wait(t, udpHolePunch(testConfig(time.Second), a.conn, a.addr(), a.addr(), nil, &a.priv, &a.pub))
	if !errors.Is(res.Err, ErrSameAdress) {
		t.Fatalf("err = %v, want ErrSameAdress", res.Err)
	}
}

// Detects (F-07): the attempt deadline being extended by every junk datagram, so that anyone
// who can reach the port keeps the hole punch (and the UI) hanging forever.
func TestHolePunchTimeoutIsNotExtendedByJunkPackets(t *testing.T) {
	a, b := newNode(t), newNode(t)
	attacker, err := net.DialUDP("udp4", nil, a.addr())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { attacker.Close() })
	stop := make(chan struct{})
	t.Cleanup(func() { close(stop) })
	go func() {
		for {
			select {
			case <-stop:
				return
			case <-time.After(20 * time.Millisecond):
				attacker.Write([]byte("junk datagram that is not a punch"))
			}
		}
	}()

	start := time.Now()
	// b never answers: the only traffic a sees is junk.
	res := wait(t, punch(testConfig(500*time.Millisecond), a, b, b.addr(), b.addr()))
	if res.Err == nil {
		t.Fatal("expected a timeout")
	}
	if el := time.Since(start); el > 3*time.Second {
		t.Fatalf("attempt lasted %v although the timeout is 500ms: junk packets extended it", el)
	}
}
