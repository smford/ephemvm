# ephemvm

> **A blazing-fast, disposable Linux machine for running any command in total isolation.**
> Boots in under **50ms**, executes your command in hardware virtualization, forwards the exit code, and evaporates instantly without leaving a trace.

[![CI](https://github.com/smford/ephemvm/actions/workflows/ci.yml/badge.svg)](https://github.com/smford/ephemvm/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/smford/ephemvm)](https://goreportcard.com/report/github.com/smford/ephemvm)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

---

## What is ephemvm?

`ephemvm` (Ephemeral Virtual Machine) is a single-binary CLI written in Go that runs shell commands or scripts inside an isolated, hardware-virtualized Linux microVM powered by [AWS Firecracker](https://firecracker-microvm.github.io/) and `/dev/kvm`.

Think of it as a disposable scratchpad computer. It spins up an uncompressed Linux kernel and root filesystem in milliseconds, runs your command in hardware isolation, streams stdout/stderr unbuffered to your terminal, passes back the real exit code, and then immediately deletes every trace of the disk overlay and API socket.

---

## Real-World Problems ephemvm Solves

### 1. "Will this script trash my computer?"
- **The Problem:** Developers, SREs, and AI coding agents regularly write automation, disk partitioning, or teardown scripts. Running them locally is nerve-wracking: a mistyped `rm -rf $DIR/` where `$DIR` is unset will delete your entire system.
- **How ephemvm Solves It:** Total filesystem isolation using Copy-on-Write (CoW). The base root filesystem is stored read-only (`0444`). Every single execution generates a private, temporary overlay file. Even if your script executes `rm -rf --no-preserve-root /`, only the ephemeral overlay is destroyed. The base image and your host computer remain 100% untouched.

### 2. "Containers share your host kernel"
- **The Problem:** Docker and container runtimes isolate user space through Linux namespaces and cgroups, but they share the exact same kernel with your host operating system. A kernel vulnerability, privilege escalation exploit, or raw networking experiment can compromise the host machine.
- **How ephemvm Solves It:** Hardware-level virtualization via KVM. `ephemvm` boots an entirely independent guest kernel (`vmlinux`) with dedicated virtual CPUs and dedicated memory. Kernel panics, crashes, and malicious exploits are contained strictly inside the microVM.

### 3. "Traditional VMs take 20 to 60 seconds to boot"
- **The Problem:** Full-blown virtual machines (VirtualBox, VMware, standard QEMU) are safe, but their boot process is far too slow for interactive test loops, CI jobs, or rapid AI agent tool calls.
- **How ephemvm Solves It:** Firecracker strips away legacy PC devices (BIOS, PCI buses, IDE controllers). Combined with `ephemvm`'s sparse block and reflink overlay initialization, microVMs boot directly into your command in under **50 milliseconds**.

### 4. "Disk clutter, orphan containers, and leftover sockets"
- **The Problem:** Containers and temporary VMs frequently leave behind stopped containers, dangling disk layers, dangling Unix sockets, and gigabytes of wasted storage.
- **How ephemvm Solves It:** Guaranteed deferred cleanup. `ephemvm` traps all exit signals (`SIGINT`, `SIGTERM`, `SIGQUIT`) and hard timeouts. On exit, it cleanly halts Firecracker (falling back to `SIGKILL` after 500ms) and purges the temporary `/tmp/ephemvm-<uuid>/` directory.

---

## Prerequisites

- **Host OS:** Linux (kernel 4.14+) with KVM hardware virtualization enabled (`/dev/kvm`).
- **Firecracker:** [Firecracker binary](https://github.com/firecracker-microvm/firecracker/releases) installed in `PATH` (or specified via `--firecracker-binary`).
- **Permissions:** Your user must have read/write access to `/dev/kvm` (e.g. `sudo usermod -aG kvm $USER`).

*(Note: Binaries can be built or cross-compiled on macOS and other platforms, but microVM execution requires Linux with `/dev/kvm`.)*

---

## Installation

### Pre-built Binaries

Download the latest static binary for your architecture from [GitHub Releases](https://github.com/smford/ephemvm/releases):

```bash
# Example for Linux x86_64
curl -fsSL https://github.com/smford/ephemvm/releases/latest/download/ephemvm-linux-amd64 -o ephemvm
chmod +x ephemvm
sudo mv ephemvm /usr/local/bin/
```

### From Source

```bash
git clone https://github.com/smford/ephemvm.git
cd ephemvm
make build
sudo cp bin/ephemvm /usr/local/bin/
```

---

## Worked Examples of Common Tasks

### Example 1: Safe Playground for AI Coding Agents
Give AI coding agents (Claude, Antigravity, Cursor, OpenCode) unrestricted bash execution without risking your laptop or host filesystem:

```bash
ephemvm run -- python3 -c "import os; print('Hello from isolated guest!')"
```

### Example 2: Testing Catastrophic Teardown Scripts
Test destructive teardown automation without fear of deleting host files:

```bash
ephemvm run -- sh -c "rm -rf --no-preserve-root / 2>/dev/null; echo 'Deleted root!'"
```
*Your host filesystem and the base rootfs remain completely intact.*

### Example 3: Running Untrusted "curl | sh" Install Scripts
Inspect and verify third-party installation scripts before allowing them anywhere near your actual environment:

```bash
ephemvm run -- sh -c "curl -sSL https://get.docker.com | sh"
```

### Example 4: Executing Multi-line Local Shell Scripts
Run an entire local shell script inside the microVM without escaping quotes or worrying about command-line length limits:

```bash
ephemvm run -s ./scripts/test-automation.sh -- arg1 arg2
```

Or pipe a script directly via standard input:

```bash
cat << 'EOF' | ephemvm run -s -
echo "Running system diagnostics inside microVM:"
uname -a
df -h
free -m
EOF
```

### Example 5: Injecting Environment Variables
Securely pass environment variables from your host shell into the guest execution:

```bash
# Pass explicitly
ephemvm run -e GITHUB_TOKEN=ghp_secret123 -e ENVIRONMENT=staging -- env

# Inherit directly from the current host environment
export DEPLOY_KEY="my-secret-key"
ephemvm run -e DEPLOY_KEY -- sh -c 'echo "Key length is: ${#DEPLOY_KEY}"'
```

### Example 6: Pre-warming Assets with Cache Management
Download and verify release assets before going offline or running microVMs:

```bash
# Inspect current asset status and SHA256 integrity
ephemvm cache status

# Pre-download and cache kernel and rootfs assets
ephemvm cache pull

# Purge cache
ephemvm cache clean
```

### Example 7: Automated CI/CD Testing with Machine-Readable JSON Telemetry
Capture structured execution metadata, execution durations, and exit codes for programmatic parsing:

```bash
ephemvm run --json -- sh -c "echo 'hello json'; exit 0"
```

Output:
```json
{
  "exit_code": 0,
  "duration_ms": 78,
  "timed_out": false,
  "stdout": "hello json\n",
  "stderr": "",
  "vcpus": 1,
  "memory_mb": 128
}
```

### Example 8: Enforcing Strict Hardware and Timeout Constraints
Restrict memory-heavy or runaway processes:

```bash
# 2 dedicated vCPUs, 256MB RAM, and a hard 10-second kill timer
ephemvm run --vcpus 2 --memory 256 --timeout 10s -- sh -c "sleep 5; echo 'Done!'"
```

---

## CLI Reference

```
Usage:
  ephemvm [command]

Available Commands:
  cache       Manage local kernel and rootfs assets (status, pull, clean)
  help        Help about any command
  run         Execute a command or script inside an ephemeral microVM (alias: exec)
  version     Print version, build, and license information

Flags for 'ephemvm run':
      --cache-dir string            Directory to cache kernel and rootfs assets (default "~/.cache/ephemvm")
  -e, --env stringArray             Set environment variables in guest (KEY=VAL or KEY)
      --firecracker-binary string   Path to firecracker binary (default: search PATH)
  -h, --help                        Help for run
      --json                        Output execution telemetry and results in JSON format
      --kernel string               Path to custom uncompressed vmlinux image
      --kernel-args string          Custom kernel command line parameter overrides
      --log-level string            Firecracker log level (Error, Warning, Info, Debug) (default "Error")
      --memory int                  Memory envelope in MB (default 128)
      --rootfs string               Path to custom base ext4 rootfs image
  -s, --script string               Path to script file to execute inside guest (or '-' for stdin)
      --timeout duration            Hard kill timer duration (e.g. 30s, 2m) (default 1m0s)
      --vcpus int                   Number of dedicated virtual CPUs (default 1)
      --work-dir string             Parent directory for ephemeral overlays (default "/tmp")
```

---

## Architecture & How It Works

```
ephemvm/
├── cmd/
│   └── ephemvm/
│       └── main.go                 # Cobra CLI, signal handlers, cache subcommands
├── internal/
│   ├── config/
│   │   ├── config.go               # Runtime flag parsing and validation
│   │   └── config_test.go
│   ├── version/
│   │   ├── version.go              # Standardized version and license formatting
│   │   └── version_test.go
│   ├── vm/
│   │   ├── firecracker.go          # Boot arguments, script encoding, env builder
│   │   ├── firecracker_linux.go    # Firecracker SDK lifecycle & signal management
│   │   ├── firecracker_unsupported.go # Stub for non-Linux OSes
│   │   ├── overlay.go              # Ephemeral CoW reflink and sparse overlay engine
│   │   ├── overlay_linux.go        # Linux FICLONE ioctl implementation
│   │   ├── overlay_other.go        # Non-Linux sparse copy fallback
│   │   ├── console.go              # Streaming serial I/O filter and exit code extractor
│   │   ├── console_test.go
│   │   ├── overlay_test.go
│   │   └── vm_test.go              # Echo, destructive, and timeout test suite
│   └── assets/
│       ├── manager.go              # Kernel & rootfs caching, SHA256 integrity check
│       └── manager_test.go
├── Makefile                        # Multi-architecture builds, tests, linting
├── go.mod
└── go.sum
```

### 1. Storage: CoW Overlay & Reflink
`ephemvm` maintains a single read-only base root filesystem image (`rootfs.ext4`) marked with `0444` permissions. When `ephemvm run` starts, it creates a unique per-run directory in `/tmp/ephemvm-<uuid>/` and issues a `FICLONE` ioctl (`reflink`). On modern filesystems (Btrfs, XFS with reflink, ZFS), this creates an instant, zero-byte copy-on-write clone. On other filesystems, it scans 64KB zero-blocks to generate a sparse file copy. The guest mounts this clone as read-write (`0600`), completely shielding the base image.

### 2. Exit Code Sentinel Protocol
The microVM boots with:
```
init=/bin/sh -- -c "<env_exports>; <cmd>; echo __EXIT_CODE:$?__; reboot -f"
```
The streaming console scanner filters the `__EXIT_CODE:<exit_code>__` sentinel in real-time from the serial stream, captures the exact exit code, suppresses the sentinel string from host standard output, and forwards the exit status to the host process upon microVM shutdown.

### 3. Graceful Signal Trapping
If an execution times out or receives `SIGINT` (`Ctrl+C`), `ephemvm` catches the signal and sends an immediate shutdown signal over the Firecracker control socket. If the process does not terminate within 500ms, it falls back to `SIGKILL` and executes deferred cleanup, deleting the overlay directory and API socket.

---

## Testing

Run unit tests with race detection:

```bash
make test
```

Run static analysis and linter:

```bash
make vet
```

---

## License

MIT License. See [LICENSE](LICENSE) for details.
Copyright (c) 2026 smford.
