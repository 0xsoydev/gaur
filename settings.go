package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/pelletier/go-toml/v2"
)

// initSettings populates the settings items based on the current config
func (m *model) initSettings() {
	m.settingsItems = []SettingItem{
		{
			Label:     "AUR Helper",
			ConfigKey: "commands.aur_helper",
			Options:   []string{"paru", "yay"},
		},
		{
			Label:     "Theme",
			ConfigKey: "ui.theme",
			Options:   listThemes(), // Use existing listThemes() from styles.go
		},
		{
			Label:     "Default View",
			ConfigKey: "startup.default_mode",
			Options:   []string{"dashboard", "install", "update", "remove"},
		},
		{
			Label:     "Border Type",
			ConfigKey: "ui.border_type",
			Options:   []string{"rounded", "normal", "thick", "double"},
		},
	}

	// Set active indices based on current config
	for i, item := range m.settingsItems {
		var currentVal string
		switch item.ConfigKey {
		case "commands.aur_helper":
			currentVal = m.config.Commands.AurHelper
		case "ui.theme":
			currentVal = m.config.UI.Theme
		case "startup.default_mode":
			currentVal = m.config.Startup.DefaultMode
		case "ui.border_type":
			currentVal = m.config.UI.BorderType
		}

		for idx, opt := range item.Options {
			// Case-insensitive comparison for themes/modes
			if strings.EqualFold(opt, currentVal) || 
			   (item.Label == "Theme" && strings.EqualFold(strings.ReplaceAll(opt, " ", "-"), currentVal)) {
				m.settingsItems[i].ActiveIndex = idx
				break
			}
		}
	}
	}

	// updateConfigFromSettings applies carousel changes to the internal config struct
	func (m *model) updateConfigFromSettings() {
	item := m.settingsItems[m.settingsIndex]
	val := item.Options[item.ActiveIndex]

	switch item.ConfigKey {
	case "commands.aur_helper":
		m.config.Commands.AurHelper = val
	case "ui.theme":
		m.config.UI.Theme = val
		// Instant update: apply theme
		if t, ok := getThemeByName(val); ok {
			setTheme(t)
		}
	case "startup.default_mode":
		m.config.Startup.DefaultMode = val
	case "ui.border_type":
		m.config.UI.BorderType = val
	}

	// We no longer save instantly to disk to allow real-time theme preview without I/O churn
	}

// saveSettingsToDisk marshals current config to TOML and writes to XDG path
func (m *model) saveSettingsToDisk() {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return
	}

	configPath := filepath.Join(configDir, "gaur", "config.toml")
	
	data, err := toml.Marshal(m.config)
	if err != nil {
		return
	}

	_ = os.WriteFile(configPath, data, 0644)
}

// getBorderStyle returns the lipgloss.Border based on configuration
func (m *model) getBorderStyle() lipgloss.Border {
	switch m.config.UI.BorderType {
	case "normal":
		return lipgloss.NormalBorder()
	case "thick":
		return lipgloss.ThickBorder()
	case "double":
		return lipgloss.DoubleBorder()
	case "rounded":
		return lipgloss.RoundedBorder()
	default:
		return lipgloss.RoundedBorder()
	}
}

// renderSettings renders the btop-style settings overlay
func (m *model) renderSettings(innerWidth, innerHeight int) string {
	overlayWidth := 60
	if overlayWidth > innerWidth-4 {
		overlayWidth = innerWidth - 4
	}

	style := lipgloss.NewStyle().
		BorderStyle(m.getBorderStyle()).
		BorderForeground(currentTheme.SelectedColor).
		Padding(1, 2).
		Width(overlayWidth)

	var content strings.Builder
	
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(currentTheme.TitleColor).
		MarginBottom(1).
		Align(lipgloss.Center).
		Width(overlayWidth - 4)

	content.WriteString(titleStyle.Render("\ue690 gaur Settings"))
	content.WriteString("\n\n")

	for i, item := range m.settingsItems {
		isFocused := i == m.settingsIndex
		
		labelStyle := lipgloss.NewStyle().Width(20).Foreground(currentTheme.TextColor)
		if isFocused {
			labelStyle = labelStyle.Foreground(currentTheme.SelectedColor).Bold(true)
		}

		label := labelStyle.Render(item.Label)
		
		val := item.Options[item.ActiveIndex]
		
		// Carousel rendering: < value >
		leftArrow := "  "
		rightArrow := "  "
		if isFocused {
			leftArrow = lipgloss.NewStyle().Foreground(currentTheme.SelectedColor).Render("< ")
			rightArrow = lipgloss.NewStyle().Foreground(currentTheme.SelectedColor).Render(" >")
		}

		valStyle := lipgloss.NewStyle().Foreground(currentTheme.TextColor)
		if isFocused {
			valStyle = valStyle.Foreground(currentTheme.SelectedColor).Bold(true)
		}
		
		carousel := fmt.Sprintf("%s%s%s", leftArrow, valStyle.Render(fmt.Sprintf(" %s ", val)), rightArrow)
		
		row := fmt.Sprintf("%s %s", label, carousel)
		
		// Ensure the entire row has a consistent background when focused
		var renderedRow string
		if isFocused {
			bgColor := lipgloss.Color("235")
			
			// Pad the row manually before styling to ensure background covers full area
			targetWidth := overlayWidth - 4
			rowWidth := lipgloss.Width(row)
			paddedRow := row
			if rowWidth < targetWidth {
				paddedRow += strings.Repeat(" ", targetWidth-rowWidth)
			}

			// Apply background maintenance to the entire padded row content
			maintainedRow := maintainBackground(paddedRow, bgColor)
			
			renderedRow = lipgloss.NewStyle().
				Background(bgColor).
				Render(maintainedRow)
		} else {
			renderedRow = lipgloss.NewStyle().
				Width(overlayWidth - 4).
				Render(row)
		}
		
		content.WriteString(renderedRow + "\n")
	}

	content.WriteString("\n")
	helpStyle := lipgloss.NewStyle().Foreground(currentTheme.SubtleColor).Italic(true)
	
	hints := fmt.Sprintf("↑/↓:navigate • ←/→:cycle • %s • %s", 
		renderKeyHint("close", m.keys.Cancel, helpStyle),
		renderKeyHint("quit", m.keys.Quit, helpStyle))
	
	// Apply layout style (centering) to the entire hints line
	centeredHints := lipgloss.NewStyle().Width(overlayWidth - 4).Align(lipgloss.Center).Render(hints)
	content.WriteString(centeredHints)

	dialog := style.Render(content.String())
	return dialog
}
