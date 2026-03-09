package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func getInstalledPackages() tea.Cmd {
	return func() tea.Msg {

		cmd := exec.Command("pacman", "-Qi")
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out

		err := cmd.Run()
		if err != nil {
			return installedPackagesMsg{err: err}
		}

		packages, err := parseInstalledPackages(out.String())
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
	cmd := exec.Command("pacman", "-Sl")
	var repoOut bytes.Buffer
	cmd.Stdout = &repoOut
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to query package repositories: %w", err)
	}
	for _, line := range strings.Split(repoOut.String(), "\n") {
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

	cmd = exec.Command("pacman", "-Qm")
	var foreignOut bytes.Buffer
	cmd.Stdout = &foreignOut
	if err := cmd.Run(); err == nil {
		foreignPkgs := make(map[string]bool)
		for _, line := range strings.Split(foreignOut.String(), "\n") {
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

	cmd = exec.Command("pacman", "-Qe")
	var explicitOut bytes.Buffer
	cmd.Stdout = &explicitOut
	if err := cmd.Run(); err == nil {
		explicitPkgs := make(map[string]bool)
		for _, line := range strings.Split(explicitOut.String(), "\n") {
			parts := strings.Fields(line)
			if len(parts) >= 1 {
				explicitPkgs[parts[0]] = true
			}
		}
		for i := range packages {
			packages[i].Explicit = explicitPkgs[packages[i].Name]
		}
	}

	cmd = exec.Command("pacman", "-Qdt")
	var orphanOut bytes.Buffer
	cmd.Stdout = &orphanOut
	if err := cmd.Run(); err == nil {
		orphanPkgs := make(map[string]bool)
		for _, line := range strings.Split(orphanOut.String(), "\n") {
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

		cmd := exec.Command("paru", "-Rns", "--noconfirm", pkg.Name)
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out

		err := cmd.Run()
		if err != nil {
			return actionCompleteMsg{
				message: fmt.Sprintf("Failed to uninstall %s: %s", pkg.Name, out.String()),
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
		cmd := exec.Command("paru", args...)
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out

		err := cmd.Run()
		if err != nil {
			return actionCompleteMsg{
				message: fmt.Sprintf("Failed to uninstall packages: %s", out.String()),
				err:     err,
			}
		}

		return actionCompleteMsg{
			message: fmt.Sprintf("Successfully uninstalled %d packages", len(validNames)),
		}
	}
}
