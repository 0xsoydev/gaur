package main

import (
	"strings"
	"testing"
)

func TestViewNoCrash(t *testing.T) {
	m := initialModel(modeInstall, DefaultConfig())
	m.width = 100
	m.height = 40
	m.loading = false
	
	// Test all modes
	modes := []viewMode{modeInstall, modeUninstall, modeUpdate, modeInstalled}
	for _, mode := range modes {
		m.mode = mode
		t.Run("view mode "+string(rune(mode)), func(t *testing.T) {
			view := m.View()
			if view == "" {
				t.Errorf("View() returned empty string for mode %v", mode)
			}
		})
	}

	// Test with confirmation dialog
	m.showConfirmation = true
	m.confirmType = confirmInstall
	m.confirmPackages = []string{"pkg1", "pkg2"}
	t.Run("confirmation dialog", func(t *testing.T) {
		view := m.View()
		if !strings.Contains(view, "Confirm Installation") {
			t.Errorf("View() didn't contain 'Confirm Installation' in confirmation mode")
		}
	})

	// Test with error overlay
	m.showConfirmation = false
	m.showErrorOverlay = true
	m.errorTitle = "Error Title"
	m.errorMessage = "Error message"
	t.Run("error overlay", func(t *testing.T) {
		view := m.View()
		if !strings.Contains(view, "Error Title") {
			t.Errorf("View() didn't contain 'Error Title' in error mode")
		}
	})
}
