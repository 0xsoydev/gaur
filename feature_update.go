package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func updateSystem() tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("paru", "-Syu", "--noconfirm")
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out

		err := cmd.Run()
		output := out.String()

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

		cmd := exec.Command("pacman", "-Qm")
		var foreignOut bytes.Buffer
		cmd.Stdout = &foreignOut
		foreignPkgs := make(map[string]bool)
		if err := cmd.Run(); err == nil {
			for _, line := range strings.Split(foreignOut.String(), "\n") {
				line = strings.TrimSpace(line)
				if line != "" {

					parts := strings.Fields(line)
					if len(parts) >= 1 {
						foreignPkgs[parts[0]] = true
					}
				}
			}
		}

		cmd = exec.Command("pacman", "-Sl")
		var repoOut bytes.Buffer
		cmd.Stdout = &repoOut
		repoMap := make(map[string]string)
		if err := cmd.Run(); err == nil {
			for _, line := range strings.Split(repoOut.String(), "\n") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {

					repoName := parts[0]
					pkgName := parts[1]
					repoMap[pkgName] = repoName
				}
			}
		}

		cmd = exec.Command("paru", "-Qu")
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {

			if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {

				return updateCheckMsg{packages: []Package{}}
			}

			return updateCheckMsg{packages: nil, err: fmt.Errorf("failed to check for updates: %w", err)}
		}

		// Step 4: Parse updates and assign accurate source
		var packages []Package
		for _, line := range strings.Split(strings.TrimSpace(stdout.String()), "\n") {
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
