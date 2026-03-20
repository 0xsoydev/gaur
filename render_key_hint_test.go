package main

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func TestRenderKeyHint(t *testing.T) {
	// Initialize lipgloss for deterministic output in tests
	lipgloss.SetColorProfile(termenv.Ascii)
	style := lipgloss.NewStyle()

	tests := []struct {
		name     string
		label    string
		key      string
		expected string
	}{
		{
			name:     "key exists in label (middle)",
			label:    "dash",
			key:      "a",
			expected: "d[a]sh",
		},
		{
			name:     "key exists in label (start)",
			label:    "dash",
			key:      "d",
			expected: "[d]ash",
		},
		{
			name:     "key exists in label (end)",
			label:    "dash",
			key:      "h",
			expected: "das[h]",
		},
		{
			name:     "key does not exist in label",
			label:    "dash",
			key:      "x",
			expected: "[x]:dash",
		},
		{
			name:     "key exists in label (case insensitive)",
			label:    "Dash",
			key:      "d",
			expected: "[D]ash",
		},
		{
			name:     "multi-character key (no match)",
			label:    "quit",
			key:      "ctrl+c",
			expected: "[^c]:quit",
		},
		{
			name:     "special key (no match)",
			label:    "mark",
			key:      "tab",
			expected: "[tab]:mark",
		},
		{
			name:     "key is multiple characters, no embedding even if substring exists",
			label:    "install",
			key:      "ins",
			expected: "[ins]:install",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			binding := key.NewBinding(key.WithKeys(tt.key))
			got := renderKeyHint(tt.label, binding, style)
			// Strip any remaining ANSI if any (though Ascii profile should minimize them)
			cleanGot := stripAnsi(got)
			if cleanGot != tt.expected {
				t.Errorf("renderKeyHint(%q, %q) = %q, want %q", tt.label, tt.key, cleanGot, tt.expected)
			}
		})
	}
}

func TestConfigKeyReflectionInView(t *testing.T) {
	lipgloss.SetColorProfile(termenv.Ascii)
	cfg := DefaultConfig()
	// Change some keys
	cfg.Keys.InstallMode = []string{"x"}
	cfg.Keys.DashboardMode = []string{"v"} // 'v' is in 'dash'
	cfg.Keys.Quit = []string{"ctrl+q"}

	m := initialModel(modeInstall, cfg)
	view := stripAnsi(m.renderHelpText(lipgloss.Color("7")))

	if !strings.Contains(view, "[x]:install") {
		t.Errorf("Expected [x]:install in help text for changed key, got %q", view)
	}
	if !strings.Contains(view, "dash") || !strings.Contains(view, "[v]") {
		// Wait, 'v' is NOT in 'dash'. It should be [v]:dash
		if !strings.Contains(view, "[v]:dash") {
			t.Errorf("Expected [v]:dash in help text, got %q", view)
		}
	}

	// Test a key that IS in the name
	cfg.Keys.RemoveMode = []string{"r"} // 'r' is in 'remove'
	m = initialModel(modeInstall, cfg)
	view = stripAnsi(m.renderHelpText(lipgloss.Color("7")))
	if !strings.Contains(view, "[r]emove") {
		t.Errorf("Expected [r]emove in help text, got %q", view)
	}

	if !strings.Contains(view, "[^q]:quit") {
		t.Errorf("Expected [^q]:quit in help text for changed multi-key, got %q", view)
	}
}

func TestConfigKeyReflectionInConfirmation(t *testing.T) {
	lipgloss.SetColorProfile(termenv.Ascii)
	cfg := DefaultConfig()
	cfg.Keys.Confirm = "y" // 'y' in 'yes'
	cfg.Keys.Cancel = "x"  // 'x' not in 'no'

	m := initialModel(modeInstall, cfg)
	m.showConfirmation = true
	m.confirmType = confirmInstall

	view := stripAnsi(m.renderConfirmationDialog(80, 24, lipgloss.Color("7")))

	if !strings.Contains(view, "[y]es") {
		t.Errorf("Expected [y]es in confirmation dialog, got %q", view)
	}
	if !strings.Contains(view, "[x]:no") {
		t.Errorf("Expected [x]:no in confirmation dialog, got %q", view)
	}
}

func TestConfigKeyReflectionInSettings(t *testing.T) {
	lipgloss.SetColorProfile(termenv.Ascii)
	cfg := DefaultConfig()
	cfg.Keys.Cancel = "c"         // 'c' in 'close'
	cfg.Keys.Quit = []string{"q"} // 'q' in 'quit'

	m := initialModel(modeInstall, cfg)
	m.mode = modeSettings

	view := stripAnsi(m.renderSettings(80, 24))

	if !strings.Contains(view, "[c]lose") {
		t.Errorf("Expected [c]lose in settings view, got %q", view)
	}
	if !strings.Contains(view, "[q]uit") {
		t.Errorf("Expected [q]uit in settings view, got %q", view)
	}
}

// Helper to strip ANSI escape codes for easier comparison
func stripAnsi(s string) string {
	var b strings.Builder
	inEsc := false
	for _, r := range s {
		if r == '\x1b' {
			inEsc = true
			continue
		}
		if inEsc {
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
				inEsc = false
			}
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
