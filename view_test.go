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

func TestRepoSummary(t *testing.T) {
	m := initialModel(modeInstall, DefaultConfig())
	pkgList := []Package{
		{Source: "core", Name: "linux"},
		{Source: "extra", Name: "vim"},
		{Source: "extra", Name: "libpng"},
		{Source: "aur", Name: "google-chrome"},
	}

	summary := m.renderRepoSummary(pkgList)

	// Should contain the counts and repo names
	// renderRepoSummary uses sourceStyle(r).Render(r) which might add ANSI codes
	if !strings.Contains(summary, "1") || !strings.Contains(summary, "core") {
		t.Errorf("Expected '1 core' in summary, got %q", summary)
	}
	if !strings.Contains(summary, "2") || !strings.Contains(summary, "extra") {
		t.Errorf("Expected '2 extra' in summary, got %q", summary)
	}
	if !strings.Contains(summary, "1") || !strings.Contains(summary, "aur") {
		t.Errorf("Expected '1 aur' in summary, got %q", summary)
	}

	// Test empty list
	if m.renderRepoSummary([]Package{}) != "" {
		t.Error("Empty package list should return empty summary")
	}
}
