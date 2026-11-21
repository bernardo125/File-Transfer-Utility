package transfer

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"ftu/internal/testutil"
)

func assertNoPartFiles(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), partPrefix) && strings.HasSuffix(e.Name(), partSuffix) {
			t.Errorf("temporary file %q left behind", e.Name())
		}
	}
}

// receiveInto sends data written by testutil (name, size, seed) and receives it into dstDir under
// peerName. expectHash overrides the announced hash when non-empty.
func receiveInto(t *testing.T, dstDir, peerName string, size int64, seed uint64, expectHash string) (string, recvResult) {
	t.Helper()
	ca, cb := testutil.DirectPair(t)
	srcDir := t.TempDir()
	src, sum := testutil.WriteFile(t, srcDir, "src.bin", size, seed)
	if expectHash != "" {
		sum = expectHash
	}
	f, err := os.Open(src)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { f.Close() })
	go SendFile(f, ca, nil)
	type out struct {
		path string
		r    recvResult
	}
	rc := make(chan out, 1)
	go func() {
		p, n, err := ReceiveFile(dstDir, peerName, cb, size, sum, nil)
		rc <- out{p, recvResult{n, err}}
	}()
	select {
	case o := <-rc:
		return o.path, o.r
	case <-time.After(guard):
		t.Fatal("receiver did not finish")
		return "", recvResult{}
	}
}

// Detects (F-01): a received file replacing a file the user already has.
func TestReceiveNeverOverwritesExistingFile(t *testing.T) {
	dst := t.TempDir()
	existing := filepath.Join(dst, "report.pdf")
	if err := os.WriteFile(existing, []byte("precious"), 0o644); err != nil {
		t.Fatal(err)
	}
	path, r := receiveInto(t, dst, "report.pdf", 4096, 1, "")
	if r.err != nil {
		t.Fatalf("receive: %v", r.err)
	}
	if filepath.Base(path) != "report (1).pdf" {
		t.Fatalf("stored as %q, want %q", filepath.Base(path), "report (1).pdf")
	}
	if b, _ := os.ReadFile(existing); string(b) != "precious" {
		t.Fatalf("existing file was modified: %q", b)
	}
	assertNoPartFiles(t, dst)
}

// Detects (F-01): a FAILED transfer truncating or deleting a file that already had that name.
func TestFailedReceiveKeepsExistingFile(t *testing.T) {
	dst := t.TempDir()
	existing := filepath.Join(dst, "photo.jpg")
	if err := os.WriteFile(existing, []byte("original photo"), 0o644); err != nil {
		t.Fatal(err)
	}
	wrongHash := strings.Repeat("0", 64)
	_, r := receiveInto(t, dst, "photo.jpg", 100_000, 2, wrongHash)
	if r.err == nil {
		t.Fatal("receiver accepted a wrong hash")
	}
	if b, err := os.ReadFile(existing); err != nil || string(b) != "original photo" {
		t.Fatalf("existing file damaged: %q err=%v", b, err)
	}
	assertNoPartFiles(t, dst)
}

// Detects (F-01): a pre-planted symlink at the destination being written through.
func TestReceiveDoesNotFollowSymlinkAtDestination(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation needs privileges on Windows")
	}
	dst, outside := t.TempDir(), t.TempDir()
	target := filepath.Join(outside, "victim.txt")
	if err := os.WriteFile(target, []byte("do not touch"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(dst, "report.pdf")); err != nil {
		t.Fatal(err)
	}
	path, r := receiveInto(t, dst, "report.pdf", 2048, 3, "")
	if r.err != nil {
		t.Fatalf("receive: %v", r.err)
	}
	if b, _ := os.ReadFile(target); string(b) != "do not touch" {
		t.Fatalf("data was written through the symlink: %q", b)
	}
	if filepath.Base(path) != "report (1).pdf" {
		t.Fatalf("stored as %q", filepath.Base(path))
	}
}

// Detects (F-01, F-05): hostile names escaping the download directory or creating dotfiles.
func TestReceiveHostileNamesStayInsideDirectory(t *testing.T) {
	cases := map[string]string{
		"../../evil.txt":    "evil.txt",
		`..\..\evil.txt`:    "evil.txt",
		"/etc/cron.d/x":     "x",
		".bashrc":           "bashrc",
		"CON.txt":           "_CON.txt",
		"report.pdf:stream": "report.pdf_stream",
	}
	for in, want := range cases {
		t.Run(in, func(t *testing.T) {
			parent := t.TempDir()
			dst := filepath.Join(parent, "downloads")
			if err := os.Mkdir(dst, 0o755); err != nil {
				t.Fatal(err)
			}
			path, r := receiveInto(t, dst, in, 1024, 4, "")
			if r.err != nil {
				t.Fatalf("receive: %v", r.err)
			}
			if filepath.Dir(path) != dst || filepath.Base(path) != want {
				t.Fatalf("stored at %q, want %q inside %q", path, want, dst)
			}
			entries, _ := os.ReadDir(parent)
			if len(entries) != 1 {
				t.Fatalf("something was created outside the download dir: %v", entries)
			}
		})
	}
}

// Detects (F-01): received files being readable by other local users, and partial data being
// visible under a final name.
func TestReceivedFileIsPrivate(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits")
	}
	dst := t.TempDir()
	path, r := receiveInto(t, dst, "a.bin", 2048, 5, "")
	if r.err != nil {
		t.Fatal(r.err)
	}
	st, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %v, want 0600", st.Mode().Perm())
	}
}

// Detects (F-14): verification silently skipped when no/garbled hash is supplied.
func TestReceiveFailsClosedWithoutValidHash(t *testing.T) {
	for _, h := range []string{"", "abc", strings.Repeat("G", 64), strings.Repeat("A", 64)} {
		_, cb := testutil.DirectPair(t)
		dst := t.TempDir()
		if _, _, err := ReceiveFile(dst, "x.bin", cb, 10, h, nil); err == nil {
			t.Errorf("hash %q accepted", h)
		}
		assertNoPartFiles(t, dst)
	}
}

// Detects (F-09): a peer that was granted consent but never opens the data stream parking the
// receiver (and a file handle) forever.
func TestReceiveTimesOutWhenNoDataStreamArrives(t *testing.T) {
	old := dataStreamAcceptTimeout
	dataStreamAcceptTimeout = 300 * time.Millisecond
	t.Cleanup(func() { dataStreamAcceptTimeout = old })
	_, cb := testutil.DirectPair(t)
	dst := t.TempDir()
	start := time.Now()
	if _, _, err := ReceiveFile(dst, "x.bin", cb, 10, strings.Repeat("0", 64), nil); err == nil {
		t.Fatal("expected a timeout error")
	}
	if time.Since(start) > 5*time.Second {
		t.Fatalf("took %v", time.Since(start))
	}
	entries, _ := os.ReadDir(dst)
	if len(entries) != 0 {
		t.Fatalf("files created before any data arrived: %v", entries)
	}
}

// Detects (F-09): a sender that stalls mid-stream holding the receiver forever.
func TestReceiveTimesOutOnStalledSender(t *testing.T) {
	old := dataStreamStallTimeout
	dataStreamStallTimeout = 400 * time.Millisecond
	t.Cleanup(func() { dataStreamStallTimeout = old })
	ca, cb := testutil.DirectPair(t)
	s, err := ca.OpenUniStream()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Write([]byte("0123456789")); err != nil {
		t.Fatal(err)
	}
	dst := t.TempDir()
	rc := make(chan error, 1)
	go func() {
		_, _, err := ReceiveFile(dst, "x.bin", cb, 1000, strings.Repeat("0", 64), nil)
		rc <- err
	}()
	select {
	case err := <-rc:
		if err == nil {
			t.Fatal("stalled transfer reported success")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("receiver not released by the stall timeout")
	}
	entries, _ := os.ReadDir(dst)
	if len(entries) != 0 {
		t.Fatalf("partial data left: %v", entries)
	}
}

func TestSweepStalePartsRemovesOnlyOldPartFiles(t *testing.T) {
	dir := t.TempDir()
	oldPart := filepath.Join(dir, ".ftu-aaaa.part")
	newPart := filepath.Join(dir, ".ftu-bbbb.part")
	keep := filepath.Join(dir, "important.txt")
	for _, p := range []string{oldPart, newPart, keep} {
		if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	past := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(oldPart, past, past); err != nil {
		t.Fatal(err)
	}
	SweepStaleParts(dir, 24*time.Hour)
	if _, err := os.Stat(oldPart); !os.IsNotExist(err) {
		t.Error("stale part file not removed")
	}
	for _, p := range []string{newPart, keep} {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("%s was removed: %v", p, err)
		}
	}
}

func TestEnsureFreeSpace(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("not implemented on this platform")
	}
	dir := t.TempDir()
	if err := EnsureFreeSpace(dir, 1); err != nil {
		t.Fatalf("1 byte should fit: %v", err)
	}
	if err := EnsureFreeSpace(dir, 1<<62); err == nil {
		t.Fatal("an absurd size must not fit")
	}
}
