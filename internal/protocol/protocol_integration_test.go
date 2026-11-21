package protocol

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ftu/internal/testutil"

	"github.com/quic-go/quic-go"
)

const guard = 30 * time.Second

func answers(s string) *bufio.Reader { return bufio.NewReader(strings.NewReader(s)) }

// session creates a connected pair and moves the process into a fresh directory, because the
// receiver saves into the working directory. Tests using it must not run in parallel.
func session(t *testing.T) (sender, receiver *quic.Conn, srcDir, rcvDir string) {
	t.Helper()
	ca, cb := testutil.DirectPair(t)
	srcDir, rcvDir = t.TempDir(), t.TempDir()
	t.Chdir(rcvDir)
	return ca, cb, srcDir, rcvDir
}

type result struct{ err error }

func receiveOnce(t *testing.T, cb *quic.Conn, in string) <-chan result {
	t.Helper()
	ch := make(chan result, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), guard)
		defer cancel()
		st, err := cb.AcceptStream(ctx)
		if err != nil {
			ch <- result{err}
			return
		}
		ch <- result{HandleIncomingTransfer(st, cb, answers(in))}
	}()
	return ch
}

func await(t *testing.T, ch <-chan result, what string) error {
	t.Helper()
	select {
	case r := <-ch:
		return r.err
	case <-time.After(guard):
		t.Fatalf("%s hung", what)
		return nil
	}
}

// Detects: a regression anywhere in request -> accept -> payload -> verify -> final status.
func TestSendAndReceiveFileEndToEnd(t *testing.T) {
	ca, cb, src, dst := session(t)
	path, sum := testutil.WriteFile(t, src, "photo.bin", 1<<20+123, 1)
	rc := receiveOnce(t, cb, "Y\n")

	if err := SendFileToPeer(ca, path); err != nil {
		t.Fatalf("sender: %v", err)
	}
	if err := await(t, rc, "receiver"); err != nil {
		t.Fatalf("receiver: %v", err)
	}
	if got := testutil.SHA256File(t, filepath.Join(dst, "photo.bin")); got != sum {
		t.Fatalf("received content differs: %s != %s", got, sum)
	}
}

// Detects: a rejected transfer still writing a file, or the sender believing it succeeded.
func TestRejectedTransferReportsForbiddenAndWritesNothing(t *testing.T) {
	ca, cb, src, dst := session(t)
	path, _ := testutil.WriteFile(t, src, "secret.bin", 5000, 2)
	rc := receiveOnce(t, cb, "N\n")

	err := SendFileToPeer(ca, path)
	if err == nil || !strings.Contains(err.Error(), "403") {
		t.Fatalf("sender error = %v, want a 403 refusal", err)
	}
	if err := await(t, rc, "receiver"); err != nil {
		t.Fatalf("receiver: %v", err)
	}
	if entries, _ := os.ReadDir(dst); len(entries) != 0 {
		t.Fatalf("rejected transfer left files: %v", entries)
	}
}

// rawRequest speaks the wire protocol by hand so tests can play a misbehaving peer.
func rawRequest(t *testing.T, ca *quic.Conn, payload []byte) *quic.Stream {
	t.Helper()
	st, err := ca.OpenStreamSync(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := sendControlMessage(payload, st); err != nil {
		t.Fatal(err)
	}
	return st
}

// Detects: path traversal through the announced file name (writes outside the target dir).
func TestMaliciousFileNameCannotEscapeDestinationDirectory(t *testing.T) {
	ca, cb, src, dst := session(t)
	path, sum := testutil.WriteFile(t, src, "payload.bin", 4096, 3)
	rc := receiveOnce(t, cb, "Y\n")

	req, _ := json.Marshal(Request{FileName: "../../../escaped.bin", FileSize: 4096, FileHash: sum})
	st := rawRequest(t, ca, req)
	if r, err := ReceiveMetadataResponse(st); err != nil || r.Status != StatusContinue {
		t.Fatalf("response=%v err=%v", r, err)
	}
	f, _ := os.Open(path)
	defer f.Close()
	if err := sendRaw(f, ca); err != nil {
		t.Fatal(err)
	}
	if err := await(t, rc, "receiver"); err != nil {
		t.Fatalf("receiver: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dst, "escaped.bin")); err != nil {
		t.Fatalf("file should be saved inside the destination dir under its base name: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dst, "..", "..", "..", "escaped.bin")); err == nil {
		t.Fatal("file escaped the destination directory")
	}
}

func sendRaw(f *os.File, ca *quic.Conn) error {
	us, err := ca.OpenUniStream()
	if err != nil {
		return err
	}
	buf := make([]byte, 32*1024)
	for {
		n, rerr := f.Read(buf)
		if n > 0 {
			if _, err := us.Write(buf[:n]); err != nil {
				return err
			}
		}
		if rerr != nil {
			return us.Close()
		}
	}
}

// Detects: a sender that announces one hash and sends other bytes being accepted; the sender
// must be told (409) and the receiver must not keep the file.
func TestReceiverRejectsPayloadThatDoesNotMatchAnnouncedHash(t *testing.T) {
	ca, cb, src, dst := session(t)
	path, _ := testutil.WriteFile(t, src, "a.bin", 4096, 4)
	_, lie := testutil.WriteFile(t, src, "b.bin", 4096, 99)
	rc := receiveOnce(t, cb, "Y\n")

	req, _ := json.Marshal(Request{FileName: "a.bin", FileSize: 4096, FileHash: lie})
	st := rawRequest(t, ca, req)
	if r, err := ReceiveMetadataResponse(st); err != nil || r.Status != StatusContinue {
		t.Fatalf("response=%v err=%v", r, err)
	}
	f, _ := os.Open(path)
	defer f.Close()
	if err := sendRaw(f, ca); err != nil {
		t.Fatal(err)
	}
	final, err := ReceiveMetadataResponse(st)
	if err != nil || final.Status != StatusConflict {
		t.Fatalf("final status=%v err=%v, want %d", final, err, StatusConflict)
	}
	if err := await(t, rc, "receiver"); err == nil {
		t.Fatal("receiver reported success for a corrupted transfer")
	}
	if _, err := os.Stat(filepath.Join(dst, "a.bin")); !os.IsNotExist(err) {
		t.Fatalf("corrupted file kept on disk (err=%v)", err)
	}
}

// Detects: the receiver crashing/hanging or acting on invalid metadata; it must answer 400.
func TestReceiverAnswersBadRequestToInvalidMetadata(t *testing.T) {
	cases := map[string][]byte{
		"not json":      []byte("<html>"),
		"negative size": []byte(`{"FileName":"x","FileSize":-5,"FileHash":"` + validHash + `"}`),
		"missing hash":  []byte(`{"FileName":"x","FileSize":5}`),
	}
	for name, payload := range cases {
		t.Run(name, func(t *testing.T) {
			ca, cb, _, dst := session(t)
			rc := receiveOnce(t, cb, "Y\n")
			st := rawRequest(t, ca, payload)
			r, err := ReceiveMetadataResponse(st)
			if err != nil || r.Status != StatusBadRequest {
				t.Fatalf("response=%v err=%v, want %d", r, err, StatusBadRequest)
			}
			if err := await(t, rc, "receiver"); err == nil {
				t.Fatal("receiver returned success for invalid metadata")
			}
			if entries, _ := os.ReadDir(dst); len(entries) != 0 {
				t.Fatalf("files created: %v", entries)
			}
		})
	}
}

// Detects: the sender panicking (nil response) or hanging when the receiver disappears
// instead of answering.
func TestSenderFailsCleanlyWhenReceiverConnectionDies(t *testing.T) {
	ca, cb, src, _ := session(t)
	path, _ := testutil.WriteFile(t, src, "a.bin", 4096, 6)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), guard)
		defer cancel()
		if _, err := cb.AcceptStream(ctx); err == nil {
			cb.CloseWithError(7, "receiver crashed")
		}
	}()
	done := make(chan error, 1)
	go func() { done <- SendFileToPeer(ca, path) }()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("sender reported success without a receiver answer")
		}
	case <-time.After(guard):
		t.Fatal("sender hung after the receiver connection died")
	}
}

// Detects: the "wait for more files" session feature failing after the first file. The sender
// opens a new control stream per file; the receiver loop must follow it.
func TestReceiverLoopHandlesSeveralFilesInOneSession(t *testing.T) {
	ca, cb, src, dst := session(t)
	p1, s1 := testutil.WriteFile(t, src, "one.bin", 70_000, 31)
	p2, s2 := testutil.WriteFile(t, src, "two.bin", 5, 32)

	loop := make(chan error, 1)
	// accept #1, continue=Y, accept #2, continue=N
	go func() { loop <- ReceiverLoop(cb, answers("Y\nY\nY\nN\n")) }()

	for _, p := range []string{p1, p2} {
		errc := make(chan error, 1)
		go func(p string) { errc <- SendFileToPeer(ca, p) }(p)
		select {
		case err := <-errc:
			if err != nil {
				t.Fatalf("send %s: %v", filepath.Base(p), err)
			}
		case <-time.After(guard):
			t.Fatalf("sending %s hung", filepath.Base(p))
		}
	}
	select {
	case err := <-loop:
		if err != nil {
			t.Fatalf("receiver loop: %v", err)
		}
	case <-time.After(guard):
		t.Fatal("receiver loop did not finish")
	}
	for name, sum := range map[string]string{"one.bin": s1, "two.bin": s2} {
		if got := testutil.SHA256File(t, filepath.Join(dst, name)); got != sum {
			t.Errorf("%s corrupted: %s != %s", name, got, sum)
		}
	}
}

// Detects: the sender telling the user "sent successfully" when the receiver reports that the
// file arrived corrupted or could not be written. The receiver here is a scripted peer that
// accepts the transfer, drains the payload and then answers with the given final status.
func TestSenderReportsFailureWhenReceiverRejectsFinalIntegrityCheck(t *testing.T) {
	for _, final := range []StatusCode{StatusConflict, StatusInternalError, StatusBadRequest} {
		t.Run(statusName(final), func(t *testing.T) {
			ca, cb, src, _ := session(t)
			path, _ := testutil.WriteFile(t, src, "a.bin", 50_000, 8)
			peerDone := make(chan error, 1)
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), guard)
				defer cancel()
				st, err := cb.AcceptStream(ctx)
				if err != nil {
					peerDone <- err
					return
				}
				if _, err := ReceiveMetaDataRequest(st); err != nil {
					peerDone <- err
					return
				}
				if err := SendCodeResponse(StatusContinue, st); err != nil {
					peerDone <- err
					return
				}
				us, err := cb.AcceptUniStream(ctx)
				if err != nil {
					peerDone <- err
					return
				}
				_, _ = io.Copy(io.Discard, us)
				peerDone <- SendCodeResponse(final, st)
			}()

			errc := make(chan error, 1)
			go func() { errc <- SendFileToPeer(ca, path) }()
			select {
			case err := <-errc:
				if err == nil {
					t.Fatalf("sender reported success although the receiver answered status %d", final)
				}
				if !strings.Contains(err.Error(), fmt.Sprint(int(final))) {
					t.Fatalf("error %q does not tell the user the receiver's status %d", err, final)
				}
			case <-time.After(guard):
				t.Fatal("sender hung waiting for the final status")
			}
			select {
			case err := <-peerDone:
				if err != nil {
					t.Fatalf("scripted receiver: %v", err)
				}
			case <-time.After(guard):
				t.Fatal("scripted receiver hung")
			}
		})
	}
}

func statusName(c StatusCode) string { return fmt.Sprintf("status_%d", int(c)) }

// Detects: peer-announced 4 GiB frame length reaching make([]byte, n) on a live connection.
func TestReceiverSurvivesOversizedFrameAnnouncement(t *testing.T) {
	ca, cb, _, _ := session(t)
	rc := receiveOnce(t, cb, "Y\n")
	st, err := ca.OpenStreamSync(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.Write([]byte{0xFF, 0xFF, 0xFF, 0xFF, '{'}); err != nil {
		t.Fatal(err)
	}
	if err := await(t, rc, "receiver"); err == nil {
		t.Fatal("receiver accepted an oversized frame")
	}
}

// Detects: the receiver blocking forever in the data phase when the sender drops after
// the accept (the process would need to be killed).
func TestReceiverFailsWhenSenderDisappearsAfterAccept(t *testing.T) {
	ca, cb, _, dst := session(t)
	rc := receiveOnce(t, cb, "Y\n")
	req, _ := json.Marshal(Request{FileName: "a.bin", FileSize: 10, FileHash: validHash})
	st := rawRequest(t, ca, req)
	if r, err := ReceiveMetadataResponse(st); err != nil || r.Status != StatusContinue {
		t.Fatalf("response=%v err=%v", r, err)
	}
	ca.CloseWithError(1, "gone")
	if err := await(t, rc, "receiver"); err == nil {
		t.Fatal("receiver reported success without payload")
	}
	if entries, _ := os.ReadDir(dst); len(entries) != 0 {
		t.Fatalf("files left behind: %v", entries)
	}
}
