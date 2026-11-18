// Package testutil holds helpers shared by the integration tests. Everything here uses the
// real production transport (UDP sockets + QuicUpgrade); nothing is mocked.
package testutil

import (
	"context"
	"crypto/ed25519"
	crand "crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"math/rand/v2"
	"net"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"ftu/internal/transport"

	"github.com/quic-go/quic-go"
)

// Peer is one side of a test connection: a real UDP socket plus an Ed25519 identity.
type Peer struct {
	Conn *net.UDPConn
	Priv ed25519.PrivateKey
	Pub  ed25519.PublicKey
}

// Addr returns the loopback address other peers must use to reach this peer.
func (p *Peer) Addr() *net.UDPAddr {
	return &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: p.Conn.LocalAddr().(*net.UDPAddr).Port}
}

// NewPeer opens a socket with the production initializer and generates a fresh identity.
func NewPeer(t testing.TB) *Peer {
	t.Helper()
	conn := transport.InitializeUDPSocket()
	t.Cleanup(func() { conn.Close() })
	pub, priv, err := ed25519.GenerateKey(crand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	return &Peer{Conn: conn, Priv: priv, Pub: pub}
}

// Connect performs the real QUIC upgrade between a and b concurrently. aTo is the address a
// dials (b's address, or a proxy in front of it) and bTo the address b dials.
// Connections are closed on test cleanup.
func Connect(t testing.TB, a, b *Peer, aTo, bTo *net.UDPAddr) (ca, cb *quic.Conn) {
	t.Helper()
	type res struct {
		c   *quic.Conn
		err error
	}
	ra, rb := make(chan res, 1), make(chan res, 1)
	go func() {
		c, err := transport.QuicUpgrade(a.Conn, aTo, &a.Priv, &b.Pub)
		ra <- res{c, err}
	}()
	go func() {
		c, err := transport.QuicUpgrade(b.Conn, bTo, &b.Priv, &a.Pub)
		rb <- res{c, err}
	}()
	x, y := <-ra, <-rb
	if x.err != nil || y.err != nil {
		t.Fatalf("QUIC upgrade failed: a=%v b=%v", x.err, y.err)
	}
	t.Cleanup(func() {
		x.c.CloseWithError(0, "test cleanup")
		y.c.CloseWithError(0, "test cleanup")
	})
	return x.c, y.c
}

// DirectPair returns two connected peers over plain loopback.
func DirectPair(t testing.TB) (ca, cb *quic.Conn) {
	t.Helper()
	a, b := NewPeer(t), NewPeer(t)
	return Connect(t, a, b, b.Addr(), a.Addr())
}

// WriteFile creates a deterministic pseudo-random file (same seed, same bytes) without ever
// holding it in memory, and returns its path and SHA-256 (lowercase hex).
func WriteFile(t testing.TB, dir, name string, size int64, seed uint64) (path, sum string) {
	t.Helper()
	path = filepath.Join(dir, name)
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create %s: %v", path, err)
	}
	defer f.Close()
	h := sha256.New()
	src := io.LimitReader(rand.NewChaCha8([32]byte{byte(seed), byte(seed >> 8), byte(seed >> 16), byte(seed >> 24)}), size)
	if _, err := io.Copy(io.MultiWriter(f, h), src); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path, hex.EncodeToString(h.Sum(nil))
}

// SHA256File hashes a file on disk.
func SHA256File(t testing.TB, path string) string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		t.Fatalf("hash %s: %v", path, err)
	}
	return hex.EncodeToString(h.Sum(nil))
}

// Impairment describes deterministic network faults applied per datagram by Proxy.
type Impairment struct {
	Loss      float64       // probability a datagram is dropped
	Duplicate float64       // probability a datagram is delivered twice
	Reorder   float64       // probability a datagram is held back an extra 2*Delay+5ms
	Delay     time.Duration // base one-way delay
	Jitter    time.Duration // uniform random extra delay in [0, Jitter)
	Seed      uint64        // makes the fault pattern reproducible
}

// Proxy is a UDP middlebox placed between two peers. Each peer dials one of the proxy's two
// sockets and believes it is talking to the other peer. It only forwards datagrams (applying
// Impairment); it provides no discovery or relaying capability the product itself lacks.
type Proxy struct {
	toB, toA *net.UDPConn // toB: the socket peer A dials; toA: the socket peer B dials
	imp      Impairment
	wg       sync.WaitGroup
	mu       sync.Mutex
	timers   map[*time.Timer]struct{}
	closed   bool

	forwarded, dropped, duplicated, reordered atomic.Int64
}

// ProxyStats counts what the proxy actually did to the datagrams. Tests use it to prove the
// impairment really happened (a "survives packet loss" test over a proxy that dropped nothing
// would prove nothing).
type ProxyStats struct{ Forwarded, Dropped, Duplicated, Reordered int64 }

func (p *Proxy) Stats() ProxyStats {
	return ProxyStats{p.forwarded.Load(), p.dropped.Load(), p.duplicated.Load(), p.reordered.Load()}
}

// NewProxy starts forwarding between a and b. Use AddrForA/AddrForB as the dial targets.
func NewProxy(t testing.TB, a, b *Peer, imp Impairment) *Proxy {
	t.Helper()
	listen := func() *net.UDPConn {
		c, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
		if err != nil {
			t.Fatalf("proxy socket: %v", err)
		}
		return c
	}
	p := &Proxy{toB: listen(), toA: listen(), imp: imp, timers: map[*time.Timer]struct{}{}}
	p.wg.Add(2)
	go p.pump(p.toB, p.toA, b.Addr(), imp.Seed)   // A -> proxy.toB ... leaves via toA -> B
	go p.pump(p.toA, p.toB, a.Addr(), imp.Seed+1) // B -> proxy.toA ... leaves via toB -> A
	t.Cleanup(p.Close)
	return p
}

// AddrForA is the address peer A must dial to reach B.
func (p *Proxy) AddrForA() *net.UDPAddr { return p.toB.LocalAddr().(*net.UDPAddr) }

// AddrForB is the address peer B must dial to reach A.
func (p *Proxy) AddrForB() *net.UDPAddr { return p.toA.LocalAddr().(*net.UDPAddr) }

// Close stops forwarding and cancels in-flight delayed datagrams.
func (p *Proxy) Close() {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	p.closed = true
	for tm := range p.timers {
		tm.Stop()
	}
	p.mu.Unlock()
	p.toA.Close()
	p.toB.Close()
	p.wg.Wait()
}

func (p *Proxy) pump(in, out *net.UDPConn, dst *net.UDPAddr, seed uint64) {
	defer p.wg.Done()
	rng := rand.New(rand.NewPCG(seed, 0x6674752d70726f78))
	buf := make([]byte, 65535)
	for {
		n, _, err := in.ReadFromUDP(buf)
		if err != nil {
			return
		}
		if rng.Float64() < p.imp.Loss {
			p.dropped.Add(1)
			continue
		}
		copies := 1
		if rng.Float64() < p.imp.Duplicate {
			copies = 2
			p.duplicated.Add(1)
		}
		for i := 0; i < copies; i++ {
			d := p.imp.Delay
			if p.imp.Jitter > 0 {
				d += time.Duration(rng.Int64N(int64(p.imp.Jitter)))
			}
			if rng.Float64() < p.imp.Reorder {
				d += 2*p.imp.Delay + 5*time.Millisecond
				p.reordered.Add(1)
			}
			p.forwarded.Add(1)
			pkt := append([]byte(nil), buf[:n]...)
			p.send(out, dst, pkt, d)
		}
	}
}

func (p *Proxy) send(out *net.UDPConn, dst *net.UDPAddr, pkt []byte, d time.Duration) {
	if d <= 0 {
		out.WriteToUDP(pkt, dst)
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return
	}
	var tm *time.Timer
	tm = time.AfterFunc(d, func() {
		out.WriteToUDP(pkt, dst)
		p.mu.Lock()
		delete(p.timers, tm)
		p.mu.Unlock()
	})
	p.timers[tm] = struct{}{}
}

// Ctx returns a context cancelled on test cleanup or after d, whichever comes first.
func Ctx(t testing.TB, d time.Duration) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), d)
	t.Cleanup(cancel)
	return ctx
}
