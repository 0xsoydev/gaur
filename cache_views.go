package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m *model) renderCacheMenu(helpText string, innerWidth, innerHeight int) string {
	menuWidth := 60
	menuHeight := 24

	if menuWidth > innerWidth-4 {
		menuWidth = innerWidth - 4
	}
	if menuHeight > innerHeight-4 {
		menuHeight = innerHeight - 4
	}

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("229")).
		MarginBottom(1).
		Align(lipgloss.Center)

	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("246")).
		Italic(true)

	normalItemStyle := lipgloss.NewStyle().
		PaddingLeft(2).
		Foreground(lipgloss.Color("250"))

	selectedItemStyle := lipgloss.NewStyle().
		PaddingLeft(2).
		Foreground(lipgloss.Color("42")).
		Bold(true)

	options := []struct {
		title string
		desc  string
	}{
		{"Safe Clean (Keep 3)", "Leaves the installed version and 2 fallbacks (paccache -r)"},
		{"Aggressive (Keep 1)", "Keeps only the currently installed version (paccache -rk1)"},
		{"Orphaned Cache", "Removes cached tarballs for uninstalled packages (paccache -ruk0)"},
		{"Nuke Everything", "Empties the entire cache directory (paccache -rk0)"},
		{"Selective Clean", "Manually select specific packages to delete"},
	}

	var menuContent strings.Builder
	menuContent.WriteString(titleStyle.Width(menuWidth - 4).Render("🧹 Cache Management Menu"))
	menuContent.WriteString("\n")

	for i, opt := range options {
		var itemStr string
		if i == m.cacheMenuIndex {
			itemStr = selectedItemStyle.Render("> " + opt.title)
		} else {
			itemStr = normalItemStyle.Render("  " + opt.title)
		}
		menuContent.WriteString(itemStr + "\n")
		
		descPadding := 4
		if i == m.cacheMenuIndex {
			menuContent.WriteString(descStyle.PaddingLeft(descPadding).Foreground(lipgloss.Color("42")).Render(opt.desc) + "\n\n")
		} else {
			menuContent.WriteString(descStyle.PaddingLeft(descPadding).Render(opt.desc) + "\n\n")
		}
	}

	menuBox := lipgloss.NewStyle().
		Width(menuWidth).
		Height(menuHeight).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("42")).
		Padding(1, 2).
		Render(menuContent.String())

	// Overlay menu onto dashboard
	return lipgloss.Place(innerWidth, innerHeight, lipgloss.Center, lipgloss.Center, menuBox, lipgloss.WithWhitespaceChars(" "), lipgloss.WithWhitespaceForeground(lipgloss.Color("235")))
}

func (m *model) renderSelectiveCacheView(helpText string, innerWidth, innerHeight int, activeColor lipgloss.Color) string {
	borderStyle := baseBorderStyle.BorderForeground(activeColor)

	headerHeight := 1
	footerHeight := 1

	availableHeight := innerHeight - headerHeight - footerHeight
	if availableHeight < 6 {
		availableHeight = 6
	}

	targetBottomPanelHeight := availableHeight
	bottomInnerHeight := targetBottomPanelHeight - 2
	resultsHeight := bottomInnerHeight - 3

	if resultsHeight < 1 {
		resultsHeight = 1
	}

	contentWidth := innerWidth - 2
	if contentWidth < 10 {
		contentWidth = 10
	}

	// Build results list
	var results strings.Builder
	pkgList := m.dashboard.AllCacheHogs

	if len(pkgList) == 0 {
		results.WriteString("  No packages found in cache")
	} else {
		startIdx := 0
		if m.selectedIndex >= resultsHeight {
			startIdx = m.selectedIndex - resultsHeight + 1
		}
		endIdx := startIdx + resultsHeight
		if endIdx > len(pkgList) {
			endIdx = len(pkgList)
		}

		var lines []string
		for i := startIdx; i < endIdx; i++ {
			pkg := pkgList[i]

			marker := "[ ]"
			if m.markedPackages[pkg.Name] {
				marker = "[x]"
			}
			prefix := "  " + marker
			if i == m.selectedIndex {
				prefix = "> " + marker
			}

			nameStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
			if m.markedPackages[pkg.Name] {
				nameStyle = nameStyle.Foreground(lipgloss.Color("42"))
			}

			line := fmt.Sprintf("%s %s",
				prefix,
				lipgloss.NewStyle().Width(contentWidth-25).Render(nameStyle.Render(truncateWithAnsi(pkg.Name, contentWidth-25))),
			)

			// Add size
			sizeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
			line += " " + sizeStyle.Render(fmt.Sprintf("%10s", pkg.Size))

			if i == m.selectedIndex {
				line = selectedStyle.Render(line)
			}

			lines = append(lines, line)
		}

		for i := len(lines) - 1; i >= 0; i-- {
			results.WriteString(lines[i])
			if i > 0 {
				results.WriteString("\n")
			}
		}
	}

	resultsBox := lipgloss.NewStyle().
		Width(contentWidth).
		Height(resultsHeight).
		Render(results.String())

	// Footer status text
	var statusMsg string
	if len(m.markedPackages) > 0 {
		statusMsg = fmt.Sprintf("Space to be freed: %s / %s total cache", formatBytes(m.cacheToFree), m.dashboard.CleanerSize)
	} else {
		statusMsg = fmt.Sprintf("Select packages to remove... (Total Cache: %s)", m.dashboard.CleanerSize)
	}
	
	statusLine := statusStyle.Render(statusMsg)
	
	// Hide input text in selective cache mode
	inputLine := ""

	bottomContent := lipgloss.JoinVertical(
		lipgloss.Left,
		resultsBox,
		"",
		statusLine,
		inputLine,
	)

	bottomPanel := borderStyle.
		Width(innerWidth - 2).
		Height(targetBottomPanelHeight - 2).
		Render(bottomContent)

	header := lipgloss.NewStyle().Bold(true).Foreground(activeColor).Render(" \uf0c7 Selective Cache Clean")

	helpWidth := lipgloss.Width(helpText)
	padding := innerWidth - helpWidth
	if padding < 0 {
		padding = 0
	}
	footer := strings.Repeat(" ", padding) + helpText
	if lipgloss.Width(footer) > innerWidth {
		footer = truncateWithAnsi(footer, innerWidth)
	}

	var layoutSections []string
	layoutSections = append(layoutSections, header, bottomPanel, footer)

	return lipgloss.JoinVertical(lipgloss.Left, layoutSections...)
}