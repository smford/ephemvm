package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/smford/ephemvm/internal/assets"
	"github.com/smford/ephemvm/internal/config"
	"github.com/smford/ephemvm/internal/vm"
)

var (
	version = "0.1.0"
)

func main() {
	cfg := config.NewDefaultConfig()

	rootCmd := &cobra.Command{
		Use:   "ephemvm",
		Short: "Execute commands in an isolated Linux microVM in <100ms",
		Long: `ephemvm is an open-source single-binary CLI in Go that executes arbitrary
shell commands or scripts inside an isolated, hardware-virtualized Linux microVM
in under 100ms using Firecracker and ephemeral copy-on-write rootfs overlays.`,
		Version: version,
	}

	runCmd := &cobra.Command{
		Use:   "run [flags] -- <command> [args...]",
		Short: "Execute a command inside an ephemeral microVM",
		Long: `Runs the specified command inside a hardware-isolated Linux microVM with an
ephemeral copy-on-write overlay. Any filesystem mutations made by the command are
isolated to the overlay and destroyed immediately upon exit.`,
		Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			cfg.Args = args
			if err := cfg.Validate(); err != nil {
				fmt.Fprintf(os.Stderr, "ephemvm: configuration error: %v\n", err)
				os.Exit(1)
			}

			// Setup OS signal trapping: os.Interrupt (SIGINT), SIGTERM, SIGQUIT
			ctx, stopNotify := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
			defer stopNotify()

			// Apply hard execution timeout
			var runCtx context.Context
			var cancelTimeout context.CancelFunc
			if cfg.Timeout > 0 {
				runCtx, cancelTimeout = context.WithTimeout(ctx, cfg.Timeout)
				defer cancelTimeout()
			} else {
				runCtx = ctx
			}

			// Ensure kernel and base rootfs assets are available and integrity-verified
			mgr := assets.NewManager(cfg.CacheDir)
			kernelPath, rootfsPath, err := mgr.EnsureAssets(runCtx, cfg.KernelPath, cfg.RootfsPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "ephemvm: asset manager error: %v\n", err)
				os.Exit(1)
			}
			cfg.KernelPath = kernelPath
			cfg.RootfsPath = rootfsPath

			// Execute inside ephemeral microVM
			exitCode, err := vm.Run(runCtx, cfg, os.Stdin, os.Stdout, os.Stderr)
			if err != nil {
				if errors.Is(err, context.DeadlineExceeded) {
					fmt.Fprintf(os.Stderr, "ephemvm: command timed out after %v\n", cfg.Timeout)
					os.Exit(124)
				}
				if errors.Is(err, context.Canceled) {
					os.Exit(130)
				}
				if !errors.Is(err, vm.ErrUnsupportedPlatform) {
					fmt.Fprintf(os.Stderr, "ephemvm: runtime error: %v\n", err)
				} else {
					fmt.Fprintf(os.Stderr, "ephemvm: %v\n", err)
				}
			}

			os.Exit(exitCode)
		},
	}

	// Register CLI flags
	runCmd.Flags().DurationVar(&cfg.Timeout, "timeout", config.DefaultTimeout, "Hard kill timer duration (e.g. 30s, 2m)")
	runCmd.Flags().Int64Var(&cfg.VCPUs, "vcpus", config.DefaultVCPUs, "Number of dedicated virtual CPUs")
	runCmd.Flags().Int64Var(&cfg.MemoryMB, "memory", config.DefaultMemoryMB, "Memory envelope in MB")
	runCmd.Flags().StringVar(&cfg.KernelArgs, "kernel-args", "", "Custom kernel command line parameter overrides")
	runCmd.Flags().StringVar(&cfg.KernelPath, "kernel", "", "Path to custom uncompressed vmlinux image")
	runCmd.Flags().StringVar(&cfg.RootfsPath, "rootfs", "", "Path to custom base ext4 rootfs image")
	runCmd.Flags().StringVar(&cfg.FirecrackerBin, "firecracker-binary", "", "Path to firecracker binary (default: search PATH)")
	runCmd.Flags().StringVar(&cfg.CacheDir, "cache-dir", cfg.CacheDir, "Directory to cache kernel and rootfs assets")
	runCmd.Flags().StringVar(&cfg.WorkDir, "work-dir", cfg.WorkDir, "Parent directory for ephemeral overlays")
	runCmd.Flags().StringVar(&cfg.LogLevel, "log-level", config.DefaultLogLevel, "Firecracker log level (Error, Warning, Info, Debug)")

	rootCmd.AddCommand(runCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
