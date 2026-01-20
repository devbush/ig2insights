package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Defaults.Model != "small" {
		t.Errorf("Default model = %s, want small", cfg.Defaults.Model)
	}
	if cfg.Defaults.Format != "text" {
		t.Errorf("Default format = %s, want text", cfg.Defaults.Format)
	}
	if cfg.Defaults.CacheTTL != "7d" {
		t.Errorf("Default cache TTL = %s, want 7d", cfg.Defaults.CacheTTL)
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input    string
		wantSecs int64
		wantErr  bool
	}{
		{"24h", 86400, false},
		{"7d", 604800, false},
		{"30d", 2592000, false},
		{"1h", 3600, false},
		{"invalid", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			dur, err := ParseDuration(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseDuration(%s) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if err == nil && int64(dur.Seconds()) != tt.wantSecs {
				t.Errorf("ParseDuration(%s) = %v, want %d seconds", tt.input, dur, tt.wantSecs)
			}
		})
	}
}

func TestConfig_Save_Load(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	cfg := DefaultConfig()
	cfg.Defaults.Model = "large"

	err := cfg.Save(configPath)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loaded.Defaults.Model != "large" {
		t.Errorf("Loaded model = %s, want large", loaded.Defaults.Model)
	}
}

func TestAppDir(t *testing.T) {
	dir := AppDir()
	if dir == "" {
		t.Error("AppDir() returned empty string")
	}

	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".ig2insights")
	if dir != expected {
		t.Errorf("AppDir() = %s, want %s", dir, expected)
	}
}
