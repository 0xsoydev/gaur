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

	initialMode := modeInstall
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

	if *themeFlag != "" {
		if t, ok := getThemeByName(*themeFlag); ok {
			setTheme(t)
		} else {
			fmt.Printf("Unknown theme: %s\nAvailable themes:\n", *themeFlag)
			for _, name := range listThemes() {
				fmt.Printf("  - %s\n", name)
			}
			os.Exit(1)
		}
	}

	m := initialModel(initialMode)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseAllMotion())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}
