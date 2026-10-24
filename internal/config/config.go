package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	DefaultTimeout  = 60 * time.Second
	DefaultVCPUs    = 1
	DefaultMemoryMB = 128
	DefaultLogLevel = "Error"
	MinMemoryMB     = 64
	MinVCPUs        = 1
)

// Config represents runtime configuration for ephemvm microVM execution.
type Config struct {
	// Timeout is the maximum duration to allow the microVM to execute.
	Timeout time.Duration

	// VCPUs defines the number of dedicated virtual CPUs.
	VCPUs int64

	// MemoryMB defines the guest memory envelope in megabytes.
	MemoryMB int64

	// KernelArgs contains custom kernel command line overrides.
	KernelArgs string

	// KernelPath specifies an explicit path to an uncompressed vmlinux binary.
	KernelPath string

	// RootfsPath specifies an explicit path to a base ext4 rootfs image.
	RootfsPath string

	// FirecrackerBin specifies the path to the firecracker binary.
	FirecrackerBin string

	// CacheDir defines the base directory where assets (vmlinux, rootfs) are cached.
	CacheDir string

	// WorkDir defines the parent directory where ephemeral overlays are created.
	WorkDir string

	// LogLevel defines Firecracker log level ("Error", "Warning", "Info", "Debug").
	LogLevel string

	// ScriptPath specifies an optional local script file to run inside the guest.
	ScriptPath string

	// Env holds key=value environment variables to inject into the guest.
	Env []string

	// JSONOutput enables structured JSON reporting of execution telemetry.
	JSONOutput bool

	// Args contains the command and arguments to execute inside the microVM.
	Args []string
}

// NewDefaultConfig returns a Config initialized with sensible production defaults.
func NewDefaultConfig() *Config {
	home, err := os.UserHomeDir()
	cacheDir := ""
	if err == nil {
		cacheDir = filepath.Join(home, ".cache", "ephemvm")
	} else {
		cacheDir = filepath.Join(os.TempDir(), "ephemvm-cache")
	}

	return &Config{
		Timeout:  DefaultTimeout,
		VCPUs:    DefaultVCPUs,
		MemoryMB: DefaultMemoryMB,
		CacheDir: cacheDir,
		WorkDir:  os.TempDir(),
		LogLevel: DefaultLogLevel,
		Args:     nil,
	}
}

// Validate verifies that all configuration values are valid.
func (c *Config) Validate() error {
	if c.Timeout <= 0 {
		return errors.New("timeout must be greater than zero")
	}
	if c.VCPUs < MinVCPUs {
		return fmt.Errorf("vcpus must be at least %d (got %d)", MinVCPUs, c.VCPUs)
	}
	if c.MemoryMB < MinMemoryMB {
		return fmt.Errorf("memory must be at least %d MB (got %d)", MinMemoryMB, c.MemoryMB)
	}
	if len(c.Args) == 0 && c.ScriptPath == "" {
		return errors.New("command arguments or --script must be provided")
	}
	return nil
}
