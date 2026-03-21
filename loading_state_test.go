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
			initialMode:    modeRemove,
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
			initialMode:    modeDashboard,
			triggerMsg:     dashboardMsg{data: DashboardData{}},
			expectedLoaded: false,
		},
		{
			name:           "Action complete (background install/remove)",
			initialMode:    modeInstall,
			triggerMsg:     actionCompleteMsg{message: "done"},
			expectedLoaded: true, // triggers refreshAll
		},
		{
			name:           "Sync success triggers refresh",
			initialMode:    modeUpdate,
			triggerMsg:     syncRepositoriesMsg{err: nil},
			expectedLoaded: true, // triggers refreshAll
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
	m.pendingUpdates = []Package{{Name: "old-pkg"}}

	// Switch to Update Mode
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(cfg.Keys.UpdateMode[0])})
	m = newModel.(*model)

	if !m.loading {
		t.Error("expected switching to update mode to set loading = true")
	}
	if len(m.pendingUpdates) != 0 {
		t.Error("expected switching to update mode to clear pendingUpdates")
	}

	// Switch to Dashboard (n)
	m.loading = false
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(cfg.Keys.DashboardMode[0])})
	m = newModel.(*model)
	if !m.loading {
		t.Error("expected switching to dashboard mode to set loading = true")
	}

	// Switch to Remove (r)
	m.loading = false
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(cfg.Keys.RemoveMode[0])})
	m = newModel.(*model)
	if !m.loading {
		t.Error("expected switching to remove mode to set loading = true")
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
