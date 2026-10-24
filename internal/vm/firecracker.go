package vm

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/smford/ephemvm/internal/config"
)

// ErrUnsupportedPlatform indicates that ephemvm requires Linux with KVM.
var ErrUnsupportedPlatform = errorsNew("ephemvm requires Linux with KVM hardware virtualization (/dev/kvm)")

func errorsNew(text string) error {
	return &customError{text: text}
}

type customError struct {
	text string
}

func (e *customError) Error() string {
	return e.text
}

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

// BuildGuestScript constructs the guest command string including environment exports and scripts.
func BuildGuestScript(cfg *config.Config, stdin io.Reader) (string, error) {
	var envBuilder strings.Builder
	for _, envVar := range cfg.Env {
		var key, val string
		if idx := strings.Index(envVar, "="); idx != -1 {
			key = envVar[:idx]
			val = envVar[idx+1:]
		} else {
			key = envVar
			val = os.Getenv(key)
		}
		if key != "" {
			envBuilder.WriteString(fmt.Sprintf("export %s=%s; ", key, ShellEscape(val)))
		}
	}

	var cmdStr string
	if cfg.ScriptPath != "" {
		var scriptBytes []byte
		if cfg.ScriptPath == "-" {
			if stdin == nil {
				return "", fmt.Errorf("stdin is required when script path is '-'")
			}
			var err error
			scriptBytes, err = io.ReadAll(stdin)
			if err != nil {
				return "", fmt.Errorf("failed to read script from stdin: %w", err)
			}
		} else {
			var err error
			scriptBytes, err = os.ReadFile(cfg.ScriptPath)
			if err != nil {
				return "", fmt.Errorf("failed to read script file %s: %w", cfg.ScriptPath, err)
			}
		}
		encoded := base64.StdEncoding.EncodeToString(scriptBytes)
		cmdArgs := ShellEscapeArgs(cfg.Args)
		if cmdArgs != "" {
			cmdStr = fmt.Sprintf("echo %s | base64 -d > /tmp/run.sh && chmod +x /tmp/run.sh && /bin/sh /tmp/run.sh %s", encoded, cmdArgs)
		} else {
			cmdStr = fmt.Sprintf("echo %s | base64 -d > /tmp/run.sh && chmod +x /tmp/run.sh && /bin/sh /tmp/run.sh", encoded)
		}
	} else {
		cmdStr = ShellEscapeArgs(cfg.Args)
	}

	fullScript := fmt.Sprintf("%s%s; echo %s$?%s; reboot -f", envBuilder.String(), cmdStr, sentinelPrefix, sentinelSuffix)
	return fullScript, nil
}

// BuildBootArgs formats the kernel command line parameters for a Config.
func BuildBootArgs(cfg *config.Config, arch string, stdin io.Reader) (string, error) {
	var baseArgs string
	if cfg.KernelArgs != "" {
		baseArgs = cfg.KernelArgs
	} else {
		// Default fast-boot arguments
		baseArgs = "console=ttyS0 reboot=k panic=1 quiet loglevel=1 pci=off"
		if arch == "arm64" {
			baseArgs = "keep_bootcon " + baseArgs
		}
	}

	guestScript, err := BuildGuestScript(cfg, stdin)
	if err != nil {
		return "", err
	}

	// Wrap in double quotes for init=/bin/sh -- -c "..."
	// Escape any backslashes and double quotes
	escapedScript := strings.ReplaceAll(guestScript, `\`, `\\`)
	escapedScript = strings.ReplaceAll(escapedScript, `"`, `\"`)

	return fmt.Sprintf("%s init=/bin/sh -- -c \"%s\"", baseArgs, escapedScript), nil
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

	// Check standard install locations and cache directory
	homeDir, _ := os.UserHomeDir()
	candidates := []string{
		filepath.Join(homeDir, ".cache", "ephemvm", "bin", "firecracker"),
		"/usr/local/bin/firecracker",
		"/usr/bin/firecracker",
		filepath.Join(homeDir, ".local", "bin", "firecracker"),
	}

	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c, nil
		}
	}

	return "", fmt.Errorf("firecracker binary not found in PATH or standard paths; please install firecracker or specify --firecracker-binary")
}
