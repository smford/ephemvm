package vm

import (
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

// Overlay encapsulates an ephemeral per-run workspace containing a CoW/sparse rootfs copy and socket.
type Overlay struct {
	// ID is the unique identifier for this overlay.
	ID string

	// Dir is the per-run directory (e.g. /tmp/ephemvm-<uuid>/).
	Dir string

	// RootfsPath is the ephemeral read-write overlay image.
	RootfsPath string

	// SocketPath is the path for Firecracker API socket.
	SocketPath string

	cleanedUp bool
	mu        sync.Mutex
}

// CreateOverlay prepares a temporary directory with a CoW/sparse clone of baseRootfsPath.
func CreateOverlay(baseRootfsPath string, parentDir string) (*Overlay, error) {
	if parentDir == "" {
		parentDir = os.TempDir()
	}

	id, err := generateUUID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate overlay uuid: %w", err)
	}

	dir := filepath.Join(parentDir, fmt.Sprintf("ephemvm-%s", id))
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create overlay directory %s: %w", dir, err)
	}

	rootfsPath := filepath.Join(dir, "rootfs.ext4")
	socketPath := filepath.Join(dir, "firecracker.sock")

	// Create ephemeral copy (via reflink/FICLONE if available, otherwise sparse copy)
	if err := cloneOrSparseCopy(baseRootfsPath, rootfsPath); err != nil {
		_ = os.RemoveAll(dir)
		return nil, fmt.Errorf("failed to create ephemeral rootfs overlay: %w", err)
	}

	// Ensure the ephemeral overlay is writable by the microVM
	if err := os.Chmod(rootfsPath, 0600); err != nil {
		_ = os.RemoveAll(dir)
		return nil, fmt.Errorf("failed to set read-write permissions on rootfs overlay: %w", err)
	}

	return &Overlay{
		ID:         id,
		Dir:        dir,
		RootfsPath: rootfsPath,
		SocketPath: socketPath,
	}, nil
}

// Cleanup removes all ephemeral resources associated with the overlay.
func (o *Overlay) Cleanup() error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.cleanedUp {
		return nil
	}
	o.cleanedUp = true

	if o.Dir != "" {
		return os.RemoveAll(o.Dir)
	}
	return nil
}

// sparseCopy copies srcPath to dstPath while preserving sparse holes.
func sparseCopy(srcPath, dstPath string) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	srcInfo, err := src.Stat()
	if err != nil {
		return err
	}

	dst, err := os.OpenFile(dstPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer dst.Close()

	// 64KB block buffer for scanning sparse zero regions
	buf := make([]byte, 64*1024)

	for {
		n, rErr := src.Read(buf)
		if n > 0 {
			if isAllZeroes(buf[:n]) {
				// Sparse hole: advance file offset without writing physical blocks
				if _, err := dst.Seek(int64(n), io.SeekCurrent); err != nil {
					return fmt.Errorf("failed to seek sparse hole: %w", err)
				}
			} else {
				if _, err := dst.Write(buf[:n]); err != nil {
					return fmt.Errorf("failed to write sparse block: %w", err)
				}
			}
		}

		if rErr != nil {
			if rErr == io.EOF {
				break
			}
			return rErr
		}
	}

	// Set exact file length to match original source file
	if err := dst.Truncate(srcInfo.Size()); err != nil {
		return fmt.Errorf("failed to truncate sparse file: %w", err)
	}

	return dst.Sync()
}

func isAllZeroes(b []byte) bool {
	for _, v := range b {
		if v != 0 {
			return false
		}
	}
	return true
}

func generateUUID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	// UUID v4 format
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}
