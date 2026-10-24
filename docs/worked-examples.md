# Worked Examples

Explore real-world scenarios and production-tested patterns using `ephemvm`.

---

## 1. Autonomous AI Coding Agent Sandbox

**The Problem:** Modern AI developer agents (such as Claude Code, Cursor, AutoGPT, or local LLMs through MCP tools) require terminal execution privileges to run tests, compile code, and inspect systems. Giving an autonomous agent direct shell access on your developer laptop or production server carries immense risk.

**The Solution:** Route all agent tool calls through `ephemvm`. The agent receives real Linux execution with full root privileges inside the microVM, but cannot access your workstation, files, SSH keys, or host kernel.

![AI Agent Sandbox Preview](assets/screenshots/ai-agent-terminal.svg)

```bash
# Agent runs Python scripts and inspects network or system info
ephemvm run --json --vcpus 2 --memory 256 -- python3 -c "
import socket, os
print(f'Isolated Guest: {socket.gethostname()}, PID: {os.getpid()}')
"
```

The CLI returns machine-readable JSON telemetry directly to the agent runtime:

```json
{
  "command": "python3 -c \"import socket, os; print(f'Isolated Guest: {socket.gethostname()}, PID: {os.getpid()}')\"",
  "exit_code": 0,
  "duration_ms": 47,
  "timed_out": false,
  "vcpus": 2,
  "memory_mb": 256,
  "stdout": "Isolated Guest: ephemvm-guest, PID: 1\n",
  "stderr": ""
}
```

---

## 2. Destructive Teardown & Uninstall Testing

**The Problem:** Testing system cleaners, uninstallation scripts, or destructive recovery routines locally can accidentally wipe files or corrupt system state.

**The Solution:** Execute destructive actions inside `ephemvm`. Mutations are written only to the ephemeral copy-on-write overlay. Even wiping the entire filesystem has zero effect on the host machine or the base rootfs image.

![Destructive Teardown Preview](assets/screenshots/destructive-terminal.svg)

```bash
ephemvm "rm -rf --no-preserve-root / 2>&1; ls -la /"
```

Output:
```
rm: cannot remove '/proc/1': Operation not permitted
rm: cannot remove '/sys/fs/cgroup': Operation not permitted
ls: /: No such file or directory
[ephemvm] guest exited with status 1
[ephemvm] overlay /tmp/ephemvm-9a3d discarded completely
[ephemvm] base rootfs ~/.cache/ephemvm/rootfs.ext4 SHA256 verified untouched
```

---

## 3. Untrusted Script & Malware Detonation

**The Problem:** Modern software commonly instructs users to execute `curl -fsSL https://... | sh`. Even with trusted vendors, compromised CDNs or typo-squatted URLs can lead to machine compromise.

**The Solution:** Detonate and inspect third-party installers inside hardware isolation before allowing them near your workstation:

```bash
ephemvm run -- sh -c "
echo '=== Downloading and inspecting installer ==='
curl -fsSL https://example.com/untrusted-install.sh -o /tmp/installer.sh
head -n 20 /tmp/installer.sh
sh /tmp/installer.sh --dry-run
"
```

Once the test completes, the microVM evaporates and all downloaded files vanish without a trace.

---

## 4. Multi-Line Scripts & Standard Input Piping

**The Problem:** Running complex bash scripts inside Docker or virtual machines often requires managing temporary container volumes, mounting host directories, or dealing with nested shell escaping quotes.

**The Solution:** Pass a local script file with `-s / --script`, or pipe scripts directly from standard input using `-s -`:

### Option A: Execute Local Script File
```bash
ephemvm run -s ./scripts/verify-environment.sh -- --verbose --dry-run
```

### Option B: Pipe via Heredoc
```bash
cat << 'EOF' | ephemvm run -s -
#!/bin/sh
set -e

echo "=== System Diagnostic Report ==="
echo "Kernel: $(uname -r)"
echo "Uptime: $(uptime)"
echo "Memory Available:"
free -m
echo "Disk Usage:"
df -h /
EOF
```

---

## 5. Secret & Host Environment Variable Injection

**The Problem:** You need to test code that requires specific configuration flags or API tokens, but you do not want to bake secrets into image templates.

**The Solution:** Inject variables explicitly using `-e KEY=VAL`, or inherit them dynamically from your host shell using `-e KEY`:

```bash
# Inject explicit value
ephemvm run -e API_ENDPOINT="https://api.internal.local" -- env | grep API_

# Inherit existing host variable
export CI_JOB_TOKEN="secret-build-token-998"
ephemvm run -e CI_JOB_TOKEN -- sh -c 'echo "Token length is: ${#CI_JOB_TOKEN}"'
```

---

## 6. Air-Gapped Environments & Cache Management

**The Problem:** CI runners and production servers in air-gapped or restricted networks cannot download dependencies at runtime.

**The Solution:** Pre-warm the `ephemvm` asset cache beforehand. Once cached, `ephemvm` runs with zero internet connectivity:

![Cache Status Preview](assets/screenshots/cache-terminal.svg)

```bash
# 1. Inspect cache status and checksums
ephemvm cache status

# 2. Pre-fetch and cryptographically verify all assets
ephemvm cache pull

# 3. Clean up cache when reclaiming disk space
ephemvm cache clean
```

---

## 7. High-Throughput Automated CI/CD Pipelines

**The Problem:** Running integration tests across hundreds of commits can quickly exhaust Docker daemon resources, accumulate dangling volumes, or hit container startup bottlenecks.

**The Solution:** Use `ephemvm` with the `--json` flag to execute unit or integration tests in parallel with sub-50ms overhead:

```bash
#!/usr/bin/env bash
set -e

RESULT=$(ephemvm run --json --timeout 15s -- ./run-test-suite.sh)

EXIT_CODE=$(echo "$RESULT" | jq .exit_code)
DURATION=$(echo "$RESULT" | jq .duration_ms)

if [ "$EXIT_CODE" -eq 0 ]; then
  echo "✅ Tests passed in ${DURATION}ms"
else
  echo "❌ Tests failed with exit code $EXIT_CODE"
  echo "$RESULT" | jq -r .stdout
  exit 1
fi
```

---

## 8. Resource Enforcement & Hard Deadlines

**The Problem:** Rogue scripts with infinite loops or fork bombs can lock up host CPU cores or consume all available memory.

**The Solution:** Enforce strict hardware constraints and hard execution timeouts:

```bash
# Limit to 2 vCPUs, 256MB RAM, and abort if not finished in 5 seconds
ephemvm run --vcpus 2 --memory 256 --timeout 5s -- sh -c "
echo 'Simulating long-running workload...'
sleep 30
echo 'This line will never be reached'
"
```

Output:
```
Simulating long-running workload...
Error: microvm execution timed out after 5s
```

`ephemvm` automatically issues a graceful shutdown, escalates to `SIGKILL` after 500ms if needed, purges the temporary overlay, and exits with standard timeout exit code `124`.
