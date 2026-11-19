package transfer

import (
	"fmt"
	"os"
	"path/filepath"
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
