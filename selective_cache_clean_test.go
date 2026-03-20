package main

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestSelectiveCacheCleanFlow(t *testing.T) {
	// 1. Initial setup
	m := initialModel(modeInstalled, DefaultConfig())
	m.loading = false
	m.dashboard.AllCacheHogs = []PackageSize{
		{Name: "pkg1", Size: "100 MiB", SizeBytes: 100 * 1024 * 1024},
		{Name: "pkg2", Size: "200 MiB", SizeBytes: 200 * 1024 * 1024},
	}
	m.dashboard.PacmanCachePath = "/tmp/pacman"
	m.dashboard.AurCachePath = "/tmp/paru"

	// 2. Press 'c' to enter Cache Menu
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	m = newModel.(*model)
	if m.mode != modeCacheMenu {
		t.Errorf("Expected modeCacheMenu, got %v", m.mode)
	}

	// 3. Move to Selective Clean (index 4) and press Enter
	m.cacheMenuIndex = 4
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(*model)
	if m.mode != modeCacheSelective {
		t.Errorf("Expected modeCacheSelective, got %v", m.mode)
	}

	// 4. Mark 'pkg1'
	m.textInput.Blur()
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = newModel.(*model)
	if !m.markedPackages["pkg1"] {
		t.Error("Expected pkg1 to be marked")
	}
	if m.cacheToFree != 100*1024*1024 {
		t.Errorf("Expected cacheToFree 100MiB, got %d", m.cacheToFree)
	}

	// 5. Press Enter to show confirmation
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(*model)
	if !m.showConfirmation {
		t.Error("Expected confirmation dialog")
	}
	if m.confirmType != confirmCleanSelective {
		t.Errorf("Expected confirmCleanSelective, got %v", m.confirmType)
	}

	// 6. Simulate successful execution completion
	// This should clear markedPackages and cacheToFree
	newModel, _ = m.Update(execCompleteMsg{operation: confirmCleanSelective, err: nil})
	m = newModel.(*model)
	if len(m.markedPackages) != 0 {
		t.Error("Expected markedPackages to be cleared after selective clean")
	}
	if m.cacheToFree != 0 {
		t.Error("Expected cacheToFree to be reset after selective clean")
	}

	// 7. Simulate dashboard data refresh
	// This should update m.filtered with the new list
	freshData := DashboardData{
		AllCacheHogs: []PackageSize{{Name: "pkg2", Size: "200 MiB", SizeBytes: 200 * 1024 * 1024}},
	}
	m.mode = modeCacheSelective
	newModel, _ = m.Update(dashboardMsg{data: freshData})
	m = newModel.(*model)

	if len(m.filtered) != 1 || m.filtered[0].Name != "pkg2" {
		t.Errorf("Expected filtered list to be updated from fresh dashboard data. Got: %v", m.filtered)
	}
}
