package main

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/pelletier/go-toml/v2"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

//go:embed themes/*.toml
var embeddedThemes embed.FS

type Theme struct {
	Name string

	BorderColor   lipgloss.Color
	SelectedColor lipgloss.Color
	TextColor     lipgloss.Color
	SubtleColor   lipgloss.Color
	TitleColor    lipgloss.Color

	ScrollbarTrack lipgloss.Color
	ScrollbarThumb lipgloss.Color
	SelectionBG    lipgloss.Color
	DimText        lipgloss.Color

	InstallColor   lipgloss.Color
	DashboardColor lipgloss.Color
	RemoveColor    lipgloss.Color
	UpdateColor    lipgloss.Color
	CacheColor     lipgloss.Color

	CoreColor     lipgloss.Color
	ExtraColor    lipgloss.Color
	MultilibColor lipgloss.Color
	AurColor      lipgloss.Color

	SuccessColor   lipgloss.Color
	WarningColor   lipgloss.Color
	ErrorColor     lipgloss.Color
	HighlightColor lipgloss.Color

	DashboardLabel   lipgloss.Color
	DashboardValue   lipgloss.Color
	DashboardWarning lipgloss.Color
	DashboardDesc    lipgloss.Color

	DialogBorder     lipgloss.Color
	ConfirmInstall   lipgloss.Color
	ConfirmRemove    lipgloss.Color
	ConfirmClean     lipgloss.Color
	ConfirmNuke      lipgloss.Color
	ConfirmSelective lipgloss.Color

	ButtonBg       lipgloss.Color
	ButtonFg       lipgloss.Color
	ButtonDangerBg lipgloss.Color
	ProgressTrack  lipgloss.Color
	AccentColor    lipgloss.Color
	SpinnerColor   lipgloss.Color
}

type tomlTheme struct {
	Border   string `toml:"border"`
	Selected string `toml:"selected"`
	Text     string `toml:"text"`
	Subtle   string `toml:"subtle"`
	Title    string `toml:"title"`

	ScrollbarTrack string `toml:"scrollbar_track"`
	ScrollbarThumb string `toml:"scrollbar_thumb"`
	SelectionBG    string `toml:"selection_bg"`
	DimText        string `toml:"dim_text"`

	Install   string `toml:"install"`
	Dashboard string `toml:"dashboard"`
	Remove    string `toml:"remove"`
	Update    string `toml:"update"`
	Cache     string `toml:"cache"`

	Core     string `toml:"core"`
	Extra    string `toml:"extra"`
	Multilib string `toml:"multilib"`
	Aur      string `toml:"aur"`

	Success   string `toml:"success"`
	Warning   string `toml:"warning"`
	Error     string `toml:"error"`
	Highlight string `toml:"highlight"`

	DashboardLabel   string `toml:"dashboard_label"`
	DashboardValue   string `toml:"dashboard_value"`
	DashboardWarning string `toml:"dashboard_warning"`
	DashboardDesc    string `toml:"dashboard_desc"`

	DialogBorder     string `toml:"dialog_border"`
	ConfirmInstall   string `toml:"confirm_install"`
	ConfirmRemove    string `toml:"confirm_remove"`
	ConfirmClean     string `toml:"confirm_clean"`
	ConfirmNuke      string `toml:"confirm_nuke"`
	ConfirmSelective string `toml:"confirm_selective"`

	ButtonBg       string `toml:"button_bg"`
	ButtonFg       string `toml:"button_fg"`
	ButtonDangerBg string `toml:"button_danger_bg"`
	ProgressTrack  string `toml:"progress_track"`
	AccentColor    string `toml:"accent"`
	SpinnerColor   string `toml:"spinner"`
}

type ThemeLoader struct {
	themes        map[string]Theme
	userThemesDir string
}

var globalThemeLoader *ThemeLoader

func InitThemeLoader() (*ThemeLoader, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("could not determine config directory: %w", err)
	}

	tl := &ThemeLoader{
		themes:        make(map[string]Theme),
		userThemesDir: filepath.Join(configDir, "gaur", "themes"),
	}

	if err := tl.loadAll(); err != nil {
		return nil, err
	}

	globalThemeLoader = tl
	return tl, nil
}

func GetThemeLoader() *ThemeLoader {
	return globalThemeLoader
}

func (tl *ThemeLoader) loadAll() error {
	if err := tl.loadEmbedded(); err != nil {
		return fmt.Errorf("failed to load embedded themes: %w", err)
	}

	tl.loadUserThemes()

	return nil
}

func (tl *ThemeLoader) loadEmbedded() error {
	entries, err := embeddedThemes.ReadDir("themes")
	if err != nil {
		return fmt.Errorf("could not read embedded themes: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".toml") {
			continue
		}

		data, err := embeddedThemes.ReadFile("themes/" + entry.Name())
		if err != nil {
			LogWarn("THEME", "Could not read embedded theme %s: %v", entry.Name(), err)
			continue
		}

		theme, err := tl.parseTheme(data, entry.Name())
		if err != nil {
			LogWarn("THEME", "Could not parse embedded theme %s: %v", entry.Name(), err)
			continue
		}

		displayName := filenameToDisplayName(entry.Name())
		theme.Name = displayName
		tl.themes[displayName] = theme
		LogDebug("THEME", "Loaded embedded theme: %s", displayName)
	}

	return nil
}

func (tl *ThemeLoader) loadUserThemes() {
	if _, err := os.Stat(tl.userThemesDir); os.IsNotExist(err) {
		LogDebug("THEME", "User themes directory does not exist: %s", tl.userThemesDir)
		return
	}

	entries, err := os.ReadDir(tl.userThemesDir)
	if err != nil {
		LogWarn("THEME", "Could not read user themes directory: %v", err)
		return
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".toml") {
			continue
		}

		filePath := filepath.Join(tl.userThemesDir, entry.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			LogWarn("THEME", "Could not read user theme %s: %v", entry.Name(), err)
			continue
		}

		theme, err := tl.parseTheme(data, entry.Name())
		if err != nil {
			LogWarn("THEME", "Could not parse user theme %s: %v", entry.Name(), err)
			continue
		}

		displayName := filenameToDisplayName(entry.Name())
		theme.Name = displayName
		tl.themes[displayName] = theme
		LogDebug("THEME", "Loaded user theme: %s (overrides embedded if exists)", displayName)
	}
}

func (tl *ThemeLoader) parseTheme(data []byte, filename string) (Theme, error) {
	var tt tomlTheme
	if err := toml.Unmarshal(data, &tt); err != nil {
		return Theme{}, fmt.Errorf("TOML parse error: %w", err)
	}

	theme := Theme{
		BorderColor:      lipgloss.Color(sanitizeColor(tt.Border)),
		SelectedColor:    lipgloss.Color(sanitizeColor(tt.Selected)),
		TextColor:        lipgloss.Color(sanitizeColor(tt.Text)),
		SubtleColor:      lipgloss.Color(sanitizeColor(tt.Subtle)),
		TitleColor:       lipgloss.Color(sanitizeColor(tt.Title)),
		ScrollbarTrack:   lipgloss.Color(sanitizeColor(tt.ScrollbarTrack)),
		ScrollbarThumb:   lipgloss.Color(sanitizeColor(tt.ScrollbarThumb)),
		SelectionBG:      lipgloss.Color(sanitizeColor(tt.SelectionBG)),
		DimText:          lipgloss.Color(sanitizeColor(tt.DimText)),
		InstallColor:     lipgloss.Color(sanitizeColor(tt.Install)),
		DashboardColor:   lipgloss.Color(sanitizeColor(tt.Dashboard)),
		RemoveColor:      lipgloss.Color(sanitizeColor(tt.Remove)),
		UpdateColor:      lipgloss.Color(sanitizeColor(tt.Update)),
		CacheColor:       lipgloss.Color(sanitizeColor(tt.Cache)),
		CoreColor:        lipgloss.Color(sanitizeColor(tt.Core)),
		ExtraColor:       lipgloss.Color(sanitizeColor(tt.Extra)),
		MultilibColor:    lipgloss.Color(sanitizeColor(tt.Multilib)),
		AurColor:         lipgloss.Color(sanitizeColor(tt.Aur)),
		SuccessColor:     lipgloss.Color(sanitizeColor(tt.Success)),
		WarningColor:     lipgloss.Color(sanitizeColor(tt.Warning)),
		ErrorColor:       lipgloss.Color(sanitizeColor(tt.Error)),
		HighlightColor:   lipgloss.Color(sanitizeColor(tt.Highlight)),
		DashboardLabel:   lipgloss.Color(sanitizeColor(tt.DashboardLabel)),
		DashboardValue:   lipgloss.Color(sanitizeColor(tt.DashboardValue)),
		DashboardWarning: lipgloss.Color(sanitizeColor(tt.DashboardWarning)),
		DashboardDesc:    lipgloss.Color(sanitizeColor(tt.DashboardDesc)),
		DialogBorder:     lipgloss.Color(sanitizeColor(tt.DialogBorder)),
		ConfirmInstall:   lipgloss.Color(sanitizeColor(tt.ConfirmInstall)),
		ConfirmRemove:    lipgloss.Color(sanitizeColor(tt.ConfirmRemove)),
		ConfirmClean:     lipgloss.Color(sanitizeColor(tt.ConfirmClean)),
		ConfirmNuke:      lipgloss.Color(sanitizeColor(tt.ConfirmNuke)),
		ConfirmSelective: lipgloss.Color(sanitizeColor(tt.ConfirmSelective)),
		ButtonBg:         lipgloss.Color(sanitizeColor(tt.ButtonBg)),
		ButtonFg:         lipgloss.Color(sanitizeColor(tt.ButtonFg)),
		ButtonDangerBg:   lipgloss.Color(sanitizeColor(tt.ButtonDangerBg)),
		ProgressTrack:    lipgloss.Color(sanitizeColor(tt.ProgressTrack)),
		AccentColor:      lipgloss.Color(sanitizeColor(tt.AccentColor)),
		SpinnerColor:     lipgloss.Color(sanitizeColor(tt.SpinnerColor)),
	}

	theme = applyDefaults(theme, filename)

	return theme, nil
}

func sanitizeColor(color string) string {
	color = strings.TrimSpace(color)
	if color == "" {
		return "#ffffff"
	}
	return color
}

func applyDefaults(theme Theme, filename string) Theme {
	defaults := getFallbackDefaults(filename)

	if string(theme.BorderColor) == "" || string(theme.BorderColor) == "#ffffff" {
		theme.BorderColor = defaults.BorderColor
	}
	if string(theme.SelectedColor) == "" || string(theme.SelectedColor) == "#ffffff" {
		theme.SelectedColor = defaults.SelectedColor
	}
	if string(theme.TextColor) == "" || string(theme.TextColor) == "#ffffff" {
		theme.TextColor = defaults.TextColor
	}
	if string(theme.ScrollbarTrack) == "" || string(theme.ScrollbarTrack) == "#ffffff" {
		theme.ScrollbarTrack = defaults.ScrollbarTrack
	}
	if string(theme.ScrollbarThumb) == "" || string(theme.ScrollbarThumb) == "#ffffff" {
		theme.ScrollbarThumb = defaults.ScrollbarThumb
	}
	if string(theme.SelectionBG) == "" || string(theme.SelectionBG) == "#ffffff" {
		theme.SelectionBG = defaults.SelectionBG
	}
	if string(theme.DimText) == "" || string(theme.DimText) == "#ffffff" {
		theme.DimText = defaults.DimText
	}
	if string(theme.CacheColor) == "" || string(theme.CacheColor) == "#ffffff" {
		theme.CacheColor = defaults.CacheColor
	}
	if string(theme.ButtonBg) == "" || string(theme.ButtonBg) == "#ffffff" {
		theme.ButtonBg = defaults.ButtonBg
	}
	if string(theme.ButtonFg) == "" || string(theme.ButtonFg) == "#ffffff" {
		theme.ButtonFg = defaults.ButtonFg
	}
	if string(theme.ButtonDangerBg) == "" || string(theme.ButtonDangerBg) == "#ffffff" {
		theme.ButtonDangerBg = defaults.ButtonDangerBg
	}
	if string(theme.ProgressTrack) == "" || string(theme.ProgressTrack) == "#ffffff" {
		theme.ProgressTrack = defaults.ProgressTrack
	}
	if string(theme.AccentColor) == "" || string(theme.AccentColor) == "#ffffff" {
		theme.AccentColor = defaults.AccentColor
	}
	if string(theme.SpinnerColor) == "" || string(theme.SpinnerColor) == "#ffffff" {
		theme.SpinnerColor = defaults.SpinnerColor
	}

	return theme
}

func getFallbackDefaults(filename string) Theme {
	return Theme{
		BorderColor:      lipgloss.Color("#6c7086"),
		SelectedColor:    lipgloss.Color("#cba6f7"),
		TextColor:        lipgloss.Color("#cdd6f4"),
		SubtleColor:      lipgloss.Color("#6c7086"),
		TitleColor:       lipgloss.Color("#f9e2af"),
		ScrollbarTrack:   lipgloss.Color("#181825"),
		ScrollbarThumb:   lipgloss.Color("#6c7086"),
		SelectionBG:      lipgloss.Color("#313244"),
		DimText:          lipgloss.Color("#6c7086"),
		InstallColor:     lipgloss.Color("#89b4fa"),
		DashboardColor:   lipgloss.Color("#f5c2e7"),
		RemoveColor:      lipgloss.Color("#f38ba8"),
		UpdateColor:      lipgloss.Color("#a6e3a1"),
		CacheColor:       lipgloss.Color("#cba6f7"),
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
		DialogBorder:     lipgloss.Color("#cba6f7"),
		ConfirmInstall:   lipgloss.Color("#89b4fa"),
		ConfirmRemove:    lipgloss.Color("#fab387"),
		ConfirmClean:     lipgloss.Color("#a6e3a1"),
		ConfirmNuke:      lipgloss.Color("#f38ba8"),
		ConfirmSelective: lipgloss.Color("#cba6f7"),
		ButtonBg:         lipgloss.Color("#45475a"),
		ButtonFg:         lipgloss.Color("#cdd6f4"),
		ButtonDangerBg:   lipgloss.Color("#f38ba8"),
		ProgressTrack:    lipgloss.Color("#313244"),
		AccentColor:      lipgloss.Color("#89b4fa"),
		SpinnerColor:     lipgloss.Color("#f5c2e7"),
	}
}

func filenameToDisplayName(filename string) string {
	name := strings.TrimSuffix(filename, ".toml")
	name = strings.ReplaceAll(name, "_", " ")
	name = strings.ReplaceAll(name, "-", " ")
	return cases.Title(language.English).String(name)
}

func (tl *ThemeLoader) GetTheme(name string) (Theme, bool) {
	theme, ok := tl.themes[name]
	return theme, ok
}

func (tl *ThemeLoader) GetThemeByConfigName(configName string) (Theme, bool) {
	normalizedName := normalizeThemeName(configName)

	for displayName, theme := range tl.themes {
		if normalizeThemeName(displayName) == normalizedName {
			return theme, true
		}
	}

	return Theme{}, false
}

func normalizeThemeName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, "_", "-")
	return name
}

func (tl *ThemeLoader) ListThemes() []string {
	var names []string
	for name := range tl.themes {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (tl *ThemeLoader) ExportDefaults(destDir string) error {
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("could not create themes directory: %w", err)
	}

	entries, err := embeddedThemes.ReadDir("themes")
	if err != nil {
		return fmt.Errorf("could not read embedded themes: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".toml") {
			continue
		}

		data, err := embeddedThemes.ReadFile("themes/" + entry.Name())
		if err != nil {
			return fmt.Errorf("could not read embedded theme %s: %w", entry.Name(), err)
		}

		destPath := filepath.Join(destDir, entry.Name())
		if err := os.WriteFile(destPath, data, 0644); err != nil {
			return fmt.Errorf("could not write theme %s: %w", entry.Name(), err)
		}
	}

	return nil
}

func (tl *ThemeLoader) GetUserThemesDir() string {
	return tl.userThemesDir
}
