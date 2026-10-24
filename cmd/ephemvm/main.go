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
	"github.com/smford/ephemvm/internal/version"
	"github.com/smford/ephemvm/internal/vm"
)

func main() {
	cfg := config.NewDefaultConfig()

	rootCmd := &cobra.Command{
		Use:   "ephemvm",
		Short: "Execute commands in an isolated Linux microVM in <100ms",
		Long: `ephemvm is an open-source single-binary CLI in Go that executes arbitrary
shell commands or scripts inside an isolated, hardware-virtualized Linux microVM
in under 100ms using Firecracker and ephemeral copy-on-write rootfs overlays.`,
		Version: version.Version,
	}

	// Custom version template to match user specification
	rootCmd.SetVersionTemplate(version.Formatted() + "\n")

	// 'version' subcommand
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print version, build, and license information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Print(version.Formatted() + "\n")
		},
	}
	rootCmd.AddCommand(versionCmd)

	// 'run' subcommand
	runCmd := &cobra.Command{
		Use:     "run [flags] -- <command> [args...]",
		Aliases: []string{"exec"},
		Short:   "Execute a command or script inside an ephemeral microVM",
		Long: `Runs the specified command or script inside a hardware-isolated Linux microVM with an
ephemeral copy-on-write overlay. Any filesystem mutations made by the command are
isolated to the overlay and destroyed immediately upon exit.`,
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
					if !cfg.JSONOutput {
						fmt.Fprintf(os.Stderr, "ephemvm: command timed out after %v\n", cfg.Timeout)
					}
					os.Exit(124)
				}
				if errors.Is(err, context.Canceled) {
					os.Exit(130)
				}
				if !errors.Is(err, vm.ErrUnsupportedPlatform) {
					if !cfg.JSONOutput {
						fmt.Fprintf(os.Stderr, "ephemvm: runtime error: %v\n", err)
					}
				} else {
					fmt.Fprintf(os.Stderr, "ephemvm: %v\n", err)
				}
			}

			os.Exit(exitCode)
		},
	}

	// Register flags for run
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
	runCmd.Flags().StringVarP(&cfg.ScriptPath, "script", "s", "", "Path to script file to execute inside guest (or '-' for stdin)")
	runCmd.Flags().StringArrayVarP(&cfg.Env, "env", "e", nil, "Set environment variables in guest (KEY=VAL or KEY to inherit from host)")
	runCmd.Flags().BoolVar(&cfg.JSONOutput, "json", false, "Output execution telemetry and results in JSON format")

	rootCmd.AddCommand(runCmd)

	// 'cache' subcommand
	cacheCmd := &cobra.Command{
		Use:   "cache [status|pull|clean]",
		Short: "Manage local kernel and rootfs assets",
		Long:  "Inspect, pre-download, or purge local cached microVM assets.",
	}

	cacheStatusCmd := &cobra.Command{
		Use:   "status",
		Short: "Inspect local cached kernel and rootfs assets",
		Run: func(cmd *cobra.Command, args []string) {
			mgr := assets.NewManager(cfg.CacheDir)
			statuses, err := mgr.Status()
			if err != nil {
				fmt.Fprintf(os.Stderr, "ephemvm: failed to inspect cache: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Cache Directory: %s\n\n", cfg.CacheDir)
			for _, s := range statuses {
				existsStr := "MISSING"
				if s.Exists {
					shaStatus := "VALID"
					if !s.ValidSHA256 {
						shaStatus = "CHECKSUM MISMATCH"
					}
					existsStr = fmt.Sprintf("PRESENT (%d bytes, %s, sha256: %s)", s.Size, s.Permissions, shaStatus)
				}
				fmt.Printf("  • %-12s : %s\n    Path         : %s\n", s.Name, existsStr, s.Path)
			}
		},
	}

	cachePullCmd := &cobra.Command{
		Use:   "pull",
		Short: "Pre-download and verify all release assets into cache",
		Run: func(cmd *cobra.Command, args []string) {
			mgr := assets.NewManager(cfg.CacheDir)
			fmt.Printf("Pre-warming assets into cache (%s)...\n", cfg.CacheDir)
			k, r, err := mgr.EnsureAssets(context.Background(), "", "")
			if err != nil {
				fmt.Fprintf(os.Stderr, "ephemvm: failed to pull assets: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Assets successfully cached and verified:\n  Kernel: %s\n  Rootfs: %s\n", k, r)
		},
	}

	cacheCleanCmd := &cobra.Command{
		Use:   "clean",
		Short: "Remove all cached assets and directory",
		Run: func(cmd *cobra.Command, args []string) {
			mgr := assets.NewManager(cfg.CacheDir)
			if err := mgr.Clean(); err != nil {
				fmt.Fprintf(os.Stderr, "ephemvm: failed to clean cache: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Cleaned cache directory: %s\n", cfg.CacheDir)
		},
	}

	cacheCmd.AddCommand(cacheStatusCmd, cachePullCmd, cacheCleanCmd)
	rootCmd.AddCommand(cacheCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
