# ephemvm

`ephemvm` is a high-performance, single-binary CLI in Go that executes arbitrary shell commands or scripts inside an isolated, hardware-virtualized Linux microVM in under **100ms**.

It leverages [AWS Firecracker](https://firecracker-microvm.github.io/) over `/dev/kvm` and instantaneous copy-on-write (CoW) disk overlays to safely run untrusted code, destructive automation (`rm -rf /`), or kernel tests without risking host filesystem or kernel integrity.

---

## Key Features

- **Sub-100ms Startup**: Launches an isolated microVM with hardware virtualization in milliseconds.
- **Strict Read-Only Base Storage & Ephemeral CoW Overlays**: Base kernel (`vmlinux`) and rootfs (`rootfs.ext4`) are cached locally in `~/.cache/ephemvm/` with read-only permissions (`0444`). Every run operates on a temporary copy-on-write overlay file (`reflink`/`FICLONE` where supported, falling back to sparse cloning) that is completely destroyed on exit.
- **Unbuffered Serial Streaming & Exit Code Forwarding**: Streams console `ttyS0` standard I/O in real time while capturing and filtering the guest's actual process exit code back to the host shell.
- **Defensive Resource Management & Signal Trapping**: Traps `SIGINT`, `SIGTERM`, and `SIGQUIT`. Issues a clean shutdown via Firecracker API with a 500ms `SIGKILL` fallback, guaranteeing all overlay artifacts and API sockets are purged.
- **Automated Asset Provisioning**: Automatically downloads and verifies SHA256 integrity of release assets (Alpine vmlinux and ext4 rootfs).

---

## Requirements

- **Linux** (kernel 4.14+) with KVM hardware virtualization enabled (`/dev/kvm`).
- [Firecracker](https://github.com/firecracker-microvm/firecracker/releases) binary installed in `PATH` (or specified via `--firecracker-binary`).
- Current user granted read/write permissions to `/dev/kvm` (e.g., `sudo usermod -aG kvm $USER`).

*(Note: Binaries can be built on macOS and other platforms, but execution requires a Linux host with `/dev/kvm`.)*

---

## Installation

### From Source

```bash
git clone https://github.com/smford/ephemvm.git
cd ephemvm
make build
sudo cp bin/ephemvm /usr/local/bin/
```

### Cross-Compilation

To cross-compile static Linux binaries for `x86_64` and `aarch64`:

```bash
make build-linux
# Binaries output to bin/ephemvm-linux-amd64 and bin/ephemvm-linux-arm64
```

---

## Quickstart

Run a command inside an isolated microVM:

```bash
ephemvm run -- uname -a
```

Execute a destructive command without altering the host or base rootfs:

```bash
ephemvm run -- sh -c "rm -rf --no-preserve-root /; echo 'destroyed in vm'"
```

Forwarding exit codes:

```bash
ephemvm run -- sh -c "exit 42"
echo $? # prints 42
```

---

## CLI Reference

```
Usage:
  ephemvm run [flags] -- <command> [args...]

Flags:
      --timeout duration            Hard kill timer duration (e.g. 30s, 2m) (default 1m0s)
      --vcpus int                   Number of dedicated virtual CPUs (default 1)
      --memory int                  Memory envelope in MB (default 128)
      --kernel-args string          Custom kernel command line parameter overrides
      --kernel string               Path to custom uncompressed vmlinux image
      --rootfs string               Path to custom base ext4 rootfs image
      --firecracker-binary string   Path to firecracker binary (default: search PATH)
      --cache-dir string            Directory to cache kernel and rootfs assets (default "~/.cache/ephemvm")
      --work-dir string             Parent directory for ephemeral overlays (default "/tmp")
      --log-level string            Firecracker log level (Error, Warning, Info, Debug) (default "Error")
  -h, --help                        Help for run
```

---

## Architecture & Internals

```
ephemvm/
├── cmd/
│   └── ephemvm/
│       └── main.go                 # Cobra CLI root and signal handling
├── internal/
│   ├── config/
│   │   ├── config.go               # Runtime flag parsing and validation
│   │   └── config_test.go
│   ├── vm/
│   │   ├── firecracker.go          # Boot arguments, escaping, binary discovery
│   │   ├── firecracker_linux.go    # Firecracker SDK lifecycle & signal management
│   │   ├── firecracker_unsupported.go # Stub for non-Linux OSes
│   │   ├── overlay.go              # CoW reflink / sparse file creation & cleanup
│   │   ├── overlay_linux.go        # Linux FICLONE ioctl implementation
│   │   ├── overlay_other.go        # Non-Linux sparse copy fallback
│   │   ├── console.go              # Serial I/O pipes and exit-code filter
│   │   ├── console_test.go
│   │   ├── overlay_test.go
│   │   └── vm_test.go              # Echo, destructive, and timeout test suite
│   └── assets/
│       ├── manager.go              # Kernel and base rootfs caching and SHA256 validation
│       └── manager_test.go
├── Makefile                        # Build, test, and lint targets
├── go.mod
└── go.sum
```

### Ephemeral Storage Lifecycle

1. When `ephemvm run` starts, it creates a unique working directory `/tmp/ephemvm-<uuid>/`.
2. It generates an ephemeral copy of the base `rootfs.ext4` inside that directory using the Linux `FICLONE` ioctl (`reflink`). On filesystems that do not support block cloning, it automatically falls back to sparse block scanning.
3. The guest mounts this ephemeral copy read-write.
4. On process exit, timeout, or signal cancellation (`SIGINT`/`SIGTERM`/`SIGQUIT`), the entire overlay directory including the Firecracker API socket is purged via deferred cleanup.

### Exit Code Sentinel Protocol

The VM boots with:
```
init=/bin/sh -- -c "<cmd>; echo __EXIT_CODE:$?__; reboot -f"
```

The streaming console scanner filters the `__EXIT_CODE:<exit_code>__` sentinel in real-time from the serial stream, captures the exact exit code, suppresses the sentinel string from host standard output, and forwards the exit status to the host process upon microVM shutdown.

---

## Testing

Run unit tests with race detection:

```bash
make test
```

Run linter / static checks:

```bash
make lint
```

Run integration tests on a Linux host with `/dev/kvm`:

```bash
make test-integration
```

---

## License

Apache 2.0. See [LICENSE](LICENSE) for details.
