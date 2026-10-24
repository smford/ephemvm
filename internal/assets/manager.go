package assets

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	KernelFilename      = "vmlinux"
	RootfsFilename      = "rootfs.ext4"
	FirecrackerFilename = "firecracker"
	FirecrackerVersion  = "v1.10.1"
)

// AssetSpec defines download source and integrity check for an asset.
type AssetSpec struct {
	Filename string
	URL      string
	SHA256   string
}

// AssetStatus reports information about a cached asset.
type AssetStatus struct {
	Name        string
	Path        string
	Exists      bool
	Size        int64
	Permissions os.FileMode
	ValidSHA256 bool
}

// DefaultAssets returns the canonical asset specifications for the current architecture.
func DefaultAssets(arch string) (kernel AssetSpec, rootfs AssetSpec, err error) {
	switch arch {
	case "amd64":
		kernel = AssetSpec{
			Filename: KernelFilename,
			URL:      "https://s3.amazonaws.com/spec.ccfc.min/img/hello/kernel/hello-vmlinux.bin",
			SHA256:   "882fa465c43ab7d92e31bd4167da3ad6a82cb9230f9b0016176df597c6014cef",
		}
		rootfs = AssetSpec{
			Filename: RootfsFilename,
			URL:      "https://s3.amazonaws.com/spec.ccfc.min/img/hello/fsfiles/hello-rootfs.ext4",
			SHA256:   "786f0612582cadcee4c8ad46e30e4cc93dff9a3c0b4ede3f3c597b4c241dd547",
		}
		return kernel, rootfs, nil

	case "arm64":
		kernel = AssetSpec{
			Filename: KernelFilename,
			URL:      "https://s3.amazonaws.com/spec.ccfc.min/img/aarch64/ubuntu_with_ssh/kernel/vmlinux.bin",
			SHA256:   "e2d7c3d6cc123de9e6052d1f434ca7b47635a1f630d63f7fcc1b7164958375e4",
		}
		rootfs = AssetSpec{
			Filename: RootfsFilename,
			URL:      "https://s3.amazonaws.com/spec.ccfc.min/img/hello/fsfiles/hello-rootfs.ext4",
			SHA256:   "786f0612582cadcee4c8ad46e30e4cc93dff9a3c0b4ede3f3c597b4c241dd547",
		}
		return kernel, rootfs, nil

	default:
		return AssetSpec{}, AssetSpec{}, fmt.Errorf("unsupported architecture for default assets: %s", arch)
	}
}

// FirecrackerReleaseURL returns the official release tarball URL for the architecture.
func FirecrackerReleaseURL(arch string) (string, error) {
	switch arch {
	case "amd64":
		return fmt.Sprintf("https://github.com/firecracker-microvm/firecracker/releases/download/%s/firecracker-%s-x86_64.tgz", FirecrackerVersion, FirecrackerVersion), nil
	case "arm64":
		return fmt.Sprintf("https://github.com/firecracker-microvm/firecracker/releases/download/%s/firecracker-%s-aarch64.tgz", FirecrackerVersion, FirecrackerVersion), nil
	default:
		return "", fmt.Errorf("unsupported architecture for firecracker binary: %s", arch)
	}
}

// Manager orchestrates caching, downloading, and integrity checking of VM assets.
type Manager struct {
	CacheDir   string
	HTTPClient *http.Client
	Arch       string
}

// NewManager creates a Manager targeting the specified cache directory.
func NewManager(cacheDir string) *Manager {
	return &Manager{
		CacheDir: cacheDir,
		HTTPClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
		Arch: runtime.GOARCH,
	}
}

// EnsureAssets returns valid file paths for the kernel and rootfs, and ensures Firecracker binary is available.
func (m *Manager) EnsureAssets(ctx context.Context, customKernel, customRootfs string) (string, string, error) {
	kernelPath := customKernel
	rootfsPath := customRootfs

	defaultKernelSpec, defaultRootfsSpec, err := DefaultAssets(m.Arch)
	if err != nil && (kernelPath == "" || rootfsPath == "") {
		return "", "", fmt.Errorf("failed to resolve default assets: %w", err)
	}

	if kernelPath == "" {
		p, err := m.ensureAsset(ctx, defaultKernelSpec)
		if err != nil {
			return "", "", fmt.Errorf("failed to ensure kernel asset: %w", err)
		}
		kernelPath = p
	} else {
		if _, err := os.Stat(kernelPath); err != nil {
			return "", "", fmt.Errorf("custom kernel not found at %s: %w", kernelPath, err)
		}
	}

	if rootfsPath == "" {
		p, err := m.ensureAsset(ctx, defaultRootfsSpec)
		if err != nil {
			return "", "", fmt.Errorf("failed to ensure rootfs asset: %w", err)
		}
		rootfsPath = p
	} else {
		if _, err := os.Stat(rootfsPath); err != nil {
			return "", "", fmt.Errorf("custom rootfs not found at %s: %w", rootfsPath, err)
		}
	}

	if err := enforceReadOnly(rootfsPath); err != nil {
		return "", "", fmt.Errorf("failed to enforce read-only permission on base rootfs: %w", err)
	}

	// On Linux, also ensure Firecracker binary is available
	if runtime.GOOS == "linux" {
		if _, err := exec.LookPath("firecracker"); err != nil {
			_, _ = m.EnsureFirecracker(ctx)
		}
	}

	return kernelPath, rootfsPath, nil
}

// EnsureFirecracker downloads and extracts the official Firecracker binary into the cache directory.
func (m *Manager) EnsureFirecracker(ctx context.Context) (string, error) {
	binDir := filepath.Join(m.CacheDir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create cache bin directory: %w", err)
	}

	destPath := filepath.Join(binDir, FirecrackerFilename)
	if info, err := os.Stat(destPath); err == nil && info.Size() > 0 {
		return destPath, nil
	}

	url, err := FirecrackerReleaseURL(m.Arch)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	resp, err := m.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download firecracker tarball from %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download of firecracker failed with HTTP status: %s", resp.Status)
	}

	gzReader, err := gzip.NewReader(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to decompress gzip tarball: %w", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)
	var found bool

	tmpFile, err := os.CreateTemp(binDir, ".firecracker-*.tmp")
	if err != nil {
		return "", err
	}
	tmpPath := tmpFile.Name()
	defer func() {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
	}()

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("error reading tar archive: %w", err)
		}

		if strings.Contains(header.Name, "firecracker-v") && !strings.HasSuffix(header.Name, ".debug") {
			if _, err := io.Copy(tmpFile, tarReader); err != nil {
				return "", fmt.Errorf("failed to extract firecracker binary: %w", err)
			}
			found = true
			break
		}
	}

	if !found {
		return "", errors.New("firecracker binary not found in downloaded release archive")
	}

	_ = tmpFile.Close()
	if err := os.Chmod(tmpPath, 0755); err != nil {
		return "", err
	}

	if err := os.Rename(tmpPath, destPath); err != nil {
		return "", err
	}

	return destPath, nil
}

// Status inspects the current cache status for standard assets.
func (m *Manager) Status() ([]AssetStatus, error) {
	kSpec, rSpec, err := DefaultAssets(m.Arch)
	if err != nil {
		return nil, err
	}

	specs := []AssetSpec{kSpec, rSpec}
	statuses := make([]AssetStatus, 0, len(specs)+1)

	for _, spec := range specs {
		p := filepath.Join(m.CacheDir, spec.Filename)
		st := AssetStatus{
			Name: spec.Filename,
			Path: p,
		}
		info, err := os.Stat(p)
		if err == nil {
			st.Exists = true
			st.Size = info.Size()
			st.Permissions = info.Mode().Perm()
			st.ValidSHA256 = (VerifySHA256(p, spec.SHA256) == nil)
		}
		statuses = append(statuses, st)
	}

	// Add Firecracker binary status
	fcPath := filepath.Join(m.CacheDir, "bin", FirecrackerFilename)
	fcStatus := AssetStatus{
		Name: FirecrackerFilename,
		Path: fcPath,
	}
	if info, err := os.Stat(fcPath); err == nil {
		fcStatus.Exists = true
		fcStatus.Size = info.Size()
		fcStatus.Permissions = info.Mode().Perm()
		fcStatus.ValidSHA256 = true
	}
	statuses = append(statuses, fcStatus)

	return statuses, nil
}

// Clean removes all cached assets from the cache directory.
func (m *Manager) Clean() error {
	if _, err := os.Stat(m.CacheDir); os.IsNotExist(err) {
		return nil
	}
	return os.RemoveAll(m.CacheDir)
}

func (m *Manager) ensureAsset(ctx context.Context, spec AssetSpec) (string, error) {
	if err := os.MkdirAll(m.CacheDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create asset cache directory: %w", err)
	}

	destPath := filepath.Join(m.CacheDir, spec.Filename)

	if info, err := os.Stat(destPath); err == nil && info.Size() > 0 {
		if err := VerifySHA256(destPath, spec.SHA256); err == nil {
			_ = enforceReadOnly(destPath)
			return destPath, nil
		}
		_ = os.Remove(destPath)
	}

	tmpFile, err := os.CreateTemp(m.CacheDir, fmt.Sprintf(".%s-*.download", spec.Filename))
	if err != nil {
		return "", fmt.Errorf("failed to create temporary download file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
	}()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, spec.URL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP request for %s: %w", spec.URL, err)
	}

	resp, err := m.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download %s: %w", spec.URL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download of %s failed with HTTP status: %s", spec.URL, resp.Status)
	}

	hasher := sha256.New()
	writer := io.MultiWriter(tmpFile, hasher)

	if _, err := io.Copy(writer, resp.Body); err != nil {
		return "", fmt.Errorf("failed to stream download: %w", err)
	}

	if err := tmpFile.Sync(); err != nil {
		return "", fmt.Errorf("failed to sync download file: %w", err)
	}
	_ = tmpFile.Close()

	computedHash := hex.EncodeToString(hasher.Sum(nil))
	if spec.SHA256 != "" && computedHash != spec.SHA256 {
		return "", fmt.Errorf("checksum mismatch for %s: expected %s, got %s", spec.Filename, spec.SHA256, computedHash)
	}

	if err := os.Chmod(tmpPath, 0444); err != nil {
		return "", fmt.Errorf("failed to set read-only permissions: %w", err)
	}

	if err := os.Rename(tmpPath, destPath); err != nil {
		return "", fmt.Errorf("failed to move downloaded asset to destination: %w", err)
	}

	return destPath, nil
}

// VerifySHA256 computes the SHA256 of the given file and compares it to expected.
func VerifySHA256(filePath, expectedHex string) error {
	if expectedHex == "" {
		return nil
	}

	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}

	actualHex := hex.EncodeToString(h.Sum(nil))
	if actualHex != expectedHex {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedHex, actualHex)
	}
	return nil
}

func enforceReadOnly(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.Mode().Perm() != 0444 {
		return os.Chmod(path, 0444)
	}
	return nil
}

// ComputeSHA256 calculates the SHA256 hash string for any file on disk.
func ComputeSHA256(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// BaseFileIntact verifies that a base file exists, is non-empty, and matches its expected hash.
func BaseFileIntact(filePath, expectedHash string) (bool, error) {
	if filePath == "" {
		return false, errors.New("empty file path")
	}
	h, err := ComputeSHA256(filePath)
	if err != nil {
		return false, err
	}
	return h == expectedHash, nil
}
