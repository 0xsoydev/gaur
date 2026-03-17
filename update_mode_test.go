package main

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestUpdateNavigation(t *testing.T) {
	m := initialModel(modeInstall, DefaultConfig())
	m.filtered = []Package{{Name: "pkg1"}, {Name: "pkg2"}, {Name: "pkg3"}}
	m.loading = false
	m.selectedIndex = 0

	// In this app, "up" (k) increases index, "down" (j) decreases index
	// Test up navigation
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = newModel.(*model)
	if m.selectedIndex != 1 {
		t.Errorf("selectedIndex = %d, want 1 after Up", m.selectedIndex)
	}

	// Test down navigation
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = newModel.(*model)
	if m.selectedIndex != 0 {
		t.Errorf("selectedIndex = %d, want 0 after Down", m.selectedIndex)
	}

	// Test 'k' navigation (up)
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	m = newModel.(*model)
	if m.selectedIndex != 1 {
		t.Errorf("selectedIndex = %d, want 1 after 'k'", m.selectedIndex)
	}

	// Test 'j' navigation (down)
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m = newModel.(*model)
	if m.selectedIndex != 0 {
		t.Errorf("selectedIndex = %d, want 0 after 'j'", m.selectedIndex)
	}
}

func TestUpdateModeSwitching(t *testing.T) {
	m := initialModel(modeInstall, DefaultConfig())
	m.loading = false
	m.installed = []Package{{Name: "already-loaded"}}

	// Switch to Uninstall mode ('r')
	// It should return a command even if m.installed is NOT empty
	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	m = newModel.(*model)
	if m.mode != modeUninstall {
		t.Errorf("mode = %v, want modeUninstall after 'r'", m.mode)
	}
	if cmd == nil {
		t.Errorf("Expected command (refreshing packages) when entering Uninstall mode, got nil")
	}

	// Switch to Dashboard/Installed mode ('n')
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	m = newModel.(*model)
	if m.mode != modeInstalled {
		t.Errorf("mode = %v, want modeInstalled after 'n'", m.mode)
	}

	// Switch back to Install mode ('i')
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("i")})
	m = newModel.(*model)
	if m.mode != modeInstall {
		t.Errorf("mode = %v, want modeInstall after 'i'", m.mode)
	}
}

func TestUpdateWindowResize(t *testing.T) {
	m := initialModel(modeInstall, DefaultConfig())
	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = newModel.(*model)
	if m.width != 80 || m.height != 24 {
		t.Errorf("width/height = %d/%d, want 80/24", m.width, m.height)
	}
}
