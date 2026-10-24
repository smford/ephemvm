//go:build linux

package vm

import (
	"os"

	"golang.org/x/sys/unix"
)

// cloneOrSparseCopy attempts FICLONE ioctl for instantaneous copy-on-write,
// falling back to sparse block copying if unsupported on the filesystem.
func cloneOrSparseCopy(srcPath, dstPath string) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.OpenFile(dstPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer dst.Close()

	// Attempt FICLONE ioctl (btrfs, xfs with reflink, zfs with block cloning)
	if err := unix.IoctlFileClone(int(dst.Fd()), int(src.Fd())); err == nil {
		return nil
	}

	// FICLONE failed or not supported on this filesystem, fall back to sparse copy
	_ = dst.Close()
	_ = os.Remove(dstPath)

	return sparseCopy(srcPath, dstPath)
}
