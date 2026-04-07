package main

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestStateResetOnModeSwitch(t *testing.T) {
	cfg := DefaultConfig()
	m := testModel(t, modeInstall, cfg)

	// 1. Simulate user state in modeInstall
	m.textInput.SetValue("vim")
	m.lastQuery = "vim"
	m.packageDetails = "Some details about vim"
	m.detailsForPackage = "vim"

	// 2. Switch to Dashboard (n)
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(cfg.Keys.DashboardMode[0])})
	m = newModel.(*model)

	if m.mode != modeDashboard {
		t.Errorf("Expected modeDashboard, got %v", m.mode)
	}

	// Verify resets
	if m.textInput.Value() != "" {
		t.Error("Search input was not cleared after switching to dashboard")
	}
	if m.packageDetails != "" {
		t.Error("Package details was not cleared after switching to dashboard")
	}
}

func TestStateResetTransitionToRemove(t *testing.T) {
	cfg := DefaultConfig()
	m := testModel(t, modeInstall, cfg)
	m.textInput.SetValue("fzf")
	m.packageDetails = "fzf details"

	// Switch to Remove (r)
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(cfg.Keys.RemoveMode[0])})
	m = newModel.(*model)

	if m.mode != modeRemove {
		t.Fatalf("Expected modeRemove, got %v", m.mode)
	}

	if m.textInput.Value() != "" {
		t.Error("Search input should be empty in new mode")
	}
	if m.packageDetails != "" {
		t.Error("Package details should be cleared in new mode")
	}
}

func TestStateResetOnSelectiveExit(t *testing.T) {
	cfg := DefaultConfig()
	m := testModel(t, modeUpdate, cfg)
	m.mode = modeUpdateSelective
	m.textInput.Focus()
	m.textInput.SetValue("lib")
	m.packageDetails = "lib details"

	// Exit via ESC
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = newModel.(*model)

	if m.mode != modeUpdate {
		t.Errorf("Expected return to modeUpdate, got %v", m.mode)
	}

	if m.textInput.Value() != "" {
		t.Error("Search input should be cleared when exiting selective update")
	}
}
