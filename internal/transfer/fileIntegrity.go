package transfer

// NOTE: based on https://transloadit.com/devtips/verify-file-integrity-with-go-and-sha256/
import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
)

// HashFileSHA256 computes the SHA256 hash of an open file and rewinds the read offset to 0.
func HashFileSHA256(file *os.File) (string, error) {
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("failed to seek file start: %w", err)
	}

	hash := sha256.New()
	buf := make([]byte, 64*1024)
	if _, err := io.CopyBuffer(hash, file, buf); err != nil {
		return "", fmt.Errorf("failed to copy file content to hash: %w", err)
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("failed to reset file position: %w", err)
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}
