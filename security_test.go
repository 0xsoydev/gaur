package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// mockRunner for capturing interactive command arguments
type mockRunner struct {
	RealCommandRunner
	capturedArgs []string
}

func (m *mockRunner) Interactive(onExit func(error) tea.Msg, name string, args ...string) tea.Cmd {
	m.capturedArgs = append([]string{name}, args...)
	return func() tea.Msg { return nil }
}

func TestSecurityFixes(t *testing.T) {
	t.Run("ValidateConfig_CacheTool", func(t *testing.T) {
		cfg := &Config{
			Commands: CommandConfig{
				CacheTool: "rm -rf /",
			},
		}
		ValidateConfig(cfg)
		if cfg.Commands.CacheTool != "paccache" {
			t.Errorf("Expected CacheTool to be reset to 'paccache', got %q", cfg.Commands.CacheTool)
		}
	})

	t.Run("ValidateConfig_CacheDir_Normalization", func(t *testing.T) {
		cfg := &Config{
			Advanced: AdvancedConfig{
				CacheDir: "/tmp/gaur/../gaur",
			},
		}
		ValidateConfig(cfg)
		expected := filepath.Clean("/tmp/gaur/../gaur")
		if cfg.Advanced.CacheDir != expected {
			t.Errorf("Expected CacheDir to be normalized to %q, got %q", expected, cfg.Advanced.CacheDir)
		}
	})

	t.Run("FilePermissions_SaveConfig", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "gaur-security-test-*")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpDir)

		configPath := filepath.Join(tmpDir, "config.toml")
		cfg := DefaultConfig()
		
		err = saveConfig(configPath, cfg)
		if err != nil {
			t.Fatalf("saveConfig failed: %v", err)
		}

		info, err := os.Stat(configPath)
		if err != nil {
			t.Fatal(err)
		}

		// On Linux, 0600 is what we expect.
		mode := info.Mode().Perm()
		if mode != 0600 {
			t.Errorf("Expected config file permissions 0600, got %o", mode)
		}
	})

	t.Run("ExecuteCleanCache_NoInterpolation", func(t *testing.T) {
		// Save old runner and restore after test
		oldRunner := runner
		mRunner := &mockRunner{}
		runner = mRunner
		defer func() { runner = oldRunner }()

		m := &model{
			config: Config{
				Commands: CommandConfig{
					CacheTool: "paccache",
				},
				Advanced: AdvancedConfig{
					CacheDir: "/tmp/malicious; rm -rf /",
				},
			},
		}

		// Trigger the command
		cmd := executeCleanCache(m, confirmCleanKeep3, 3, false)
		if cmd == nil {
			t.Fatal("executeCleanCache returned nil command")
		}
		// Execute the tea.Cmd to trigger our mock runner
		cmd()

		// Verify that the malicious string is passed as an argument, NOT interpolated into the script
		if len(mRunner.capturedArgs) < 5 {
			t.Fatalf("Expected at least 5 arguments for bash -c, got %v", mRunner.capturedArgs)
		}

		script := mRunner.capturedArgs[2]
		if strings.Contains(script, "/tmp/malicious; rm -rf /") {
			t.Error("CRITICAL: Malicious string was interpolated into the shell script!")
		}

		// Verify it was passed as an argument
		found := false
		for i, arg := range mRunner.capturedArgs {
			if i > 2 && arg == "/tmp/malicious; rm -rf /" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Malicious string was not found in the arguments list")
		}
	})
}
