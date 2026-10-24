# CLI Reference

Full reference for all `ephemvm` commands and command-line flags.

---

## Global Usage

```
ephemvm [command] [flags]
```

### Global Flags

| Flag | Description |
| :--- | :--- |
| `-h, --help` | Display help for command |
| `-v, --version` | Display version, commit, date, and license information |

---

## Commands

### `ephemvm run`

Executes a command or script inside an ephemeral microVM. (Alias: `exec`).

```
ephemvm run [flags] -- <command> [args...]
ephemvm run [flags] -s <script_path> [-- [args...]]
```

#### Flags

| Flag | Default | Description |
| :--- | :--- | :--- |
| `--timeout <duration>` | `1m0s` | Maximum execution duration before hard termination. |
| `--vcpus <int>` | `1` | Number of dedicated virtual CPUs allocated to the microVM. |
| `--memory <int>` | `128` | Memory envelope in megabytes (minimum 64MB). |
| `-s, --script <path>` | `""` | Path to a local script file to run inside the guest (or `-` for stdin). |
| `-e, --env <key=val>` | `[]` | Environment variable to inject into guest (`KEY=VAL` or `KEY` to inherit from host). |
| `--json` | `false` | Emits execution telemetry and results in structured JSON format. |
| `--kernel <path>` | `""` | Custom path to an uncompressed `vmlinux` binary. |
| `--rootfs <path>` | `""` | Custom path to a base `rootfs.ext4` image. |
| `--kernel-args <string>`| `""` | Override base kernel boot arguments. |
| `--firecracker-binary <path>` | `""` | Custom path to Firecracker binary (defaults to searching `PATH`). |
| `--cache-dir <path>` | `~/.cache/ephemvm` | Directory where kernel and rootfs assets are stored. |
| `--work-dir <path>` | `/tmp` | Parent directory for temporary ephemeral overlays. |
| `--log-level <string>` | `Error` | Firecracker log level (`Error`, `Warning`, `Info`, `Debug`). |

---

### `ephemvm cache`

Manages local kernel and root filesystem assets.

```
ephemvm cache [status|pull|clean]
```

#### Subcommands

- `ephemvm cache status`: Displays the presence, size, permissions, and SHA256 integrity status of cached assets.
- `ephemvm cache pull`: Pre-downloads and verifies release assets without starting a VM.
- `ephemvm cache clean`: Deletes the cache directory and all downloaded assets.

---

### `ephemvm version`

Displays version, git commit SHA, build date, and license information:

```
ephemvm version
```

Output:
```
ephemvm 0.1.0 (commit: ..., date: ...)
Copyright (c) 2026 smford
Licensed under the MIT License
```
