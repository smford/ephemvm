package config

import (
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := NewDefaultConfig()
	if cfg.Timeout != DefaultTimeout {
		t.Fatalf("expected timeout %v, got %v", DefaultTimeout, cfg.Timeout)
	}
	if cfg.VCPUs != DefaultVCPUs {
		t.Fatalf("expected vcpus %d, got %d", DefaultVCPUs, cfg.VCPUs)
	}
	if cfg.MemoryMB != DefaultMemoryMB {
		t.Fatalf("expected memory %d MB, got %d MB", DefaultMemoryMB, cfg.MemoryMB)
	}
	if cfg.CacheDir == "" {
		t.Fatal("expected non-empty cache dir")
	}
	if cfg.WorkDir == "" {
		t.Fatal("expected non-empty work dir")
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(cfg *Config)
		wantErr bool
	}{
		{
			name: "valid config",
			modify: func(cfg *Config) {
				cfg.Args = []string{"echo", "hello"}
			},
			wantErr: false,
		},
		{
			name: "empty args",
			modify: func(cfg *Config) {
				cfg.Args = nil
			},
			wantErr: true,
		},
		{
			name: "zero timeout",
			modify: func(cfg *Config) {
				cfg.Args = []string{"ls"}
				cfg.Timeout = 0
			},
			wantErr: true,
		},
		{
			name: "negative timeout",
			modify: func(cfg *Config) {
				cfg.Args = []string{"ls"}
				cfg.Timeout = -5 * time.Second
			},
			wantErr: true,
		},
		{
			name: "zero vcpus",
			modify: func(cfg *Config) {
				cfg.Args = []string{"ls"}
				cfg.VCPUs = 0
			},
			wantErr: true,
		},
		{
			name: "insufficient memory",
			modify: func(cfg *Config) {
				cfg.Args = []string{"ls"}
				cfg.MemoryMB = 32
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := NewDefaultConfig()
			tt.modify(cfg)
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}
