# ephemvm Documentation

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

## Why ephemvm?

| Feature | Docker / Containers | Traditional VMs (VirtualBox, QEMU) | **ephemvm** |
| :--- | :--- | :--- | :--- |
| **Isolation Boundary** | Namespaces & cgroups *(shared host kernel)* | Full Hardware Virtualization | **Hardware Virtualization (KVM microVM)** |
| **Startup Latency** | 500ms – 2s | 20s – 60s | **< 50ms** |
| **Filesystem Safety** | Modifies writable container layer | Disk image modified | **Strict Read-Only Base + Reflink CoW** |
| **Cleanup Guarantee** | Lingering stopped containers & volumes | Orphan VM disks remain | **100% Guaranteed Deferred Cleanup** |
| **Kernel Crash Protection** | Host kernel panic possible | Protected | **Host Kernel Fully Protected** |

---

## Quick Tour

Run any Linux command in hardware isolation:

```bash
ephemvm run -- uname -a
```

Run destructive actions without risking host or base image:

```bash
ephemvm run -- sh -c "rm -rf --no-preserve-root /; echo 'Root destroyed!'"
```

Execute a local script inside the microVM:

```bash
ephemvm run -s ./scripts/benchmark.sh
```

Inspect and pre-download cached assets:

```bash
ephemvm cache status
ephemvm cache pull
```

---

## Navigation

- [Getting Started](getting-started.md) — Prerequisites and installation instructions.
- [Architecture & Internals](architecture.md) — How Firecracker, CoW overlays, and serial filtering work together.
- [Worked Examples](worked-examples.md) — Real-world scenarios (AI sandboxing, CI/CD, curl-to-bash inspection).
- [CLI Reference](cli-reference.md) — Comprehensive documentation of all flags and commands.
