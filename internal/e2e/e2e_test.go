// Package e2e runs two independent peers through the product's real connection pipeline:
// token -> GetPeer -> UDP hole punch -> QUIC upgrade -> transfer protocol, on loopback sockets.
// STUN is the only step replaced (there is no internet in tests): the token's "public" address
// is the peer's loopback address, which is exactly what a same-LAN peer advertises. NAT
// traversal across real NATs cannot be exercised here; see docs/TESTING.md.
package e2e

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"ftu/internal/connectivity"
	"ftu/internal/discovery"
	"ftu/internal/protocol"
	"ftu/internal/testutil"

	"github.com/quic-go/quic-go"
)

const guard = 90 * time.Second

// establish runs the full pipeline for both peers concurrently. viaA/viaB are the addresses the
// token of the *other* peer advertises (a proxy address when the link is impaired).
func establish(t testing.TB, a, b *testutil.Peer, viaA, viaB *net.UDPAddr) (ca, cb *quic.Conn) {
	t.Helper()
	tokenA := discovery.BuildToken(viaB, nil, viaB, a.Pub) // what B must dial to reach A
	tokenB := discovery.BuildToken(viaA, nil, viaA, b.Pub) // what A must dial to reach B

	type res struct {
		c   *quic.Conn
		err error
	}
	run := func(me *testutil.Peer, peerToken string) <-chan res {
		ch := make(chan res, 1)
		go func() {
			local, remote, v6, pub, err := discovery.GetPeer(peerToken)
			if err != nil {
				ch <- res{nil, fmt.Errorf("GetPeer: %w", err)}
				return
			}
			punch := <-connectivity.UDPHolePunch(me.Conn, local, remote, v6, &me.Priv, pub)
			if punch.Err != nil {
				ch <- res{nil, fmt.Errorf("hole punch: %w", punch.Err)}
				return
			}
			c, err := quicUpgrade(me, punch.Addr, pub)
			ch <- res{c, err}
		}()
		return ch
	}
	ra, rb := run(a, tokenB), run(b, tokenA)
	var x, y res
	select {
	case x = <-ra:
	case <-time.After(guard):
		t.Fatal("peer A never finished connecting")
	}
	select {
	case y = <-rb:
	case <-time.After(guard):
		t.Fatal("peer B never finished connecting")
	}
	if x.err != nil || y.err != nil {
		t.Fatalf("connect failed: a=%v b=%v", x.err, y.err)
	}
	t.Cleanup(func() {
		x.c.CloseWithError(0, "done")
		y.c.CloseWithError(0, "done")
	})
	return x.c, y.c
}

func send(t *testing.T, c *quic.Conn, path string) <-chan error {
	ch := make(chan error, 1)
	go func() { ch <- protocol.SendFileToPeer(c, path) }()
	return ch
}

func receive(c *quic.Conn) <-chan error {
	ch := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), guard)
		defer cancel()
		st, err := c.AcceptStream(ctx)
		if err != nil {
			ch <- err
			return
		}
		ch <- protocol.HandleIncomingTransfer(st, c, bufio.NewReader(strings.NewReader("Y\n")))
	}()
	return ch
}

func transfer(t *testing.T, from, to *quic.Conn, path string) {
	t.Helper()
	se, re := send(t, from, path), receive(to)
	for _, c := range []struct {
		name string
		ch   <-chan error
	}{{"sender", se}, {"receiver", re}} {
		select {
		case err := <-c.ch:
			if err != nil {
				t.Fatalf("%s: %v", c.name, err)
			}
		case <-time.After(guard):
			t.Fatalf("%s hung", c.name)
		}
	}
}

func assertReceived(t *testing.T, dir, name, wantSum string) {
	t.Helper()
	if got := testutil.SHA256File(t, filepath.Join(dir, name)); got != wantSum {
		t.Fatalf("%s corrupted in transit: sha256 %s, want %s", name, got, wantSum)
	}
}

// Detects: breakage anywhere in the serverless connection pipeline or the transfer, in both
// directions (either peer can send), verified by content hash.
func TestFullPipelineBidirectionalTransfer(t *testing.T) {
	a, b := testutil.NewPeer(t), testutil.NewPeer(t)
	ca, cb := establish(t, a, b, b.Addr(), a.Addr())
	src, dst := t.TempDir(), t.TempDir()
	t.Chdir(dst)

	p1, s1 := testutil.WriteFile(t, src, "a-to-b.bin", 3<<20+5, 1)
	p2, s2 := testutil.WriteFile(t, src, "b-to-a.bin", 700_001, 2)
	transfer(t, ca, cb, p1)
	assertReceived(t, dst, "a-to-b.bin", s1)
	transfer(t, cb, ca, p2)
	assertReceived(t, dst, "b-to-a.bin", s2)
}

// Detects: data loss/corruption or stalls when the network drops, duplicates, reorders and
// delays datagrams. Fault patterns are seeded, so a failure is reproducible.
func TestTransferSurvivesImpairedNetwork(t *testing.T) {
	cases := []struct {
		name string
		imp  testutil.Impairment
	}{
		{"latency_and_jitter", testutil.Impairment{Delay: 30 * time.Millisecond, Jitter: 20 * time.Millisecond, Seed: 1}},
		{"packet_loss_5pct", testutil.Impairment{Loss: 0.05, Seed: 2}},
		{"duplication_20pct", testutil.Impairment{Duplicate: 0.20, Seed: 3}},
		{"reordering_20pct", testutil.Impairment{Reorder: 0.20, Delay: 5 * time.Millisecond, Seed: 4}},
		{"everything_at_once", testutil.Impairment{Loss: 0.03, Duplicate: 0.05, Reorder: 0.10, Delay: 15 * time.Millisecond, Jitter: 10 * time.Millisecond, Seed: 5}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if testing.Short() {
				t.Skip("impaired-network tests skipped with -short")
			}
			a, b := testutil.NewPeer(t), testutil.NewPeer(t)
			px := testutil.NewProxy(t, a, b, tc.imp)
			ca, cb := establish(t, a, b, px.AddrForA(), px.AddrForB())
			src, dst := t.TempDir(), t.TempDir()
			t.Chdir(dst)
			path, sum := testutil.WriteFile(t, src, "data.bin", 512<<10, 42)
			transfer(t, ca, cb, path)
			assertReceived(t, dst, "data.bin", sum)
		})
	}
}

// Detects: independent sessions interfering through shared state when many peers talk at once.
func TestSimultaneousSessionsStayIsolated(t *testing.T) {
	const sessions = 4
	dst := t.TempDir()
	t.Chdir(dst)
	src := t.TempDir()

	type job struct {
		a, b *testutil.Peer
		path string
		name string
		sum  string
	}
	jobs := make([]job, sessions)
	for i := range jobs {
		name := fmt.Sprintf("session-%d.bin", i)
		p, sum := testutil.WriteFile(t, src, name, int64(512*1024+i*9973), uint64(10+i))
		jobs[i] = job{testutil.NewPeer(t), testutil.NewPeer(t), p, name, sum}
	}
	var wg sync.WaitGroup
	errs := make(chan error, sessions)
	for _, j := range jobs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ca, cb := establish(t, j.a, j.b, j.b.Addr(), j.a.Addr())
			se, re := make(chan error, 1), receive(cb)
			go func() { se <- protocol.SendFileToPeer(ca, j.path) }()
			for _, ch := range []<-chan error{se, re} {
				if err := <-ch; err != nil {
					errs <- fmt.Errorf("%s: %w", j.name, err)
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
	for _, j := range jobs {
		assertReceived(t, dst, j.name, j.sum)
	}
}

// Detects: goroutine / socket leaks across connect-transfer-disconnect cycles (a long-running
// app would slowly exhaust resources).
func TestRepeatedConnectDisconnectCyclesDoNotLeakGoroutines(t *testing.T) {
	dst, src := t.TempDir(), t.TempDir()
	t.Chdir(dst)
	path, sum := testutil.WriteFile(t, src, "cycle.bin", 100_000, 77)

	runtime.GC()
	before := runtime.NumGoroutine()
	for i := 0; i < 8; i++ {
		func() {
			a, b := testutil.NewPeer(t), testutil.NewPeer(t)
			ca, cb := establish(t, a, b, b.Addr(), a.Addr())
			_ = os.Remove(filepath.Join(dst, "cycle.bin"))
			transfer(t, ca, cb, path)
			assertReceived(t, dst, "cycle.bin", sum)
			ca.CloseWithError(0, "bye")
			cb.CloseWithError(0, "bye")
			a.Conn.Close()
			b.Conn.Close()
		}()
	}
	// goroutines unwind asynchronously after Close; poll for the settled value
	deadline := time.Now().Add(10 * time.Second)
	for {
		runtime.GC()
		now := runtime.NumGoroutine()
		if now <= before+4 {
			return
		}
		if time.Now().After(deadline) {
			buf := make([]byte, 1<<16)
			buf = buf[:runtime.Stack(buf, true)]
			t.Fatalf("goroutines grew from %d to %d after 8 cycles:\n%s", before, now, buf)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// Detects: an abrupt drop going unnoticed by the other side (it would wait for the idle timeout
// or forever), and a fresh session failing after a previous one died. Recovery in this product
// is a new session (new socket + new token); reusing the same UDP socket for a second hole
// punch after QUIC ran on it is not supported (see docs/TESTING.md).
func TestNewSessionWorksAfterAbruptDrop(t *testing.T) {
	src, dst := t.TempDir(), t.TempDir()
	t.Chdir(dst)
	p1, s1 := testutil.WriteFile(t, src, "first.bin", 200_000, 61)
	p2, s2 := testutil.WriteFile(t, src, "second.bin", 300_000, 62)

	a, b := testutil.NewPeer(t), testutil.NewPeer(t)
	ca, cb := establish(t, a, b, b.Addr(), a.Addr())
	transfer(t, ca, cb, p1)
	assertReceived(t, dst, "first.bin", s1)

	ca.CloseWithError(99, "link lost") // sender vanishes without any goodbye on the data path
	select {
	case <-cb.Context().Done():
	case <-time.After(guard):
		t.Fatal("peer never noticed the dropped connection")
	}
	if _, err := cb.OpenStreamSync(context.Background()); err == nil {
		t.Fatal("stream opened on a dead connection")
	}

	a2, b2 := testutil.NewPeer(t), testutil.NewPeer(t)
	ca2, cb2 := establish(t, a2, b2, b2.Addr(), a2.Addr())
	transfer(t, ca2, cb2, p2)
	assertReceived(t, dst, "second.bin", s2)
}
