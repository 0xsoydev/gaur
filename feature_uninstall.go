package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func getInstalledPackages() tea.Cmd {
	return func() tea.Msg {

		out, err := runner.Run("pacman", "-Qi")
		if err != nil {
			return installedPackagesMsg{err: err}
		}

		packages, err := parseInstalledPackages(string(out))
		if err != nil {
			return installedPackagesMsg{err: fmt.Errorf("failed to parse installed packages: %w", err)}
		}
		return installedPackagesMsg{packages: packages}
	}
}

func parseInstalledPackages(output string) ([]Package, error) {
	var packages []Package
	blocks := strings.Split(output, "\n\n")

	for _, block := range blocks {
		if strings.TrimSpace(block) == "" {
			continue
		}

		var pkg Package
		pkg.Installed = true
		pkg.Source = "local"

		lines := strings.Split(block, "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "Name") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					pkg.Name = strings.TrimSpace(parts[1])
				}
			} else if strings.HasPrefix(line, "Version") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					pkg.Version = strings.TrimSpace(parts[1])
				}
			} else if strings.HasPrefix(line, "Description") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					pkg.Description = strings.TrimSpace(parts[1])
				}
			}
		}

		if pkg.Name != "" {
			packages = append(packages, pkg)
		}
	}

	repoMap := make(map[string]string)
	repoOut, err := runner.Run("pacman", "-Sl")
	if err != nil {
		return nil, fmt.Errorf("failed to query package repositories: %w", err)
	}
	for _, line := range strings.Split(string(repoOut), "\n") {
		parts := strings.Fields(line)
		if len(parts) >= 2 {

			repoMap[parts[1]] = parts[0]
		}
	}

	for i := range packages {
		if repo, ok := repoMap[packages[i].Name]; ok {
			packages[i].Source = repo
		}
	}

	foreignOut, err := runner.Run("pacman", "-Qm")
	if err == nil {
		foreignPkgs := make(map[string]bool)
		for _, line := range strings.Split(string(foreignOut), "\n") {
			parts := strings.Fields(line)
			if len(parts) >= 1 {
				foreignPkgs[parts[0]] = true
			}
		}
		for i := range packages {
			if foreignPkgs[packages[i].Name] {
				packages[i].Source = "aur"
			}
		}
	}

	explicitOut, err := runner.Run("pacman", "-Qe")
	if err == nil {
		explicitPkgs := make(map[string]bool)
		for _, line := range strings.Split(string(explicitOut), "\n") {
			parts := strings.Fields(line)
			if len(parts) >= 1 {
				explicitPkgs[parts[0]] = true
			}
		}
		for i := range packages {
			packages[i].Explicit = explicitPkgs[packages[i].Name]
		}
	}

	orphanOut, err := runner.Run("pacman", "-Qdt")
	if err == nil {
		orphanPkgs := make(map[string]bool)
		for _, line := range strings.Split(string(orphanOut), "\n") {
			parts := strings.Fields(line)
			if len(parts) >= 1 {
				orphanPkgs[parts[0]] = true
			}
		}
		for i := range packages {
			packages[i].Orphan = orphanPkgs[packages[i].Name]
		}
	}

	return packages, nil
}

func uninstallPackage(pkg Package) tea.Cmd {
	return func() tea.Msg {

		if !isValidPackageName(pkg.Name) {
			return actionCompleteMsg{
				message: fmt.Sprintf("Invalid package name: %s", pkg.Name),
				err:     fmt.Errorf("invalid package name"),
			}
		}

		out, err := runner.Run("paru", "-Rns", "--noconfirm", pkg.Name)
		if err != nil {
			return actionCompleteMsg{
				message: fmt.Sprintf("Failed to uninstall %s: %s", pkg.Name, string(out)),
				err:     err,
			}
		}

		return actionCompleteMsg{
			message: fmt.Sprintf("Successfully uninstalled %s", pkg.Name),
		}
	}
}

func uninstallMultiplePackages(pkgNames []string) tea.Cmd {
	return func() tea.Msg {

		validNames, allValid := sanitizePackageNames(pkgNames)
		if !allValid {
			return actionCompleteMsg{
				message: "Some package names contain invalid characters and were skipped",
				err:     fmt.Errorf("invalid package names detected"),
			}
		}
		if len(validNames) == 0 {
			return actionCompleteMsg{
				message: "No valid package names to uninstall",
				err:     fmt.Errorf("no valid packages"),
			}
		}

		args := append([]string{"-Rns", "--noconfirm"}, validNames...)
		out, err := runner.Run("paru", args...)
		if err != nil {
			return actionCompleteMsg{
				message: fmt.Sprintf("Failed to uninstall packages: %s", string(out)),
				err:     err,
			}
		}

		return actionCompleteMsg{
			message: fmt.Sprintf("Successfully uninstalled %d packages", len(validNames)),
		}
	}
}
