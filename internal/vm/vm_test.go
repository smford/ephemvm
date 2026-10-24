package vm

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/smford/ephemvm/internal/assets"
	"github.com/smford/ephemvm/internal/config"
)

// kvmAndFirecrackerAvailable checks if real microVM integration tests can run on this host.
func kvmAndFirecrackerAvailable() bool {
	if err := CheckKVM(); err != nil {
		return false
	}
	if _, err := FindFirecrackerBinary(""); err != nil {
		return false
	}
	return true
}

func prepareTestAssets(t *testing.T) (string, string) {
	t.Helper()
	mgr := assets.NewManager(filepath.Join(os.TempDir(), "ephemvm-test-cache"))
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	k, r, err := mgr.EnsureAssets(ctx, "", "")
	if err != nil {
		t.Fatalf("failed to prepare test assets: %v", err)
	}
	return k, r
}

func TestIntegrationEcho(t *testing.T) {
	if !kvmAndFirecrackerAvailable() {
		t.Skip("skipping integration test: /dev/kvm or firecracker not available on this host")
	}

	kernelPath, rootfsPath := prepareTestAssets(t)

	cfg := config.NewDefaultConfig()
	cfg.KernelPath = kernelPath
	cfg.RootfsPath = rootfsPath
	cfg.Args = []string{"echo", "hello ephemvm"}
	cfg.Timeout = 15 * time.Second

	var stdout, stderr bytes.Buffer
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	exitCode, err := Run(ctx, cfg, nil, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d (stderr: %s)", exitCode, stderr.String())
	}
	if !strings.Contains(stdout.String(), "hello ephemvm") {
		t.Fatalf("expected stdout to contain 'hello ephemvm', got %q", stdout.String())
	}
}

func TestIntegrationDestructiveRmRf(t *testing.T) {
	if !kvmAndFirecrackerAvailable() {
		t.Skip("skipping integration test: /dev/kvm or firecracker not available on this host")
	}

	kernelPath, rootfsPath := prepareTestAssets(t)

	// Record hash of the base rootfs before destructive execution
	beforeHash, err := assets.ComputeSHA256(rootfsPath)
	if err != nil {
		t.Fatalf("failed to compute initial hash: %v", err)
	}

	cfg := config.NewDefaultConfig()
	cfg.KernelPath = kernelPath
	cfg.RootfsPath = rootfsPath
	// Execute destructive command inside guest
	cfg.Args = []string{"sh", "-c", "rm -rf --no-preserve-root / 2>/dev/null || rm -rf / 2>/dev/null; sync"}
	cfg.Timeout = 20 * time.Second

	var stdout, stderr bytes.Buffer
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	_, err = Run(ctx, cfg, nil, &stdout, &stderr)
	if err != nil {
		t.Logf("destructive command finished with error (expected during root destruction): %v", err)
	}

	// Verify the base image hash is strictly unchanged
	afterHash, err := assets.ComputeSHA256(rootfsPath)
	if err != nil {
		t.Fatalf("failed to compute after hash: %v", err)
	}

	if beforeHash != afterHash {
		t.Fatalf("CRITICAL INTEGRITY FAILURE: base rootfs was modified! before: %s, after: %s", beforeHash, afterHash)
	}
}

func TestIntegrationTimeoutCancellation(t *testing.T) {
	if !kvmAndFirecrackerAvailable() {
		t.Skip("skipping integration test: /dev/kvm or firecracker not available on this host")
	}

	kernelPath, rootfsPath := prepareTestAssets(t)

	cfg := config.NewDefaultConfig()
	cfg.KernelPath = kernelPath
	cfg.RootfsPath = rootfsPath
	cfg.Args = []string{"sleep", "30"}
	cfg.Timeout = 1 * time.Second

	var stdout, stderr bytes.Buffer
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	start := time.Now()
	exitCode, err := Run(ctx, cfg, nil, &stdout, &stderr)
	duration := time.Since(start)

	if duration > 4*time.Second {
		t.Fatalf("execution took too long to cancel: %v", duration)
	}
	if exitCode != 124 && exitCode != 130 && exitCode != 1 {
		t.Logf("exit code after timeout: %d (expected 124, 130, or 1)", exitCode)
	}
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}
