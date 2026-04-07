package main

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// Helper to create mouse wheel events with the new API
func mouseWheelDown(x, y int) tea.MouseMsg {
	return tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonWheelDown, X: x, Y: y}
}

func TestMouseScrollStandardMode(t *testing.T) {
	m := testModel(t, modeInstall, DefaultConfig())
	m.width = 80
	m.height = 24
	m.loading = false

	// Setup with package details and a list
	m.packageDetails = "Some details"
	m.maxDetailsScroll = 10
	m.detailsScrollOffset = 0
	m.filtered = []Package{{Name: "pkg1"}, {Name: "pkg2"}, {Name: "pkg3"}}
	m.selectedIndex = 1 // Start at 1 so we can go both directions

	// 1. TOP HALF (Y < height/2) should scroll details
	newModel, _ := m.Update(mouseWheelDown(10, 5))
	m = newModel.(*model)
	if m.detailsScrollOffset != 1 {
		t.Errorf("Top half: expected detailsScrollOffset 1, got %d", m.detailsScrollOffset)
	}
	if m.selectedIndex != 1 {
		t.Error("Top half: selectedIndex changed")
	}

	// 2. BOTTOM HALF (Y >= height/2) should navigate list
	// In bottom-up menus: scroll down visually moves selection down, which DECREASES index
	newModel, _ = m.Update(mouseWheelDown(10, 20))
	m = newModel.(*model)
	if m.selectedIndex != 0 {
		t.Errorf("Bottom half scroll down: expected selectedIndex 0 (bottom-up: scroll down decreases index), got %d", m.selectedIndex)
	}
}

func TestMouseScrollSelectiveUpdateMode(t *testing.T) {
	m := testModel(t, modeUpdateSelective, DefaultConfig())
	m.width = 80
	m.height = 24
	m.loading = false
	m.updatableAll = []Package{{Name: "pkg1"}, {Name: "pkg2"}, {Name: "pkg3"}}
	m.filtered = m.updatableAll
	m.selectedIndex = 1

	m.packageDetails = "Some details"
	m.maxDetailsScroll = 10
	m.detailsScrollOffset = 0

	// 1. RIGHT SIDE (X >= width/2): scroll details
	newModel, _ := m.Update(mouseWheelDown(60, 10))
	m = newModel.(*model)
	if m.detailsScrollOffset != 1 {
		t.Errorf("Right side: expected detailsScrollOffset 1, got %d", m.detailsScrollOffset)
	}

	// 2. LEFT SIDE (X < width/2): navigate list
	// In bottom-up menus: scroll down visually moves selection down, which DECREASES index
	newModel, _ = m.Update(mouseWheelDown(10, 10))
	m = newModel.(*model)
	if m.selectedIndex != 0 {
		t.Errorf("Left side scroll down: expected selectedIndex 0 (bottom-up: scroll down decreases index), got %d", m.selectedIndex)
	}
}

func TestMouseScrollSimpleUpdateMode(t *testing.T) {
	m := testModel(t, modeUpdate, DefaultConfig())
	m.width = 80
	m.height = 24
	m.loading = false
	m.pendingUpdates = []Package{{Name: "p1"}, {Name: "p2"}}
	m.updateScrollOffset = 0
	m.maxUpdateScroll = 10

	// Should scroll updateScrollOffset regardless of Y
	newModel, _ := m.Update(mouseWheelDown(10, 5))
	m = newModel.(*model)
	if m.updateScrollOffset != 1 {
		t.Errorf("Expected updateScrollOffset 1, got %d", m.updateScrollOffset)
	}
}

func TestMouseScrollConfirmation(t *testing.T) {
	m := testModel(t, modeInstall, DefaultConfig())
	m.showConfirmation = true
	m.confirmScrollOffset = 0
	m.maxConfirmScroll = 5

	newModel, _ := m.Update(mouseWheelDown(10, 5))
	m = newModel.(*model)
	if m.confirmScrollOffset != 1 {
		t.Errorf("Expected confirmScrollOffset 1, got %d", m.confirmScrollOffset)
	}
}
