package main

import (
	"fmt"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func updateSystem() tea.Cmd {
	return func() tea.Msg {
		out, err := runner.Run("paru", "-Syu", "--noconfirm")
		output := string(out)

		if err != nil {
			return updateOutputMsg{
				output: output,
				done:   true,
				err:    err,
			}
		}

		return updateOutputMsg{
			output: output,
			done:   true,
		}
	}
}

// checkUpdates fetches available updates using paru -Qu
func checkUpdates() tea.Cmd {
	return func() tea.Msg {

		foreignOut, err := runner.Run("pacman", "-Qm")
		foreignPkgs := make(map[string]bool)
		if err == nil {
			for _, line := range strings.Split(string(foreignOut), "\n") {
				line = strings.TrimSpace(line)
				if line != "" {

					parts := strings.Fields(line)
					if len(parts) >= 1 {
						foreignPkgs[parts[0]] = true
					}
				}
			}
		}

		repoOut, err := runner.Run("pacman", "-Sl")
		repoMap := make(map[string]string)
		if err == nil {
			for _, line := range strings.Split(string(repoOut), "\n") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {

					repoName := parts[0]
					pkgName := parts[1]
					repoMap[pkgName] = repoName
				}
			}
		}

		stdout, err := runner.Run("paru", "-Qu")
		if err != nil {

			if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {

				return updateCheckMsg{packages: []Package{}}
			}

			return updateCheckMsg{packages: nil, err: fmt.Errorf("failed to check for updates: %w", err)}
		}

		// Step 4: Parse updates and assign accurate source
		var packages []Package
		for _, line := range strings.Split(strings.TrimSpace(string(stdout)), "\n") {
			if line == "" {
				continue
			}
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				pkgName := parts[0]

				if !isValidPackageName(pkgName) {
					continue
				}
				pkg := Package{
					Name:      pkgName,
					Version:   strings.Join(parts[1:], " "),
					Installed: true,
				}

				if foreignPkgs[pkgName] {
					pkg.Source = "aur"
				} else if repoName, ok := repoMap[pkgName]; ok {
					pkg.Source = repoName
				} else {
					pkg.Source = "unknown"
				}
				packages = append(packages, pkg)
			}
		}
		return updateCheckMsg{packages: packages}
	}
}
