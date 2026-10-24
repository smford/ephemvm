package version

import (
	"fmt"
)

var (
	// Version holds the semantic version of ephemvm.
	Version = "0.1.0"

	// Commit holds the git commit hash injected at build time.
	Commit = "none"

	// Date holds the build timestamp injected at build time.
	Date = "unknown"

	// Copyright notice.
	Copyright = "Copyright (c) 2026 smford"

	// License notice.
	License = "Licensed under the MIT License"
)

// Formatted returns the multi-line standardized version string.
func Formatted() string {
	return fmt.Sprintf("ephemvm %s (commit: %s, date: %s)\n%s\n%s", Version, Commit, Date, Copyright, License)
}
