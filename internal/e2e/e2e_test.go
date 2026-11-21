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
		// the impairment this case promises; asserted against what the proxy really did
		wantDropped, wantDuplicated, wantReordered bool
	}{
		{"packet_loss_5pct", testutil.Impairment{Loss: 0.05, Seed: 2}, true, false, false},
		{"reordering_20pct", testutil.Impairment{Reorder: 0.20, Delay: 5 * time.Millisecond, Seed: 4}, false, false, true},
		{"everything_at_once", testutil.Impairment{Loss: 0.03, Duplicate: 0.05, Reorder: 0.10, Delay: 15 * time.Millisecond, Jitter: 10 * time.Millisecond, Seed: 5}, true, true, true},
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

			st := px.Stats()
			t.Logf("proxy: %+v", st)
			if st.Forwarded < 100 {
				t.Fatalf("only %d datagrams crossed the proxy: the transfer did not really go through it", st.Forwarded)
			}
			if tc.wantDropped && st.Dropped == 0 || tc.wantDuplicated && st.Duplicated == 0 || tc.wantReordered && st.Reordered == 0 {
				t.Fatalf("impairment never happened, test proves nothing: %+v", st)
			}
		})
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
