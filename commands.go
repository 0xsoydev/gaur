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
func debouncePackageInfo(pkgName string) tea.Cmd {
	return tea.Tick(packageInfoDebounceTime, func(t time.Time) tea.Msg {
		return debounceTickMsg{packageName: pkgName}
	})
}

func getPackageInfo(pkg Package) tea.Cmd {
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

		out, err := runner.Run("paru", "--noconfirm", arg, pkg.Name)
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
		{"paru", "AUR helper", true},
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

	if len(missing) > 0 {
		return fmt.Errorf("missing required dependencies:\n  %s", strings.Join(missing, "\n  "))
	}
	return nil
}

// executeInstallInTerminal runs paru -S interactively using tea.ExecProcess
func executeInstallInTerminal(packages []string) tea.Cmd {

	validNames, _ := sanitizePackageNames(packages)
	if len(validNames) == 0 {
		return func() tea.Msg {
			return execCompleteMsg{operation: confirmInstall, packages: packages, err: fmt.Errorf("no valid package names")}
		}
	}

	args := append([]string{"-S"}, validNames...)
	return runner.Interactive(func(err error) tea.Msg {
		return execCompleteMsg{operation: confirmInstall, packages: validNames, err: err}
	}, "paru", args...)
}

// executeUninstallInTerminal runs paru -Rns interactively using tea.ExecProcess
func executeUninstallInTerminal(packages []string) tea.Cmd {

	validNames, _ := sanitizePackageNames(packages)
	if len(validNames) == 0 {
		return func() tea.Msg {
			return execCompleteMsg{operation: confirmUninstall, packages: packages, err: fmt.Errorf("no valid package names")}
		}
	}

	args := append([]string{"-Rns"}, validNames...)
	return runner.Interactive(func(err error) tea.Msg {
		return execCompleteMsg{operation: confirmUninstall, packages: validNames, err: err}
	}, "paru", args...)
}

// executeUpdateInTerminal runs paru -Syu interactively using tea.ExecProcess
func executeUpdateInTerminal() tea.Cmd {
	return runner.Interactive(func(err error) tea.Msg {
		return execCompleteMsg{operation: confirmUpdate, err: err}
	}, "paru", "-Syu")
}

// executeCleanCache cleans both pacman and paru caches
func executeCleanCache(op confirmationType, keep int, uninstalled bool) tea.Cmd {
	return runner.Interactive(func(err error) tea.Msg {
		return execCompleteMsg{operation: op, err: err}
	}, "bash", "-c", fmt.Sprintf(`
		# Clean pacman cache
		paccache -r %s -k %d
		# Clean paru cache (recursive)
		find ~/.cache/paru/clone -maxdepth 2 -type d -exec paccache -r %s -k %d -c {} +
	`, 
	func() string { if uninstalled { return "-u" }; return "" }(),
	keep,
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

// executeRemoveOrphansInTerminal runs paru -Rns $(paru -Qdtq) interactively using tea.ExecProcess
func executeRemoveOrphansInTerminal(orphans []string) tea.Cmd {

	validNames, _ := sanitizePackageNames(orphans)
	if len(validNames) == 0 {
		return func() tea.Msg {
			return execCompleteMsg{operation: confirmRemoveOrphans, packages: orphans, err: fmt.Errorf("no valid package names")}
		}
	}

	args := append([]string{"-Rns"}, validNames...)
	return runner.Interactive(func(err error) tea.Msg {
		return execCompleteMsg{operation: confirmRemoveOrphans, packages: validNames, err: err}
	}, "paru", args...)
}

// executeSelectiveUpdateInTerminal runs paru -S interactively using tea.ExecProcess
func executeSelectiveUpdateInTerminal(packages []string) tea.Cmd {
	validNames, _ := sanitizePackageNames(packages)
	if len(validNames) == 0 {
		return func() tea.Msg {
			return execCompleteMsg{operation: confirmSelectiveUpdate, packages: packages, err: fmt.Errorf("no valid package names")}
		}
	}

	args := append([]string{"-S"}, validNames...)
	return runner.Interactive(func(err error) tea.Msg {
		return execCompleteMsg{operation: confirmSelectiveUpdate, packages: validNames, err: err}
	}, "paru", args...)
}

// syncRepositoriesInTerminal runs paru -Sy interactively to sync databases
func syncRepositoriesInTerminal() tea.Cmd {
	return runner.Interactive(func(err error) tea.Msg {
		return syncRepositoriesMsg{err: err}
	}, "paru", "-Sy")
}
