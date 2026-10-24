package vm

import (
	"strings"
	"testing"

	"github.com/smford/ephemvm/internal/config"
)

func TestShellEscape(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", "''"},
		{"echo", "echo"},
		{"hello-world", "hello-world"},
		{"foo bar", "'foo bar'"},
		{"it's", `'it'\''s'`},
		{"path/to/file", "path/to/file"},
		{"$VAR", "'$VAR'"},
	}

	for _, tt := range tests {
		got := ShellEscape(tt.input)
		if got != tt.want {
			t.Errorf("ShellEscape(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestShellEscapeArgs(t *testing.T) {
	args := []string{"echo", "hello world", "test"}
	got := ShellEscapeArgs(args)
	want := "echo 'hello world' test"
	if got != want {
		t.Errorf("ShellEscapeArgs(%v) = %q, want %q", args, got, want)
	}
}

func TestBuildBootArgs(t *testing.T) {
	cfg := config.NewDefaultConfig()
	cfg.Args = []string{"echo", "hello"}

	bootArgs, err := BuildBootArgs(cfg, "amd64", nil)
	if err != nil {
		t.Fatalf("BuildBootArgs failed: %v", err)
	}

	if !strings.Contains(bootArgs, "console=ttyS0") {
		t.Errorf("expected console=ttyS0 in boot args, got: %s", bootArgs)
	}
	if !strings.Contains(bootArgs, "reboot=k") {
		t.Errorf("expected reboot=k in boot args, got: %s", bootArgs)
	}
	if !strings.Contains(bootArgs, "panic=1") {
		t.Errorf("expected panic=1 in boot args, got: %s", bootArgs)
	}
	if !strings.Contains(bootArgs, "init=/bin/sh -- -c") {
		t.Errorf("expected init=/bin/sh in boot args, got: %s", bootArgs)
	}
	if !strings.Contains(bootArgs, "__EXIT_CODE:$?__") {
		t.Errorf("expected sentinel pattern in boot args, got: %s", bootArgs)
	}
	if !strings.Contains(bootArgs, "reboot -f") {
		t.Errorf("expected reboot -f in boot args, got: %s", bootArgs)
	}
}

func TestBuildBootArgsWithEnvAndScript(t *testing.T) {
	cfg := config.NewDefaultConfig()
	cfg.Env = []string{"API_KEY=secret-token", "DEBUG=1"}
	cfg.ScriptPath = "-"

	stdin := strings.NewReader("echo 'script running'; uname -a")
	bootArgs, err := BuildBootArgs(cfg, "amd64", stdin)
	if err != nil {
		t.Fatalf("BuildBootArgs failed: %v", err)
	}

	if !strings.Contains(bootArgs, "export API_KEY=secret-token") {
		t.Errorf("expected API_KEY in boot args, got: %s", bootArgs)
	}
	if !strings.Contains(bootArgs, "export DEBUG=1") {
		t.Errorf("expected DEBUG in boot args, got: %s", bootArgs)
	}
	if !strings.Contains(bootArgs, "base64 -d") {
		t.Errorf("expected base64 decoding for script in boot args, got: %s", bootArgs)
	}
}

func TestBuildBootArgsCustom(t *testing.T) {
	cfg := config.NewDefaultConfig()
	cfg.KernelArgs = "console=ttyS0 custom_flag=1"
	cfg.Args = []string{"ls"}

	bootArgs, err := BuildBootArgs(cfg, "amd64", nil)
	if err != nil {
		t.Fatalf("BuildBootArgs failed: %v", err)
	}

	if !strings.HasPrefix(bootArgs, cfg.KernelArgs) {
		t.Errorf("expected custom prefix in boot args, got: %s", bootArgs)
	}
}
