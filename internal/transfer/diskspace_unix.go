//go:build unix

package transfer

import (
	"fmt"
	"syscall"
)

// EnsureFreeSpace fails when the filesystem holding dir cannot fit size more bytes.
func EnsureFreeSpace(dir string, size int64) error {
	var st syscall.Statfs_t
	if err := syscall.Statfs(dir, &st); err != nil {
		return fmt.Errorf("cannot check free space: %w", err)
	}
	free := uint64(st.Bavail) * uint64(st.Bsize)
	if size < 0 || uint64(size) > free {
		return fmt.Errorf("%w: need %d bytes, %d available", ErrInsufficientSpace, size, free)
	}
	return nil
}
