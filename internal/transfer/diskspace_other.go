//go:build !unix

package transfer

// EnsureFreeSpace is not implemented on this platform: the write path still fails safely (and
// removes the partial file) if the disk fills up during the transfer.
func EnsureFreeSpace(dir string, size int64) error { return nil }
