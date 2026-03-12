package main

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestMouseScrollStandardMode(t *testing.T) {
	m := initialModel(modeInstall)
	mp := &m
	mp.width = 80
	mp.height = 24
	mp.loading = false
	
	// Setup with package info and a list
	mp.packageInfo = "Some info"
	mp.maxInfoScroll = 10
	mp.infoScrollOffset = 0
	mp.filtered = []Package{{Name: "pkg1"}, {Name: "pkg2"}}
	mp.selectedIndex = 0

	// 1. TOP HALF (Y < height/2) should scroll details
	newModel, _ := mp.Update(tea.MouseMsg{Type: tea.MouseWheelDown, Y: 5})
	mp = newModel.(*model)
	if mp.infoScrollOffset != 1 {
		t.Errorf("Top half: expected infoScrollOffset 1, got %d", mp.infoScrollOffset)
	}
	if mp.selectedIndex != 0 {
		t.Error("Top half: selectedIndex changed")
	}

	// 2. BOTTOM HALF (Y >= height/2) should navigate list
	newModel, _ = mp.Update(tea.MouseMsg{Type: tea.MouseWheelUp, Y: 20}) // Up increases index
	mp = newModel.(*model)
	if mp.selectedIndex != 1 {
		t.Errorf("Bottom half: expected selectedIndex 1, got %d", mp.selectedIndex)
	}
}

func TestMouseScrollSelectiveUpdateMode(t *testing.T) {
	m := initialModel(modeUpdateSelective)
	mp := &m
	mp.width = 80
	mp.height = 24
	mp.loading = false
	mp.updatableAll = []Package{{Name: "pkg1"}, {Name: "pkg2"}}
	mp.filtered = mp.updatableAll
	mp.selectedIndex = 0
	
	mp.packageInfo = "Some info"
	mp.maxInfoScroll = 10
	mp.infoScrollOffset = 0

	// 1. RIGHT SIDE (X >= width/2): scroll details
	newModel, _ := mp.Update(tea.MouseMsg{Type: tea.MouseWheelDown, X: 60, Y: 10})
	mp = newModel.(*model)
	if mp.infoScrollOffset != 1 {
		t.Errorf("Right side: expected infoScrollOffset 1, got %d", mp.infoScrollOffset)
	}

	// 2. LEFT SIDE (X < width/2): navigate list
	newModel, _ = mp.Update(tea.MouseMsg{Type: tea.MouseWheelUp, X: 10, Y: 10})
	mp = newModel.(*model)
	if mp.selectedIndex != 1 {
		t.Errorf("Left side: expected selectedIndex 1, got %d", mp.selectedIndex)
	}
}

func TestMouseScrollSimpleUpdateMode(t *testing.T) {
	m := initialModel(modeUpdate)
	mp := &m
	mp.width = 80
	mp.height = 24
	mp.loading = false
	mp.pendingUpdates = []Package{{Name: "p1"}, {Name: "p2"}}
	mp.updateScrollOffset = 0
	mp.maxUpdateScroll = 10

	// Should scroll updateScrollOffset regardless of Y
	newModel, _ := mp.Update(tea.MouseMsg{Type: tea.MouseWheelDown, Y: 5})
	mp = newModel.(*model)
	if mp.updateScrollOffset != 1 {
		t.Errorf("Expected updateScrollOffset 1, got %d", mp.updateScrollOffset)
	}
}

func TestMouseScrollConfirmation(t *testing.T) {
	m := initialModel(modeInstall)
	mp := &m
	mp.showConfirmation = true
	mp.confirmScrollOffset = 0
	mp.maxConfirmScroll = 5

	newModel, _ := mp.Update(tea.MouseMsg{Type: tea.MouseWheelDown, Y: 5})
	mp = newModel.(*model)
	if mp.confirmScrollOffset != 1 {
		t.Errorf("Expected confirmScrollOffset 1, got %d", mp.confirmScrollOffset)
	}
}
