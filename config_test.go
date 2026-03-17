package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Startup.DefaultMode != "install" {
		t.Errorf("Expected default mode 'install', got %q", cfg.Startup.DefaultMode)
	}
	if cfg.Commands.AurHelper != "paru" {
		t.Errorf("Expected default helper 'paru', got %q", cfg.Commands.AurHelper)
	}
	if len(cfg.Keys.Quit) == 0 {
		t.Error("Expected default quit keys to be populated")
	}
}

func TestTokenizeFlags(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"--noconfirm", []string{"--noconfirm"}},
		{"-S --needed", []string{"-S", "--needed"}},
		{"  ", nil},
		{"", nil},
	}

	for _, tt := range tests {
		res := TokenizeFlags(tt.input)
		if len(res) != len(tt.expected) {
			t.Errorf("TokenizeFlags(%q) length = %d, want %d", tt.input, len(res), len(tt.expected))
			continue
		}
		for i := range res {
			if res[i] != tt.expected[i] {
				t.Errorf("TokenizeFlags(%q)[%d] = %q, want %q", tt.input, i, res[i], tt.expected[i])
			}
		}
	}
}

func TestConfigLoadPersistence(t *testing.T) {
	// Create a temporary directory for testing config
	tmpDir, err := os.MkdirTemp("", "gaur-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "config.toml")
	
	// Test Saving
	cfg := DefaultConfig()
	cfg.UI.Theme = "dracula"
	err = saveConfig(configPath, cfg)
	if err != nil {
		t.Errorf("Failed to save config: %v", err)
	}

	// Test Manual Loading (Simulating what LoadConfig does after path resolution)
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}

	if !testing.Short() {
		// Verify content contains dracula
		if !containsString(string(data), "dracula") {
			t.Error("Saved TOML does not contain the updated theme value")
		}
	}
}

func containsString(s, substr string) bool {
	return strings.Contains(s, substr)
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid helper yay",
			input:    "yay",
			expected: "yay",
		},
		{
			name:     "valid helper paru",
			input:    "paru",
			expected: "paru",
		},
		{
			name:     "unsupported helper fallback",
			input:    "apt",
			expected: "paru",
		},
		{
			name:     "blank helper fallback",
			input:    "",
			expected: "paru",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				Commands: CommandConfig{
					AurHelper: tt.input,
				},
			}
			ValidateConfig(config)
			if config.Commands.AurHelper != tt.expected {
				t.Errorf("ValidateConfig() set helper to %q, want %q", config.Commands.AurHelper, tt.expected)
			}
		})
	}
}
