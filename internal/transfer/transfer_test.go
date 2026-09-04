package transfer

import (
	"crypto/sha256"
	"fmt"
	"os"
	"testing"
)

func TestHashFileSHA256(t *testing.T) {
	content := []byte("Hello world file content for SHA256 test")
	tmpFile, err := os.CreateTemp("", "hash_test_*")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := tmpFile.Write(content); err != nil {
		t.Fatalf("failed to write content: %v", err)
	}

	hash, err := HashFileSHA256(tmpFile)
	if err != nil {
		t.Fatalf("HashFileSHA256 failed: %v", err)
	}

	expectedHash := fmt.Sprintf("%x", sha256.Sum256(content))
	if hash != expectedHash {
		t.Errorf("expected hash %s, got %s", expectedHash, hash)
	}
}

func TestSelectFileToSend(t *testing.T) {
	content := []byte("SelectFileToSend test bytes")
	tmpFile, err := os.CreateTemp("", "select_test_*")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := tmpFile.Write(content); err != nil {
		t.Fatalf("failed to write content: %v", err)
	}
	tmpFile.Close()

	file, stats, hash, err := SelectFileToSend(tmpPath)
	if err != nil {
		t.Fatalf("SelectFileToSend failed: %v", err)
	}
	defer file.Close()

	if stats.Size() != int64(len(content)) {
		t.Errorf("expected size %d, got %d", len(content), stats.Size())
	}

	expectedHash := fmt.Sprintf("%x", sha256.Sum256(content))
	if hash != expectedHash {
		t.Errorf("expected hash %s, got %s", expectedHash, hash)
	}

	// Verify file is open and readable
	readBuf := make([]byte, len(content))
	n, err := file.Read(readBuf)
	if err != nil {
		t.Fatalf("failed to read from returned file: %v", err)
	}
	if n != len(content) || string(readBuf) != string(content) {
		t.Errorf("read content mismatch: got %s", string(readBuf))
	}
}
