package main

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestBottomUpNavigation tests navigation in bottom-up rendered menus (modeInstall, modeRemove).
// In bottom-up menus, the list is rendered with index 0 at the bottom of the screen.
// Therefore, pressing "up" should INCREASE the index (moving toward items shown at the top),
// and pressing "down" should DECREASE the index (moving toward items shown at the bottom).
func TestBottomUpNavigation(t *testing.T) {
	t.Run("modeInstall", func(t *testing.T) {
		m := initialModel(modeInstall, DefaultConfig())
		m.filtered = []Package{{Name: "pkg1"}, {Name: "pkg2"}, {Name: "pkg3"}}
		m.loading = false
		m.selectedIndex = 1 // Start in the middle so we can go both directions

		// In bottom-up menu: Up arrow should INCREASE index (visually move up toward higher indices)
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
		m = newModel.(*model)
		if m.selectedIndex != 2 {
			t.Errorf("selectedIndex = %d, want 2 after Up (bottom-up: up increases index)", m.selectedIndex)
		}

		// Down arrow should DECREASE index (visually move down toward lower indices)
		newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = newModel.(*model)
		if m.selectedIndex != 1 {
			t.Errorf("selectedIndex = %d, want 1 after Down (bottom-up: down decreases index)", m.selectedIndex)
		}

		// 'k' should behave like Up (increase index)
		newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
		m = newModel.(*model)
		if m.selectedIndex != 2 {
			t.Errorf("selectedIndex = %d, want 2 after 'k' (bottom-up: k increases index)", m.selectedIndex)
		}

		// 'j' should behave like Down (decrease index)
		newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
		m = newModel.(*model)
		if m.selectedIndex != 1 {
			t.Errorf("selectedIndex = %d, want 1 after 'j' (bottom-up: j decreases index)", m.selectedIndex)
		}
	})

	t.Run("modeRemove", func(t *testing.T) {
		m := initialModel(modeRemove, DefaultConfig())
		m.filteredInstalled = []Package{{Name: "pkg1"}, {Name: "pkg2"}, {Name: "pkg3"}}
		m.loading = false
		m.selectedIndex = 1

		// In bottom-up menu: Up should INCREASE index
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
		m = newModel.(*model)
		if m.selectedIndex != 2 {
			t.Errorf("selectedIndex = %d, want 2 after Up in modeRemove", m.selectedIndex)
		}

		// Down should DECREASE index
		newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = newModel.(*model)
		if m.selectedIndex != 1 {
			t.Errorf("selectedIndex = %d, want 1 after Down in modeRemove", m.selectedIndex)
		}
	})
}

// TestBottomUpNavigationBoundaries ensures navigation correctly clamps at boundaries.
func TestBottomUpNavigationBoundaries(t *testing.T) {
	m := initialModel(modeInstall, DefaultConfig())
	m.filtered = []Package{{Name: "pkg1"}, {Name: "pkg2"}, {Name: "pkg3"}}
	m.loading = false

	// Start at bottom (index 0), pressing Down should stay at 0 (can't go below)
	m.selectedIndex = 0
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = newModel.(*model)
	if m.selectedIndex != 0 {
		t.Errorf("selectedIndex = %d, want 0 after Down at bottom boundary", m.selectedIndex)
	}

	// Start at top (index 2), pressing Up should stay at 2 (can't go above)
	m.selectedIndex = 2
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = newModel.(*model)
	if m.selectedIndex != 2 {
		t.Errorf("selectedIndex = %d, want 2 after Up at top boundary", m.selectedIndex)
	}
}

// TestBottomUpPageNavigation tests PgUp/PgDown behavior in bottom-up menus.
func TestBottomUpPageNavigation(t *testing.T) {
	m := initialModel(modeInstall, DefaultConfig())
	// Create a list with 15 packages to test page navigation
	m.filtered = make([]Package, 15)
	for i := range m.filtered {
		m.filtered[i] = Package{Name: fmt.Sprintf("pkg%d", i)}
	}
	m.loading = false
	m.selectedIndex = 5

	// PgUp should increase index by 10 (move up visually)
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	m = newModel.(*model)
	if m.selectedIndex != 14 { // 5 + 10 = 15, clamped to 14 (max index)
		t.Errorf("selectedIndex = %d, want 14 after PgUp", m.selectedIndex)
	}

	// PgDown should decrease index by 10 (move down visually)
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	m = newModel.(*model)
	if m.selectedIndex != 4 { // 14 - 10 = 4
		t.Errorf("selectedIndex = %d, want 4 after PgDown", m.selectedIndex)
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
