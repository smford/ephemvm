# Worked Examples

Real-world scenarios and common tasks using `ephemvm`.

---

## 1. AI Coding Agent Sandbox

Give autonomous AI coding assistants (such as Claude Code, Cursor, Antigravity, or local LLMs) real bash tool execution without risking your local workstation, developer configuration, or files:

```bash
ephemvm run -- python3 -c "
import urllib.request
print('Checking external connectivity inside microVM...')
"
```

The agent can format disks, install packages, and test code, yet nothing can escape the microVM.

---

## 2. Destructive Teardown Testing

Verify cleanup, teardown, and uninstall automation scripts safely:

```bash
ephemvm run -- sh -c "
echo 'Beginning destructive disk test...'
rm -rf --no-preserve-root / 2>/dev/null
echo 'Command completed!'
"
```

Because the base image is strictly read-only and mutations only touch the ephemeral CoW overlay in `/tmp`, your host filesystem is 100% safe.

---

## 3. Untrusted Install Scripts

Before piping random scripts from the internet into your host machine, inspect their behavior in an isolated environment:

```bash
ephemvm run -- sh -c "curl -fsSL https://example.com/install.sh | sh"
```

---

## 4. Multi-line Local Scripts

Run complex multi-line bash or python scripts with arguments without CLI escaping issues:

```bash
# Execute local script
ephemvm run -s ./scripts/deploy-test.sh -- --verbose --dry-run
```

Or pipe scripts directly from standard input:

```bash
cat << 'EOF' | ephemvm run -s -
echo "=== MicroVM Diagnostic Info ==="
uname -a
cat /proc/cpuinfo | grep 'model name' | head -n 1
free -m
EOF
```

---

## 5. Environment Variables

Inject host secrets or test parameters into the guest:

```bash
# Explicit key-value pair
ephemvm run -e API_KEY=sk_test_12345 -- env

# Inherit from host environment
export REGISTRY_TOKEN="token-xyz"
ephemvm run -e REGISTRY_TOKEN -- sh -c 'echo "Token is set with length: ${#REGISTRY_TOKEN}"'
```

---

## 6. Cache Pre-warming

Verify release assets before running without internet access:

```bash
# Check status
ephemvm cache status

# Pre-download and verify SHA256 checksums
ephemvm cache pull

# Purge cache when cleaning disk space
ephemvm cache clean
```

---

## 7. Automated CI/CD with JSON

Integrate `ephemvm` into automated testing pipelines and retrieve structured JSON metrics:

```bash
ephemvm run --json -- sh -c "echo 'all tests pass'; exit 0"
```

Output:
```json
{
  "exit_code": 0,
  "duration_ms": 74,
  "timed_out": false,
  "stdout": "all tests pass\n",
  "stderr": "",
  "vcpus": 1,
  "memory_mb": 128
}
```

---

## 8. Resource Limits & Hard Timeouts

Constrain microVM resource consumption and protect against infinite loops:

```bash
# Allocate 2 vCPUs, 256MB RAM, and enforce a 5-second hard timeout
ephemvm run --vcpus 2 --memory 256 --timeout 5s -- sh -c "sleep 10"
```

The microVM is cleanly terminated at 5 seconds and returns exit code `124`.
