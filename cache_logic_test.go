package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetAURCacheDir(t *testing.T) {
	// Mock HOME for predictable testing
	oldHome := os.Getenv("HOME")
	mockHome := "/home/testuser"
	os.Setenv("HOME", mockHome)
	defer os.Setenv("HOME", oldHome)

	tests := []struct {
		name     string
		helper   string
		custom   string
		expected string
	}{
		{
			name:     "paru default cache",
			helper:   "paru",
			custom:   "",
			expected: filepath.Join(mockHome, ".cache", "paru", "clone"),
		},
		{
			name:     "yay default cache",
			helper:   "yay",
			custom:   "",
			expected: filepath.Join(mockHome, ".cache", "yay"),
		},
		{
			name:     "custom cache override",
			helper:   "paru",
			custom:   "/custom/path",
			expected: "/custom/path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				Commands: CommandConfig{
					AurHelper: tt.helper,
				},
				Advanced: AdvancedConfig{
					CacheDir: tt.custom,
				},
			}
			got, _ := GetAURCacheDir(config)
			if got != tt.expected {
				t.Errorf("GetAURCacheDir() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestParsePaccacheDryRun(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected string
	}{
		{
			name:     "no candidates",
			output:   "==> no candidate packages found for pruning",
			expected: "0 B",
		},
		{
			name:     "success mb",
			output:   "\n==> finished dry run: 78 candidates (disk space saved: 782.43 MiB)\n",
			expected: "782.43 MiB",
		},
		{
			name:     "success kb",
			output:   "\n==> finished dry run: 1 candidates (disk space saved: 141.83 KiB)\n",
			expected: "141.83 KiB",
		},
		{
			name:     "empty output",
			output:   "",
			expected: "0 B",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := parsePaccacheDryRun(tt.output)
			if actual != tt.expected {
				t.Errorf("parsePaccacheDryRun() = %v, want %v", actual, tt.expected)
			}
		})
	}
}
