package transfer

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"ftu/internal/testutil"
)

func BenchmarkHashFileSHA256(b *testing.B) {
	for _, size := range []int64{1 << 20, 64 << 20} {
		b.Run(fmt.Sprintf("%dMiB", size>>20), func(b *testing.B) {
			path, _ := testutil.WriteFile(b, b.TempDir(), "f.bin", size, 1)
			f, err := os.Open(path)
			if err != nil {
				b.Fatal(err)
			}
			defer f.Close()
			b.ReportAllocs()
			b.SetBytes(size)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := HashFileSHA256(f); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkTransferThroughput moves a file over a real loopback QUIC connection and verifies the hash.
// Throughput is machine dependent: compare runs on the same host, never assert on it.
func BenchmarkTransferThroughput(b *testing.B) {
	const size = 32 << 20
	dir := b.TempDir()
	src, sum := testutil.WriteFile(b, dir, "src.bin", size, 9)
	b.ReportAllocs()
	b.SetBytes(size)
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		ca, cb := testutil.DirectPair(b)
		f, err := os.Open(src)
		if err != nil {
			b.Fatal(err)
		}
		dest := filepath.Join(dir, "dest.bin")
		b.StartTimer()
		done := make(chan error, 1)
		go func() { done <- SendFile(f, ca, nil) }()
		if _, err := ReceiveFile(dest, cb, size, sum, nil); err != nil {
			b.Fatal(err)
		}
		if err := <-done; err != nil {
			b.Fatal(err)
		}
		b.StopTimer()
		f.Close()
	}
}

func BenchmarkConcurrentTransfers(b *testing.B) {
	for _, n := range []int{1, 4} {
		b.Run(fmt.Sprintf("%d_pairs", n), func(b *testing.B) {
			const size = 8 << 20
			dir := b.TempDir()
			src, sum := testutil.WriteFile(b, dir, "src.bin", size, 9)
			b.ReportAllocs()
			b.SetBytes(int64(size * n))
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				var wg sync.WaitGroup
				errs := make(chan error, 2*n)
				run := make([]func(), 0, n)
				for j := 0; j < n; j++ {
					ca, cb := testutil.DirectPair(b)
					j := j
					run = append(run, func() {
						defer wg.Done()
						f, err := os.Open(src)
						if err != nil {
							errs <- err
							return
						}
						defer f.Close()
						go func() { errs <- SendFile(f, ca, nil) }()
						_, err = ReceiveFile(filepath.Join(dir, fmt.Sprintf("d%d.bin", j)), cb, size, sum, nil)
						errs <- err
					})
				}
				b.StartTimer()
				wg.Add(n)
				for _, r := range run {
					go r()
				}
				wg.Wait()
				b.StopTimer()
				for k := 0; k < 2*n; k++ {
					if err := <-errs; err != nil {
						b.Fatal(err)
					}
				}
			}
		})
	}
}
