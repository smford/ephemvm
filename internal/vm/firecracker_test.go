package vm

import (
	"strings"
	"testing"
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
	args := []string{"echo", "hello"}
	bootArgs := BuildBootArgs("", args, "amd64")

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

func TestBuildBootArgsCustom(t *testing.T) {
	custom := "console=ttyS0 custom_flag=1"
	bootArgs := BuildBootArgs(custom, []string{"ls"}, "amd64")

	if !strings.HasPrefix(bootArgs, custom) {
		t.Errorf("expected custom prefix in boot args, got: %s", bootArgs)
	}
}
