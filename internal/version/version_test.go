package version

import (
	"strings"
	"testing"
)

func TestFormatted(t *testing.T) {
	v := Formatted()
	if !strings.HasPrefix(v, "ephemvm 0.1.0") {
		t.Errorf("expected version prefix, got: %s", v)
	}
	if !strings.Contains(v, "Copyright (c) 2026 smford") {
		t.Errorf("expected copyright notice, got: %s", v)
	}
	if !strings.Contains(v, "Licensed under the MIT License") {
		t.Errorf("expected MIT License notice, got: %s", v)
	}
}
