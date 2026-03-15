package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {

	if err := checkDependencies(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	cfg, err := LoadConfig()
	if err != nil {
		// Log error but proceed with default config
		fmt.Fprintf(os.Stderr, "Warning: could not load config: %v\n", err)
	}

	themeFlag := flag.String("theme", "", "Color theme (use --list-themes to see options)")
	listThemesFlag := flag.Bool("list-themes", false, "List available themes and exit")
	installFlag := flag.Bool("install", false, "Start in install mode (search and install packages)")
	installFlagShort := flag.Bool("i", false, "Short flag for install mode")
	removeFlag := flag.Bool("remove", false, "Start in remove mode (uninstall packages)")
	removeFlagShort := flag.Bool("r", false, "Short flag for remove mode")
	updateFlag := flag.Bool("update", false, "Start in update mode (system updates)")
	updateFlagShort := flag.Bool("u", false, "Short flag for update mode")
	infoFlag := flag.Bool("info", false, "Start in info mode (view installed packages)")
	infoFlagShort := flag.Bool("I", false, "Short flag for info mode (dashboard)")
	flag.Parse()

	if *listThemesFlag {
		fmt.Println("Available themes:")
		for _, name := range listThemes() {
			fmt.Printf("  - %s\n", name)
		}
		return
	}

	// Determine initial mode: Flags take precedence, then config, then default
	var initialMode viewMode
	modeFromConfig := modeInstall
	switch cfg.Startup.DefaultMode {
	case "dashboard", "installed", "info":
		modeFromConfig = modeInstalled
	case "uninstall", "remove":
		modeFromConfig = modeUninstall
	case "update":
		modeFromConfig = modeUpdate
	case "install":
		modeFromConfig = modeInstall
	}

	initialMode = modeFromConfig
	switch {
	case *installFlag || *installFlagShort:
		initialMode = modeInstall
	case *removeFlag || *removeFlagShort:
		initialMode = modeUninstall
	case *updateFlag || *updateFlagShort:
		initialMode = modeUpdate
	case *infoFlag || *infoFlagShort:
		initialMode = modeInstalled
	}

	// Theme resolution: Flag takes precedence, then config
	activeTheme := cfg.UI.Theme
	if *themeFlag != "" {
		activeTheme = *themeFlag
	}

	if activeTheme != "" {
		if t, ok := getThemeByName(activeTheme); ok {
			setTheme(t)
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
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}
