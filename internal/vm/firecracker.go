package vm

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ErrUnsupportedPlatform indicates that ephemvm requires Linux with KVM.
var ErrUnsupportedPlatform = errors.New("ephemvm requires Linux with KVM hardware virtualization (/dev/kvm)")


// ShellEscapeArgs escapes and concatenates arguments for POSIX /bin/sh.
func ShellEscapeArgs(args []string) string {
	escaped := make([]string, len(args))
	for i, a := range args {
		escaped[i] = ShellEscape(a)
	}
	return strings.Join(escaped, " ")
}

// ShellEscape safely escapes a single string argument for POSIX shells.
func ShellEscape(s string) string {
	if len(s) == 0 {
		return "''"
	}
	safe := true
	for _, r := range s {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') ||
			r == '-' || r == '_' || r == '.' || r == '/' || r == ':' || r == '=' || r == '@') {
			safe = false
			break
		}
	}
	if safe {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// BuildBootArgs formats the kernel command line parameters.
// Format: console=ttyS0 reboot=k panic=1 init=/bin/sh -- -c "<cmd>; echo __EXIT_CODE:$?__; reboot -f"
func BuildBootArgs(customArgs string, cmdArgs []string, arch string) string {
	var baseArgs string
	if customArgs != "" {
		baseArgs = customArgs
	} else {
		// Default fast-boot arguments
		baseArgs = "console=ttyS0 reboot=k panic=1 quiet loglevel=1 pci=off"
		if arch == "arm64" {
			baseArgs = "keep_bootcon " + baseArgs
		}
	}

	escapedCmd := ShellEscapeArgs(cmdArgs)
	guestScript := fmt.Sprintf("%s; echo %s$?%s; reboot -f", escapedCmd, sentinelPrefix, sentinelSuffix)

	// Wrap in double quotes for init=/bin/sh -- -c "..."
	// Escape any double quotes or backslashes
	escapedScript := strings.ReplaceAll(guestScript, `\`, `\\`)
	escapedScript = strings.ReplaceAll(escapedScript, `"`, `\"`)

	return fmt.Sprintf("%s init=/bin/sh -- -c \"%s\"", baseArgs, escapedScript)
}

// FindFirecrackerBinary resolves the path to the firecracker binary.
func FindFirecrackerBinary(customPath string) (string, error) {
	if customPath != "" {
		if _, err := os.Stat(customPath); err != nil {
			return "", fmt.Errorf("specified firecracker binary not found: %s", customPath)
		}
		return customPath, nil
	}

	// Check PATH
	if p, err := exec.LookPath("firecracker"); err == nil {
		return p, nil
	}

	// Check standard install locations
	candidates := []string{
		"/usr/local/bin/firecracker",
		"/usr/bin/firecracker",
		filepath.Join(os.Getenv("HOME"), ".local", "bin", "firecracker"),
	}

	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c, nil
		}
	}

	return "", fmt.Errorf("firecracker binary not found in PATH or standard paths; please install firecracker or specify --firecracker-binary")
}
