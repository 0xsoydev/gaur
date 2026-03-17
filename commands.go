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

// debouncePackageInfo returns a command that waits for the debounce duration
// then sends a tick message to trigger the actual fetch
func debouncePackageInfo(m *model, pkgName string) tea.Cmd {
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
		} else {
			cmd = append(cmd, "--noconfirm")
		}
	case "remove":
		cmd = []string{helper}
		if c.Commands.UninstallFlags != "" {
			cmd = append(cmd, TokenizeFlags(c.Commands.UninstallFlags)...)
		} else {
			cmd = append(cmd, "-Rns")
		}
	case "update":
		// User specified that update maps to -Qu
		cmd = []string{helper, "-Qu"}
	case "search":
		cmd = []string{helper, "-Ss", "-a"}
	case "info":
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

func getPackageInfo(m *model, pkg Package) tea.Cmd {
	return func() tea.Msg {

		if !isValidPackageName(pkg.Name) {
			return packageInfoMsg{info: "Invalid package name", packageName: pkg.Name, err: fmt.Errorf("invalid package name: %s", pkg.Name)}
		}

		// Use -Qi for purely local lookup if we explicitly want installed version info.
		// However, for updates/installs, we want -Si (remote info). 
		// We execute this with a timeout to prevent UI freezes on slow networks during rapid scrolling.
		arg := "-Si"
		if pkg.Installed && pkg.Source == "unknown" {
			// Only fallback to Qi if it's an orphaned/unknown local package
			arg = "-Qi"
		}

		args := []string{"--noconfirm", arg, pkg.Name}
		out, err := runner.Run(m.config.Commands.AurHelper, args...)
		if err != nil {
			return packageInfoMsg{info: "Failed to get package info", packageName: pkg.Name, err: err}
		}

		return packageInfoMsg{info: string(out), packageName: pkg.Name}
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

// executeUninstallInTerminal runs the AUR helper interactively using tea.ExecProcess
func executeUninstallInTerminal(m *model, packages []string) tea.Cmd {

	validNames, _ := sanitizePackageNames(packages)
	if len(validNames) == 0 {
		return func() tea.Msg {
			return execCompleteMsg{operation: confirmUninstall, packages: packages, err: fmt.Errorf("no valid package names")}
		}
	}

	args := BuildAURCommand(&m.config, "remove", validNames...)

	return runner.Interactive(func(err error) tea.Msg {
		return execCompleteMsg{operation: confirmUninstall, packages: validNames, err: err}
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
func executeCleanCache(m *model, op confirmationType, keep int, uninstalled bool) tea.Cmd {
	cacheTool := m.config.Commands.CacheTool
	if cacheTool == "" {
		cacheTool = "paccache"
	}

	aurCache, _ := GetAURCacheDir(&m.config)

	return runner.Interactive(func(err error) tea.Msg {
		return execCompleteMsg{operation: op, err: err}
	}, "bash", "-c", fmt.Sprintf(`
		# Clean pacman cache
		%s -r %s -k %d
		# Clean AUR helper cache (recursive)
		find %s -maxdepth 2 -type d -exec %s -r %s -k %d -c {} +
	`,
		cacheTool,
		func() string { if uninstalled { return "-u" }; return "" }(),
		keep,
		aurCache,
		cacheTool,
		func() string { if uninstalled { return "-u" }; return "" }(),
		keep))
}

// executeSelectiveClean specifically deletes selected cache files
func executeSelectiveClean(packages []string, pacmanCachePath string, paruCachePath string) tea.Cmd {
	return func() tea.Msg {
		if len(packages) == 0 {
			return execCompleteMsg{operation: confirmCleanSelective, err: fmt.Errorf("no packages selected")}
		}

		// Create a quick lookup map for the base names we want to delete
		toDelete := make(map[string]bool)
		for _, p := range packages {
			toDelete[p] = true
		}

		deleteMatches := func(dirPath string) error {
			entries, err := os.ReadDir(dirPath)
			if err != nil {
				return err
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
						// Found a match, delete it
						filePath := filepath.Join(dirPath, name)
						_ = os.Remove(filePath)
					}
				}
			}
			return nil
		}

		// Delete from pacman cache
		_ = deleteMatches(pacmanCachePath)

		// Delete from paru clone cache (nested)
		_ = filepath.WalkDir(paruCachePath, func(path string, d os.DirEntry, err error) error {
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
					_ = os.Remove(path)
				}
			}
			return nil
		})

		return execCompleteMsg{operation: confirmCleanSelective, err: nil}
	}
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
