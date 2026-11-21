package transfer

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/quic-go/quic-go"
)

// ErrInsufficientSpace is returned by EnsureFreeSpace when the announced size does not fit.
var ErrInsufficientSpace = errors.New("not enough free disk space")

// ErrIntegrity marks a received file whose size or SHA-256 does not match what was announced.
var ErrIntegrity = errors.New("integrity check failed")

// Timeouts for peer-driven waits. They are variables only so tests can shorten them.
var (
	// dataStreamAcceptTimeout bounds how long the receiver waits for the sender to open the
	// data stream after consent was given.
	dataStreamAcceptTimeout = 30 * time.Second
	// dataStreamStallTimeout bounds how long a single read on the data stream may block, so a
	// peer that stops sending cannot hold the receiver (and its temp file) forever.
	dataStreamStallTimeout = 60 * time.Second
)

const (
	partPrefix = ".ftu-"
	partSuffix = ".part"
	// maxPublishAttempts bounds the "name (N).ext" search when a name already exists.
	maxPublishAttempts = 1000
)

// stallReader arms a fresh read deadline before every read.
type stallReader struct {
	s       *quic.ReceiveStream
	timeout time.Duration
}

func (r *stallReader) Read(p []byte) (int, error) {
	if err := r.s.SetReadDeadline(time.Now().Add(r.timeout)); err != nil {
		return 0, err
	}
	return r.s.Read(p)
}

func validSHA256Hex(h string) bool {
	return len(h) == 64 && strings.Trim(h, "0123456789abcdef") == ""
}

// ReceiveFile reads the file payload from the next unidirectional QUIC stream and stores it in
// dir under a safe name derived from name (see SanitizeFileName).
//
// Safety properties:
//   - the data goes to a private temp file (".ftu-<random>.part", O_EXCL, mode 0600) opened
//     through an os.Root, so nothing can escape dir and a planted symlink is never followed;
//   - nothing exists under the final name until size and SHA-256 are verified;
//   - publication never replaces an existing file: on a name clash the result is "name (1).ext";
//   - every error path removes the temp file; no existing file is ever truncated or deleted;
//   - waiting for the stream and each read are bounded by timeouts, extra bytes after the
//     announced size are an error, and an empty/malformed expected hash fails closed.
//
// It returns the final path and the number of payload bytes written. onProgress is optional.
func ReceiveFile(dir, name string, conn *quic.Conn, expectedSize int64, expectedHash string, onProgress ProgressFunc) (string, int64, error) {
	if expectedSize < 0 {
		return "", 0, fmt.Errorf("invalid expected size %d", expectedSize)
	}
	if !validSHA256Hex(expectedHash) {
		return "", 0, errors.New("missing or malformed expected SHA-256")
	}
	safeName, err := SanitizeFileName(name)
	if err != nil {
		return "", 0, err
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return "", 0, fmt.Errorf("failed to open destination directory: %w", err)
	}
	defer root.Close()

	ctx, cancel := context.WithTimeout(context.Background(), dataStreamAcceptTimeout)
	dataStream, err := conn.AcceptUniStream(ctx)
	cancel()
	if err != nil {
		return "", 0, fmt.Errorf("failed to accept data stream: %w", err)
	}

	tmpName, f, err := createTemp(root)
	if err != nil {
		dataStream.CancelRead(0)
		return "", 0, fmt.Errorf("failed to create temporary file: %w", err)
	}
	fail := func(written int64, format string, args ...any) (string, int64, error) {
		f.Close()
		_ = root.Remove(tmpName)
		dataStream.CancelRead(0)
		return "", written, fmt.Errorf(format, args...)
	}

	log.Printf("[Transfer] Receiving file payload (expected %d bytes)...", expectedSize)
	start := time.Now()
	hasher := sha256.New()
	writer := io.MultiWriter(f, hasher)
	buf := make([]byte, transferBufferSize)

	// Always bound the read to the announced size (including 0) so a misbehaving sender cannot
	// stream unbounded data to disk.
	src := &stallReader{s: dataStream, timeout: dataStreamStallTimeout}
	written, err := io.CopyBuffer(writer, withProgress(io.LimitReader(src, expectedSize), onProgress), buf)
	if err != nil {
		return fail(written, "failed to read stream data: %w", err)
	}
	if written != expectedSize {
		return fail(written, "%w: incomplete file transfer: expected %d bytes, got %d bytes", ErrIntegrity, expectedSize, written)
	}
	// The sender must end the stream right after the announced size.
	var extra [1]byte
	if n, err := io.ReadFull(src, extra[:]); n > 0 {
		return fail(written, "%w: sender sent more data than the announced %d bytes", ErrIntegrity, expectedSize)
	} else if err != io.EOF {
		return fail(written, "data stream did not end cleanly: %v", err)
	}
	if err := f.Sync(); err != nil {
		return fail(written, "failed to flush file to disk: %w", err)
	}
	if err := f.Close(); err != nil {
		_ = root.Remove(tmpName)
		return "", written, fmt.Errorf("failed to close file: %w", err)
	}
	if calculated := hex.EncodeToString(hasher.Sum(nil)); calculated != expectedHash {
		_ = root.Remove(tmpName)
		return "", written, fmt.Errorf("%w: hash mismatch: expected %s, got %s", ErrIntegrity, expectedHash, calculated)
	}

	finalName, err := publish(root, tmpName, safeName)
	if err != nil {
		_ = root.Remove(tmpName)
		return "", written, fmt.Errorf("failed to store received file: %w", err)
	}

	elapsed := time.Since(start).Seconds()
	if elapsed <= 0 {
		elapsed = 0.001
	}
	log.Printf("[Transfer] Successfully received %d bytes in %.2f seconds (%.2f MB/s). Checksum verified!",
		written, elapsed, float64(written)/(1024*1024*elapsed))
	return filepath.Join(dir, finalName), written, nil
}

// createTemp creates a fresh private temp file inside root.
func createTemp(root *os.Root) (string, *os.File, error) {
	for i := 0; i < 16; i++ {
		var rnd [8]byte
		if _, err := rand.Read(rnd[:]); err != nil {
			return "", nil, err
		}
		name := partPrefix + hex.EncodeToString(rnd[:]) + partSuffix
		f, err := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err == nil {
			return name, f, nil
		}
		if !errors.Is(err, fs.ErrExist) {
			return "", nil, err
		}
	}
	return "", nil, errors.New("could not allocate a unique temporary name")
}

// candidateName returns name for i == 0 and "stem (i).ext" afterwards.
func candidateName(name string, i int) string {
	if i == 0 {
		return name
	}
	ext := filepath.Ext(name)
	stem := strings.TrimSuffix(name, ext)
	if stem == "" { // name is only an extension-like string such as ".x"
		stem, ext = name, ""
	}
	return fmt.Sprintf("%s (%d)%s", stem, i, ext)
}

// publish moves the verified temp file to a final name without ever replacing an existing file.
// os.Rename is deliberately not used: it silently replaces the destination. A hard link fails
// with "exists" instead; where links are unsupported it falls back to an O_EXCL copy.
func publish(root *os.Root, tmpName, name string) (string, error) {
	useCopy := false
	for i := 0; i < maxPublishAttempts; i++ {
		candidate := candidateName(name, i)
		if len(candidate) > MaxFileNameBytes {
			return "", fmt.Errorf("%w: no room for a unique suffix", ErrInvalidFileName)
		}
		if !useCopy {
			err := root.Link(tmpName, candidate)
			if err == nil {
				if rmErr := root.Remove(tmpName); rmErr != nil {
					log.Printf("[Transfer] could not remove temporary file: %v", rmErr)
				}
				return candidate, nil
			}
			if errors.Is(err, fs.ErrExist) {
				continue
			}
			useCopy = true // links not supported here (e.g. FAT, some network shares)
		}
		err := copyNoClobber(root, tmpName, candidate)
		if err == nil {
			_ = root.Remove(tmpName)
			return candidate, nil
		}
		if errors.Is(err, fs.ErrExist) {
			continue
		}
		return "", err
	}
	return "", errors.New("too many files with the same name")
}

func copyNoClobber(root *os.Root, from, to string) error {
	in, err := root.Open(from)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := root.OpenFile(to, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if _, err := io.CopyBuffer(out, in, make([]byte, transferBufferSize)); err != nil {
		out.Close()
		_ = root.Remove(to)
		return err
	}
	if err := out.Sync(); err != nil {
		out.Close()
		_ = root.Remove(to)
		return err
	}
	return out.Close()
}

// SweepStaleParts removes temp files left behind by a crash (".ftu-*.part") that are older
// than olderThan. Newer ones may belong to a transfer that is still running.
func SweepStaleParts(dir string, olderThan time.Duration) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return
	}
	defer root.Close()
	for _, e := range entries {
		n := e.Name()
		if !e.Type().IsRegular() || !strings.HasPrefix(n, partPrefix) || !strings.HasSuffix(n, partSuffix) {
			continue
		}
		if info, err := e.Info(); err == nil && time.Since(info.ModTime()) > olderThan {
			_ = root.Remove(n)
		}
	}
}
