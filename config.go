package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/pelletier/go-toml/v2"
)

const (
	defaultConfigDir      = "gaur"
	defaultConfigFile     = "config.toml"
	defaultLogFile        = "gaur.log"
	textInputCharLimit    = 100
	textInputDefaultWidth = 50
)

// DefaultConfig returns the fallback configuration
func DefaultConfig() Config {
	return Config{
		Startup: StartupConfig{
			DefaultMode: "install",
		},
		UI: UIConfig{
			Theme:      "catppuccin-mocha",
			BorderType: "rounded",
		},
		Commands: CommandConfig{
			AurHelper:      "paru",
			InstallFlags:   "",
			UninstallFlags: "-Rns",
			CacheTool:      "paccache",
		},
		Advanced: AdvancedConfig{
			DebounceMs: 150,
			CacheDir:   "",
		},
		Keys: KeyConfig{
			Quit:           []string{"q", "ctrl+c"},
			InstallMode:    "i",
			UninstallMode:  "r",
			UpdateMode:     "u",
			DashboardMode:  "d",
			Search:         "/",
			Mark:           "tab",
			Selective:      "s",
			Settings:       ",",
			Confirm:        "enter",
			Cancel:         "esc",
		},
	}
}

// LoadConfig resolves the config path, ensures it exists, and parses it
func LoadConfig() (Config, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return DefaultConfig(), err
	}

	fullDir := filepath.Join(configDir, defaultConfigDir)
	configPath := filepath.Join(fullDir, defaultConfigFile)

	// Ensure directory exists
	if _, err := os.Stat(fullDir); os.IsNotExist(err) {
		if err := os.MkdirAll(fullDir, 0755); err != nil {
			return DefaultConfig(), err
		}
	}

	// Ensure config file exists, if not write default
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		cfg := DefaultConfig()
		if err := saveConfig(configPath, cfg); err != nil {
			return cfg, nil // Return default even if save fails
		}
		return cfg, nil
	}

	// Read and parse
	data, err := os.ReadFile(configPath)
	if err != nil {
		return DefaultConfig(), err
	}

	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		// Log error and fallback
		logFile := filepath.Join(fullDir, defaultLogFile)
		f, _ := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if f != nil {
			defer f.Close()
			logger := log.New(f, "CONFIG ERROR: ", log.LstdFlags)
			logger.Println(err)
		}
		return DefaultConfig(), nil
	}

	ValidateConfig(&cfg)
	return cfg, nil
}

// ValidateConfig ensures the configuration values are supported and safe.
func ValidateConfig(c *Config) {
	helper := strings.ToLower(strings.TrimSpace(c.Commands.AurHelper))
	if helper == "" || (helper != "paru" && helper != "yay") {
		log.Printf("Warning: unsupported AUR helper '%s'. Resetting to 'paru'.", c.Commands.AurHelper)
		c.Commands.AurHelper = "paru"
	} else {
		c.Commands.AurHelper = helper
	}
}

func saveConfig(path string, cfg Config) error {
	data, err := toml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// NewKeyMap creates a KeyMap from the configuration
func NewKeyMap(k KeyConfig) KeyMap {
	return KeyMap{
		Quit:           newBinding("quit", k.Quit...),
		InstallMode:    newBinding("install mode", k.InstallMode),
		UninstallMode:  newBinding("uninstall mode", k.UninstallMode),
		UpdateMode:     newBinding("update mode", k.UpdateMode),
		DashboardMode:  newBinding("dashboard mode", k.DashboardMode),
		Search:         newBinding("search", k.Search),
		Mark:           newBinding("mark", k.Mark),
		Selective:      newBinding("selective", k.Selective),
		Settings:       newBinding("settings", k.Settings),
		Confirm:        newBinding("confirm", k.Confirm),
		Cancel:         newBinding("cancel", k.Cancel),
	}
}

func newBinding(help string, keys ...string) key.Binding {
	return key.NewBinding(
		key.WithKeys(keys...),
		key.WithHelp("", help),
	)
}

// TokenizeFlags safely splits a flag string into a slice of strings
// Simple implementation using strings.Fields, could be improved with shlex if needed
func TokenizeFlags(flags string) []string {
	if strings.TrimSpace(flags) == "" {
		return nil
	}
	return strings.Fields(flags)
}
