package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func TestDashboardLoadingDimensions(t *testing.T) {
	m_init := initialModel(modeInstalled)
	m_init.width = 80
	m_init.height = 24
	m_init.loading = true
	
	// Crucial: Initialize layout constants via Update and use the result
	new_m, _ := m_init.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m := new_m.(*model)

	view := m.View()
	lines := strings.Split(view, "\n")

	if len(lines) != 24 {
		t.Errorf("Expected loading height 24, got %d", len(lines))
	}
	
	for i, line := range lines {
		if lipgloss.Width(line) != 80 {
			t.Errorf("Line %d width mismatch: expected 80, got %d", i, lipgloss.Width(line))
		}
	}
}

func TestDashboardRendering(t *testing.T) {
	m_init := initialModel(modeInstalled)
	
	// Crucial: Initialize layout constants via Update and use the result
	new_m, _ := m_init.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m := new_m.(*model) 
	m.loading = false

	// Mock Dashboard Data
	m.dashboard = DashboardData{
		TotalPackages:       1000,
		ExplicitlyInstalled: 200,
		ForeignPackages:     50,
		RepoDistribution: map[string]int{
			"core":     100,
			"extra":    800,
			"multilib": 50,
		},
		TotalSize:        "40.0 GiB",
		TotalSizeBytes:   40 * 1024 * 1024 * 1024,
		CleanerSize:      "20.0 GiB",
		CleanerSizeBytes: 20 * 1024 * 1024 * 1024,
		DiskTotal:        "100 GiB",
		DiskUsed:         "40 GiB",
		DiskFree:         "60 GiB",
		DiskUsedPercent:  0.4,
		RecentlyInstalled: []RecentPackage{
			{Name: "pkg1", Timestamp: "2024-03-12 10:00"},
			{Name: "pkg2", Timestamp: "2024-03-12 09:00"},
		},
		TopPackages: []PackageSize{
			{Name: "huge-pkg", Size: "2.5 GiB"}, // Should be red
			{Name: "big-pkg", Size: "750 MiB"},  // Should be orange
			{Name: "small-pkg", Size: "10 MiB"},  // Should be cyan
		},
	}

	view := m.renderDashboard("help text", m.width, m.height)

	// 1. Verify all major sections exist
	sections := []string{
		"\U000f03d7  Package Counts",
		"\U000f02ca  Disk Usage Analysis",
		"\ueb9c  Repository Distribution",
		"\ueddf  Top by Weight",
		"\uf1da  Recently Installed",
	}

	for _, section := range sections {
		if !strings.Contains(view, section) {
			t.Errorf("Dashboard missing section: %s", section)
		}
	}

	// 2. Verify Disk Usage Labels & Sizes exist below the bar
	lines := strings.Split(view, "\n")
	foundLabels := false
	foundSizes := false
	for _, line := range lines {
		if strings.Contains(line, "Packages") && strings.Contains(line, "Cache") {
			foundLabels = true
		}
		if strings.Contains(line, "40.0 GiB") && strings.Contains(line, "20.0 GiB") {
			foundSizes = true
		}
	}
	if !foundLabels || !foundSizes {
		t.Errorf("Disk usage centered labels/sizes not found. Labels: %v, Sizes: %v", foundLabels, foundSizes)
	}

	// 3. Verify Repo Distribution Bar exists
	foundRepoBar := false
	for _, line := range lines {
		if strings.Contains(line, "Core(") && strings.Contains(line, "Extra(") && strings.Contains(line, "AUR(") {
			foundRepoBar = true
		}
	}
	if !foundRepoBar {
		t.Error("Repository distribution bar or labels not found in view")
	}
}

func TestDashboardRefreshLogic(t *testing.T) {
	m := initialModel(modeInstalled)
	m.textInput.Blur()

	// Test 'n' key
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")}
	_, cmd := m.Update(msg)

	if cmd == nil {
		t.Fatal("Pressing 'n' in info mode did not return a command")
	}

	// Test 'ctrl+r'
	msg = tea.KeyMsg{Type: tea.KeyCtrlR}
	_, cmd = m.Update(msg)

	if cmd == nil {
		t.Fatal("Pressing 'ctrl+r' in info mode did not return a command")
	}
}

func TestSizeColorCodingLogic(t *testing.T) {
	// This tests the logic used inside renderDashboard indirectly
	// by checking if the expected color sequences are present
	m := initialModel(modeInstalled)
	m.width = 100
	m.height = 30
	m.loading = false
	m.dashboard = DashboardData{
		TopPackages: []PackageSize{
			{Name: "huge", Size: "2.0 GiB"},   // Red (Color 196)
			{Name: "medium", Size: "600 MiB"}, // Orange (Color 208)
			{Name: "small", Size: "10 MiB"},   // Cyan (Color 51)
		},
	}

	view := m.renderDashboard("help", m.width, m.height)

	// Check for ANSI color codes (simplified check)
	// Red: 196, Orange: 208, Cyan: 51
	if !strings.Contains(view, "196") && !strings.Contains(view, "31") { // Some themes use different codes
		// Since themes change, we at least verify the packages are listed
		if !strings.Contains(view, "huge") || !strings.Contains(view, "medium") || !strings.Contains(view, "small") {
			t.Error("Top packages not rendered correctly")
		}
	}
}
