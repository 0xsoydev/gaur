package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m *model) renderCacheMenu(helpText string, innerWidth, innerHeight int) string {
	menuWidth := 80
	menuHeight := 22

	if menuWidth > innerWidth-4 {
		menuWidth = innerWidth - 4
	}
	if menuHeight > innerHeight-4 {
		menuHeight = innerHeight - 4
	}

	selectedColor := lipgloss.Color("42") // default green
	switch m.cacheMenuIndex {
	case 0:
		selectedColor = lipgloss.Color("39") // Blue
	case 1:
		selectedColor = lipgloss.Color("208") // Orange
	case 2:
		selectedColor = lipgloss.Color("42") // Green
	case 3:
		selectedColor = lipgloss.Color("196") // Red
	case 4:
		selectedColor = lipgloss.Color("135") // Purple
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
		Foreground(selectedColor).
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
	menuContent.WriteString(titleStyle.Width(menuWidth - 4).Render(" \U000f00e2 Cache Management Menu"))
	menuContent.WriteString("\n\n")

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
			menuContent.WriteString(descStyle.PaddingLeft(descPadding).Foreground(selectedColor).Render(opt.desc) + "\n\n")
		} else {
			menuContent.WriteString(descStyle.PaddingLeft(descPadding).Render(opt.desc) + "\n\n")
		}
	}

	menuBox := lipgloss.NewStyle().
		Width(menuWidth).
		Height(menuHeight).
		Border(m.getBorderStyle()).
		BorderForeground(selectedColor).
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
	pkgList := m.filtered

	if len(pkgList) == 0 {
		if m.textInput.Value() != "" {
			results.WriteString("  No matches for '" + m.textInput.Value() + "'")
		} else {
			results.WriteString("  No packages found in cache")
		}
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

			// Define consistent name width
			// Left gutter: prefix (5) + space (1) = 6 chars
			// Right gutter: we want 6 chars as well.
			// Total layout: prefix (5) + space (1) + name (nameWidth) + space (1) + size (10) + right gutter (6) = contentWidth
			// nameWidth = contentWidth - 23
			nameWidth := contentWidth - 23
			if nameWidth < 10 {
				nameWidth = 10
			}

			nameStr := pkg.Name
			if indices, ok := m.matchIndices[i]; ok && m.textInput.Value() != "" {
				nameStr = highlightMatches(pkg.Name, indices)
			}
			nameStr = truncateWithAnsi(nameStr, nameWidth)
			
			// Ensure name is padded to exact width for alignment
			visualNameWidth := lipgloss.Width(nameStr)
			paddedName := nameStr
			if visualNameWidth < nameWidth {
				paddedName += strings.Repeat(" ", nameWidth-visualNameWidth)
			}
			
			sizeStr := fmt.Sprintf("%10s", pkg.Size)

			var line string
			if i == m.selectedIndex || m.markedPackages[pkg.Name] {
				bgColor := lipgloss.Color("237") // Default marked grey
				if i == m.selectedIndex {
					bgColor = lipgloss.Color("135") // Selection purple
				}
				fgColor := lipgloss.Color("255")
				
				// Apply background maintenance to name which contains resets from highlighting/truncation
				maintainedName := maintainBackground(paddedName, bgColor)
				
				// Construct line content with consistent spacing: prefix + " " + name + " " + size
				lineContent := fmt.Sprintf("%s %s %s", prefix, maintainedName, sizeStr)
				
				line = lipgloss.NewStyle().
					Background(bgColor).
					Foreground(fgColor).
					Bold(i == m.selectedIndex).
					Width(contentWidth).
					Render(lineContent)
			} else {
				// Normal items: Styled parts but same layout for alignment
				namePart := lipgloss.NewStyle().Foreground(lipgloss.Color("250")).Render(paddedName)
				sizePart := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(sizeStr)
				line = fmt.Sprintf("%s %s %s", prefix, namePart, sizePart)
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
	
	inputLine := m.textInput.View()

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
