package transfer

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/quic-go/quic-go"
)

const transferBufferSize = 64 * 1024 // 64 KB chunk buffer

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
func SendFile(file *os.File, conn *quic.Conn) error {
	log.Println("[Transfer] Sending file payload to peer...")
	dataStream, err := conn.OpenUniStream()
	if err != nil {
		return err
	}
	start := time.Now()

	buf := make([]byte, transferBufferSize)
	written, err := io.CopyBuffer(dataStream, file, buf)
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

// ReceiveFile reads file payload from the QUIC stream, writes it to disk in chunks,
// and validates its SHA256 checksum on-the-fly. If verification fails, the partial/corrupted file is deleted.
func ReceiveFile(destPath string, conn *quic.Conn, expectedSize int64, expectedHash string) (int64, error) {
	newFile, err := os.Create(destPath)
	if err != nil {
		return 0, fmt.Errorf("failed to create destination file: %w", err)
	}

	// Hash on-the-fly while saving chunks to disk
	hasher := sha256.New()
	writer := io.MultiWriter(newFile, hasher)

	dataStream, err := conn.AcceptUniStream(context.Background())

	log.Printf("[Transfer] Receiving file payload (expected %d bytes)...", expectedSize)
	start := time.Now()

	buf := make([]byte, transferBufferSize)
	var src io.Reader = dataStream
	if expectedSize > 0 {
		src = io.LimitReader(dataStream, expectedSize)
	}

	written, err := io.CopyBuffer(writer, src, buf)
	if err != nil {
		newFile.Close()
		_ = os.Remove(destPath)
		return written, fmt.Errorf("failed to read stream data: %w", err)
	}

	if expectedSize > 0 && written != expectedSize {
		newFile.Close()
		_ = os.Remove(destPath)
		return written, fmt.Errorf("incomplete file transfer: expected %d bytes, got %d bytes", expectedSize, written)
	}

	if err := newFile.Sync(); err != nil {
		newFile.Close()
		_ = os.Remove(destPath)
		return written, fmt.Errorf("failed to flush file to disk: %w", err)
	}
	newFile.Close()

	calculatedHash := fmt.Sprintf("%x", hasher.Sum(nil))
	if expectedHash != "" && calculatedHash != expectedHash {
		_ = os.Remove(destPath)
		return written, fmt.Errorf("hash mismatch: expected %s, got %s", expectedHash, calculatedHash)
	}

	elapsed := time.Since(start).Seconds()
	if elapsed <= 0 {
		elapsed = 0.001
	}
	speedMBs := float64(written) / (1024 * 1024 * elapsed)
	log.Printf("[Transfer] Successfully received %d bytes in %.2f seconds (%.2f MB/s). Checksum verified!", written, elapsed, speedMBs)
	return written, nil
}
