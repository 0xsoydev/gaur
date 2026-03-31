package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// Parse flags first to check for list-themes (which doesn't need deps)
	themeFlag := flag.String("theme", "", "Color theme (use --list-themes to see options)")
	listThemesFlag := flag.Bool("list-themes", false, "List available themes and exit")
	installFlag := flag.Bool("install", false, "Start in install mode (search and install packages)")
	installFlagShort := flag.Bool("i", false, "Short flag for install mode")
	removeFlag := flag.Bool("remove", false, "Start in remove mode (remove packages)")
	removeFlagShort := flag.Bool("r", false, "Short flag for remove mode")
	updateFlag := flag.Bool("update", false, "Start in update mode (system updates)")
	updateFlagShort := flag.Bool("u", false, "Short flag for update mode")
	dashFlag := flag.Bool("dash", false, "Start in dashboard mode (view system stats)")
	dashFlagShort := flag.Bool("d", false, "Short flag for dashboard mode")
	flag.Parse()

	if *listThemesFlag {
		fmt.Println("Available themes:")
		for _, name := range listThemes() {
			fmt.Printf("  - %s\n", name)
		}
		return
	}

	// Load configuration
	cfg, err := LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not load config: %v\n", err)
	}

	// Initialize logger based on config
	logLevel := LogLevelFromString(cfg.Logging.Level)
	if err := InitLogger(logLevel, ""); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not initialize logger: %v\n", err)
	}
	defer CloseLogger()

	// Check dependencies
	if err := checkDependencies(); err != nil {
		LogError("STARTUP", "Dependency check failed: %v", err)
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	LogDebug("STARTUP", "All dependencies satisfied")

	// Determine initial mode: Flags take precedence, then config, then default
	var initialMode viewMode
	modeFromConfig := modeInstall
	switch cfg.Startup.DefaultMode {
	case "dashboard", "dash":
		modeFromConfig = modeDashboard
	case "remove":
		modeFromConfig = modeRemove
	case "update":
		modeFromConfig = modeUpdate
	case "install":
		modeFromConfig = modeInstall
	}

	initialMode = modeFromConfig
	switch {
	case *installFlag || *installFlagShort:
		initialMode = modeInstall
		LogInfo("STARTUP", "Starting in install mode (flag override)")
	case *removeFlag || *removeFlagShort:
		initialMode = modeRemove
		LogInfo("STARTUP", "Starting in remove mode (flag override)")
	case *updateFlag || *updateFlagShort:
		initialMode = modeUpdate
		LogInfo("STARTUP", "Starting in update mode (flag override)")
	case *dashFlag || *dashFlagShort:
		initialMode = modeDashboard
		LogInfo("STARTUP", "Starting in dashboard mode (flag override)")
	default:
		LogInfo("STARTUP", "Starting in %s mode (from config)", modeString(initialMode))
	}

	// Theme resolution: Flag takes precedence, then config
	activeTheme := cfg.UI.Theme
	if *themeFlag != "" {
		activeTheme = *themeFlag
		LogInfo("STARTUP", "Theme override from flag: %s", activeTheme)
	}

	if activeTheme != "" {
		if t, ok := getThemeByName(activeTheme); ok {
			setTheme(t)
			LogDebug("STARTUP", "Theme set to: %s", activeTheme)
		} else if *themeFlag != "" {
			fmt.Printf("Unknown theme: %s\nAvailable themes:\n", *themeFlag)
			for _, name := range listThemes() {
				fmt.Printf("  - %s\n", name)
			}
			os.Exit(1)
		}
	}

	m := initialModel(initialMode, cfg)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseAllMotion())
	LogInfo("STARTUP", "TUI program starting")
	if _, err := p.Run(); err != nil {
		LogError("STARTUP", "Program error: %v", err)
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
	LogInfo("STARTUP", "Program exited normally")
}
