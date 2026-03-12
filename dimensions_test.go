package main

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestViewDimensions(t *testing.T) {
	tests := []struct {
		name   string
		width  int
		height int
		mode   viewMode
		setup  func(*model)
	}{
		{
			name:   "Standard Mode Install",
			width:  80,
			height: 24,
			mode:   modeInstall,
		},
		{
			name:   "Small Terminal",
			width:  80,
			height: 15,
			mode:   modeUninstall,
		},
		{
			name:   "Large Terminal",
			width:  120,
			height: 40,
			mode:   modeInstalled,
		},
		{
			name:   "Confirmation Dialog",
			width:  80,
			height: 24,
			mode:   modeInstall,
			setup: func(m *model) {
				m.showConfirmation = true
				m.confirmType = confirmInstall
				m.confirmPackages = []string{"pkg1", "pkg2"}
			},
		},
		{
			name:   "Error Overlay",
			width:  80,
			height: 24,
			mode:   modeInstall,
			setup: func(m *model) {
				m.showErrorOverlay = true
				m.errorTitle = "Error"
				m.errorMessage = "Message"
			},
		},
		{
			name:   "Simple Update View",
			width:  80,
			height: 24,
			mode:   modeUpdate,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := initialModel(tt.mode)
			m.width = tt.width
			m.height = tt.height
			m.loading = false
			if tt.setup != nil {
				tt.setup(&m)
			}

			view := m.View()
			lines := strings.Split(view, "\n")

			// Check height (number of lines)
			if len(lines) != tt.height {
				t.Errorf("Expected height %d, got %d", tt.height, len(lines))
			}

			// Check width of each line
			for i, line := range lines {
				actualWidth := lipgloss.Width(line)
				if actualWidth != tt.width {
					// Some lines might be empty if strings.Split encounters a trailing \n,
					// but lipgloss.Place usually ensures it fills the area.
					// Let's check if it's just the last empty line from Split.
					if i == len(lines)-1 && actualWidth == 0 {
						continue
					}
					t.Errorf("Line %d: expected width %d, got %d (line: %q)", i, tt.width, actualWidth, line)
				}
			}
		})
	}
}

func TestBorderConsistency(t *testing.T) {
	// Verify that the rounded border is used and reaches the edges
	width := 80
	height := 20
	m := initialModel(modeInstall)
	m.width = width
	m.height = height
	m.loading = false

	view := m.View()
	lines := strings.Split(view, "\n")

	// The top line should have border characters at the edges
	// Rounded border characters: ╭ ╮ ╯ ╰ ─ │
	// Note: Lipgloss might use different characters based on terminal support, 
	// but we expect a border of 'width' length.
	
	firstLine := lines[0]
	lastMainLine := lines[height-2] // height-1 is the footer, height-2 is bottom border

	if lipgloss.Width(firstLine) != width {
		t.Errorf("Top border width mismatch: got %d, want %d", lipgloss.Width(firstLine), width)
	}
	
	if lipgloss.Width(lastMainLine) != width {
		t.Errorf("Bottom border width mismatch: got %d, want %d", lipgloss.Width(lastMainLine), width)
	}
}
