package transfer

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func sha(b []byte) string { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }

func writeTemp(t *testing.T, content []byte) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "f.bin")
	if err := os.WriteFile(p, content, 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// Detects: a wrong digest (algorithm, encoding, or buffer-boundary bug). Known SHA-256 vectors
// are used so the expectation does not reuse the production code path.
func TestHashFileSHA256KnownVectors(t *testing.T) {
	tests := []struct {
		name    string
		content []byte
		want    string
	}{
		{"empty", nil, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},
		{"abc", []byte("abc"), "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f, err := os.Open(writeTemp(t, tc.content))
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()
			got, err := HashFileSHA256(f)
			if err != nil || got != tc.want {
				t.Fatalf("got %q err=%v, want %q", got, err, tc.want)
			}
		})
	}
}

// Detects: hashing only the unread tail of a file (when the offset is not 0), failing to rewind
// afterwards (the transfer would then send an empty payload), and chunk-boundary errors around
// the 64 KiB hashing buffer.
func TestHashFileSHA256HashesWholeFileAndRewinds(t *testing.T) {
	for _, size := range []int{1, 64*1024 - 1, 64 * 1024, 64*1024 + 1, 200_000} {
		content := bytes.Repeat([]byte{0x5A, 0xA5, 0x01}, size/3+1)[:size]
		f, err := os.Open(writeTemp(t, content))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Seek(int64(size/2), io.SeekStart); err != nil { // leave the offset mid-file
			t.Fatal(err)
		}
		got, err := HashFileSHA256(f)
		if err != nil || got != sha(content) {
			t.Fatalf("size %d: hash=%q err=%v, want %q", size, got, err, sha(content))
		}
		if pos, _ := f.Seek(0, io.SeekCurrent); pos != 0 {
			t.Fatalf("size %d: offset after hashing is %d, want 0", size, pos)
		}
		f.Close()
	}
}

// Detects: SelectFileToSend returning a file that is not ready to stream (offset != 0), metadata
// that disagrees with the file, or a digest that does not match the content.
func TestSelectFileToSendReturnsReadyFileWithCorrectMetadata(t *testing.T) {
	content := bytes.Repeat([]byte("ftu-"), 40_000) // 160 KB, several hash chunks
	path := writeTemp(t, content)

	f, stats, hash, err := SelectFileToSend(path)
	if err != nil {
		t.Fatalf("SelectFileToSend: %v", err)
	}
	defer f.Close()
	if stats.Name() != "f.bin" || stats.Size() != int64(len(content)) {
		t.Fatalf("stats = %s/%d, want f.bin/%d", stats.Name(), stats.Size(), len(content))
	}
	if hash != sha(content) {
		t.Fatalf("hash = %s, want %s", hash, sha(content))
	}
	got, err := io.ReadAll(f)
	if err != nil || !bytes.Equal(got, content) {
		t.Fatalf("file not positioned at 0 / content differs (read %d bytes, err=%v)", len(got), err)
	}
}

// Detects: invalid user input (typo, directory) being accepted or crashing; error must be
// returned and no file handle handed to the caller.
func TestSelectFileToSendRejectsInvalidPaths(t *testing.T) {
	dir := t.TempDir()
	tests := map[string]string{
		"missing file": filepath.Join(dir, "does-not-exist.bin"),
		"directory":    dir,
		"empty path":   "",
	}
	for name, p := range tests {
		t.Run(name, func(t *testing.T) {
			f, stats, hash, err := SelectFileToSend(p)
			if err == nil {
				f.Close()
				t.Fatalf("accepted %q", p)
			}
			if f != nil || stats != nil || hash != "" {
				t.Fatalf("error returned together with values: %v %v %q", f, stats, hash)
			}
		})
	}
}

// Detects: an I/O failure being reported as a valid hash.
func TestHashFileSHA256ClosedFileReturnsError(t *testing.T) {
	f, err := os.Open(writeTemp(t, []byte("data")))
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	if got, err := HashFileSHA256(f); err == nil {
		t.Fatalf("hashing a closed file returned %q with no error", got)
	}
}
