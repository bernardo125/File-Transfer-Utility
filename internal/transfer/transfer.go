package transfer

import (
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/quic-go/quic-go"
)

const transferBufferSize = 64 * 1024 // 64 KB chunk buffer

// ProgressFunc receives the cumulative number of payload bytes moved so far.
// It is invoked synchronously from the copy loop, so implementations must be cheap
// (callers are expected to throttle any expensive work such as UI events).
type ProgressFunc func(bytesDone int64)

// progressReader counts bytes read from the underlying reader and reports them.
// It only observes the stream; the bytes themselves are passed through untouched.
type progressReader struct {
	r    io.Reader
	done int64
	fn   ProgressFunc
}

func (p *progressReader) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)
	if n > 0 {
		p.done += int64(n)
		p.fn(p.done)
	}
	return n, err
}

// withProgress wraps r so that onProgress is notified as data flows. A nil callback returns r unchanged.
func withProgress(r io.Reader, onProgress ProgressFunc) io.Reader {
	if onProgress == nil {
		return r
	}
	return &progressReader{r: r, fn: onProgress}
}

// SelectFileToSend opens a file for read-only access, extracts its metadata and SHA256 checksum,
// and leaves the file pointer positioned at offset 0, ready to be transmitted.
// The caller is responsible for closing the returned file.
func SelectFileToSend(filePath string) (file *os.File, stats os.FileInfo, hash string, err error) {
	file, err = os.Open(filePath)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to open file: %w", err)
	}

	stats, err = file.Stat()
	if err != nil {
		file.Close()
		return nil, nil, "", fmt.Errorf("failed to get file stats: %w", err)
	}

	if stats.IsDir() {
		file.Close()
		return nil, nil, "", fmt.Errorf("path is a directory: %s", filePath)
	}

	hash, err = HashFileSHA256(file)
	if err != nil {
		file.Close()
		return nil, nil, "", fmt.Errorf("failed to calculate hash: %w", err)
	}

	// Reset file pointer to beginning for subsequent transfer
	if _, err = file.Seek(0, io.SeekStart); err != nil {
		file.Close()
		return nil, nil, "", fmt.Errorf("failed to rewind file: %w", err)
	}

	return file, stats, hash, nil
}

// SendFile streams the file content from disk to the QUIC stream in memory-efficient chunks.
// unidirecional data stream (only for data tranfer)
// onProgress is optional and reports bytes handed to the QUIC stream.
func SendFile(file *os.File, conn *quic.Conn, onProgress ProgressFunc) error {
	log.Println("[Transfer] Sending file payload to peer...")
	dataStream, err := conn.OpenUniStream()
	if err != nil {
		return err
	}
	start := time.Now()

	buf := make([]byte, transferBufferSize)
	written, err := io.CopyBuffer(dataStream, withProgress(file, onProgress), buf)
	if err != nil {
		return fmt.Errorf("error during file streaming: %w", err)
	}
	dataStream.Close()

	elapsed := time.Since(start).Seconds()
	if elapsed <= 0 {
		elapsed = 0.001
	}
	speedMBs := float64(written) / (1024 * 1024 * elapsed)
	log.Printf("[Transfer] Finished sending %d bytes in %.2f seconds (%.2f MB/s)", written, elapsed, speedMBs)
	return nil
}
