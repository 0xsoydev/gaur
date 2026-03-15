package main

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestLoadingStateTransitions(t *testing.T) {
	cfg := DefaultConfig()

	tests := []struct {
		name           string
		initialMode    viewMode
		triggerMsg     tea.Msg
		expectedLoaded bool
	}{
		{
			name:           "Repo packages loaded",
			initialMode:    modeInstall,
			triggerMsg:     repoPackagesMsg{packages: []Package{}},
			expectedLoaded: false,
		},
		{
			name:           "Installed packages loaded",
			initialMode:    modeUninstall,
			triggerMsg:     installedPackagesMsg{packages: []Package{}},
			expectedLoaded: false,
		},
		{
			name:           "Update check completed",
			initialMode:    modeUpdate,
			triggerMsg:     updateCheckMsg{packages: []Package{}},
			expectedLoaded: false,
		},
		{
			name:           "Update check error",
			initialMode:    modeUpdate,
			triggerMsg:     updateCheckMsg{err: fmt.Errorf("error")},
			expectedLoaded: false,
		},
		{
			name:           "Dashboard data loaded",
			initialMode:    modeInstalled,
			triggerMsg:     dashboardMsg{data: DashboardData{}},
			expectedLoaded: false,
		},
		{
			name:           "Action complete (background install/uninstall)",
			initialMode:    modeInstall,
			triggerMsg:     actionCompleteMsg{message: "done"},
			expectedLoaded: false,
		},
		{
			name:           "Sync failed clears loading",
			initialMode:    modeUpdate,
			triggerMsg:     syncRepositoriesMsg{err: fmt.Errorf("sync error")},
			expectedLoaded: false,
		},
		{
			name:           "Exec complete (interactive update) triggers refresh",
			initialMode:    modeUpdate,
			triggerMsg:     execCompleteMsg{operation: confirmUpdate},
			expectedLoaded: true, // It sets loading=true to refresh the list
		},
		{
			name:           "Exec complete error clears loading",
			initialMode:    modeUpdate,
			triggerMsg:     execCompleteMsg{operation: confirmUpdate, err: fmt.Errorf("fail")},
			expectedLoaded: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := initialModel(tt.initialMode, cfg)
			m.loading = true // Simulate being in loading state

			newModel, _ := m.Update(tt.triggerMsg)
			res := newModel.(*model)

			if res.loading != tt.expectedLoaded {
				t.Errorf("expected loading to be %v, got %v", tt.expectedLoaded, res.loading)
			}
		})
	}
}

func TestModeSwitchesSetLoading(t *testing.T) {
	cfg := DefaultConfig()
	m := initialModel(modeInstall, cfg)
	m.loading = false

	// Switch to Update Mode
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(cfg.Keys.UpdateMode)})
	m = newModel.(*model)
	// Success logic for update mode: it triggers syncRepositoriesInTerminal, but doesn't immediately set m.loading=true 
	// until syncRepositoriesMsg or checkUpdates is received or handled.
	// Actually, Update() logic for UpdateMode:
	/*
		case key.Matches(msg, m.keys.UpdateMode):
			m.mode = modeUpdate
			return m, syncRepositoriesInTerminal(m)
	*/
	// It doesn't set m.loading = true explicitly in the key handler, 
	// let's check if it SHOULD.

	// Switch to Dashboard (n)
	m.loading = false
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(cfg.Keys.DashboardMode)})
	m = newModel.(*model)
	if !m.loading {
		t.Error("expected switching to dashboard mode to set loading = true")
	}

	// Switch to Uninstall (r)
	m.loading = false
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(cfg.Keys.UninstallMode)})
	m = newModel.(*model)
	if !m.loading {
		t.Error("expected switching to uninstall mode to set loading = true")
	}
}

func TestSyncTransitionsLoading(t *testing.T) {
	cfg := DefaultConfig()
	m := initialModel(modeUpdate, cfg)
	
	// 1. Receive Sync Success
	// case syncRepositoriesMsg: if msg.err == nil { m.loading = true; return m, checkUpdates() }
	newModel, _ := m.Update(syncRepositoriesMsg{err: nil})
	m = newModel.(*model)
	if !m.loading {
		t.Error("expected successful sync to maintain or set loading = true")
	}

	// 2. Receive Update Check Result
	newModel, _ = m.Update(updateCheckMsg{packages: []Package{}})
	m = newModel.(*model)
	if m.loading {
		t.Error("expected update check result to set loading = false")
	}
}
