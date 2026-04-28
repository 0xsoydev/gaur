package main

import (
	"github.com/charmbracelet/lipgloss"
)

var currentTheme Theme

func setTheme(theme Theme) {
	currentTheme = theme

	defaultBorderColor = currentTheme.BorderColor
	modeColors = getModeColors()
	sourceColors = getSourceColors()

	updateColorVariables()

	selectedStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(currentTheme.SelectedColor)

	statusStyle = lipgloss.NewStyle().
		Foreground(currentTheme.SubtleColor)

	helpStyle = lipgloss.NewStyle().
		Foreground(currentTheme.SubtleColor).
		Bold(true)

	installedBadge = lipgloss.NewStyle().
		Foreground(currentTheme.SuccessColor).
		Bold(true)

	matchHighlightStyle = lipgloss.NewStyle().
		Foreground(currentTheme.HighlightColor).
		Bold(true)
}

func updateColorVariables() {
	colorDimGray = currentTheme.DimText
	colorLightGray = currentTheme.SubtleColor
	colorMediumGray = currentTheme.SubtleColor
	colorWhite = currentTheme.TextColor
	colorRed = currentTheme.ErrorColor
	colorOrange = currentTheme.WarningColor
	colorYellow = currentTheme.WarningColor
	colorGreen = currentTheme.SuccessColor
	colorCyan = currentTheme.DashboardValue
	colorBlue = currentTheme.InstallColor
	colorPurple = currentTheme.SelectedColor
	colorMagenta = currentTheme.SelectedColor
}

func getModeColors() map[viewMode]lipgloss.Color {
	return map[viewMode]lipgloss.Color{
		modeDashboard:       currentTheme.DashboardColor,
		modeInstall:         currentTheme.InstallColor,
		modeUpdate:          currentTheme.UpdateColor,
		modeUpdateSelective: currentTheme.UpdateColor,
		modeRemove:          currentTheme.RemoveColor,
		modeCacheSelective:  currentTheme.CacheColor,
	}
}

func getSourceColors() map[string]lipgloss.Color {
	return map[string]lipgloss.Color{
		"core":     currentTheme.CoreColor,
		"extra":    currentTheme.ExtraColor,
		"multilib": currentTheme.MultilibColor,
		"aur":      currentTheme.AurColor,
	}
}

var (
	defaultBorderColor lipgloss.Color

	modeColors = getModeColors()

	sourceColors = getSourceColors()

	baseBorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder())

	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(currentTheme.SelectedColor)

	statusStyle = lipgloss.NewStyle().
			Foreground(currentTheme.SubtleColor)

	helpStyle = lipgloss.NewStyle().
			Foreground(currentTheme.SubtleColor).
			Bold(true)

	installedBadge = lipgloss.NewStyle().
			Foreground(currentTheme.SuccessColor).
			Bold(true)

	updateBadge = lipgloss.NewStyle().
			Foreground(currentTheme.WarningColor).
			Bold(true)

	matchHighlightStyle = lipgloss.NewStyle().
				Foreground(currentTheme.HighlightColor).
				Bold(true)
)

var (
	colorDimGray    lipgloss.Color
	colorLightGray  lipgloss.Color
	colorMediumGray lipgloss.Color
	colorWhite      lipgloss.Color
	colorRed        lipgloss.Color
	colorOrange     lipgloss.Color
	colorYellow     lipgloss.Color
	colorGreen      lipgloss.Color
	colorCyan       lipgloss.Color
	colorBlue       lipgloss.Color
	colorPurple     lipgloss.Color
	colorMagenta    lipgloss.Color
)

func styleWithWidth(width int) lipgloss.Style {
	return lipgloss.NewStyle().Width(width)
}

func styleWithForeground(color lipgloss.Color) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(color)
}

func styleBoldWithForeground(color lipgloss.Color) lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(color)
}

func styleItalicDim() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(currentTheme.SubtleColor).Italic(true)
}
