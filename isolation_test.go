package main

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

func TestMarkedPackagesIsolationOnModeSwitch(t *testing.T) {
	// Start in Install mode
	m := initialModel(modeInstall, DefaultConfig())
	m.packages = []Package{{Source: "extra", Name: "vim", Version: "1.0"}}
	m.filtered = m.packages
	m.selectedIndex = 0
	
	// Set up keys
	m.keys.UninstallMode = key.NewBinding(key.WithKeys("u"))
	m.keys.DashboardMode = key.NewBinding(key.WithKeys("d"))
	m.keys.InstallMode = key.NewBinding(key.WithKeys("i"))
	m.keys.UpdateMode = key.NewBinding(key.WithKeys("U"))

	// 1. Mark a package in Install mode
	m.handleMarking()
	if !m.markedPackages["vim"] {
		t.Fatal("Expected vim to be marked in Install mode")
	}

	// 2. Switch to Uninstall mode
	resModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("u")})
	m = resModel.(*model)
	
	if m.mode != modeUninstall {
		t.Fatalf("Expected modeUninstall, got %v", m.mode)
	}
	if len(m.markedPackages) != 0 {
		t.Errorf("markedPackages should be empty after switching to Uninstall mode, got %v", m.markedPackages)
	}

	// 3. Mark something else in Uninstall mode (mock installed list)
	m.installed = []Package{{Source: "core", Name: "linux", Version: "6.0"}}
	m.filteredInstalled = m.installed
	m.selectedIndex = 0
	m.handleMarking()
	if !m.markedPackages["linux"] {
		t.Fatal("Expected linux to be marked in Uninstall mode")
	}

	// 4. Switch to Dashboard (DashboardMode)
	resModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	m = resModel.(*model)
	if m.mode != modeInstalled {
		t.Fatalf("Expected modeInstalled, got %v", m.mode)
	}
	if len(m.markedPackages) != 0 {
		t.Error("markedPackages should be empty after switching to Dashboard")
	}

	// 5. Switch to Update mode
	resModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("U")})
	m = resModel.(*model)
	if m.mode != modeUpdate {
		t.Fatalf("Expected modeUpdate, got %v", m.mode)
	}
	if len(m.markedPackages) != 0 {
		t.Error("markedPackages should be empty after switching to Update mode")
	}
}

func TestMarkedPackagesPersistenceDuringFiltering(t *testing.T) {
	// Marks should persist during search/filtering in the SAME mode
	m := initialModel(modeInstall, DefaultConfig())
	m.packages = []Package{
		{Source: "extra", Name: "vim", Version: "1.0"},
		{Source: "extra", Name: "neovim", Version: "0.8"},
	}
	m.filtered = m.packages
	
	// Mark vim
	m.selectedIndex = 0
	m.handleMarking()
	if !m.markedPackages["vim"] {
		t.Fatal("vim should be marked")
	}

	// Filter list
	m.textInput.SetValue("neo")
	m.performFiltering()
	
	if !m.markedPackages["vim"] {
		t.Error("vim should remain marked even if filtered out of view")
	}
}

func TestMarkedPackagesClearedOnCancel(t *testing.T) {
	m := initialModel(modeInstall, DefaultConfig())
	m.markedPackages["vim"] = true
	
	m.keys.Cancel = key.NewBinding(key.WithKeys("esc"))
	resModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = resModel.(*model)
	
	if len(m.markedPackages) != 0 {
		t.Error("markedPackages should be cleared on Cancel (Esc) when items are selected")
	}
}
