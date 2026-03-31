package main

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestUpdateNavigation(t *testing.T) {
	m := initialModel(modeInstall, DefaultConfig())
	m.filtered = []Package{{Name: "pkg1"}, {Name: "pkg2"}, {Name: "pkg3"}}
	m.loading = false
	m.selectedIndex = 1 // Start in the middle so we can go both directions

	// Up/k decreases index, Down/j increases index (standard navigation)
	// Test up navigation (should decrease index)
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = newModel.(*model)
	if m.selectedIndex != 0 {
		t.Errorf("selectedIndex = %d, want 0 after Up", m.selectedIndex)
	}

	// Test down navigation (should increase index)
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = newModel.(*model)
	if m.selectedIndex != 1 {
		t.Errorf("selectedIndex = %d, want 1 after Down", m.selectedIndex)
	}

	// Test 'k' navigation (up - should decrease index)
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	m = newModel.(*model)
	if m.selectedIndex != 0 {
		t.Errorf("selectedIndex = %d, want 0 after 'k'", m.selectedIndex)
	}

	// Test 'j' navigation (down - should increase index)
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m = newModel.(*model)
	if m.selectedIndex != 1 {
		t.Errorf("selectedIndex = %d, want 1 after 'j'", m.selectedIndex)
	}
}

func TestUpdateModeSwitching(t *testing.T) {
	m := initialModel(modeInstall, DefaultConfig())
	m.loading = false
	m.installed = []Package{{Name: "already-loaded"}}

	// Switch to Remove mode ('r')
	// It should return a command even if m.installed is NOT empty
	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	m = newModel.(*model)
	if m.mode != modeRemove {
		t.Errorf("mode = %v, want modeRemove after 'r'", m.mode)
	}
	if cmd == nil {
		t.Errorf("Expected command (refreshing packages) when entering Remove mode, got nil")
	}

	// Switch to Dashboard/Installed mode ('d')
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	m = newModel.(*model)
	if m.mode != modeDashboard {
		t.Errorf("mode = %v, want modeDashboard after 'd'", m.mode)
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
