package main

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestStateResetOnModeSwitch(t *testing.T) {
	cfg := DefaultConfig()
	m := initialModel(modeInstall, cfg)
	
	// 1. Simulate user state in modeInstall
	m.textInput.SetValue("vim")
	m.lastQuery = "vim"
	m.packageDash = "Some dashboard about vim"
	m.dashForPackage = "vim"
	
	// 2. Switch to Dashboard (n)
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(cfg.Keys.DashboardMode)})
	m = newModel.(*model)
	
	if m.mode != modeDashboard {
		t.Errorf("Expected modeDashboard, got %v", m.mode)
	}
	
	// Verify resets
	if m.textInput.Value() != "" {
		t.Error("Search input was not cleared after switching to dashboard")
	}
	if m.packageDash != "" {
		t.Error("Package dash was not cleared after switching to dashboard")
	}
}

func TestStateResetTransitionToUninstall(t *testing.T) {
	cfg := DefaultConfig()
	m := initialModel(modeInstall, cfg)
	m.textInput.SetValue("fzf")
	m.packageDash = "fzf dashboard"
	
	// Switch to Uninstall (r)
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(cfg.Keys.UninstallMode)})
	m = newModel.(*model)
	
	if m.mode != modeUninstall {
		t.Fatalf("Expected modeUninstall, got %v", m.mode)
	}
	
	if m.textInput.Value() != "" {
		t.Error("Search input should be empty in new mode")
	}
	if m.packageDash != "" {
		t.Error("Package dash should be cleared in new mode")
	}
}

func TestStateResetOnSelectiveExit(t *testing.T) {
	cfg := DefaultConfig()
	m := initialModel(modeUpdate, cfg)
	m.mode = modeUpdateSelective
	m.textInput.Focus()
	m.textInput.SetValue("lib")
	m.packageDash = "lib dashboard"
	
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
