package main

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func TestErrorOverlayCentering(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	
	m := initialModel(modeInstall, DefaultConfig())
	m.errorTitle = "Test Error"
	m.errorMessage = "This is a multi-line\nerror message that should\nbe perfectly centered."
	
	width := 100
	height := 20
	output := m.renderErrorOverlay(width, height)
	
	lines := strings.Split(output, "\n")
	
	// Check if at least one line contains the error message and is centered
	found := false
	for _, line := range lines {
		if strings.Contains(line, "multi-line") {
			found = true
			// Calculate padding
			trimmed := strings.TrimSpace(line)
			contentWidth := lipgloss.Width(trimmed)
			
			// The line should be roughly in the middle of the 'width'
			lineOffset := strings.Index(line, trimmed)
			rightOffset := width - lineOffset - contentWidth
			
			// Allow for 1 character difference due to rounding
			if diff := lineOffset - rightOffset; diff > 1 || diff < -1 {
				t.Errorf("Line not centered: left padding %d, right padding %d. Line: %q", lineOffset, rightOffset, line)
			}
		}
	}
	
	if !found {
		t.Error("Error message lines not found in output")
	}
}

func TestConfirmationCentering(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	
	m := initialModel(modeInstall, DefaultConfig())
	m.confirmType = confirmCleanNuke
	// We don't hardcode paths here because they are now dynamic
	m.dashboard.PacmanCachePath = "/var/cache/pacman/pkg"
	path, _ := GetAURCacheDir(&m.config)
	m.dashboard.ParuCachePath = path
	
	width := 100
	height := 30
	output := m.renderConfirmationDialog(width, height, lipgloss.Color("6"))
	
	lines := strings.Split(output, "\n")
	
	// Check if the WARNING line is centered
	found := false
	for _, line := range lines {
		if strings.Contains(line, "WARNING") {
			found = true
			trimmed := strings.TrimSpace(line)
			contentWidth := lipgloss.Width(trimmed)
			lineOffset := strings.Index(line, trimmed)
			rightOffset := width - lineOffset - contentWidth
			
			if diff := lineOffset - rightOffset; diff > 1 || diff < -1 {
				t.Errorf("Warning line not centered: left padding %d, right padding %d. Line: %q", lineOffset, rightOffset, line)
			}
		}
	}
	
	if !found {
		t.Error("Warning message not found in confirmation output")
	}
}
