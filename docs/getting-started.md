# Getting Started

## Prerequisites

To run microVMs with `ephemvm`, your system must meet the following requirements:

1. **Host Operating System:** Linux (kernel 4.14 or later).
2. **Hardware Virtualization:** KVM enabled in BIOS/UEFI (`/dev/kvm` must exist).
3. **User Access:** The executing user must have read and write permissions to `/dev/kvm`:
   ```bash
   # Add your user to the kvm group
   sudo usermod -aG kvm $USER
   # Verify access
   test -r /dev/kvm && test -w /dev/kvm && echo "KVM access OK"
   ```
4. **Firecracker:** Install the [Firecracker](https://github.com/firecracker-microvm/firecracker/releases) binary and place it in your `PATH` (or point to it via `--firecracker-binary`).

*(Note: Binaries can be built or cross-compiled on macOS and Windows, but microVM execution requires Linux with `/dev/kvm`.)*

---

## Installation

### Static Binary (Recommended)

Download the latest static binary for your architecture from [GitHub Releases](https://github.com/smford/ephemvm/releases):

```bash
# For x86_64
curl -fsSL https://github.com/smford/ephemvm/releases/latest/download/ephemvm_linux_amd64.tar.gz | tar -xz
sudo mv ephemvm /usr/local/bin/

# For ARM64 (e.g. AWS Graviton, Raspberry Pi 5)
curl -fsSL https://github.com/smford/ephemvm/releases/latest/download/ephemvm_linux_arm64.tar.gz | tar -xz
sudo mv ephemvm /usr/local/bin/
```

### Homebrew (macOS / Linux)

If using Homebrew on Linux:

```bash
brew tap smford/tap
brew install ephemvm
```

### Building From Source

Requires Go 1.22+:

```bash
git clone https://github.com/smford/ephemvm.git
cd ephemvm
make build
sudo cp bin/ephemvm /usr/local/bin/
```

---

## Your First Ephemeral Command

Run your first command inside a fresh microVM:

```bash
ephemvm run -- uname -a
```

When you run this:
1. `ephemvm` automatically downloads tested release assets (Alpine Linux `vmlinux` kernel and minimal `rootfs.ext4`) into `~/.cache/ephemvm/` and verifies their SHA256 checksums.
2. It generates a temporary overlay in `/tmp/ephemvm-<uuid>/`.
3. It launches Firecracker, boots the Linux guest kernel, executes `uname -a`, and streams the output directly to your terminal.
4. The moment the command finishes, Firecracker shuts down and the temporary overlay directory is permanently deleted.
