package transfer

// NOTE: based on https://transloadit.com/devtips/verify-file-integrity-with-go-and-sha256/
import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
)

// hashFileSHA256 computes the SHA256 hash of a file.
func hashFileSHA256(filePath string) (string, error) {
	// Use os.Open for read-only access.
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	// Ensure the file is closed even if errors occur later.
	// Capture the close error only if no other error occurred.
	defer func() {
		if cerr := file.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("failed to close file: %w", cerr)
		}
	}()

	// Create a new SHA256 hash interface.
	hash := sha256.New()

	// io.Copy efficiently copies data from the file to the hash function.
	// It handles buffering internally.
	if _, err = io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("failed to copy file content to hash: %w", err)
	}

	// Get the resulting hash sum as a byte slice and format it as hex.
	// hash.Sum(nil) appends the hash to a nil slice.
	hashInBytes := hash.Sum(nil)
	hashString := fmt.Sprintf("%x", hashInBytes)

	return hashString, err // err will be nil unless file.Close() failed
}
