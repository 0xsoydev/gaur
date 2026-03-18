package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetAURCacheDirXDG(t *testing.T) {
	// 1. Test XDG_CACHE_HOME override
	oldXDG := os.Getenv("XDG_CACHE_HOME")
	mockXDG := "/tmp/gaur-test-cache"
	os.Setenv("XDG_CACHE_HOME", mockXDG)
	defer os.Setenv("XDG_CACHE_HOME", oldXDG)

	config := &Config{
		Commands: CommandConfig{AurHelper: "yay"},
	}

	expected := filepath.Join(mockXDG, "yay")
	got, err := GetAURCacheDir(config)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if got != expected {
		t.Errorf("GetAURCacheDir() with XDG_CACHE_HOME = %q, want %q", got, expected)
	}

	// 2. Test paru specific path under XDG
	config.Commands.AurHelper = "paru"
	expected = filepath.Join(mockXDG, "paru", "clone")
	got, _ = GetAURCacheDir(config)
	if got != expected {
		t.Errorf("GetAURCacheDir() for paru with XDG_CACHE_HOME = %q, want %q", got, expected)
	}

	// 3. Test explicit config override (should trump XDG)
	customPath := "/custom/path"
	config.Advanced.CacheDir = customPath
	got, _ = GetAURCacheDir(config)
	if got != customPath {
		t.Errorf("GetAURCacheDir() with explicit config = %q, want %q", got, customPath)
	}
}
