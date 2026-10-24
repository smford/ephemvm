package vm

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestOverlayLifecycleAndIntegrity(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a base image containing a mix of data and sparse zeros
	baseImagePath := filepath.Join(tmpDir, "base_rootfs.ext4")
	baseData := make([]byte, 1024*1024) // 1MB
	copy(baseData[0:], []byte("ephemvm-base-header-data"))
	copy(baseData[500*1024:], []byte("ephemvm-base-middle-data"))
	copy(baseData[1024*1024-32:], []byte("ephemvm-base-trailer-data"))

	if err := os.WriteFile(baseImagePath, baseData, 0444); err != nil {
		t.Fatalf("failed to create base image: %v", err)
	}

	h := sha256.New()
	h.Write(baseData)
	initialHash := hex.EncodeToString(h.Sum(nil))

	// Create overlay
	overlay, err := CreateOverlay(baseImagePath, tmpDir)
	if err != nil {
		t.Fatalf("CreateOverlay failed: %v", err)
	}

	// Verify overlay directory and files exist
	if _, err := os.Stat(overlay.Dir); err != nil {
		t.Fatalf("overlay dir does not exist: %v", err)
	}
	if _, err := os.Stat(overlay.RootfsPath); err != nil {
		t.Fatalf("overlay rootfs does not exist: %v", err)
	}

	// Verify overlay contents match base image initially
	overlayData, err := os.ReadFile(overlay.RootfsPath)
	if err != nil {
		t.Fatalf("failed to read overlay rootfs: %v", err)
	}
	if !bytes.Equal(baseData, overlayData) {
		t.Fatal("overlay rootfs content does not match base image")
	}

	// Perform destructive mutation on the overlay image (simulating guest writes)
	mutatedData := []byte("DESTRUCTIVE_OVERWRITE_MUTATION_TO_SIMULATE_GUEST_WRITES")
	f, err := os.OpenFile(overlay.RootfsPath, os.O_WRONLY, 0600)
	if err != nil {
		t.Fatalf("failed to open overlay for mutation: %v", err)
	}
	if _, err := f.WriteAt(mutatedData, 0); err != nil {
		t.Fatalf("failed to write mutation: %v", err)
	}
	_ = f.Close()

	// Verify that the base image is strictly unchanged!
	currentBaseData, err := os.ReadFile(baseImagePath)
	if err != nil {
		t.Fatalf("failed to read base image: %v", err)
	}
	hAfter := sha256.New()
	hAfter.Write(currentBaseData)
	afterHash := hex.EncodeToString(hAfter.Sum(nil))

	if afterHash != initialHash {
		t.Fatalf("CRITICAL: base image was mutated! before: %s, after: %s", initialHash, afterHash)
	}

	// Verify Cleanup deletes the entire overlay directory
	if err := overlay.Cleanup(); err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}

	if _, err := os.Stat(overlay.Dir); !os.IsNotExist(err) {
		t.Fatalf("expected overlay dir to be deleted, got err: %v", err)
	}

	// Cleanup should be idempotent
	if err := overlay.Cleanup(); err != nil {
		t.Fatalf("second Cleanup call failed: %v", err)
	}
}
