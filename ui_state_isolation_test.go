package main

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

func TestUIStateIsolationOnModeSwitch(t *testing.T) {
	// Start in Dashboard mode
	m := initialModel(modeDashboard, DefaultConfig())
	
	// Set up keys for switching
	m.keys.InstallMode = key.NewBinding(key.WithKeys("i"))
	m.keys.UninstallMode = key.NewBinding(key.WithKeys("r"))
	m.keys.DashboardMode = key.NewBinding(key.WithKeys("d"))
	m.keys.UpdateMode = key.NewBinding(key.WithKeys("u"))

	if m.textInput.Placeholder != "View system dashboard" {
		t.Errorf("Dashboard mode: expected placeholder 'View system dashboard', got %q", m.textInput.Placeholder)
	}

	// 1. Switch to Install mode
	resModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("i")})
	m = resModel.(*model)
	
	if m.mode != modeInstall {
		t.Fatalf("Expected modeInstall, got %v", m.mode)
	}
	if m.textInput.Placeholder != "Search packages..." {
		t.Errorf("Install mode: expected placeholder 'Search packages...', got %q", m.textInput.Placeholder)
	}

	// Blur input before switching back (since modeInstall focuses it)
	m.textInput.Blur()

	// 2. Switch back to Dashboard mode
	resModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	m = resModel.(*model)
	if m.mode != modeDashboard {
		t.Fatalf("Expected modeDashboard, got %v", m.mode)
	}
	if m.textInput.Placeholder != "View system dashboard" {
		t.Errorf("Dashboard mode: expected placeholder 'View system dashboard', got %q", m.textInput.Placeholder)
	}

	// 3. Switch to Uninstall mode
	resModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	m = resModel.(*model)
	if m.mode != modeUninstall {
		t.Fatalf("Expected modeUninstall, got %v", m.mode)
	}
	if m.textInput.Placeholder != "Filter installed packages..." {
		t.Errorf("Uninstall mode: expected placeholder 'Filter installed packages...', got %q", m.textInput.Placeholder)
	}

	// Blur input (modeUninstall also focuses it via resetState -> updatePlaceholder? No, Uninstall switch focuses it manually if it wants)
	// Wait, Uninstall switch doesn't focus it.
	
	// 4. Switch to Update mode
	resModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("u")})
	m = resModel.(*model)
	if m.mode != modeUpdate {
		t.Fatalf("Expected modeUpdate, got %v", m.mode)
	}
	if m.textInput.Placeholder != "Checking for updates..." {
		t.Errorf("Update mode: expected placeholder 'Checking for updates...', got %q", m.textInput.Placeholder)
	}
}
