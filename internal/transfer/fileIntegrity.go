package transfer

// NOTE: based on https://transloadit.com/devtips/verify-file-integrity-with-go-and-sha256/
import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
)

// hashFileSHA256 computes the SHA256 hash of a file.
func hashFileSHA256(file *os.File) (string, error) {

	// Create a new SHA256 hash interface.
	hash := sha256.New()

	// io.Copy efficiently copies data from the file to the hash function.
	// It handles buffering internally.
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("failed to copy file content to hash: %w", err)
	}

	// Get the resulting hash sum as a byte slice and format it as hex.
	// hash.Sum(nil) appends the hash to a nil slice.
	hashInBytes := hash.Sum(nil)
	hashString := fmt.Sprintf("%x", hashInBytes)

	return hashString, nil
}
