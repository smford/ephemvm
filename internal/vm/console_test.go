package vm

import (
	"bytes"
	"fmt"
	"testing"
)

func TestConsoleFilterNormalOutput(t *testing.T) {
	var out bytes.Buffer
	filter := NewConsoleFilter(&out)

	input := "hello world\nthis is a normal command output\n"
	n, err := filter.Write([]byte(input))
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != len(input) {
		t.Fatalf("expected n=%d, got %d", len(input), n)
	}
	if err := filter.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	if out.String() != input {
		t.Fatalf("expected %q, got %q", input, out.String())
	}

	code, found := filter.ExitCode()
	if found || code != -1 {
		t.Fatalf("expected no exit code, got (%d, %v)", code, found)
	}
}

func TestConsoleFilterWithExitCodeZero(t *testing.T) {
	var out bytes.Buffer
	filter := NewConsoleFilter(&out)

	input := "guest boot ok\ncommand finished successfully\n__EXIT_CODE:0__\n"
	_, err := filter.Write([]byte(input))
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if err := filter.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	expectedOut := "guest boot ok\ncommand finished successfully\n"
	if out.String() != expectedOut {
		t.Fatalf("expected %q, got %q", expectedOut, out.String())
	}

	code, found := filter.ExitCode()
	if !found {
		t.Fatal("expected exit code to be found")
	}
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestConsoleFilterWithNonZeroExitCode(t *testing.T) {
	var out bytes.Buffer
	filter := NewConsoleFilter(&out)

	input := "sh: command not found: foobar\n__EXIT_CODE:127__\r\n"
	_, err := filter.Write([]byte(input))
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if err := filter.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	expectedOut := "sh: command not found: foobar\n"
	if out.String() != expectedOut {
		t.Fatalf("expected %q, got %q", expectedOut, out.String())
	}

	code, found := filter.ExitCode()
	if !found {
		t.Fatal("expected exit code to be found")
	}
	if code != 127 {
		t.Fatalf("expected exit code 127, got %d", code)
	}
}

func TestConsoleFilterChunkedSplitAcrossSentinel(t *testing.T) {
	var out bytes.Buffer
	filter := NewConsoleFilter(&out)

	chunks := []string{
		"output line 1\n",
		"output line 2: __EX",
		"IT_CO",
		"DE:4",
		"2__",
		"\nmore post-execution data\n",
	}

	for _, c := range chunks {
		if _, err := filter.Write([]byte(c)); err != nil {
			t.Fatalf("Write failed: %v", err)
		}
	}
	if err := filter.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	expectedOut := "output line 1\noutput line 2: more post-execution data\n"
	if out.String() != expectedOut {
		t.Fatalf("expected %q, got %q", expectedOut, out.String())
	}

	code, found := filter.ExitCode()
	if !found {
		t.Fatal("expected exit code to be found")
	}
	if code != 42 {
		t.Fatalf("expected exit code 42, got %d", code)
	}
}

func TestConsoleFilterSingleByteStreaming(t *testing.T) {
	var out bytes.Buffer
	filter := NewConsoleFilter(&out)

	input := "streaming test\n__EXIT_CODE:1__\n"
	for i := 0; i < len(input); i++ {
		if _, err := filter.Write([]byte{input[i]}); err != nil {
			t.Fatalf("byte write failed: %v", err)
		}
	}
	if err := filter.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	expectedOut := "streaming test\n"
	if out.String() != expectedOut {
		t.Fatalf("expected %q, got %q", expectedOut, out.String())
	}

	code, found := filter.ExitCode()
	if !found || code != 1 {
		t.Fatalf("expected exit code (1, true), got (%d, %v)", code, found)
	}
}

func TestConsoleFilterFalsePositivePrefix(t *testing.T) {
	var out bytes.Buffer
	filter := NewConsoleFilter(&out)

	// Output containing characters that match prefixes of the sentinel but aren't the sentinel
	input := "__EXTRA_VARIABLE=true\n__EXIT_WITHOUT_CODE\n"
	if _, err := filter.Write([]byte(input)); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if err := filter.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	if out.String() != input {
		t.Fatalf("expected %q, got %q", input, out.String())
	}

	code, found := filter.ExitCode()
	if found {
		t.Fatalf("unexpected exit code found: %d", code)
	}
}

func TestFindPartialPrefixMatch(t *testing.T) {
	tests := []struct {
		data   string
		target string
		want   int
	}{
		{"hello", "__EXIT_CODE:", 0},
		{"hello_", "__EXIT_CODE:", 1},
		{"hello__", "__EXIT_CODE:", 2},
		{"hello__E", "__EXIT_CODE:", 3},
		{"hello__EXIT_CODE", "__EXIT_CODE:", 11},
		{"hello__EXIT_CODE:", "__EXIT_CODE:", 0}, // full match, not strictly a partial prefix
		{"", "__EXIT_CODE:", 0},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_%s", tt.data, tt.target), func(t *testing.T) {
			got := findPartialPrefixMatch([]byte(tt.data), tt.target)
			if got != tt.want {
				t.Errorf("findPartialPrefixMatch(%q, %q) = %d, want %d", tt.data, tt.target, got, tt.want)
			}
		})
	}
}
