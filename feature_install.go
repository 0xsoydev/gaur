package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Commands
// loadRepoPackages loads all packages from local pacman database
func loadRepoPackages() tea.Cmd {
	return func() tea.Msg {

		cmd := exec.Command("pacman", "-Sl")
		var stdout bytes.Buffer
		cmd.Stdout = &stdout
		if err := cmd.Run(); err != nil {
			return repoPackagesMsg{err: err}
		}

		installedCmd := exec.Command("pacman", "-Qq")
		var installedOut bytes.Buffer
		installedCmd.Stdout = &installedOut
		if err := installedCmd.Run(); err != nil {
			return repoPackagesMsg{err: fmt.Errorf("failed to get installed packages list: %w", err)}
		}

		installedSet := make(map[string]bool)
		for _, name := range strings.Split(installedOut.String(), "\n") {
			name = strings.TrimSpace(name)
			if name != "" {
				installedSet[name] = true
			}
		}

		// Parse "repo name version [installed]" format
		var packages []Package
		for _, line := range strings.Split(stdout.String(), "\n") {
			parts := strings.Fields(line)
			if len(parts) < 3 {
				continue
			}
			pkg := Package{
				Source:    parts[0],
				Name:      parts[1],
				Version:   parts[2],
				Installed: installedSet[parts[1]] || (len(parts) > 3 && parts[3] == "[installed]"),
			}
			packages = append(packages, pkg)
		}

		return repoPackagesMsg{packages: packages}
	}
}

// filterAllPackages combines repo and AUR packages, then fuzzy filters together
// This ensures fzf ranks all packages by relevance to the query
// Supports repo filtering with prefixes: c (core), e (extra), m (multilib), a (aur)
// Filters can be combined: ae:, cem:, aem: etc.
func (m *model) filterAllPackages(query string) {
	if query == "" {
		m.filtered = []Package{}
		m.matchIndices = nil
		return
	}

	repoFilters, searchQuery := parseRepoFilter(query)

	allPackages := make([]Package, 0, len(m.repoPackages)+len(m.aurPackages))
	allPackages = append(allPackages, m.repoPackages...)
	allPackages = append(allPackages, m.aurPackages...)

	if len(repoFilters) > 0 {
		var filtered []Package
		for _, pkg := range allPackages {
			if repoFilters[pkg.Source] {
				filtered = append(filtered, pkg)
			}
		}
		allPackages = filtered
	}

	if len(allPackages) == 0 {
		m.filtered = []Package{}
		m.matchIndices = nil
		return
	}

	if searchQuery == "" {
		m.filtered = allPackages
		m.matchIndices = nil
		return
	}

	m.filtered = fuzzyFilter(allPackages, searchQuery)

	m.matchIndices = computeAllMatchIndices(m.filtered, searchQuery)
}

// searchAUR searches the AUR via paru (network call)
func searchAUR(query string) tea.Cmd {
	return func() tea.Msg {
		if query == "" {
			return aurSearchMsg{packages: []Package{}, query: query}
		}

		// Sanitize search query - only allow safe characters for search
		// This prevents command injection through the search query
		var sanitized strings.Builder
		for _, r := range query {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
				(r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
				sanitized.WriteRune(r)
			} else if r == ' ' {
				sanitized.WriteRune('-')
			}

		}
		searchQuery := sanitized.String()
		if searchQuery == "" {
			return aurSearchMsg{packages: []Package{}, query: query}
		}

		cmd := exec.Command("paru", "-Ss", "-a", searchQuery)
		var stdout bytes.Buffer
		cmd.Stdout = &stdout
		if err := cmd.Run(); err != nil {
			return aurSearchMsg{packages: nil, query: query, err: fmt.Errorf("AUR search command failed: %w", err)}
		}

		if stdout.Len() == 0 {
			return aurSearchMsg{packages: []Package{}, query: query}
		}

		packages := parseAUROutput(stdout.String())
		return aurSearchMsg{packages: packages, query: query}
	}
}

func installPackage(pkg Package) tea.Cmd {
	return func() tea.Msg {

		if !isValidPackageName(pkg.Name) {
			return actionCompleteMsg{
				message: fmt.Sprintf("Invalid package name: %s", pkg.Name),
				err:     fmt.Errorf("invalid package name"),
			}
		}

		cmd := exec.Command("paru", "-S", "--noconfirm", pkg.Name)
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out

		err := cmd.Run()
		if err != nil {
			return actionCompleteMsg{
				message: fmt.Sprintf("Failed to install %s: %s", pkg.Name, out.String()),
				err:     err,
			}
		}

		return actionCompleteMsg{
			message: fmt.Sprintf("Successfully installed %s", pkg.Name),
		}
	}
}

func installMultiplePackages(pkgNames []string) tea.Cmd {
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
				message: "No valid package names to install",
				err:     fmt.Errorf("no valid packages"),
			}
		}

		args := append([]string{"-S", "--noconfirm"}, validNames...)
		cmd := exec.Command("paru", args...)
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out

		err := cmd.Run()
		if err != nil {
			return actionCompleteMsg{
				message: fmt.Sprintf("Failed to install packages: %s", out.String()),
				err:     err,
			}
		}

		return actionCompleteMsg{
			message: fmt.Sprintf("Successfully installed %d packages", len(validNames)),
		}
	}
}
