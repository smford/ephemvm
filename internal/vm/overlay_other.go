//go:build !linux

package vm

// cloneOrSparseCopy on non-Linux platforms falls back to sparse copy.
func cloneOrSparseCopy(srcPath, dstPath string) error {
	return sparseCopy(srcPath, dstPath)
}
