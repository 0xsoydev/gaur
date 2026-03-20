package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// layoutScenario defines a specific state of the application to test
type layoutScenario struct {
	name   string
	width  int
	height int
	mode   viewMode
	setup  func(*model)
}

func TestGlobalLayoutIntegrity(t *testing.T) {
	scenarios := []layoutScenario{
		{
			name:   "Install Mode - Standard",
			width:  80,
			height: 24,
			mode:   modeInstall,
		},
		{
			name:   "Remove Mode - Standard",
			width:  80,
			height: 24,
			mode:   modeRemove,
		},
		{
			name:   "Dashboard - Loading State",
			width:  80,
			height: 24,
			mode:   modeDashboard,
			setup: func(m *model) {
				m.loading = true
			},
		},
		{
			name:   "Dashboard - Fully Loaded",
			width:  100,
			height: 29,
			mode:   modeDashboard,
			setup: func(m *model) {
				m.loading = false
				m.dashboard = DashboardData{
					TotalPackages:    500,
					TotalSize:        "5.00 GiB",
					DiskTotal:        "100.00 GiB",
					DiskUsed:         "50.00 GiB",
					DiskUsedPercent:  0.5,
					RepoDistribution: map[string]int{"core": 100},
					TopPackages:      []PackageSize{{Name: "test", Size: "100.00 MiB"}},
				}
			},
		},
		{
			name:   "Update Mode - Empty",
			width:  80,
			height: 24,
			mode:   modeUpdate,
			setup: func(m *model) {
				m.loading = false
				m.pendingUpdates = nil
			},
		},
		{
			name:   "Update Mode - With Updates",
			width:  80,
			height: 24,
			mode:   modeUpdate,
			setup: func(m *model) {
				m.loading = false
				m.pendingUpdates = []Package{{Name: "pkg1", Version: "1.0", Source: "extra"}}
			},
		},
		{
			name:   "Selective Update Mode",
			width:  100,
			height: 30,
			mode:   modeUpdateSelective,
			setup: func(m *model) {
				m.loading = false
				m.filtered = []Package{{Name: "pkg1", Source: "extra"}}
			},
		},
		{
			name:   "Confirmation Overlay - List",
			width:  80,
			height: 24,
			mode:   modeInstall,
			setup: func(m *model) {
				m.showConfirmation = true
				m.confirmType = confirmInstall
				m.confirmPackages = []string{"pkg1", "pkg2", "pkg3"}
			},
		},
		{
			name:   "Confirmation Overlay - Simple",
			width:  80,
			height: 24,
			mode:   modeDashboard,
			setup: func(m *model) {
				m.showConfirmation = true
				m.confirmType = confirmCleanKeep3
			},
		},
		{
			name:   "Error Overlay",
			width:  80,
			height: 24,
			mode:   modeInstall,
			setup: func(m *model) {
				m.showErrorOverlay = true
				m.errorTitle = "Critial Failure"
				m.errorMessage = "Something went wrong"
			},
		},
		{
			name:   "Small Terminal Constraint",
			width:  60,
			height: 15,
			mode:   modeInstall,
		},
		{
			name:   "Large Terminal Workspace",
			width:  150,
			height: 50,
			mode:   modeDashboard,
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			m := initialModel(s.mode, DefaultConfig())

			// Initialize window size properly
			m.Update(tea.WindowSizeMsg{Width: s.width, Height: s.height})

			if s.setup != nil {
				s.setup(m)
			}

			view := m.View()
			lines := strings.Split(view, "\n")

			// 1. HEIGHT CHECK: Must be exactly m.height
			if len(lines) != s.height {
				t.Errorf("[%s] Expected height %d, got %d", s.name, s.height, len(lines))
			}

			// 2. WIDTH & CONTENT CHECK
			for i, line := range lines {
				// Strip ANSI codes for width checking if necessary, though lipgloss.Width handles it
				actualWidth := lipgloss.Width(line)
				if actualWidth != s.width {
					// We allow the last line to be 0 width if it's an empty line after a trailing newline
					if i == len(lines)-1 && actualWidth == 0 {
						continue
					}
					t.Errorf("[%s] Line %d: expected width %d, got %d (line: %q)", s.name, i, s.width, actualWidth, line)
				}
			}

			// 3. FOOTER PINNING: The footer must be on the last line (unless overlay is active)
			// Overlay dialogs might center content, but standard views MUST pin the footer.
			if !m.showConfirmation && !m.showErrorOverlay {
				lastLine := lines[len(lines)-1]
				footerFound := strings.Contains(lastLine, "quit") || strings.Contains(lastLine, "search")
				if !footerFound {
					t.Errorf("[%s] Footer elements not found on the last line of the view", s.name)
				}
			}
		})
	}
}

func TestSplitRatioIntegrity(t *testing.T) {
	// Tests that the split between Dash and List panels is sane
	m := initialModel(modeInstall, DefaultConfig())
	m.width = 80
	m.height = 24
	m.loading = false

	view := m.View()

	// Count occurrences of top-left border corner "╭"
	// In split layout, we expect exactly 2 panels, so 2 top borders
	corners := strings.Count(view, "╭")
	if corners < 2 {
		t.Errorf("Expected at least 2 panels in split view, found %d corners", corners)
	}
}
