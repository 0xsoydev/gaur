package main

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestMouseScrollStandardMode(t *testing.T) {
	m := initialModel(modeInstall, DefaultConfig())
	m.width = 80
	m.height = 24
	m.loading = false
	
	// Setup with package dash and a list
	m.packageDash = "Some dashboard"
	m.maxDashScroll = 10
	m.dashScrollOffset = 0
	m.filtered = []Package{{Name: "pkg1"}, {Name: "pkg2"}}
	m.selectedIndex = 0

	// 1. TOP HALF (Y < height/2) should scroll details
	newModel, _ := m.Update(tea.MouseMsg{Type: tea.MouseWheelDown, Y: 5})
	m = newModel.(*model)
	if m.dashScrollOffset != 1 {
		t.Errorf("Top half: expected dashScrollOffset 1, got %d", m.dashScrollOffset)
	}
	if m.selectedIndex != 0 {
		t.Error("Top half: selectedIndex changed")
	}

	// 2. BOTTOM HALF (Y >= height/2) should navigate list
	newModel, _ = m.Update(tea.MouseMsg{Type: tea.MouseWheelUp, Y: 20}) // Up increases index
	m = newModel.(*model)
	if m.selectedIndex != 1 {
		t.Errorf("Bottom half: expected selectedIndex 1, got %d", m.selectedIndex)
	}
}

func TestMouseScrollSelectiveUpdateMode(t *testing.T) {
	m := initialModel(modeUpdateSelective, DefaultConfig())
	m.width = 80
	m.height = 24
	m.loading = false
	m.updatableAll = []Package{{Name: "pkg1"}, {Name: "pkg2"}}
	m.filtered = m.updatableAll
	m.selectedIndex = 0
	
	m.packageDash = "Some dashboard"
	m.maxDashScroll = 10
	m.dashScrollOffset = 0

	// 1. RIGHT SIDE (X >= width/2): scroll details
	newModel, _ := m.Update(tea.MouseMsg{Type: tea.MouseWheelDown, X: 60, Y: 10})
	m = newModel.(*model)
	if m.dashScrollOffset != 1 {
		t.Errorf("Right side: expected dashScrollOffset 1, got %d", m.dashScrollOffset)
	}

	// 2. LEFT SIDE (X < width/2): navigate list
	newModel, _ = m.Update(tea.MouseMsg{Type: tea.MouseWheelUp, X: 10, Y: 10})
	m = newModel.(*model)
	if m.selectedIndex != 1 {
		t.Errorf("Left side: expected selectedIndex 1, got %d", m.selectedIndex)
	}
}

func TestMouseScrollSimpleUpdateMode(t *testing.T) {
	m := initialModel(modeUpdate, DefaultConfig())
	m.width = 80
	m.height = 24
	m.loading = false
	m.pendingUpdates = []Package{{Name: "p1"}, {Name: "p2"}}
	m.updateScrollOffset = 0
	m.maxUpdateScroll = 10

	// Should scroll updateScrollOffset regardless of Y
	newModel, _ := m.Update(tea.MouseMsg{Type: tea.MouseWheelDown, Y: 5})
	m = newModel.(*model)
	if m.updateScrollOffset != 1 {
		t.Errorf("Expected updateScrollOffset 1, got %d", m.updateScrollOffset)
	}
}

func TestMouseScrollConfirmation(t *testing.T) {
	m := initialModel(modeInstall, DefaultConfig())
	m.showConfirmation = true
	m.confirmScrollOffset = 0
	m.maxConfirmScroll = 5

	newModel, _ := m.Update(tea.MouseMsg{Type: tea.MouseWheelDown, Y: 5})
	m = newModel.(*model)
	if m.confirmScrollOffset != 1 {
		t.Errorf("Expected confirmScrollOffset 1, got %d", m.confirmScrollOffset)
	}
}
