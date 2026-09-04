package protocol

import (
	"bytes"
	"os"
	"testing"
)

func TestBuildAndReceiveMetadata(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "ftu_test_*")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := tmpFile.WriteString("test content for hashing"); err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}

	stats, err := tmpFile.Stat()
	if err != nil {
		t.Fatalf("failed to get stat: %v", err)
	}

	hash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	payload, err := BuildMetaDataSendFile(stats, hash)
	if err != nil {
		t.Fatalf("failed to build metadata: %v", err)
	}

	req, err := ReceiveMetaDataFile(payload)
	if err != nil {
		t.Fatalf("failed to parse metadata: %v", err)
	}

	if req.Method != SEND {
		t.Errorf("expected SEND, got %s", req.Method)
	}
	if req.FileName != stats.Name() {
		t.Errorf("expected %s, got %s", stats.Name(), req.FileName)
	}
	if req.FileSize != stats.Size() {
		t.Errorf("expected %d, got %d", stats.Size(), req.FileSize)
	}
	if req.FileHash != hash {
		t.Errorf("expected %s, got %s", hash, req.FileHash)
	}
}

func TestResponseMetadata(t *testing.T) {
	respBytes, err := BuildMetadataResponse(StatusOK)
	if err != nil {
		t.Fatalf("failed to build response: %v", err)
	}

	code := ReceiveMetadataResponse(respBytes)
	if code != StatusOK {
		t.Errorf("expected %d, got %d", StatusOK, code)
	}

	resp, err := ParseMetadataResponse(respBytes)
	if err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Status != StatusOK {
		t.Errorf("expected %d, got %d", StatusOK, resp.Status)
	}
}

func TestFraming(t *testing.T) {
	buf := new(bytes.Buffer)
	testData := []byte("Hello, QUIC framing!")

	if err := WriteFrame(buf, testData); err != nil {
		t.Fatalf("WriteFrame failed: %v", err)
	}

	readData, err := ReadFrame(buf)
	if err != nil {
		t.Fatalf("ReadFrame failed: %v", err)
	}

	if !bytes.Equal(testData, readData) {
		t.Errorf("expected %s, got %s", string(testData), string(readData))
	}
}
