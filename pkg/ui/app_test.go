package ui

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"ftu/internal/testutil"
)

// These tests run two real Apps (the Go backend of the desktop UI) against each other over real
// UDP hole punching and QUIC on loopback. The only thing replaced is the Wails event channel: a
// recorder captures exactly what the frontend would receive. No browser involved, so they are
// fast enough for every commit and can hit error paths the UI cannot reach.

const wait = 45 * time.Second

type ev struct {
	name string
	data any
}

type recorder struct {
	mu     sync.Mutex
	events []ev
	signal chan struct{}
}

func newRecorder() *recorder { return &recorder{signal: make(chan struct{}, 1)} }

func (r *recorder) emit(name string, data ...any) {
	var d any
	if len(data) > 0 {
		d = data[0]
	}
	r.mu.Lock()
	r.events = append(r.events, ev{name, d})
	r.mu.Unlock()
	select {
	case r.signal <- struct{}{}:
	default:
	}
}

func (r *recorder) all() []ev {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]ev(nil), r.events...)
}

// waitFor blocks until an event satisfying match has been emitted (scanning history first), or fails.
func (r *recorder) waitFor(t *testing.T, what string, match func(ev) bool) ev {
	t.Helper()
	deadline := time.After(wait)
	seen := 0
	for {
		r.mu.Lock()
		evs := r.events[seen:]
		seen = len(r.events)
		r.mu.Unlock()
		for _, e := range evs {
			if match(e) {
				return e
			}
		}
		select {
		case <-r.signal:
		case <-deadline:
			t.Fatalf("timed out waiting for %s; events so far: %v", what, names(r.all()))
		}
	}
}

func names(evs []ev) []string {
	out := make([]string, len(evs))
	for i, e := range evs {
		out[i] = e.name
		switch d := e.data.(type) {
		case string:
			out[i] += "=" + d
		case ConnectionStateDTO:
			out[i] += "=" + d.State
		}
	}
	return out
}

func status(s string) func(ev) bool {
	return func(e ev) bool { return e.name == EventTransferStatus && e.data == s }
}
func named(n string) func(ev) bool { return func(e ev) bool { return e.name == n } }

type peer struct {
	app *App
	rec *recorder
	dir string
}

func newPeer(t *testing.T) *peer {
	t.Helper()
	p := &peer{app: NewApp(), rec: newRecorder(), dir: t.TempDir()}
	p.app.SetEventEmitter(p.rec.emit)
	p.app.SetDownloadDirectory(p.dir)
	t.Cleanup(func() { p.app.Shutdown(context.Background()) })
	return p
}

func (p *peer) token(t *testing.T) string {
	t.Helper()
	tok, err := p.app.GetLocalToken()
	if err != nil || tok == "" {
		t.Fatalf("GetLocalToken: %q %v", tok, err)
	}
	return tok
}

// pair connects two fresh Apps the way two users would: exchange tokens, both press Connect.
func pair(t *testing.T) (a, b *peer) {
	t.Helper()
	a, b = newPeer(t), newPeer(t)
	ta, tb := a.token(t), b.token(t)
	if err := a.app.ConnectPeer(tb); err != nil {
		t.Fatal(err)
	}
	if err := b.app.ConnectPeer(ta); err != nil {
		t.Fatal(err)
	}
	a.rec.waitFor(t, "A connected", named(EventConnectionSuccess))
	b.rec.waitFor(t, "B connected", named(EventConnectionSuccess))
	return a, b
}

func (p *peer) received(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir(p.dir)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range entries {
		out = append(out, e.Name())
	}
	return out
}

// send starts SendSelectedFile in the background and returns its result channel.
func (p *peer) send(path string) <-chan error {
	ch := make(chan error, 1)
	go func() { ch <- p.app.SendSelectedFile(path) }()
	return ch
}

func result(t *testing.T, ch <-chan error, what string) error {
	t.Helper()
	select {
	case err := <-ch:
		return err
	case <-time.After(wait):
		t.Fatalf("%s did not return", what)
		return nil
	}
}

// Detects: any break in the UI backend's real flow, and a mismatch between what the frontend is
// told (events) and what really happened (bytes on the receiver's disk).
func TestTransferEndToEndEventsAndBytes(t *testing.T) {
	a, b := pair(t)
	src, sum := testutil.WriteFile(t, t.TempDir(), "report.bin", 3<<20+11, 1)

	sent := a.send(src)
	offer := b.rec.waitFor(t, "incoming offer", named(EventTransferIncoming)).data.(IncomingTransferDTO)
	if offer.FileName != "report.bin" || offer.FileSize != 3<<20+11 || offer.FileHash != sum {
		t.Fatalf("offer shown to the receiving user is wrong: %+v (want report.bin, %d, %s)", offer, 3<<20+11, sum)
	}
	if err := b.app.RespondIncomingTransfer(true); err != nil {
		t.Fatal(err)
	}
	if err := result(t, sent, "SendSelectedFile"); err != nil {
		t.Fatalf("sender: %v", err)
	}

	done := b.rec.waitFor(t, "receiver completed", named(EventTransferCompleted)).data.(TransferCompletedDTO)
	a.rec.waitFor(t, "sender completed", named(EventTransferCompleted))
	want := filepath.Join(b.dir, "report.bin")
	if done.DestPath != want || done.FileSize != 3<<20+11 || !done.HashVerified {
		t.Fatalf("completed event %+v does not match reality (dest %s)", done, want)
	}
	if got := testutil.SHA256File(t, want); got != sum {
		t.Fatalf("file on disk differs from the original: %s != %s", got, sum)
	}

	// The sender's status sequence is what drives the sending screen.
	var seq []string
	for _, e := range a.rec.all() {
		if e.name == EventTransferStatus {
			seq = append(seq, e.data.(string))
		}
	}
	if strings.Join(seq, ",") != "requesting,waiting_approval,transferring,verifying" {
		t.Fatalf("sender status sequence = %v", seq)
	}
	// Progress must be monotonic and finish at exactly 100 %, or the progress bar goes backwards/stalls.
	var last float64 = -1
	for _, e := range b.rec.all() {
		if e.name != EventTransferProgress {
			continue
		}
		p := e.data.(TransferProgressDTO)
		if p.Percentage < last {
			t.Fatalf("receiver progress went backwards: %.1f after %.1f", p.Percentage, last)
		}
		last = p.Percentage
	}
	if last != 100 {
		t.Fatalf("receiver progress ended at %.1f, want 100", last)
	}
}

// Detects: a decline writing data, or the sender UI not learning about it.
func TestDeclinedTransferWritesNothingAndSenderIsTold(t *testing.T) {
	a, b := pair(t)
	src, _ := testutil.WriteFile(t, t.TempDir(), "no.bin", 100_000, 2)

	sent := a.send(src)
	b.rec.waitFor(t, "incoming", named(EventTransferIncoming))
	if err := b.app.RespondIncomingTransfer(false); err != nil {
		t.Fatal(err)
	}
	if err := result(t, sent, "SendSelectedFile"); err == nil || !strings.Contains(err.Error(), "rejected by peer") {
		t.Fatalf("sender error = %v, want 'rejected by peer'", err)
	}
	a.rec.waitFor(t, "sender rejected status", status("rejected"))
	b.rec.waitFor(t, "receiver rejected status", status("rejected"))
	if got := b.received(t); len(got) != 0 {
		t.Fatalf("declined transfer left files: %v", got)
	}
}

// Detects: consent bypass. A "yes" given while no offer is pending must not be remembered and
// silently accept the NEXT file someone sends.
func TestRespondWithoutPendingOfferDoesNotPreApproveNextFile(t *testing.T) {
	a, b := pair(t)
	if err := b.app.RespondIncomingTransfer(true); err == nil {
		// The call was accepted: prove the consequence, so the failure explains itself.
		src, _ := testutil.WriteFile(t, t.TempDir(), "surprise.bin", 50_000, 3)
		sent := a.send(src)
		select {
		case err := <-sent:
			if err == nil {
				t.Fatalf("a stale 'accept' auto-accepted an offer the user never saw (received %v)", b.received(t))
			}
		case <-time.After(wait):
		}
		t.Fatal("RespondIncomingTransfer with no pending offer must return an error")
	}
}

// Detects: nil-pointer / confusing failures when the user tries to send while not connected.
func TestSendWithoutConnectionOrWithBadFileFailsCleanly(t *testing.T) {
	solo := newPeer(t)
	src, _ := testutil.WriteFile(t, t.TempDir(), "x.bin", 10, 4)
	if err := solo.app.SendSelectedFile(src); err == nil || !strings.Contains(err.Error(), "no active peer connection") {
		t.Fatalf("send without a peer: %v", err)
	}

	a, _ := pair(t)
	for name, path := range map[string]string{"missing file": filepath.Join(t.TempDir(), "nope"), "directory": t.TempDir()} {
		if err := a.app.SendSelectedFile(path); err == nil {
			t.Fatalf("%s: accepted", name)
		}
	}
	for _, e := range a.rec.all() {
		if e.name == EventTransferStatus {
			t.Fatalf("a send that failed validation must not move the UI into a transfer state, got %v", names(a.rec.all()))
		}
	}
}

// Detects: bad tokens hanging the UI in "connecting" instead of reporting a failure.
func TestConnectPeerBadTokens(t *testing.T) {
	p := newPeer(t)
	own := p.token(t)

	if err := p.app.ConnectPeer("short"); err == nil {
		t.Fatal("a token of 5 characters must be rejected synchronously")
	}
	for name, tok := range map[string]string{
		"garbage":   strings.Repeat("!", 80),
		"own token": own,
	} {
		t.Run(name, func(t *testing.T) {
			q := newPeer(t)
			q.token(t)
			if name == "own token" {
				tok = q.token(t)
			}
			if err := q.app.ConnectPeer(tok); err != nil {
				t.Fatalf("ConnectPeer returns asynchronously; got %v", err)
			}
			failed := q.rec.waitFor(t, "connection failure", named(EventConnectionFailed)).data.(ConnectionFailedDTO)
			if failed.Reason == "" {
				t.Fatal("failure without a reason shown to the user")
			}
		})
	}
}

// Detects: Cancel leaving the app unable to connect afterwards (stuck socket / stale goroutines).
func TestCancelConnectionThenConnectForReal(t *testing.T) {
	a, b := newPeer(t), newPeer(t)
	ta, tb := a.token(t), b.token(t)

	if err := a.app.ConnectPeer(tb); err != nil { // B is not dialing back
		t.Fatal(err)
	}
	a.rec.waitFor(t, "hole punching started", func(e ev) bool {
		d, ok := e.data.(ConnectionStateDTO)
		return ok && d.State == "hole_punching"
	})
	if err := a.app.CancelConnection(); err != nil {
		t.Fatal(err)
	}
	a.rec.waitFor(t, "cancel reported", named(EventConnectionFailed))

	if err := a.app.ConnectPeer(tb); err != nil {
		t.Fatal(err)
	}
	if err := b.app.ConnectPeer(ta); err != nil {
		t.Fatal(err)
	}
	a.rec.waitFor(t, "A connected after cancel", named(EventConnectionSuccess))
	b.rec.waitFor(t, "B connected", named(EventConnectionSuccess))
}

// Detects: the other UI never learning that the peer left, and the local side keeping a dead session.
func TestDisconnectIsReportedToThePeerAndSessionEnds(t *testing.T) {
	a, b := pair(t)
	if err := a.app.Disconnect(); err != nil {
		t.Fatal(err)
	}
	b.rec.waitFor(t, "peer-lost on B", func(e ev) bool {
		d, ok := e.data.(ConnectionStateDTO)
		return ok && d.State == "error" && d.Detail == "Peer disconnected"
	})
	src, _ := testutil.WriteFile(t, t.TempDir(), "x.bin", 10, 5)
	if err := a.app.SendSelectedFile(src); err == nil {
		t.Fatal("sending after Disconnect must fail")
	}
}

// Detects: a partial file or a false "completed" when the sender's app dies mid-transfer.
func TestSenderDiesMidTransfer(t *testing.T) {
	a, b := pair(t)
	src, _ := testutil.WriteFile(t, t.TempDir(), "huge.bin", 256<<20, 6)

	a.send(src)
	b.rec.waitFor(t, "incoming", named(EventTransferIncoming))
	if err := b.app.RespondIncomingTransfer(true); err != nil {
		t.Fatal(err)
	}
	b.rec.waitFor(t, "data flowing", func(e ev) bool {
		p, ok := e.data.(TransferProgressDTO)
		return ok && p.BytesTransferred > 0 && p.Percentage < 100
	})
	a.app.Shutdown(context.Background())

	errEv := b.rec.waitFor(t, "receive error", named(EventTransferError)).data.(TransferErrorDTO)
	if errEv.Code != "RECEIVE_FAILED" || errEv.Message == "" {
		t.Fatalf("error event = %+v", errEv)
	}
	if got := b.received(t); len(got) != 0 {
		t.Fatalf("partial file left behind: %v", got)
	}
	for _, e := range b.rec.all() {
		if e.name == EventTransferCompleted {
			t.Fatal("a failed transfer was reported as completed")
		}
	}
}

// Detects: the sender's call hanging or reporting success when the receiver's app dies mid-transfer.
func TestReceiverDiesMidTransfer(t *testing.T) {
	a, b := pair(t)
	src, _ := testutil.WriteFile(t, t.TempDir(), "huge.bin", 256<<20, 7)

	sent := a.send(src)
	b.rec.waitFor(t, "incoming", named(EventTransferIncoming))
	if err := b.app.RespondIncomingTransfer(true); err != nil {
		t.Fatal(err)
	}
	b.rec.waitFor(t, "data flowing", func(e ev) bool {
		p, ok := e.data.(TransferProgressDTO)
		return ok && p.BytesTransferred > 0
	})
	b.app.Shutdown(context.Background())

	if err := result(t, sent, "SendSelectedFile"); err == nil {
		t.Fatal("sender reported success although the receiver died")
	}
	for _, e := range a.rec.all() {
		if e.name == EventTransferCompleted {
			t.Fatal("sender UI shows 'delivered' for a transfer that never completed")
		}
	}
}

// Detects: the receiver loop serving only the first file of a session (UI path of the CLI bug).
func TestSeveralFilesInOneSession(t *testing.T) {
	a, b := pair(t)
	for i, name := range []string{"one.bin", "two.bin", "three.bin"} {
		src, sum := testutil.WriteFile(t, t.TempDir(), name, int64(40_000+i*7), uint64(20+i))
		sent := a.send(src)
		b.rec.waitFor(t, "offer "+name, func(e ev) bool {
			d, ok := e.data.(IncomingTransferDTO)
			return ok && d.FileName == name
		})
		if err := b.app.RespondIncomingTransfer(true); err != nil {
			t.Fatal(err)
		}
		if err := result(t, sent, name); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if got := testutil.SHA256File(t, filepath.Join(b.dir, name)); got != sum {
			t.Fatalf("%s corrupted", name)
		}
	}
}

// Detects: the progress stream flooding the frontend (it re-renders per event) on a big file.
func TestProgressEventsAreThrottled(t *testing.T) {
	a, b := pair(t)
	src, _ := testutil.WriteFile(t, t.TempDir(), "big.bin", 96<<20, 8)
	sent := a.send(src)
	b.rec.waitFor(t, "incoming", named(EventTransferIncoming))
	if err := b.app.RespondIncomingTransfer(true); err != nil {
		t.Fatal(err)
	}
	if err := result(t, sent, "send"); err != nil {
		t.Fatal(err)
	}
	b.rec.waitFor(t, "completed", named(EventTransferCompleted))
	n := 0
	for _, e := range b.rec.all() {
		if e.name == EventTransferProgress {
			n++
		}
	}
	// 150 ms cap: a 96 MiB loopback transfer cannot need thousands of events.
	if n > 300 {
		t.Fatalf("%d progress events for one transfer: throttling is broken", n)
	}
}

// Detects: data races on the App's shared state (tokens, sockets) under concurrent UI calls.
func TestConcurrentTokenRequestsAreSafeAndStable(t *testing.T) {
	p := newPeer(t)
	var wg sync.WaitGroup
	tokens := make([]string, 16)
	for i := range tokens {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tokens[i], _ = p.app.GetLocalToken()
		}()
	}
	wg.Wait()
	for _, tk := range tokens {
		if tk == "" || tk != tokens[0] {
			t.Fatalf("concurrent GetLocalToken returned different tokens: %v", tokens)
		}
	}
}
