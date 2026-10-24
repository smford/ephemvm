//go:build linux

package vm

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"syscall"
	"time"

	"github.com/firecracker-microvm/firecracker-go-sdk"
	models "github.com/firecracker-microvm/firecracker-go-sdk/client/models"
	"github.com/sirupsen/logrus"
	"github.com/smford/ephemvm/internal/config"
)

// CheckKVM verifies that KVM is accessible on the host.
func CheckKVM() error {
	f, err := os.OpenFile("/dev/kvm", os.O_RDWR, 0)
	if err != nil {
		if os.IsNotExist(err) {
			return errors.New("/dev/kvm does not exist: hardware virtualization is not enabled or supported")
		}
		if os.IsPermission(err) {
			return errors.New("cannot access /dev/kvm: permission denied (ensure current user has access to /dev/kvm or is in the kvm group)")
		}
		return fmt.Errorf("error accessing /dev/kvm: %w", err)
	}
	_ = f.Close()
	return nil
}

// Run executes the command inside an ephemeral Firecracker microVM.
func Run(ctx context.Context, cfg *config.Config, stdin io.Reader, stdout io.Writer, stderr io.Writer) (int, error) {
	if err := CheckKVM(); err != nil {
		return 1, err
	}

	fcBin, err := FindFirecrackerBinary(cfg.FirecrackerBin)
	if err != nil {
		return 1, err
	}

	// Create ephemeral CoW/sparse overlay workspace
	overlay, err := CreateOverlay(cfg.RootfsPath, cfg.WorkDir)
	if err != nil {
		return 1, fmt.Errorf("failed to create overlay: %w", err)
	}
	defer func() {
		_ = overlay.Cleanup()
	}()

	// Construct kernel boot arguments
	bootArgs := BuildBootArgs(cfg.KernelArgs, cfg.Args, runtime.GOARCH)

	// Wrap stdout with streaming console exit code filter
	filter := NewConsoleFilter(stdout)
	defer func() {
		_ = filter.Close()
	}()

	// Build firecracker execution command
	fcCmd := firecracker.VMCommandBuilder{}.
		WithBin(fcBin).
		WithSocketPath(overlay.SocketPath).
		WithStdin(stdin).
		WithStdout(filter).
		WithStderr(stderr).
		Build(ctx)

	// Setup silent logger for firecracker-go-sdk
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	logEntry := logrus.NewEntry(logger)

	fcCfg := firecracker.Config{
		SocketPath:      overlay.SocketPath,
		KernelImagePath: cfg.KernelPath,
		KernelArgs:      bootArgs,
		Drives: []models.Drive{
			{
				DriveID:      firecracker.String("rootfs"),
				PathOnHost:   firecracker.String(overlay.RootfsPath),
				IsRootDevice: firecracker.Bool(true),
				IsReadOnly:   firecracker.Bool(false),
			},
		},
		MachineCfg: models.MachineConfiguration{
			VcpuCount:  firecracker.Int64(cfg.VCPUs),
			MemSizeMib: firecracker.Int64(cfg.MemoryMB),
		},
	}

	machine, err := firecracker.NewMachine(ctx, fcCfg,
		firecracker.WithProcessRunner(fcCmd),
		firecracker.WithLogger(logEntry),
	)
	if err != nil {
		return 1, fmt.Errorf("failed to configure microvm machine: %w", err)
	}

	// Start the microVM
	if err := machine.Start(ctx); err != nil {
		return 1, fmt.Errorf("failed to start microvm: %w", err)
	}

	// Channel for machine process termination
	waitDone := make(chan error, 1)
	go func() {
		waitDone <- machine.Wait(context.Background())
	}()

	// Monitor lifecycle: handle normal completion vs cancellation/timeout/signal
	select {
	case err := <-waitDone:
		_ = filter.Close()
		if code, found := filter.ExitCode(); found {
			return code, nil
		}
		if err != nil {
			return 1, err
		}
		return 0, nil

	case <-ctx.Done():
		// Context was cancelled by timeout or signal trap
		// 1. Issue immediate clean shutdown via Firecracker API
		shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 500*time.Millisecond)
		_ = machine.Shutdown(shutdownCtx)
		cancelShutdown()

		// 2. Wait up to 500ms grace period for clean exit, then SIGKILL fallback
		select {
		case <-waitDone:
			// Machine terminated within grace period
		case <-time.After(500 * time.Millisecond):
			// Force kill fallback
			_ = machine.StopVMM()
			if pid, err := machine.PID(); err == nil && pid > 0 {
				_ = syscall.Kill(pid, syscall.SIGKILL)
			}
			<-waitDone
		}

		_ = filter.Close()

		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return 124, fmt.Errorf("command timed out after %v", cfg.Timeout)
		}
		return 130, ctx.Err()
	}
}
