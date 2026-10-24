//go:build !linux

package vm

import (
	"context"
	"io"

	"github.com/smford/ephemvm/internal/config"
)

// CheckKVM is a stub on non-Linux platforms.
func CheckKVM() error {
	return ErrUnsupportedPlatform
}

// Run returns an error on non-Linux platforms.
func Run(ctx context.Context, cfg *config.Config, stdin io.Reader, stdout io.Writer, stderr io.Writer) (int, error) {
	return 1, ErrUnsupportedPlatform
}
