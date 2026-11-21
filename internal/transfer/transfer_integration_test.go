package transfer

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"ftu/internal/testutil"

	"github.com/quic-go/quic-go"
)

const guard = 30 * time.Second

type sendResult struct{ err error }
type recvResult struct {
	n   int64
	err error
}

// runTransfer sends srcPath over a real QUIC connection pair and receives it to dest.
// expectSize/expectHash are what the receiver was told (they may deliberately lie).
func runTransfer(t *testing.T, ca, cb *quic.Conn, srcPath, dest string, expectSize int64, expectHash string, onRecv ProgressFunc) (sendResult, recvResult) {
	t.Helper()
	f, err := os.Open(srcPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { f.Close() })

	sc, rc := make(chan sendResult, 1), make(chan recvResult, 1)
	go func() { sc <- sendResult{SendFile(f, ca, nil)} }()
	go func() {
		_, n, err := ReceiveFile(filepath.Dir(dest), filepath.Base(dest), cb, expectSize, expectHash, onRecv)
		rc <- recvResult{n, err}
	}()
	var s sendResult
	var r recvResult
	for i := 0; i < 2; i++ {
		select {
		case s = <-sc:
		case r = <-rc:
		case <-time.After(guard):
			t.Fatal("transfer deadlocked: sender/receiver did not return")
		}
	}
	return s, r
}

// Detects: corruption, truncation or off-by-one at chunk boundaries (buffer is 64 KiB), and the
// empty file case. Success is judged by SHA-256 of the received bytes, not by size or existence.
func TestTransferPreservesContentAtBoundarySizes(t *testing.T) {
	sizes := []int64{0, 1, transferBufferSize, transferBufferSize + 1, 5 << 20}
	for _, size := range sizes {
		t.Run(fmt.Sprintf("%d_bytes", size), func(t *testing.T) {
			ca, cb := testutil.DirectPair(t)
			dir := t.TempDir()
			src, sum := testutil.WriteFile(t, dir, "src.bin", size, uint64(size)+1)
			dest := filepath.Join(dir, "dest.bin")

			s, r := runTransfer(t, ca, cb, src, dest, size, sum, nil)
			if s.err != nil || r.err != nil {
				t.Fatalf("send=%v recv=%v", s.err, r.err)
			}
			if r.n != size {
				t.Fatalf("received %d bytes, want %d", r.n, size)
			}
			if got := testutil.SHA256File(t, dest); got != sum {
				t.Fatalf("content differs: sha256 %s, want %s", got, sum)
			}
		})
	}
}

// Detects: corrupted content being delivered (receiver must reject and leave no file behind).
func TestReceiveRejectsHashMismatchAndRemovesFile(t *testing.T) {
	ca, cb := testutil.DirectPair(t)
	dir := t.TempDir()
	src, sum := testutil.WriteFile(t, dir, "src.bin", 200_000, 7)
	_, otherSum := testutil.WriteFile(t, dir, "other.bin", 200_000, 8)
	if sum == otherSum {
		t.Fatal("test files must differ")
	}
	dest := filepath.Join(dir, "dest.bin")

	_, r := runTransfer(t, ca, cb, src, dest, 200_000, otherSum, nil)
	if r.err == nil {
		t.Fatal("receiver accepted a file whose hash does not match the announced hash")
	}
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Fatalf("corrupt file left on disk (stat err=%v)", err)
	}
}

// Detects: a short transfer being reported as success (sender announces more than it sends).
func TestReceiveRejectsTruncatedPayload(t *testing.T) {
	ca, cb := testutil.DirectPair(t)
	dir := t.TempDir()
	src, sum := testutil.WriteFile(t, dir, "src.bin", 100_000, 3)
	dest := filepath.Join(dir, "dest.bin")

	_, r := runTransfer(t, ca, cb, src, dest, 100_001, sum, nil)
	if r.err == nil {
		t.Fatal("receiver accepted fewer bytes than announced")
	}
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Fatalf("partial file left on disk (stat err=%v)", err)
	}
}

// Detects: a sender streaming more than it announced growing the receiver's disk usage (the
// receiver used to disable its limit when the announced size was 0). The receiver must stop at
// the announced size, treat the surplus as a protocol violation (F-14) and keep no file. Only the
// receiver is asserted: a sender that over-sends is left blocked by QUIC flow control.
func TestReceiveRejectsAndNeverStoresMoreThanAnnouncedSize(t *testing.T) {
	for _, announced := range []int64{0, 1000} {
		t.Run(fmt.Sprintf("announced_%d", announced), func(t *testing.T) {
			ca, cb := testutil.DirectPair(t)
			dir := t.TempDir()
			src, _ := testutil.WriteFile(t, dir, "big.bin", 2<<20, 5)
			_, wantSum := testutil.WriteFile(t, dir, "prefix.bin", announced, 5) // same stream => same prefix
			f, err := os.Open(src)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { f.Close() })

			go SendFile(f, ca, nil) // blocks on flow control once the receiver stops reading
			rc := make(chan recvResult, 1)
			go func() {
				_, n, err := ReceiveFile(dir, "dest.bin", cb, announced, wantSum, nil)
				rc <- recvResult{n, err}
			}()
			select {
			case r := <-rc:
				if !errors.Is(r.err, ErrIntegrity) {
					t.Fatalf("surplus data must be an integrity error, got %v", r.err)
				}
				if r.n > announced {
					t.Fatalf("wrote %d bytes, more than the announced %d", r.n, announced)
				}
			case <-time.After(guard):
				t.Fatal("receiver did not finish")
			}
			if _, err := os.Stat(filepath.Join(dir, "dest.bin")); !os.IsNotExist(err) {
				t.Fatalf("file kept although the sender over-sent (stat err=%v)", err)
			}
			assertNoPartFiles(t, dir)
		})
	}
}

// Detects: hangs or leftover partial files when the connection dies mid-transfer; both sides
// must return an error and the receiver must not keep a partial file.
func TestConnectionClosedMidTransferFailsBothSides(t *testing.T) {
	ca, cb := testutil.DirectPair(t)
	dir := t.TempDir()
	const size = 32 << 20
	src, sum := testutil.WriteFile(t, dir, "src.bin", size, 11)
	dest := filepath.Join(dir, "dest.bin")

	var once sync.Once
	kill := func(done int64) {
		if done >= 1<<20 {
			once.Do(func() { ca.CloseWithError(42, "sender crashed") })
		}
	}
	s, r := runTransfer(t, ca, cb, src, dest, size, sum, kill)
	if r.err == nil {
		t.Fatal("receiver reported success although the sender vanished mid-transfer")
	}
	if s.err == nil {
		t.Fatal("sender reported success although its connection was closed")
	}
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Fatalf("partial file left on disk (stat err=%v)", err)
	}
}

// Detects: shared state between transfers (globals, buffers, file names) corrupting data when
// several independent peer pairs transfer at once. Run with -race.
func TestConcurrentTransfersAreIsolated(t *testing.T) {
	const n = 6
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			ca, cb := testutil.DirectPair(t) // helper only uses t.Cleanup / Fatalf-on-setup
			dir := t.TempDir()
			size := int64(1<<20 + i*131_071)
			src, sum := testutil.WriteFile(t, dir, "src.bin", size, uint64(100+i))
			dest := filepath.Join(dir, "dest.bin")
			f, err := os.Open(src)
			if err != nil {
				errs <- err
				return
			}
			defer f.Close()
			sc := make(chan error, 1)
			go func() { sc <- SendFile(f, ca, nil) }()
			if _, _, err := ReceiveFile(filepath.Dir(dest), filepath.Base(dest), cb, size, sum, nil); err != nil {
				errs <- fmt.Errorf("pair %d recv: %w", i, err)
				return
			}
			if err := <-sc; err != nil {
				errs <- fmt.Errorf("pair %d send: %w", i, err)
				return
			}
			if got := testutil.SHA256File(t, dest); got != sum {
				errs <- fmt.Errorf("pair %d: content corrupted (%s != %s)", i, got, sum)
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
}

// Detects: the streaming path buffering the file in memory (multi-GB transfers would OOM).
// Peak heap growth while moving 96 MiB must stay far below the file size.
func TestReceiveMemoryStaysBounded(t *testing.T) {
	if testing.Short() {
		t.Skip("skipped with -short: moves 96 MiB")
	}
	ca, cb := testutil.DirectPair(t)
	dir := t.TempDir()
	const size = 96 << 20
	src, sum := testutil.WriteFile(t, dir, "src.bin", size, 21)
	dest := filepath.Join(dir, "dest.bin")

	runtime.GC()
	var base runtime.MemStats
	runtime.ReadMemStats(&base)
	var peak uint64
	var next int64 = 4 << 20
	sample := func(done int64) {
		if done < next {
			return
		}
		next += 4 << 20
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		if m.HeapAlloc > peak {
			peak = m.HeapAlloc
		}
	}
	s, r := runTransfer(t, ca, cb, src, dest, size, sum, sample)
	if s.err != nil || r.err != nil {
		t.Fatalf("send=%v recv=%v", s.err, r.err)
	}
	growth := int64(peak) - int64(base.HeapAlloc)
	t.Logf("peak heap growth during 96 MiB transfer: %d KiB", growth>>10)
	if growth > size/3 {
		t.Fatalf("heap grew by %d MiB while streaming %d MiB: file is being buffered", growth>>20, size>>20)
	}
}
