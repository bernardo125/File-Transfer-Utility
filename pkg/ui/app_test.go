package ui

import (
	"context"
	"ftu/internal/discovery"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
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
	return r.waitWithin(t, wait, what, match)
}

func (r *recorder) waitWithin(t *testing.T, d time.Duration, what string, match func(ev) bool) ev {
	t.Helper()
	deadline := time.After(d)
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
	p.app.setEventEmitter(p.rec.emit)
	p.app.setDownloadDirectory(p.dir)
	t.Cleanup(func() { p.app.shutdown(context.Background()) })
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

func awaiting(e ev) bool {
	d, ok := e.data.(ConnectionStateDTO)
	return e.name == EventConnectionState && ok && d.State == "awaiting_confirmation"
}

// connectUnconfirmed does what two users do up to (and including) seeing the fingerprint, but
// does NOT confirm it. It returns the fingerprints shown to a and b.
func connectUnconfirmed(t *testing.T) (a, b *peer, fpA, fpB string) {
	t.Helper()
	a, b = newPeer(t), newPeer(t)
	ta, tb := a.token(t), b.token(t)
	if err := a.app.ConnectPeer(tb); err != nil {
		t.Fatal(err)
	}
	if err := b.app.ConnectPeer(ta); err != nil {
		t.Fatal(err)
	}
	fpA = a.rec.waitFor(t, "A awaiting confirmation", awaiting).data.(ConnectionStateDTO).Fingerprint
	fpB = b.rec.waitFor(t, "B awaiting confirmation", awaiting).data.(ConnectionStateDTO).Fingerprint
	return a, b, fpA, fpB
}

// pair connects two fresh Apps the way two users would: exchange tokens, both press Connect,
// compare the fingerprint and both confirm it.
func pair(t *testing.T) (a, b *peer) {
	t.Helper()
	a, b, fpA, fpB := connectUnconfirmed(t)
	if fpA == "" || fpA != fpB {
		t.Fatalf("users would see different fingerprints: %q vs %q", fpA, fpB)
	}
	if err := a.app.ConfirmPeerFingerprint(true); err != nil {
		t.Fatal(err)
	}
	if err := b.app.ConfirmPeerFingerprint(true); err != nil {
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

// answer responds to the most recent offer shown to this peer, like a user clicking the dialog.
func (p *peer) answer(t *testing.T, accept bool) error {
	t.Helper()
	evs := p.rec.all()
	for i := len(evs) - 1; i >= 0; i-- {
		if evs[i].name == EventTransferIncoming {
			return p.app.RespondIncomingTransfer(evs[i].data.(IncomingTransferDTO).OfferID, accept)
		}
	}
	t.Fatal("no incoming offer to answer")
	return nil
}

// sendNow is SendSelectedFile after the user "picked" the file in the native dialog.
func (p *peer) sendNow(path string) error {
	p.app.allowSendPath(path)
	return p.app.SendSelectedFile(path)
}

// send starts SendSelectedFile in the background and returns its result channel.
func (p *peer) send(path string) <-chan error {
	ch := make(chan error, 1)
	p.app.allowSendPath(path) // what the native file dialog does
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
	if err := b.app.RespondIncomingTransfer(offer.OfferID, true); err != nil {
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

// Detects: consent bypass. A "yes" given while no offer is pending must not be remembered and
// silently accept the NEXT file someone sends.
func TestRespondWithoutPendingOfferDoesNotPreApproveNextFile(t *testing.T) {
	a, b := pair(t)
	if err := b.app.RespondIncomingTransfer("00000000000000000000000000000000", true); err == nil {
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
	if err := solo.sendNow(src); err == nil || !strings.Contains(err.Error(), "no active peer connection") {
		t.Fatalf("send without a peer: %v", err)
	}

	a, _ := pair(t)
	for name, path := range map[string]string{"missing file": filepath.Join(t.TempDir(), "nope"), "directory": t.TempDir()} {
		if err := a.sendNow(path); err == nil {
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
	a.rec.waitFor(t, "A awaiting confirmation after cancel", awaiting)
	b.rec.waitFor(t, "B awaiting confirmation", awaiting)
	if err := a.app.ConfirmPeerFingerprint(true); err != nil {
		t.Fatal(err)
	}
	if err := b.app.ConfirmPeerFingerprint(true); err != nil {
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
	// Must be prompt: a peer that only notices through the 30 s idle timeout means Disconnect
	// did not close the QUIC session.
	b.rec.waitWithin(t, 10*time.Second, "peer-lost on B", func(e ev) bool {
		d, ok := e.data.(ConnectionStateDTO)
		return ok && d.State == "error" && d.Detail == "Peer disconnected"
	})
	src, _ := testutil.WriteFile(t, t.TempDir(), "x.bin", 10, 5)
	if err := a.sendNow(src); err == nil {
		t.Fatal("sending after Disconnect must fail")
	}
}

// Detects: a partial file or a false "completed" when the sender's app dies mid-transfer.
func TestSenderDiesMidTransfer(t *testing.T) {
	a, b := pair(t)
	src, _ := testutil.WriteFile(t, t.TempDir(), "huge.bin", 256<<20, 6)

	a.send(src)
	b.rec.waitFor(t, "incoming", named(EventTransferIncoming))
	if err := b.answer(t, true); err != nil {
		t.Fatal(err)
	}
	b.rec.waitFor(t, "data flowing", func(e ev) bool {
		p, ok := e.data.(TransferProgressDTO)
		return ok && p.BytesTransferred > 0 && p.Percentage < 100
	})
	a.app.shutdown(context.Background())

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
	if err := b.answer(t, true); err != nil {
		t.Fatal(err)
	}
	b.rec.waitFor(t, "data flowing", func(e ev) bool {
		p, ok := e.data.(TransferProgressDTO)
		return ok && p.BytesTransferred > 0
	})
	b.app.shutdown(context.Background())

	if err := result(t, sent, "SendSelectedFile"); err == nil {
		t.Fatal("sender reported success although the receiver died")
	}
	for _, e := range a.rec.all() {
		if e.name == EventTransferCompleted {
			t.Fatal("sender UI shows 'delivered' for a transfer that never completed")
		}
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

// Detects: a decline writing data, or the sender UI not learning about it.
func TestDeclinedTransferWritesNothingAndSenderIsTold(t *testing.T) {
	a, b := pair(t)
	src, _ := testutil.WriteFile(t, t.TempDir(), "no.bin", 100_000, 2)

	sent := a.send(src)
	b.rec.waitFor(t, "incoming", named(EventTransferIncoming))
	if err := b.answer(t, false); err != nil {
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

// Detects (F-02): a decision for a different/unknown/already answered offer being applied.
func TestRespondIsBoundToTheShownOfferID(t *testing.T) {
	a, b := pair(t)
	src, _ := testutil.WriteFile(t, t.TempDir(), "one.bin", 50_000, 8)
	sent := a.send(src)
	offer := b.rec.waitFor(t, "offer", named(EventTransferIncoming)).data.(IncomingTransferDTO)
	if offer.OfferID == "" {
		t.Fatal("offer carries no id")
	}
	if err := b.app.RespondIncomingTransfer("not-the-id", true); err == nil {
		t.Fatal("decision with a wrong offer id was accepted")
	}
	if err := b.app.RespondIncomingTransfer(offer.OfferID, true); err != nil {
		t.Fatalf("correct id refused: %v", err)
	}
	if err := b.app.RespondIncomingTransfer(offer.OfferID, true); err == nil {
		t.Fatal("a second decision for the same offer was accepted")
	}
	if err := result(t, sent, "SendSelectedFile"); err != nil {
		t.Fatalf("sender: %v", err)
	}
	if err := b.app.RespondIncomingTransfer(offer.OfferID, true); err == nil {
		t.Fatal("a stale decision after completion was accepted")
	}
}

// Detects (F-02): a second offer sharing (and racing for) the decision of the first one.
func TestSecondOfferWhileOnePendingIsRefused(t *testing.T) {
	a, b := pair(t)
	srcs := t.TempDir()
	one, _ := testutil.WriteFile(t, srcs, "one.bin", 50_000, 9)
	two, _ := testutil.WriteFile(t, srcs, "two.bin", 50_000, 10)

	first := a.send(one)
	offer := b.rec.waitFor(t, "first offer", named(EventTransferIncoming)).data.(IncomingTransferDTO)

	err := result(t, a.send(two), "second SendSelectedFile")
	if err == nil || !strings.Contains(err.Error(), "429") {
		t.Fatalf("second offer should be refused with 429, got %v", err)
	}
	count := 0
	for _, e := range b.rec.all() {
		if e.name == EventTransferIncoming {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("user was shown %d offers, want exactly 1", count)
	}
	if err := b.app.RespondIncomingTransfer(offer.OfferID, true); err != nil {
		t.Fatal(err)
	}
	if err := result(t, first, "first SendSelectedFile"); err != nil {
		t.Fatalf("first transfer: %v", err)
	}
	if got := b.received(t); len(got) != 1 || got[0] != "one.bin" {
		t.Fatalf("received %v, want only one.bin", got)
	}
}

// Detects (F-03): the data channel opening before the users confirmed the fingerprint.
func TestNothingFlowsBeforeTheFingerprintIsConfirmed(t *testing.T) {
	a, b, fpA, fpB := connectUnconfirmed(t)
	if fpA != fpB || fpA == "" {
		t.Fatalf("fingerprints differ: %q %q", fpA, fpB)
	}
	src, _ := testutil.WriteFile(t, t.TempDir(), "early.bin", 1000, 11)
	if err := a.sendNow(src); err == nil || !strings.Contains(err.Error(), "not been confirmed") {
		t.Fatalf("send before confirmation = %v, want a 'not confirmed' error", err)
	}
	for _, p := range []*peer{a, b} {
		for _, e := range p.rec.all() {
			if e.name == EventConnectionSuccess {
				t.Fatal("connection reported as successful before confirmation")
			}
		}
	}
	// Only A confirms: B still refuses to send, and B's side has no receiver loop yet.
	if err := a.app.ConfirmPeerFingerprint(true); err != nil {
		t.Fatal(err)
	}
	if err := b.sendNow(src); err == nil {
		t.Fatal("B sent although B's user never confirmed")
	}
	if got := b.received(t); len(got) != 0 {
		t.Fatalf("files appeared on B: %v", got)
	}
}

// Detects (F-03): a mismatch not tearing the session down, or leaving a usable channel behind.
func TestRefusingTheFingerprintClosesTheSession(t *testing.T) {
	a, b, _, _ := connectUnconfirmed(t)
	if err := a.app.ConfirmPeerFingerprint(false); err != nil {
		t.Fatal(err)
	}
	failed := a.rec.waitFor(t, "failure reported", named(EventConnectionFailed)).data.(ConnectionFailedDTO)
	if !strings.Contains(failed.Reason, "did not match") {
		t.Fatalf("reason = %q", failed.Reason)
	}
	src, _ := testutil.WriteFile(t, t.TempDir(), "x.bin", 100, 12)
	if err := a.sendNow(src); err == nil {
		t.Fatal("send worked after the pairing was refused")
	}
	if err := a.app.ConfirmPeerFingerprint(true); err == nil {
		t.Fatal("a refused pairing could be confirmed afterwards")
	}
	// The other side learns the peer is gone.
	b.rec.waitFor(t, "B told about the lost peer", named(EventConnectionFailed))
}

func TestConfirmWithoutPendingPairingFails(t *testing.T) {
	p := newPeer(t)
	if err := p.app.ConfirmPeerFingerprint(true); err == nil {
		t.Fatal("confirming nothing succeeded")
	}
}

// Detects (F-03): an expired token being used to start a connection.
func TestExpiredTokenIsRejected(t *testing.T) {
	a, b := newPeer(t), newPeer(t)
	old := discovery.TokenValidity
	discovery.TokenValidity = -time.Minute
	expired := b.token(t)
	discovery.TokenValidity = old
	_ = a.token(t)
	if err := a.app.ConnectPeer(expired); err != nil {
		t.Fatal(err)
	}
	f := a.rec.waitFor(t, "failure", named(EventConnectionFailed)).data.(ConnectionFailedDTO)
	if !strings.Contains(f.Reason, "expired") {
		t.Fatalf("reason = %q, want it to mention the expiry", f.Reason)
	}
}

// Detects (F-06): the renderer naming an arbitrary path (e.g. an SSH key) to send or inspect.
func TestOnlyFilesPickedInTheDialogCanBeInspectedOrSent(t *testing.T) {
	a, b := pair(t)
	dir := t.TempDir()
	secret, _ := testutil.WriteFile(t, dir, "id_ed25519", 500, 20)
	picked, _ := testutil.WriteFile(t, dir, "picked.bin", 500, 21)

	if _, err := a.app.InspectFile(secret); err == nil {
		t.Fatal("InspectFile accepted a path nobody picked")
	}
	if err := a.app.SendSelectedFile(secret); err == nil {
		t.Fatal("SendSelectedFile accepted a path nobody picked")
	}
	if got := b.rec.all(); countEvents(got, EventTransferIncoming) != 0 {
		t.Fatal("an offer for an unpicked file reached the peer")
	}
	a.app.allowSendPath(picked)
	// the allowed path may be spelled differently but is the same file
	if _, err := a.app.InspectFile(filepath.Join(dir, ".", "sub", "..", "picked.bin")); err != nil {
		t.Fatalf("picked file refused: %v", err)
	}
	if _, err := a.app.InspectFile("picked.bin"); err == nil {
		t.Fatal("a relative path must not match an absolute selection")
	}
}

func countEvents(evs []ev, name string) int {
	n := 0
	for _, e := range evs {
		if e.name == name {
			n++
		}
	}
	return n
}

// Detects (F-06): OpenFolder used to launch URLs, URI handlers or unrelated directories.
func TestOpenFolderIsConfinedToKnownFolders(t *testing.T) {
	p := newPeer(t) // download dir = p.dir
	inside := filepath.Join(p.dir, "sub")
	if err := os.Mkdir(inside, 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(inside, "a.txt")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	pickedDir := t.TempDir()
	picked, _ := testutil.WriteFile(t, pickedDir, "p.bin", 10, 30)
	p.app.allowSendPath(picked)

	for _, ok := range []string{p.dir, inside, file, picked, pickedDir} {
		if _, err := p.app.openableFolder(ok); err != nil {
			t.Errorf("openableFolder(%q) refused: %v", ok, err)
		}
	}
	bad := []string{"", "https://evil.example/x", "ms-msdt:/id PCWDiagnostic", "file:///etc/passwd", "relative/dir", "-h", "/", "/etc",
		outside, filepath.Join(p.dir, "missing"), filepath.Join(p.dir, "..")}
	for _, path := range bad {
		if got, err := p.app.openableFolder(path); err == nil {
			t.Errorf("openableFolder(%q) = %q, want refusal", path, got)
		}
	}
	if runtime.GOOS != "windows" {
		link := filepath.Join(p.dir, "escape")
		if err := os.Symlink(outside, link); err != nil {
			t.Fatal(err)
		}
		if got, err := p.app.openableFolder(link); err == nil {
			t.Errorf("a symlink out of the download folder was followed: %q", got)
		}
	}
}

// Detects (F-06): methods added to *App silently becoming callable from the renderer. Wails binds
// EVERY exported method; test hooks, lifecycle callbacks and setters must stay unexported.
func TestRendererCallableSurfaceIsTheReviewedList(t *testing.T) {
	want := []string{"CancelConnection", "ConfirmPeerFingerprint", "ConnectPeer", "CopyToClipboard", "Disconnect",
		"GetClipboardText", "GetDownloadDirectory", "GetLocalToken", "GetMyToken", "InspectFile", "OpenFolder",
		"OpenGitHub", "QuitApplication", "RespondIncomingTransfer", "SelectDownloadDirectory", "SelectFileNative",
		"SendSelectedFile"}
	typ := reflect.TypeOf(&App{})
	var got []string
	for i := 0; i < typ.NumMethod(); i++ {
		got = append(got, typ.Method(i).Name)
	}
	sort.Strings(got)
	sort.Strings(want)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("renderer-callable methods changed:\n got  %v\n want %v\nreview each new method for what a hostile renderer could do with it", got, want)
	}
}
