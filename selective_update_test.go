package main

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestSelectiveUpdateFlow(t *testing.T) {
	// 1. Initial setup: Mode Update with some pending updates
	pkgs := []Package{
		{Source: "extra", Name: "firefox", Version: "120.0"},
		{Source: "extra", Name: "vlc", Version: "3.0.18"},
		{Source: "aur", Name: "yay", Version: "12.5.7"},
	}
	m := initialModel(modeUpdate, DefaultConfig())
	m.loading = false
	m.pendingUpdates = pkgs
	m.updatableAll = pkgs
	m.filtered = pkgs
	m.selectedIndex = 0
	m.width = 80
	m.height = 24

	// 2. Press 's' to enter Selective Update mode
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	m = newModel.(*model)

	if m.mode != modeUpdateSelective {
		t.Errorf("Expected modeUpdateSelective, got %v", m.mode)
	}

	// 3. Blur text input to enable navigation
	// We call Blur() manually to avoid the 'esc' logic that resets the mode.
	m.textInput.Blur()
	if m.textInput.Focused() {
		t.Error("Expected text input to be blurred")
	}

	// 4. Mark 'firefox' (index 0) and 'yay' (index 2)
	// Current selectedIndex should be 0 (firefox)
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab}) // Mark firefox
	m = newModel.(*model)

	if len(m.markedPackages) != 1 {
		t.Errorf("Expected 1 marked package, got %d. Marked: %v", len(m.markedPackages), m.markedPackages)
	}

	// 'j' (down) increases selectedIndex (standard navigation)
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}) // move to vlc (index 1)
	m = newModel.(*model)
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}) // move to yay (index 2)
	m = newModel.(*model)

	if m.selectedIndex != 2 {
		t.Errorf("Expected selectedIndex 2, got %d", m.selectedIndex)
	}

	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab}) // Mark yay
	m = newModel.(*model)

	if len(m.markedPackages) != 2 {
		t.Errorf("Expected 2 marked packages, got %d. Marked: %v", len(m.markedPackages), m.markedPackages)
	}
	if !m.markedPackages["firefox"] || !m.markedPackages["yay"] {
		t.Error("Marked packages mismatch")
	}

	// 5. Press Enter to show confirmation
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(*model)

	if !m.showConfirmation {
		t.Error("Expected confirmation dialog to be shown")
	}
	if m.confirmType != confirmSelectiveUpdate {
		t.Errorf("Expected confirmSelectiveUpdate, got %v", m.confirmType)
	}
	if len(m.confirmPackages) != 2 {
		t.Errorf("Expected 2 packages in confirmation, got %d", len(m.confirmPackages))
	}
}

func TestSelectiveUpdateMouseScrollWithDetails(t *testing.T) {
	m := initialModel(modeUpdateSelective, DefaultConfig())
	m.width = 80
	m.height = 24
	m.loading = false
	m.updatableAll = []Package{{Name: "pkg1"}, {Name: "pkg2"}}
	m.filtered = m.updatableAll
	m.selectedIndex = 0

	// Set package details - this SHOULD prevent list scrolling
	m.packageDetails = "Some multi-line\npackage details\nto scroll through."
	m.maxDetailsScroll = 5
	m.detailsScrollOffset = 0

	// Scroll mouse wheel up - RIGHT SIDE (X=60) should scroll details
	newModel, _ := m.Update(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonWheelUp, X: 60})
	m = newModel.(*model)

	if m.selectedIndex != 0 {
		t.Errorf("Expected selectedIndex 0, got %d (should NOT move list when scrolling on right side)", m.selectedIndex)
	}

	// Scroll mouse wheel down - RIGHT SIDE (X=60) should scroll details
	newModel, _ = m.Update(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonWheelDown, X: 60})
	m = newModel.(*model)

	if m.selectedIndex != 0 {
		t.Errorf("Expected selectedIndex 0, got %d (should NOT move list when scrolling on right side)", m.selectedIndex)
	}
	if m.detailsScrollOffset != 1 {
		t.Errorf("Expected detailsScrollOffset 1, got %d", m.detailsScrollOffset)
	}
}

func TestSelectiveUpdateMouseScrollWithoutDetails(t *testing.T) {
	m := initialModel(modeUpdateSelective, DefaultConfig())
	m.width = 80
	m.height = 24
	m.loading = false
	m.updatableAll = []Package{{Name: "pkg1"}, {Name: "pkg2"}}
	m.filtered = m.updatableAll
	m.selectedIndex = 1 // Start at 1 so we can scroll up

	// EMPTY package details
	m.packageDetails = ""

	// Scroll mouse wheel up - LEFT SIDE (X=10) should move list cursor up (decrease index)
	newModel, _ := m.Update(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonWheelUp, X: 10})
	m = newModel.(*model)

	if m.selectedIndex != 0 {
		t.Errorf("Expected selectedIndex 0, got %d (list SHOULD move up when scrolling wheel up on left side)", m.selectedIndex)
	}
}
