package main

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestUpdateNavigation(t *testing.T) {
	m := initialModel(modeInstall)
	mp := &m
	mp.filtered = []Package{{Name: "pkg1"}, {Name: "pkg2"}, {Name: "pkg3"}}
	mp.loading = false
	mp.selectedIndex = 0

	// In this app, "up" (k) increases index, "down" (j) decreases index
	// Test up navigation
	newModel, _ := mp.Update(tea.KeyMsg{Type: tea.KeyUp})
	mp = newModel.(*model)
	if mp.selectedIndex != 1 {
		t.Errorf("selectedIndex = %d, want 1 after Up", mp.selectedIndex)
	}

	// Test down navigation
	newModel, _ = mp.Update(tea.KeyMsg{Type: tea.KeyDown})
	mp = newModel.(*model)
	if mp.selectedIndex != 0 {
		t.Errorf("selectedIndex = %d, want 0 after Down", mp.selectedIndex)
	}

	// Test 'k' navigation (up)
	newModel, _ = mp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	mp = newModel.(*model)
	if mp.selectedIndex != 1 {
		t.Errorf("selectedIndex = %d, want 1 after 'k'", mp.selectedIndex)
	}

	// Test 'j' navigation (down)
	newModel, _ = mp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	mp = newModel.(*model)
	if mp.selectedIndex != 0 {
		t.Errorf("selectedIndex = %d, want 0 after 'j'", mp.selectedIndex)
	}
}

func TestUpdateModeSwitching(t *testing.T) {
	m := initialModel(modeInstall)
	mp := &m
	mp.loading = false

	// Switch to Uninstall mode ('r')
	newModel, _ := mp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	mp = newModel.(*model)
	if mp.mode != modeUninstall {
		t.Errorf("mode = %v, want modeUninstall after 'r'", mp.mode)
	}

	// Switch to Dashboard/Installed mode ('n')
	newModel, _ = mp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	mp = newModel.(*model)
	if mp.mode != modeInstalled {
		t.Errorf("mode = %v, want modeInstalled after 'n'", mp.mode)
	}

	// Switch back to Install mode ('i')
	newModel, _ = mp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("i")})
	mp = newModel.(*model)
	if mp.mode != modeInstall {
		t.Errorf("mode = %v, want modeInstall after 'i'", mp.mode)
	}
}

func TestUpdateWindowResize(t *testing.T) {
	m := initialModel(modeInstall)
	mp := &m
	newModel, _ := mp.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	mp = newModel.(*model)
	if mp.width != 80 || mp.height != 24 {
		t.Errorf("width/height = %d/%d, want 80/24", mp.width, mp.height)
	}
}
