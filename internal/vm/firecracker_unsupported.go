//go:build !linux

package vm

import (
	"context"
	"errors"
	"io"

	"github.com/smford/ephemvm/internal/config"
)

const unsupportedPlatformMessage = `ephemvm requires Linux with KVM hardware virtualization (/dev/kvm).

Why this happens:
  ephemvm executes commands inside microVMs using AWS Firecracker, which communicates
  directly with the Linux Kernel-based Virtual Machine (/dev/kvm) ioctl interfaces.
  macOS uses the Apple Hypervisor.framework and does not provide /dev/kvm.

How to run ephemvm:
  1. On a Linux Machine (Recommended):
     Run ephemvm on any Linux workstation, server, or cloud instance (Ubuntu, Debian, Fedora, Arch, etc.).
     Ensure your user has access to /dev/kvm:
       sudo usermod -aG kvm $USER

  2. On macOS via a Linux VM with Nested Virtualization:
     Spin up a local Linux VM (e.g. via UTM with virtualization enabled, Multipass, Lima, or Docker Desktop)
     and run ephemvm inside that Linux VM.

  3. Asset Inspection & Pre-Warming:
     You can still inspect, download, and verify microVM release assets on macOS using:
       ephemvm cache status
       ephemvm cache pull`

// CheckKVM returns a descriptive error on non-Linux platforms.
func CheckKVM() error {
	return errors.New(unsupportedPlatformMessage)
}

// Run returns a descriptive error on non-Linux platforms.
func Run(ctx context.Context, cfg *config.Config, stdin io.Reader, stdout io.Writer, stderr io.Writer) (int, error) {
	return 1, errors.New(unsupportedPlatformMessage)
}
