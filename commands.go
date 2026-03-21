package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// debouncePackageDash returns a command that waits for the debounce duration
// then sends a tick message to trigger the actual fetch
func debouncePackageDash(m *model, pkgName string) tea.Cmd {
	duration := time.Duration(m.config.Advanced.DebounceMs) * time.Millisecond
	return tea.Tick(duration, func(t time.Time) tea.Msg {
		return debounceTickMsg{packageName: pkgName}
	})
}

// BuildAURCommand constructs the full command slice for the configured AUR helper.
func BuildAURCommand(c *Config, action string, args ...string) []string {
	helper := c.Commands.AurHelper
	if helper == "" {
		helper = "paru"
	}

	var cmd []string
	switch action {
	case "install":
		cmd = []string{helper, "-S"}
		if c.Commands.InstallFlags != "" {
			cmd = append(cmd, TokenizeFlags(c.Commands.InstallFlags)...)
		}
	case "remove":
		cmd = []string{helper}
		if c.Commands.RemoveFlags != "" {
			cmd = append(cmd, TokenizeFlags(c.Commands.RemoveFlags)...)
		} else {
			cmd = append(cmd, "-Rns")
		}
	case "update":
		// User specified that update maps to -Qu
		cmd = []string{helper, "-Qu"}
	case "search":
		cmd = []string{helper, "-Ss", "-a"}
	case "dash":
		cmd = []string{helper, "-Si"}
	case "check-updates":
		cmd = []string{helper, "-Qu"}
	case "sync":
		cmd = []string{helper, "-Sy"}
	case "full-update":
		cmd = []string{helper, "-Syu"}
	default:
		cmd = []string{helper}
	}

	return append(cmd, args...)
}

func getPackageDash(m *model, pkg Package) tea.Cmd {
	return func() tea.Msg {

		if !isValidPackageName(pkg.Name) {
			return packageDashMsg{dash: "Invalid package name", packageName: pkg.Name, err: fmt.Errorf("invalid package name: %s", pkg.Name)}
		}

		// Use -Qi for purely local lookup if we explicitly want installed version dash.
		// However, for updates/installs, we want -Si (remote dash).
		// We execute this with a timeout to prevent UI freezes on slow networks during rapid scrolling.
		arg := "-Si"
		if pkg.Installed && pkg.Source == "unknown" {
			// Only fallback to Qi if it's an orphaned/unknown local package
			arg = "-Qi"
		}

		args := []string{"--noconfirm", arg, pkg.Name}
		out, err := runner.Run(m.config.Commands.AurHelper, args...)
		if err != nil {
			return packageDashMsg{dash: "Failed to get package dashboard", packageName: pkg.Name, err: err}
		}

		return packageDashMsg{dash: string(out), packageName: pkg.Name}
	}
}

// checkDependencies verifies that all required external commands are available
func checkDependencies() error {
	requiredCommands := []struct {
		name     string
		desc     string
		required bool
	}{
		{"pacman", "system package manager", true},
		{"fzf", "fuzzy finder", true},
		{"paccache", "cache management tool (pacman-contrib)", true},
	}

	var missing []string
	for _, cmd := range requiredCommands {
		if _, err := exec.LookPath(cmd.name); err != nil {
			if cmd.required {
				missing = append(missing, fmt.Sprintf("%s (%s)", cmd.name, cmd.desc))
			}
		}
	}

	// We don't check for aur_helper here yet because config is not loaded,
	// main.go handles it by checking standard ones or we can check later.
	// For now, let's keep it simple.

	if len(missing) > 0 {
		return fmt.Errorf("missing required dependencies:\n  %s", strings.Join(missing, "\n  "))
	}
	return nil
}

// executeInstallInTerminal runs the AUR helper interactively using tea.ExecProcess
func executeInstallInTerminal(m *model, packages []string) tea.Cmd {

	validNames, _ := sanitizePackageNames(packages)
	if len(validNames) == 0 {
		return func() tea.Msg {
			return execCompleteMsg{operation: confirmInstall, packages: packages, err: fmt.Errorf("no valid package names")}
		}
	}

	args := BuildAURCommand(&m.config, "install", validNames...)

	return runner.Interactive(func(err error) tea.Msg {
		return execCompleteMsg{operation: confirmInstall, packages: validNames, err: err}
	}, args[0], args[1:]...)
}

func contains(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

// executeRemoveInTerminal runs the AUR helper interactively using tea.ExecProcess
func executeRemoveInTerminal(m *model, packages []string) tea.Cmd {

	validNames, _ := sanitizePackageNames(packages)
	if len(validNames) == 0 {
		return func() tea.Msg {
			return execCompleteMsg{operation: confirmRemove, packages: packages, err: fmt.Errorf("no valid package names")}
		}
	}

	args := BuildAURCommand(&m.config, "remove", validNames...)

	return runner.Interactive(func(err error) tea.Msg {
		return execCompleteMsg{operation: confirmRemove, packages: validNames, err: err}
	}, args[0], args[1:]...)
}

// executeUpdateInTerminal runs the AUR helper interactively using tea.ExecProcess
func executeUpdateInTerminal(m *model) tea.Cmd {
	args := BuildAURCommand(&m.config, "full-update")
	return runner.Interactive(func(err error) tea.Msg {
		return execCompleteMsg{operation: confirmUpdate, err: err}
	}, args[0], args[1:]...)
}

// executeCleanCache cleans both pacman and AUR helper caches
func executeCleanCache(m *model, op confirmationType, keep int, removed bool) tea.Cmd {
	cacheTool := m.config.Commands.CacheTool
	if cacheTool == "" {
		cacheTool = "paccache"
	}

	aurCache, _ := GetAURCacheDir(&m.config)

	// Check if AUR cache exists to avoid 'find' errors
	aurCacheExists := false
	if aurCache != "" {
		if _, err := os.Stat(aurCache); err == nil {
			aurCacheExists = true
		}
	}

	removedFlag := ""
	if removed {
		removedFlag = "-u"
	}

	// Build a script that provides feedback and handles subdirectories correctly
	script := fmt.Sprintf("echo 'Cleaning system cache...'\nsudo %s -r %s -k %d || true", cacheTool, removedFlag, keep)
	if aurCacheExists {
		// Use \; instead of + to ensure paccache is called for each subdirectory
		// We use -q to suppress "no candidate packages" noise from empty subdirs
		script += fmt.Sprintf("\necho 'Cleaning AUR helper cache...'\nfind %s -maxdepth 2 -type d -exec %s -q -r %s -k %d -c {} \\;\necho 'Done.'",
			aurCache, cacheTool, removedFlag, keep)
	}

	return runner.Interactive(func(err error) tea.Msg {
		return execCompleteMsg{operation: op, err: err}
	}, "bash", "-c", script)
}

// executeSelectiveClean specifically deletes selected cache files
func executeSelectiveClean(m *model, packages []string, pacmanCachePath string, aurCachePath string) tea.Cmd {
	if len(packages) == 0 {
		return func() tea.Msg {
			return execCompleteMsg{operation: confirmCleanSelective, err: fmt.Errorf("no packages selected")}
		}
	}

	// 1. Find all matching files across both caches
	toDelete := make(map[string]bool)
	for _, p := range packages {
		toDelete[p] = true
	}

	var files []string

	findMatches := func(dirPath string) {
		entries, err := os.ReadDir(dirPath)
		if err != nil {
			return
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			if !strings.HasSuffix(name, ".pkg.tar.zst") && !strings.HasSuffix(name, ".pkg.tar.xz") {
				continue
			}
			parts := strings.Split(name, "-")
			if len(parts) > 3 {
				baseName := strings.Join(parts[:len(parts)-3], "-")
				if toDelete[baseName] {
					files = append(files, filepath.Join(dirPath, name))
				}
			}
		}
	}

	findMatches(pacmanCachePath)
	_ = filepath.WalkDir(aurCachePath, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		name := d.Name()
		if !strings.HasSuffix(name, ".pkg.tar.zst") && !strings.HasSuffix(name, ".pkg.tar.xz") {
			return nil
		}
		parts := strings.Split(name, "-")
		if len(parts) > 3 {
			baseName := strings.Join(parts[:len(parts)-3], "-")
			if toDelete[baseName] {
				files = append(files, path)
			}
		}
		return nil
	})

	if len(files) == 0 {
		return func() tea.Msg {
			return execCompleteMsg{operation: confirmCleanSelective, err: fmt.Errorf("no matching cache files found")}
		}
	}

	// 2. Decide if we need sudo
	needsSudo := false
	for _, f := range files {
		if strings.HasPrefix(f, "/var/cache") {
			needsSudo = true
			break
		}
	}

	args := append([]string{"rm", "-f"}, files...)
	execCmd := "rm"
	execArgs := args[1:]
	if needsSudo {
		execCmd = "sudo"
		execArgs = args
	}

	return runner.Interactive(func(err error) tea.Msg {
		return execCompleteMsg{operation: confirmCleanSelective, err: err}
	}, execCmd, execArgs...)
}

// executeRemoveOrphansInTerminal runs the AUR helper interactively using tea.ExecProcess
func executeRemoveOrphansInTerminal(m *model, orphans []string) tea.Cmd {

	validNames, _ := sanitizePackageNames(orphans)
	if len(validNames) == 0 {
		return func() tea.Msg {
			return execCompleteMsg{operation: confirmRemoveOrphans, packages: orphans, err: fmt.Errorf("no valid package names")}
		}
	}

	args := BuildAURCommand(&m.config, "remove", validNames...)
	return runner.Interactive(func(err error) tea.Msg {
		return execCompleteMsg{operation: confirmRemoveOrphans, packages: validNames, err: err}
	}, args[0], args[1:]...)
}

// executeSelectiveUpdateInTerminal runs the AUR helper interactively using tea.ExecProcess
func executeSelectiveUpdateInTerminal(m *model, packages []string) tea.Cmd {
	validNames, _ := sanitizePackageNames(packages)
	if len(validNames) == 0 {
		return func() tea.Msg {
			return execCompleteMsg{operation: confirmSelectiveUpdate, packages: packages, err: fmt.Errorf("no valid package names")}
		}
	}

	args := BuildAURCommand(&m.config, "install", validNames...)
	return runner.Interactive(func(err error) tea.Msg {
		return execCompleteMsg{operation: confirmSelectiveUpdate, packages: validNames, err: err}
	}, args[0], args[1:]...)
}

// syncRepositoriesInTerminal runs the AUR helper interactively to sync databases
func syncRepositoriesInTerminal(m *model) tea.Cmd {
	args := BuildAURCommand(&m.config, "sync")
	return runner.Interactive(func(err error) tea.Msg {
		return syncRepositoriesMsg{err: err}
	}, args[0], args[1:]...)
}
