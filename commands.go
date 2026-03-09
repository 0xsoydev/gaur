package main

import (
	"bytes"
	"fmt"
	"os/exec"
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

		cmd := exec.Command("paru", "-Si", pkg.Name)
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out

		err := cmd.Run()
		if err != nil {
			return packageInfoMsg{info: "Failed to get package info", packageName: pkg.Name, err: err}
		}

		return packageInfoMsg{info: out.String(), packageName: pkg.Name}
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
	c := exec.Command("paru", args...)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return execCompleteMsg{operation: confirmInstall, packages: validNames, err: err}
	})
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
	c := exec.Command("paru", args...)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return execCompleteMsg{operation: confirmUninstall, packages: validNames, err: err}
	})
}

// executeUpdateInTerminal runs paru -Syu interactively using tea.ExecProcess
func executeUpdateInTerminal() tea.Cmd {
	c := exec.Command("paru", "-Syu")
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return execCompleteMsg{operation: confirmUpdate, err: err}
	})
}

// executeCleanCacheInTerminal runs paru -Sc interactively using tea.ExecProcess
func executeCleanCacheInTerminal() tea.Cmd {
	c := exec.Command("paru", "-Sc")
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return execCompleteMsg{operation: confirmCleanCache, err: err}
	})
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
	c := exec.Command("paru", args...)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return execCompleteMsg{operation: confirmRemoveOrphans, packages: validNames, err: err}
	})
}

// syncRepositoriesInTerminal runs paru -Sy interactively to sync databases
func syncRepositoriesInTerminal() tea.Cmd {
	c := exec.Command("paru", "-Sy")
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return syncRepositoriesMsg{err: err}
	})
}
