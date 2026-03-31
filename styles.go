package main

import (
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Theme type for TUI theming
type themeType int

const (
	themeBasic themeType = iota
	themeCatppuccinFrappe
	themeCatppuccinMacchiato
	themeCatppuccinMocha
	themeDracula
	themeGruvboxDark
	themeOneDark
	themeMonokaiPro
	themeRosePine
	themeSolarizedDark
	themeTokyonightNight
	themeTokyonightStorm
)

// Theme holds all color definitions for the UI
type Theme struct {
	Name string

	// Base colors
	BorderColor   lipgloss.Color
	SelectedColor lipgloss.Color
	TextColor     lipgloss.Color
	SubtleColor   lipgloss.Color
	TitleColor    lipgloss.Color

	// Mode colors
	InstallColor   lipgloss.Color
	DashboardColor lipgloss.Color
	RemoveColor    lipgloss.Color
	UpdateColor    lipgloss.Color

	// Source colors
	CoreColor     lipgloss.Color
	ExtraColor    lipgloss.Color
	MultilibColor lipgloss.Color
	AurColor      lipgloss.Color

	// Status colors
	SuccessColor   lipgloss.Color
	WarningColor   lipgloss.Color
	ErrorColor     lipgloss.Color
	HighlightColor lipgloss.Color

	// Dashboard colors
	DashboardLabel   lipgloss.Color
	DashboardValue   lipgloss.Color
	DashboardWarning lipgloss.Color
	DashboardDesc    lipgloss.Color
}

// Available themes
var themes = map[themeType]Theme{
	themeCatppuccinFrappe: {
		Name:             "Catppuccin Frappe",
		BorderColor:      lipgloss.Color("#737994"),
		SelectedColor:    lipgloss.Color("#ca9ee6"),
		TextColor:        lipgloss.Color("#c6d0f5"),
		SubtleColor:      lipgloss.Color("#737994"),
		TitleColor:       lipgloss.Color("#e5c890"),
		InstallColor:     lipgloss.Color("#8caaee"),
		DashboardColor:   lipgloss.Color("#f4b8e4"),
		RemoveColor:      lipgloss.Color("#e78284"),
		UpdateColor:      lipgloss.Color("#a6d189"),
		CoreColor:        lipgloss.Color("#a6d189"),
		ExtraColor:       lipgloss.Color("#8caaee"),
		MultilibColor:    lipgloss.Color("#ef9f76"),
		AurColor:         lipgloss.Color("#ca9ee6"),
		SuccessColor:     lipgloss.Color("#a6d189"),
		WarningColor:     lipgloss.Color("#e5c890"),
		ErrorColor:       lipgloss.Color("#e78284"),
		HighlightColor:   lipgloss.Color("#e5c890"),
		DashboardLabel:   lipgloss.Color("#c6d0f5"),
		DashboardValue:   lipgloss.Color("#99d1db"),
		DashboardWarning: lipgloss.Color("#e78284"),
		DashboardDesc:    lipgloss.Color("#a5adce"),
	},
	themeCatppuccinMacchiato: {
		Name:             "Catppuccin Macchiato",
		BorderColor:      lipgloss.Color("#6e738d"),
		SelectedColor:    lipgloss.Color("#c6a0f6"),
		TextColor:        lipgloss.Color("#cad3f5"),
		SubtleColor:      lipgloss.Color("#6e738d"),
		TitleColor:       lipgloss.Color("#eed49f"),
		InstallColor:     lipgloss.Color("#8aadf4"),
		DashboardColor:   lipgloss.Color("#f5bde6"),
		RemoveColor:      lipgloss.Color("#ed8796"),
		UpdateColor:      lipgloss.Color("#a6da95"),
		CoreColor:        lipgloss.Color("#a6da95"),
		ExtraColor:       lipgloss.Color("#8aadf4"),
		MultilibColor:    lipgloss.Color("#f5a97f"),
		AurColor:         lipgloss.Color("#c6a0f6"),
		SuccessColor:     lipgloss.Color("#a6da95"),
		WarningColor:     lipgloss.Color("#eed49f"),
		ErrorColor:       lipgloss.Color("#ed8796"),
		HighlightColor:   lipgloss.Color("#eed49f"),
		DashboardLabel:   lipgloss.Color("#cad3f5"),
		DashboardValue:   lipgloss.Color("#91d7e3"),
		DashboardWarning: lipgloss.Color("#ed8796"),
		DashboardDesc:    lipgloss.Color("#a5adcb"),
	},
	themeCatppuccinMocha: {
		Name:             "Catppuccin Mocha",
		BorderColor:      lipgloss.Color("#6c7086"),
		SelectedColor:    lipgloss.Color("#cba6f7"),
		TextColor:        lipgloss.Color("#cdd6f4"),
		SubtleColor:      lipgloss.Color("#6c7086"),
		TitleColor:       lipgloss.Color("#f9e2af"),
		InstallColor:     lipgloss.Color("#89b4fa"),
		DashboardColor:   lipgloss.Color("#f5c2e7"),
		RemoveColor:      lipgloss.Color("#f38ba8"),
		UpdateColor:      lipgloss.Color("#a6e3a1"),
		CoreColor:        lipgloss.Color("#a6e3a1"),
		ExtraColor:       lipgloss.Color("#89b4fa"),
		MultilibColor:    lipgloss.Color("#fab387"),
		AurColor:         lipgloss.Color("#cba6f7"),
		SuccessColor:     lipgloss.Color("#a6e3a1"),
		WarningColor:     lipgloss.Color("#f9e2af"),
		ErrorColor:       lipgloss.Color("#f38ba8"),
		HighlightColor:   lipgloss.Color("#f9e2af"),
		DashboardLabel:   lipgloss.Color("#cdd6f4"),
		DashboardValue:   lipgloss.Color("#89dceb"),
		DashboardWarning: lipgloss.Color("#f38ba8"),
		DashboardDesc:    lipgloss.Color("#a6adc8"),
	},
	themeDracula: {
		Name:             "Dracula",
		BorderColor:      lipgloss.Color("#6272a4"),
		SelectedColor:    lipgloss.Color("#bd93f9"),
		TextColor:        lipgloss.Color("#f8f8f2"),
		SubtleColor:      lipgloss.Color("#6272a4"),
		TitleColor:       lipgloss.Color("#f1fa8c"),
		InstallColor:     lipgloss.Color("#8be9fd"),
		DashboardColor:   lipgloss.Color("#ff79c6"),
		RemoveColor:      lipgloss.Color("#ff5555"),
		UpdateColor:      lipgloss.Color("#50fa7b"),
		CoreColor:        lipgloss.Color("#50fa7b"),
		ExtraColor:       lipgloss.Color("#8be9fd"),
		MultilibColor:    lipgloss.Color("#ffb86c"),
		AurColor:         lipgloss.Color("#bd93f9"),
		SuccessColor:     lipgloss.Color("#50fa7b"),
		WarningColor:     lipgloss.Color("#f1fa8c"),
		ErrorColor:       lipgloss.Color("#ff5555"),
		HighlightColor:   lipgloss.Color("#f1fa8c"),
		DashboardLabel:   lipgloss.Color("#f8f8f2"),
		DashboardValue:   lipgloss.Color("#8be9fd"),
		DashboardWarning: lipgloss.Color("#ff5555"),
		DashboardDesc:    lipgloss.Color("#6272a4"),
	},
	themeGruvboxDark: {
		Name:             "Gruvbox Dark",
		BorderColor:      lipgloss.Color("#665c54"),
		SelectedColor:    lipgloss.Color("#d3869b"),
		TextColor:        lipgloss.Color("#ebdbb2"),
		SubtleColor:      lipgloss.Color("#665c54"),
		TitleColor:       lipgloss.Color("#fabd2f"),
		InstallColor:     lipgloss.Color("#83a598"),
		DashboardColor:   lipgloss.Color("#d3869b"),
		RemoveColor:      lipgloss.Color("#fb4934"),
		UpdateColor:      lipgloss.Color("#b8bb26"),
		CoreColor:        lipgloss.Color("#b8bb26"),
		ExtraColor:       lipgloss.Color("#83a598"),
		MultilibColor:    lipgloss.Color("#fe8019"),
		AurColor:         lipgloss.Color("#d3869b"),
		SuccessColor:     lipgloss.Color("#b8bb26"),
		WarningColor:     lipgloss.Color("#fabd2f"),
		ErrorColor:       lipgloss.Color("#fb4934"),
		HighlightColor:   lipgloss.Color("#fabd2f"),
		DashboardLabel:   lipgloss.Color("#ebdbb2"),
		DashboardValue:   lipgloss.Color("#8ec07c"),
		DashboardWarning: lipgloss.Color("#fb4934"),
		DashboardDesc:    lipgloss.Color("#a89984"),
	},
	themeOneDark: {
		Name:             "One Dark",
		BorderColor:      lipgloss.Color("#5c6370"),
		SelectedColor:    lipgloss.Color("#c678dd"),
		TextColor:        lipgloss.Color("#abb2bf"),
		SubtleColor:      lipgloss.Color("#5c6370"),
		TitleColor:       lipgloss.Color("#e5c07b"),
		InstallColor:     lipgloss.Color("#61afef"),
		DashboardColor:   lipgloss.Color("#c678dd"),
		RemoveColor:      lipgloss.Color("#e06c75"),
		UpdateColor:      lipgloss.Color("#98c379"),
		CoreColor:        lipgloss.Color("#98c379"),
		ExtraColor:       lipgloss.Color("#61afef"),
		MultilibColor:    lipgloss.Color("#d19a66"),
		AurColor:         lipgloss.Color("#c678dd"),
		SuccessColor:     lipgloss.Color("#98c379"),
		WarningColor:     lipgloss.Color("#e5c07b"),
		ErrorColor:       lipgloss.Color("#e06c75"),
		HighlightColor:   lipgloss.Color("#e5c07b"),
		DashboardLabel:   lipgloss.Color("#abb2bf"),
		DashboardValue:   lipgloss.Color("#56b6c2"),
		DashboardWarning: lipgloss.Color("#e06c75"),
		DashboardDesc:    lipgloss.Color("#5c6370"),
	},
	themeMonokaiPro: {
		Name:             "Monokai Pro",
		BorderColor:      lipgloss.Color("#727072"),
		SelectedColor:    lipgloss.Color("#ab9df2"),
		TextColor:        lipgloss.Color("#fcfcfa"),
		SubtleColor:      lipgloss.Color("#727072"),
		TitleColor:       lipgloss.Color("#ffd866"),
		InstallColor:     lipgloss.Color("#78dce8"),
		DashboardColor:   lipgloss.Color("#ff6188"),
		RemoveColor:      lipgloss.Color("#ff6188"),
		UpdateColor:      lipgloss.Color("#a9dc76"),
		CoreColor:        lipgloss.Color("#a9dc76"),
		ExtraColor:       lipgloss.Color("#78dce8"),
		MultilibColor:    lipgloss.Color("#fc9867"),
		AurColor:         lipgloss.Color("#ab9df2"),
		SuccessColor:     lipgloss.Color("#a9dc76"),
		WarningColor:     lipgloss.Color("#ffd866"),
		ErrorColor:       lipgloss.Color("#ff6188"),
		HighlightColor:   lipgloss.Color("#ffd866"),
		DashboardLabel:   lipgloss.Color("#fcfcfa"),
		DashboardValue:   lipgloss.Color("#78dce8"),
		DashboardWarning: lipgloss.Color("#ff6188"),
		DashboardDesc:    lipgloss.Color("#727072"),
	},
	themeRosePine: {
		Name:             "Rose Pine",
		BorderColor:      lipgloss.Color("#6e6a86"),
		SelectedColor:    lipgloss.Color("#c4a7e7"),
		TextColor:        lipgloss.Color("#e0def4"),
		SubtleColor:      lipgloss.Color("#6e6a86"),
		TitleColor:       lipgloss.Color("#f6c177"),
		InstallColor:     lipgloss.Color("#31748f"),
		DashboardColor:   lipgloss.Color("#ebbcba"),
		RemoveColor:      lipgloss.Color("#eb6f92"),
		UpdateColor:      lipgloss.Color("#9ccfd8"),
		CoreColor:        lipgloss.Color("#9ccfd8"),
		ExtraColor:       lipgloss.Color("#31748f"),
		MultilibColor:    lipgloss.Color("#f6c177"),
		AurColor:         lipgloss.Color("#c4a7e7"),
		SuccessColor:     lipgloss.Color("#9ccfd8"),
		WarningColor:     lipgloss.Color("#f6c177"),
		ErrorColor:       lipgloss.Color("#eb6f92"),
		HighlightColor:   lipgloss.Color("#f6c177"),
		DashboardLabel:   lipgloss.Color("#e0def4"),
		DashboardValue:   lipgloss.Color("#31748f"),
		DashboardWarning: lipgloss.Color("#eb6f92"),
		DashboardDesc:    lipgloss.Color("#908caa"),
	},
	themeSolarizedDark: {
		Name:             "Solarized Dark",
		BorderColor:      lipgloss.Color("#586e75"),
		SelectedColor:    lipgloss.Color("#6c71c4"),
		TextColor:        lipgloss.Color("#839496"),
		SubtleColor:      lipgloss.Color("#586e75"),
		TitleColor:       lipgloss.Color("#b58900"),
		InstallColor:     lipgloss.Color("#268bd2"),
		DashboardColor:   lipgloss.Color("#d33682"),
		RemoveColor:      lipgloss.Color("#dc322f"),
		UpdateColor:      lipgloss.Color("#859900"),
		CoreColor:        lipgloss.Color("#859900"),
		ExtraColor:       lipgloss.Color("#268bd2"),
		MultilibColor:    lipgloss.Color("#cb4b16"),
		AurColor:         lipgloss.Color("#6c71c4"),
		SuccessColor:     lipgloss.Color("#859900"),
		WarningColor:     lipgloss.Color("#b58900"),
		ErrorColor:       lipgloss.Color("#dc322f"),
		HighlightColor:   lipgloss.Color("#b58900"),
		DashboardLabel:   lipgloss.Color("#839496"),
		DashboardValue:   lipgloss.Color("#2aa198"),
		DashboardWarning: lipgloss.Color("#dc322f"),
		DashboardDesc:    lipgloss.Color("#657b83"),
	},
	themeTokyonightNight: {
		Name:             "Tokyonight Night",
		BorderColor:      lipgloss.Color("#565f89"),
		SelectedColor:    lipgloss.Color("#bb9af7"),
		TextColor:        lipgloss.Color("#c0caf5"),
		SubtleColor:      lipgloss.Color("#565f89"),
		TitleColor:       lipgloss.Color("#e0af68"),
		InstallColor:     lipgloss.Color("#7aa2f7"),
		DashboardColor:   lipgloss.Color("#bb9af7"),
		RemoveColor:      lipgloss.Color("#f7768e"),
		UpdateColor:      lipgloss.Color("#9ece6a"),
		CoreColor:        lipgloss.Color("#9ece6a"),
		ExtraColor:       lipgloss.Color("#7aa2f7"),
		MultilibColor:    lipgloss.Color("#ff9e64"),
		AurColor:         lipgloss.Color("#bb9af7"),
		SuccessColor:     lipgloss.Color("#9ece6a"),
		WarningColor:     lipgloss.Color("#e0af68"),
		ErrorColor:       lipgloss.Color("#f7768e"),
		HighlightColor:   lipgloss.Color("#e0af68"),
		DashboardLabel:   lipgloss.Color("#c0caf5"),
		DashboardValue:   lipgloss.Color("#7dcfff"),
		DashboardWarning: lipgloss.Color("#f7768e"),
		DashboardDesc:    lipgloss.Color("#a9b1d6"),
	},
	themeTokyonightStorm: {
		Name:             "Tokyonight Storm",
		BorderColor:      lipgloss.Color("#565f89"),
		SelectedColor:    lipgloss.Color("#bb9af7"),
		TextColor:        lipgloss.Color("#c0caf5"),
		SubtleColor:      lipgloss.Color("#565f89"),
		TitleColor:       lipgloss.Color("#e0af68"),
		InstallColor:     lipgloss.Color("#7aa2f7"),
		DashboardColor:   lipgloss.Color("#bb9af7"),
		RemoveColor:      lipgloss.Color("#f7768e"),
		UpdateColor:      lipgloss.Color("#9ece6a"),
		CoreColor:        lipgloss.Color("#9ece6a"),
		ExtraColor:       lipgloss.Color("#7aa2f7"),
		MultilibColor:    lipgloss.Color("#ff9e64"),
		AurColor:         lipgloss.Color("#bb9af7"),
		SuccessColor:     lipgloss.Color("#9ece6a"),
		WarningColor:     lipgloss.Color("#e0af68"),
		ErrorColor:       lipgloss.Color("#f7768e"),
		HighlightColor:   lipgloss.Color("#e0af68"),
		DashboardLabel:   lipgloss.Color("#c0caf5"),
		DashboardValue:   lipgloss.Color("#7dcfff"),
		DashboardWarning: lipgloss.Color("#f7768e"),
		DashboardDesc:    lipgloss.Color("#a9b1d6"),
	},
}

// Current active theme
var currentTheme = themes[themeCatppuccinMocha]

// setTheme changes the active theme and updates all styles
func setTheme(t themeType) {
	if theme, ok := themes[t]; ok {
		currentTheme = theme

		defaultBorderColor = currentTheme.BorderColor
		modeColors = getModeColors()
		sourceColors = getSourceColors()

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
}

// getThemeByName returns a theme type by its name (case-insensitive)
func getThemeByName(name string) (themeType, bool) {
	nameLower := strings.ToLower(name)
	for t, theme := range themes {
		if strings.ToLower(theme.Name) == nameLower ||
			strings.ToLower(strings.ReplaceAll(theme.Name, " ", "-")) == nameLower ||
			strings.ToLower(strings.ReplaceAll(theme.Name, " ", "")) == nameLower {
			return t, true
		}
	}
	return themeBasic, false
}

// listThemes returns a list of available theme names
func listThemes() []string {
	var names []string
	for _, theme := range themes {
		names = append(names, theme.Name)
	}
	sort.Strings(names)
	return names
}

// getModeColors returns the mode colors based on current theme
func getModeColors() map[viewMode]lipgloss.Color {
	return map[viewMode]lipgloss.Color{
		modeDashboard:       currentTheme.DashboardColor,
		modeInstall:         currentTheme.InstallColor,
		modeUpdate:          currentTheme.UpdateColor,
		modeUpdateSelective: currentTheme.UpdateColor,
		modeRemove:          currentTheme.RemoveColor,
		modeCacheSelective:  lipgloss.Color("135"),
	}
}

// getSourceColors returns the source colors based on current theme
func getSourceColors() map[string]lipgloss.Color {
	return map[string]lipgloss.Color{
		"core":     currentTheme.CoreColor,
		"extra":    currentTheme.ExtraColor,
		"multilib": currentTheme.MultilibColor,
		"aur":      currentTheme.AurColor,
	}
}

// Styles - initialized with theme colors
var (
	defaultBorderColor = currentTheme.BorderColor

	// Mode-specific colors for active view highlighting
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

	matchHighlightStyle = lipgloss.NewStyle().
				Foreground(currentTheme.HighlightColor).
				Bold(true)
)

// ══════════════════════════════════════════════════════════════════════════════
// Reusable Style Helpers - avoid inline lipgloss.NewStyle() calls
// ══════════════════════════════════════════════════════════════════════════════

// styleWithWidth returns a style with the specified width.
func styleWithWidth(width int) lipgloss.Style {
	return lipgloss.NewStyle().Width(width)
}

// styleWithForeground returns a style with the specified foreground color.
func styleWithForeground(color lipgloss.Color) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(color)
}

// styleBoldWithForeground returns a bold style with foreground color.
func styleBoldWithForeground(color lipgloss.Color) lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(color)
}

// styleItalicDim returns an italic style with dimmed foreground color.
func styleItalicDim() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(currentTheme.SubtleColor).Italic(true)
}

// ══════════════════════════════════════════════════════════════════════════════
// Common Color Constants - avoid repeated color string definitions
// ══════════════════════════════════════════════════════════════════════════════

var (
	// UI Colors
	colorDimGray    = lipgloss.Color("236")
	colorLightGray  = lipgloss.Color("240")
	colorMediumGray = lipgloss.Color("241")
	colorWhite      = lipgloss.Color("252")

	// Semantic Colors
	colorRed     = lipgloss.Color("196")
	colorOrange  = lipgloss.Color("208")
	colorYellow  = lipgloss.Color("214")
	colorGreen   = lipgloss.Color("42")
	colorCyan    = lipgloss.Color("51")
	colorBlue    = lipgloss.Color("39")
	colorPurple  = lipgloss.Color("135")
	colorMagenta = lipgloss.Color("205")
)
