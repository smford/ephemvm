# Architecture & Internals

`ephemvm` combines AWS Firecracker, the Linux KVM hypervisor interface, copy-on-write storage techniques, and real-time serial streaming into a single, high-performance binary.

```
+------------------------------------------------------------------------+
|                              HOST SHELL                                |
|   $ ephemvm run [flags] -- <cmd>                                       |
+-----------------------------------+------------------------------------+
                                    |
                                    v
+------------------------------------------------------------------------+
|                           ephemvm CORE                                 |
|                                                                        |
|  1. Asset Manager (~/.cache/ephemvm/)                                  |
|     - vmlinux (kernel, read-only 0444)                                 |
|     - rootfs.ext4 (base image, read-only 0444, SHA256 verified)        |
|                                                                        |
|  2. Overlay Allocator (/tmp/ephemvm-<uuid>/)                           |
|     - FICLONE ioctl (instant reflink CoW clone)                        |
|     - Sparse block scanner fallback                                    |
|     - firecracker.sock API control socket                              |
|                                                                        |
|  3. Streaming Console Filter                                           |
|     - Direct serial UART ttyS0 bridge                                  |
|     - Extracts & removes __EXIT_CODE:$?__                              |
|     - Real-time unbuffered stream to host os.Stdout                    |
|                                                                        |
|  4. Signal Trapper & SRE Watchdog                                      |
|     - Traps SIGINT / SIGTERM / SIGQUIT                                 |
|     - Clean shutdown -> 500ms grace period -> SIGKILL fallback         |
|     - Deferred cleanup: guarantees zero disk/socket leakage            |
+-----------------------------------+------------------------------------+
                                    |
                                    v
+------------------------------------------------------------------------+
|                   FIRECRACKER MICROVM (over /dev/kvm)                  |
|                                                                        |
|   - Dedicated vCPU & RAM envelope                                      |
|   - Stripped-down Linux Kernel (no BIOS, no PCI, no ACPI)              |
|   - Ephemeral CoW Rootfs (/dev/vda)                                    |
|   - Serial Console UART (ttyS0)                                        |
|                                                                        |
|   Guest Exec:                                                          |
|   init=/bin/sh -- -c "<cmd>; echo __EXIT_CODE:$?__; reboot -f"         |
+------------------------------------------------------------------------+
```

---

## Storage & Copy-on-Write Overlay

1. **Strict Base Image Protection:**  
   The base root filesystem (`rootfs.ext4`) and kernel (`vmlinux`) are stored in `~/.cache/ephemvm/` with read-only permissions (`0444`). The base image is never opened in write mode or attached to Firecracker directly.

2. **Reflink / FICLONE:**  
   When an execution starts, `ephemvm` calls the Linux `FICLONE` ioctl on filesystems supporting block-level copy-on-write (Btrfs, XFS with reflink enabled, ZFS with block cloning). This generates an instant, zero-byte file clone that only allocates physical blocks when mutated.

3. **Sparse Copy Fallback:**  
   If the filesystem does not support `FICLONE`, `ephemvm` reads the base image in 64KB blocks. Runs of zero bytes are skipped via `Seek` calls rather than written, preserving sparse disk allocation and copying at gigabytes per second.

4. **Deferred Cleanup:**  
   The entire per-run directory `/tmp/ephemvm-<uuid>/` is removed via deferred cleanup before process termination.

---

## Exit Code Sentinel Protocol

To accurately return the guest command's exit code without requiring a guest daemon or agent:

1. `ephemvm` instructs the kernel to boot with:
   ```
   console=ttyS0 reboot=k panic=1 init=/bin/sh -- -c "<cmd>; echo __EXIT_CODE:$?__; reboot -f"
   ```
2. When `<cmd>` finishes, `/bin/sh` captures `$?` and emits:
   ```
   __EXIT_CODE:<exit_code>__
   ```
   followed immediately by `reboot -f`.
3. The host-side `ConsoleFilter` wraps Firecracker's stdout stream. It uses a rolling sliding buffer to detect and extract the sentinel pattern across arbitrary streaming chunk boundaries.
4. The sentinel string is suppressed so it never appears in the user's terminal, and the integer exit code is returned to the host shell.

---

## Signal Trapping & Lifecycle

`ephemvm` intercepts `os.Interrupt` (`SIGINT`), `syscall.SIGTERM`, and `syscall.SIGQUIT`:

1. On signal or timeout, `ephemvm` issues an immediate clean shutdown request to Firecracker's Unix socket API.
2. A 500ms watchdog timer starts.
3. If the microVM has not stopped after 500ms, `ephemvm` calls `StopVMM()` and issues a `SIGKILL` to Firecracker's process ID.
4. The deferred overlay cleanup executes, guaranteeing no orphaned sockets or disk files remain.
