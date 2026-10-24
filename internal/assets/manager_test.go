package assets

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestManagerEnsureAssetsLocal(t *testing.T) {
	tmpDir := t.TempDir()

	customKernel := filepath.Join(tmpDir, "custom-vmlinux")
	customRootfs := filepath.Join(tmpDir, "custom-rootfs.ext4")

	if err := os.WriteFile(customKernel, []byte("fake-kernel-content"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(customRootfs, []byte("fake-rootfs-content"), 0644); err != nil {
		t.Fatal(err)
	}

	mgr := NewManager(filepath.Join(tmpDir, "cache"))
	k, r, err := mgr.EnsureAssets(context.Background(), customKernel, customRootfs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if k != customKernel {
		t.Errorf("expected kernel %s, got %s", customKernel, k)
	}
	if r != customRootfs {
		t.Errorf("expected rootfs %s, got %s", customRootfs, r)
	}

	info, err := os.Stat(customRootfs)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0444 {
		t.Errorf("expected 0444 permissions on base rootfs, got %v", info.Mode().Perm())
	}
}

func TestManagerStatusAndClean(t *testing.T) {
	cacheDir := t.TempDir()
	mgr := NewManager(cacheDir)

	statuses, err := mgr.Status()
	if err != nil {
		t.Fatalf("Status failed: %v", err)
	}
	if len(statuses) != 2 {
		t.Fatalf("expected 2 asset statuses, got %d", len(statuses))
	}
	for _, st := range statuses {
		if st.Exists {
			t.Errorf("expected asset %s to not exist initially", st.Name)
		}
	}

	// Create fake asset and verify status
	fakeKernelPath := filepath.Join(cacheDir, "vmlinux")
	if err := os.WriteFile(fakeKernelPath, []byte("fake"), 0444); err != nil {
		t.Fatal(err)
	}

	statuses, err = mgr.Status()
	if err != nil {
		t.Fatal(err)
	}
	if !statuses[0].Exists {
		t.Errorf("expected %s to exist", statuses[0].Name)
	}

	// Test Clean
	if err := mgr.Clean(); err != nil {
		t.Fatalf("Clean failed: %v", err)
	}
	if _, err := os.Stat(cacheDir); !os.IsNotExist(err) {
		t.Error("expected cacheDir to be removed")
	}
}

func TestManagerDownloadAndVerify(t *testing.T) {
	mockContent := []byte("minimal-mock-image-data-for-testing")
	h := sha256.New()
	h.Write(mockContent)
	expectedHash := hex.EncodeToString(h.Sum(nil))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(mockContent)
	}))
	defer server.Close()

	cacheDir := t.TempDir()
	mgr := NewManager(cacheDir)

	spec := AssetSpec{
		Filename: "test.img",
		URL:      server.URL,
		SHA256:   expectedHash,
	}

	dest, err := mgr.ensureAsset(context.Background(), spec)
	if err != nil {
		t.Fatalf("failed to ensure asset: %v", err)
	}

	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(mockContent) {
		t.Errorf("expected content %s, got %s", string(mockContent), string(data))
	}

	server.Close()
	dest2, err := mgr.ensureAsset(context.Background(), spec)
	if err != nil {
		t.Fatalf("cached asset ensure failed: %v", err)
	}
	if dest2 != dest {
		t.Errorf("expected same path %s, got %s", dest, dest2)
	}
}

func TestManagerDownloadChecksumMismatch(t *testing.T) {
	mockContent := []byte("bad-content")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(mockContent)
	}))
	defer server.Close()

	cacheDir := t.TempDir()
	mgr := NewManager(cacheDir)

	spec := AssetSpec{
		Filename: "corrupted.img",
		URL:      server.URL,
		SHA256:   "0000000000000000000000000000000000000000000000000000000000000000",
	}

	_, err := mgr.ensureAsset(context.Background(), spec)
	if err == nil {
		t.Fatal("expected checksum error, got nil")
	}
}

func TestVerifySHA256(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test.txt")
	content := []byte("ephemvm test content")
	if err := os.WriteFile(tmpFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	h := sha256.New()
	h.Write(content)
	correctHash := hex.EncodeToString(h.Sum(nil))

	if err := VerifySHA256(tmpFile, correctHash); err != nil {
		t.Errorf("expected verification success, got: %v", err)
	}

	if err := VerifySHA256(tmpFile, "badhash"); err == nil {
		t.Error("expected error for bad hash, got nil")
	}
}
