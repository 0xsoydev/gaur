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
	m := initialModel(modeUpdate)
	mp := &m
	mp.loading = false
	mp.pendingUpdates = pkgs
	mp.updatableAll = pkgs
	mp.filtered = pkgs
	mp.selectedIndex = 0
	mp.width = 80
	mp.height = 24

	// 2. Press 's' to enter Selective Update mode
	newModel, _ := mp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	mp = newModel.(*model)

	if mp.mode != modeUpdateSelective {
		t.Errorf("Expected modeUpdateSelective, got %v", mp.mode)
	}

	// 3. Blur text input to enable navigation
	// We call Blur() manually to avoid the 'esc' logic that resets the mode.
	mp.textInput.Blur()
	if mp.textInput.Focused() {
		t.Error("Expected text input to be blurred")
	}

	// 4. Mark 'firefox' (index 0) and 'yay' (index 2)
	// Current selectedIndex should be 0 (firefox)
	newModel, _ = mp.Update(tea.KeyMsg{Type: tea.KeyTab}) // Mark firefox
	mp = newModel.(*model)

	if len(mp.markedPackages) != 1 {
		t.Errorf("Expected 1 marked package, got %d. Marked: %v", len(mp.markedPackages), mp.markedPackages)
	}

	// In this app, 'k' (up) increases selectedIndex.
	newModel, _ = mp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")}) // move to vlc (index 1)
	mp = newModel.(*model)
	newModel, _ = mp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")}) // move to yay (index 2)
	mp = newModel.(*model)
	
	if mp.selectedIndex != 2 {
		t.Errorf("Expected selectedIndex 2, got %d", mp.selectedIndex)
	}

	newModel, _ = mp.Update(tea.KeyMsg{Type: tea.KeyTab}) // Mark yay
	mp = newModel.(*model)

	if len(mp.markedPackages) != 2 {
		t.Errorf("Expected 2 marked packages, got %d. Marked: %v", len(mp.markedPackages), mp.markedPackages)
	}
	if !mp.markedPackages["firefox"] || !mp.markedPackages["yay"] {
		t.Error("Marked packages mismatch")
	}

	// 5. Press Enter to show confirmation
	newModel, _ = mp.Update(tea.KeyMsg{Type: tea.KeyEnter})
	mp = newModel.(*model)

	if !mp.showConfirmation {
		t.Error("Expected confirmation dialog to be shown")
	}
	if mp.confirmType != confirmSelectiveUpdate {
		t.Errorf("Expected confirmSelectiveUpdate, got %v", mp.confirmType)
	}
	if len(mp.confirmPackages) != 2 {
		t.Errorf("Expected 2 packages in confirmation, got %d", len(mp.confirmPackages))
	}
}

func TestSelectiveUpdateMouseScrollWithInfo(t *testing.T) {
	m := initialModel(modeUpdateSelective)
	mp := &m
	mp.width = 80
	mp.height = 24
	mp.loading = false
	mp.updatableAll = []Package{{Name: "pkg1"}, {Name: "pkg2"}}
	mp.filtered = mp.updatableAll
	mp.selectedIndex = 0
	
	// Set package info - this SHOULD prevent list scrolling
	mp.packageInfo = "Some multi-line\npackage info\nto scroll through."
	mp.maxInfoScroll = 5
	mp.infoScrollOffset = 0

	// Scroll mouse wheel up - should NOT change selectedIndex
	newModel, _ := mp.Update(tea.MouseMsg{Type: tea.MouseWheelUp})
	mp = newModel.(*model)
	
	if mp.selectedIndex != 0 {
		t.Errorf("Expected selectedIndex 0, got %d (should NOT move list when info is present)", mp.selectedIndex)
	}
	
	// Scroll mouse wheel down - should NOT change selectedIndex
	newModel, _ = mp.Update(tea.MouseMsg{Type: tea.MouseWheelDown})
	mp = newModel.(*model)
	
	if mp.selectedIndex != 0 {
		t.Errorf("Expected selectedIndex 0, got %d (should NOT move list when info is present)", mp.selectedIndex)
	}
	if mp.infoScrollOffset != 1 {
		t.Errorf("Expected infoScrollOffset 1, got %d", mp.infoScrollOffset)
	}
}

func TestSelectiveUpdateMouseScrollWithoutInfo(t *testing.T) {
	m := initialModel(modeUpdateSelective)
	mp := &m
	mp.width = 80
	mp.height = 24
	mp.loading = false
	mp.updatableAll = []Package{{Name: "pkg1"}, {Name: "pkg2"}}
	mp.filtered = mp.updatableAll
	mp.selectedIndex = 0
	
	// EMPTY package info
	mp.packageInfo = ""

	// Scroll mouse wheel up - this SHOULD move list cursor in current code
	newModel, _ := mp.Update(tea.MouseMsg{Type: tea.MouseWheelUp})
	mp = newModel.(*model)
	
	if mp.selectedIndex != 1 {
		t.Errorf("Expected selectedIndex 1, got %d (list SHOULD move when info is empty)", mp.selectedIndex)
	}
}
