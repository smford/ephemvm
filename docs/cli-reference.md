# CLI Reference

Comprehensive reference guide for the `ephemvm` command-line interface.

---

## Global Synopsis

```bash
ephemvm [command] [flags]
```

### Direct Command Routing
For convenience, `ephemvm` automatically routes commands without requiring explicit subcommands:

```bash
# These two commands are completely identical:
ephemvm "uname -a"
ephemvm run -- uname -a
```

### Global Flags

| Flag | Shorthand | Type | Default | Description |
| :--- | :---: | :--- | :--- | :--- |
| `--help` | `-h` | `bool` | `false` | Displays help message and available options |
| `--version` | `-v` | `bool` | `false` | Displays version, commit SHA, build date, and license |

---

## Commands

### `ephemvm run`

Executes a command or script inside an isolated hardware microVM. (Alias: `exec`).

```bash
ephemvm run [flags] -- <command> [args...]
ephemvm run [flags] -s <script_path> [-- [args...]]
```

#### Execution Flags

| Flag | Shorthand | Type | Default | Description |
| :--- | :---: | :--- | :--- | :--- |
| `--timeout` | | `duration` | `1m0s` | Maximum execution duration before hard termination (e.g. `10s`, `5m`, `1h`). |
| `--vcpus` | | `int` | `1` | Number of dedicated virtual CPUs allocated to the microVM. |
| `--memory` | | `int` | `128` | Memory envelope in megabytes (minimum 64MB). |
| `--script` | `-s` | `string` | `""` | Path to a local script to execute inside guest (use `-` for standard input). |
| `--env` | `-e` | `stringArray`| `[]` | Environment variable to inject (`KEY=VAL` or `KEY` to inherit from host). |
| `--json` | | `bool` | `false` | Emits execution telemetry and results in structured JSON format. |

#### Storage & Path Flags

| Flag | Type | Default | Description |
| :--- | :--- | :--- | :--- |
| `--cache-dir` | `string` | `~/.cache/ephemvm` | Directory where kernel, rootfs, and Firecracker binary are stored. |
| `--work-dir` | `string` | `/tmp` | Parent directory where temporary ephemeral overlays are allocated. |
| `--kernel` | `string` | `""` | Custom path to an uncompressed `vmlinux` binary (overrides cache). |
| `--rootfs` | `string` | `""` | Custom path to a base `rootfs.ext4` image (overrides cache). |
| `--firecracker-binary` | `string` | `""` | Path to Firecracker executable (defaults to checking cache or `PATH`). |
| `--kernel-args` | `string` | `""` | Custom boot arguments passed directly to the guest Linux kernel. |
| `--log-level` | `string` | `Error` | Firecracker VMM log level (`Error`, `Warning`, `Info`, `Debug`). |

---

### `ephemvm cache`

Manages local kernel, root filesystem, and Firecracker hypervisor assets.

```bash
ephemvm cache [status|pull|clean]
```

| Subcommand | Description |
| :--- | :--- |
| `ephemvm cache status` | Displays file size, permissions, and cryptographic SHA256 integrity verification for all cached assets. |
| `ephemvm cache pull` | Pre-downloads release assets and verifies their integrity without starting a microVM. Essential for air-gapped setups. |
| `ephemvm cache clean` | Deletes the `~/.cache/ephemvm` directory and all downloaded assets to free disk space. |

---

### `ephemvm version`

Displays detailed build and licensing information:

```bash
ephemvm version
```

Example Output:
```
ephemvm 0.1.1 (commit: 297cc5016e788bc5f2d9c490a187a55d491fbb4a, date: 2026-10-24T01:41:10Z)
Copyright (c) 2026 smford
Licensed under the MIT License
```

---

## Exit Code Contract

`ephemvm` preserves exact exit statuses to ensure compatibility with shell scripts, CI/CD runners, and build systems:

| Exit Code | Meaning | Description |
| :---: | :--- | :--- |
| `0` | Success | Guest command finished successfully with zero exit status. |
| `1 – 123` | Guest Command Failure | Direct exit status forwarded from the command executed inside the microVM. |
| `124` | Command Timeout | Execution exceeded the duration specified by `--timeout`. |
| `125` | Virtual Machine Failure | MicroVM failed to initialize, boot, or configure hardware resources. |
| `130` | Interrupted (`SIGINT`) | Execution was cancelled by `SIGINT` (e.g. user pressed `Ctrl+C`). |
| `143` | Terminated (`SIGTERM`) | MicroVM process received a termination signal from the host operating system. |

---

## JSON Telemetry Schema

When invoking `ephemvm` with `--json`, stdout and stderr are captured, and execution metrics are emitted as a single JSON object:

```json
{
  "command": "python3 -c \"print('Hello from microVM')\"",
  "exit_code": 0,
  "duration_ms": 42,
  "timed_out": false,
  "stdout": "Hello from microVM\n",
  "stderr": "",
  "vcpus": 1,
  "memory_mb": 128
}
```

### Field Definitions

| Field | Type | Description |
| :--- | :--- | :--- |
| `command` | `string` | The exact command or script string executed inside the guest. |
| `exit_code` | `integer` | Numerical exit code returned by the guest command. |
| `duration_ms` | `integer` | Total execution duration from boot to shutdown in milliseconds. |
| `timed_out` | `boolean` | `true` if execution was aborted due to `--timeout` expiration. |
| `stdout` | `string` | Standard output stream emitted by the guest command. |
| `stderr` | `string` | Standard error stream emitted by the guest command. |
| `vcpus` | `integer` | Number of dedicated virtual CPUs allocated to the run. |
| `memory_mb` | `integer` | Memory envelope allocated to the run in megabytes. |
