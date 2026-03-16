package main

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func TestHighlightMatches(t *testing.T) {
	// matchHighlightStyle is defined in styles.go
	lipgloss.SetColorProfile(termenv.TrueColor)
	setTheme(themeCatppuccinMocha)
	s := "aur/vim"
	matchedIndices := []int{4, 5, 6} // "vim"
	
	result := highlightMatches(s, matchedIndices)
	
	if !strings.Contains(result, "v") || !strings.Contains(result, "i") || !strings.Contains(result, "m") {
		t.Errorf("highlightMatches failed to include matched characters: %q", result)
	}
	
	// Check if the number of highlighted characters matches our expectation
	// This is tricky because of ANSI codes, but we can check for their presence
	if !strings.Contains(result, "\x1b[") {
		t.Error("highlightMatches didn't include ANSI escape codes for highlighting")
	}
}

func TestHighlightMatchesWithSourceColor(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	setTheme(themeCatppuccinMocha)
	pkg := Package{Source: "aur", Name: "vim"}
	matchedIndices := []int{4, 5, 6} // "vim"
	
	result := highlightMatchesWithSourceColor(pkg, matchedIndices)
	
	if lipgloss.Width(result) != len("aur/vim") {
		t.Errorf("highlightMatchesWithSourceColor width = %d, want %d", lipgloss.Width(result), len("aur/vim"))
	}
	
	// Check without matches
	resultNoMatch := highlightMatchesWithSourceColor(pkg, nil)
	if lipgloss.Width(resultNoMatch) != len("aur/vim") {
		t.Errorf("highlightMatchesWithSourceColor (no match) width = %d, want %d", lipgloss.Width(resultNoMatch), len("aur/vim"))
	}
}

func TestTruncateWithAnsi(t *testing.T) {
	red := lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Render("Hello World")
	
	tests := []struct {
		name     string
		input    string
		width    int
		expected string // visual check
	}{
		{"no truncate", "hello", 10, "hello"},
		{"simple truncate", "hello world", 5, "hello"},
		{"ansi truncate", red, 5, "Hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateWithAnsi(tt.input, tt.width)
			actualWidth := lipgloss.Width(result)
			if actualWidth > tt.width {
				t.Errorf("truncateWithAnsi(%q, %d) width = %d, want <= %d", tt.input, tt.width, actualWidth, tt.width)
			}
		})
	}
}

func TestSubstringAnsi(t *testing.T) {
	red := lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Render("Hello World")
	
	tests := []struct {
		name  string
		input string
		skip  int
		want  string
	}{
		{"no skip", "hello", 0, "hello"},
		{"simple skip", "hello", 2, "llo"},
		{"ansi skip", red, 6, "World"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := substringAnsi(tt.input, tt.skip)
			actualWidth := lipgloss.Width(result)
			expectedWidth := lipgloss.Width(tt.input) - tt.skip
			if actualWidth != expectedWidth {
				t.Errorf("substringAnsi(%q, %d) width = %d, want %d", tt.input, tt.skip, actualWidth, expectedWidth)
			}
		})
	}
}

func TestMaintainBackground(t *testing.T) {
	bgColor := lipgloss.Color("235")
	input := "\x1b[31mRed\x1b[0m Text"
	
	result := maintainBackground(input, bgColor)
	
	// Result should contain the background color sequence after the reset
	if !strings.Contains(result, "\x1b[0m") {
		t.Error("maintainBackground stripped the reset code")
	}
	
	// Ensure it still has the original color code too
	if !strings.Contains(result, "\x1b[31m") {
		t.Error("maintainBackground stripped original foreground code")
	}
}
