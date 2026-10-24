//go:build linux

package vm

import (
	"bytes"
	"context"
	"encoding/json"
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

// ExecutionResult holds telemetry when running in structured mode.
type ExecutionResult struct {
	ExitCode   int    `json:"exit_code"`
	DurationMS int64  `json:"duration_ms"`
	TimedOut   bool   `json:"timed_out"`
	Stdout     string `json:"stdout,omitempty"`
	Stderr     string `json:"stderr,omitempty"`
	VCPUs      int64  `json:"vcpus"`
	MemoryMB   int64  `json:"memory_mb"`
}

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
	startTime := time.Now()

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
	bootArgs, err := BuildBootArgs(cfg, runtime.GOARCH, stdin)
	if err != nil {
		return 1, fmt.Errorf("failed to build boot args: %w", err)
	}

	var jsonStdoutBuf bytes.Buffer
	var jsonStderrBuf bytes.Buffer

	effectiveStdout := stdout
	effectiveStderr := stderr

	if cfg.JSONOutput {
		effectiveStdout = &jsonStdoutBuf
		effectiveStderr = &jsonStderrBuf
	}

	// Wrap stdout with streaming console exit code filter
	filter := NewConsoleFilter(effectiveStdout)
	defer func() {
		_ = filter.Close()
	}()

	// Build firecracker execution command
	fcCmd := firecracker.VMCommandBuilder{}.
		WithBin(fcBin).
		WithSocketPath(overlay.SocketPath).
		WithStdin(stdin).
		WithStdout(filter).
		WithStderr(effectiveStderr).
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

	var finalExitCode int
	var finalErr error
	var timedOut bool

	// Monitor lifecycle: handle normal completion vs cancellation/timeout/signal
	select {
	case err := <-waitDone:
		_ = filter.Close()
		if code, found := filter.ExitCode(); found {
			finalExitCode = code
		} else if err != nil {
			finalExitCode = 1
			finalErr = err
		} else {
			finalExitCode = 0
		}

	case <-ctx.Done():
		// Context was cancelled by timeout or signal trap
		shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 500*time.Millisecond)
		_ = machine.Shutdown(shutdownCtx)
		cancelShutdown()

		select {
		case <-waitDone:
		case <-time.After(500 * time.Millisecond):
			_ = machine.StopVMM()
			if pid, err := machine.PID(); err == nil && pid > 0 {
				_ = syscall.Kill(pid, syscall.SIGKILL)
			}
			<-waitDone
		}

		_ = filter.Close()

		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			timedOut = true
			finalExitCode = 124
			finalErr = fmt.Errorf("command timed out after %v", cfg.Timeout)
		} else {
			finalExitCode = 130
			finalErr = ctx.Err()
		}
	}

	if cfg.JSONOutput {
		result := ExecutionResult{
			ExitCode:   finalExitCode,
			DurationMS: time.Since(startTime).Milliseconds(),
			TimedOut:   timedOut,
			Stdout:     jsonStdoutBuf.String(),
			Stderr:     jsonStderrBuf.String(),
			VCPUs:      cfg.VCPUs,
			MemoryMB:   cfg.MemoryMB,
		}
		data, _ := json.MarshalIndent(result, "", "  ")
		_, _ = stdout.Write(append(data, '\n'))
	}

	return finalExitCode, finalErr
}
