package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderHelpText creates the help menu with the active mode highlighted
func (m model) renderHelpText(activeColor lipgloss.Color) string {
	dimStyle := helpStyle
	activeStyle := lipgloss.NewStyle().
		Foreground(activeColor).
		Bold(true)

	var parts []string

	parts = append(parts, dimStyle.Render("[/] search  [tab] mark  "))

	if m.mode == modeInstall {
		parts = append(parts, activeStyle.Render("[i]nstall"))
	} else {
		parts = append(parts, dimStyle.Render("[i]nstall"))
	}
	parts = append(parts, dimStyle.Render("  "))

	if m.mode == modeInstalled {
		parts = append(parts, activeStyle.Render("i[n]fo"))
	} else {
		parts = append(parts, dimStyle.Render("i[n]fo"))
	}
	parts = append(parts, dimStyle.Render("  "))

	if m.mode == modeUninstall {
		parts = append(parts, activeStyle.Render("[r]emove"))
	} else {
		parts = append(parts, dimStyle.Render("[r]emove"))
	}
	parts = append(parts, dimStyle.Render("  "))

	if m.mode == modeUpdate {
		parts = append(parts, activeStyle.Render("[u]pdate"))
	} else {
		parts = append(parts, dimStyle.Render("[u]pdate"))
	}
	parts = append(parts, dimStyle.Render("  "))

	parts = append(parts, dimStyle.Render("[q]uit"))

	return strings.Join(parts, "")
}

func (m model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	contentWidth := m.width - 4
	contentHeight := m.height - 4

	activeColor := modeColors[m.mode]
	if activeColor == "" {
		activeColor = defaultBorderColor
	}

	titleStyle := baseTitleStyle.Background(activeColor)
	borderStyle := baseBorderStyle.BorderForeground(activeColor)

	modeText := ""
	switch m.mode {
	case modeInstall:
		modeText = "INSTALL"
	case modeInstalled:
		modeText = "INFO"
	case modeUninstall:
		modeText = "UNINSTALL"
	case modeUpdate:
		modeText = "UPDATE"
	}

	header := titleStyle.Render(" GAUR - " + modeText + " ")

	helpText := m.renderHelpText(activeColor)

	if m.showConfirmation {
		return m.renderConfirmationDialog(contentWidth, contentHeight, activeColor)
	}

	if m.showErrorOverlay {
		return m.renderErrorOverlay(contentWidth, contentHeight)
	}

	if m.mode == modeInstalled {
		return m.renderDashboard(helpText, contentWidth, contentHeight)
	}

	infoHeight := contentHeight / 2
	infoContent := ""
	if m.mode == modeUpdate {
		if m.loadingInfo {
			infoContent = fmt.Sprintf("Loading details for %s...", m.infoForPackage)
		} else if m.packageInfo != "" {
			infoContent = m.packageInfo
		} else {
			infoContent = "Select an update to see details"
		}
	} else if m.loadingInfo {
		infoContent = fmt.Sprintf("Loading details for %s...", m.infoForPackage)
	} else if m.packageInfo != "" {
		infoContent = m.packageInfo
	} else {
		infoContent = "Select a package to see details"
	}

	infoLines := strings.Split(infoContent, "\n")
	if len(infoLines) > infoHeight-2 {
		infoLines = infoLines[:infoHeight-2]
	}
	infoContent = strings.Join(infoLines, "\n")

	infoBox := lipgloss.NewStyle().
		Width(contentWidth-2).
		Height(infoHeight-2).
		Padding(0, 1).
		Render(infoContent)

	infoPanel := borderStyle.
		Width(contentWidth).
		Height(infoHeight).
		Render(infoBox)

	bottomHeight := contentHeight - infoHeight - 1
	resultsHeight := bottomHeight - 3
	if resultsHeight < 1 {
		resultsHeight = 1
	}

	// Build results list
	var results strings.Builder
	var pkgList []Package
	if m.mode == modeInstall {
		pkgList = m.filtered
	} else if m.mode == modeUninstall {
		pkgList = m.filteredInstalled
	} else if m.mode == modeUpdate {
		pkgList = m.filtered
	}

	if m.loading {
		results.WriteString("  Loading...")
	} else if m.mode == modeUpdate && len(pkgList) == 0 && !m.loading {
		results.WriteString("  " + m.statusMessage)
	} else if len(pkgList) == 0 {
		results.WriteString("  No packages to display")
	} else {

		startIdx := 0
		if m.selectedIndex >= resultsHeight {
			startIdx = m.selectedIndex - resultsHeight + 1
		}
		endIdx := startIdx + resultsHeight
		if endIdx > len(pkgList) {
			endIdx = len(pkgList)
		}

		// Get the appropriate match indices map
		var matchIndicesMap map[int][]int
		if m.mode == modeInstall {
			matchIndicesMap = m.matchIndices
		} else if m.mode == modeUninstall {
			matchIndicesMap = m.installedMatchIndices
		} else if m.mode == modeUpdate {
			matchIndicesMap = m.matchIndices
		}

		// Build lines in reverse order (most relevant at bottom, near input field)
		var lines []string
		for i := startIdx; i < endIdx; i++ {
			pkg := pkgList[i]

			marker := " "
			if m.markedPackages[pkg.Name] {
				marker = "*"
			}
			prefix := " " + marker
			if i == m.selectedIndex {
				prefix = ">" + marker
			}

			sourceStyle := lipgloss.NewStyle()
			if color, ok := sourceColors[pkg.Source]; ok {
				sourceStyle = sourceStyle.Foreground(color)
			}

			// Apply highlighting with source colors
			var displayPkgStr string
			if matchIndicesMap != nil {
				if indices, ok := matchIndicesMap[i]; ok {

					displayPkgStr = highlightMatchesWithSourceColor(pkg, indices)
				} else {
					displayPkgStr = sourceStyle.Render(pkg.Source) + "/" + pkg.Name
				}
			} else {
				displayPkgStr = sourceStyle.Render(pkg.Source) + "/" + pkg.Name
			}

			line := fmt.Sprintf("%s%s %s",
				prefix,
				displayPkgStr,
				lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(pkg.Version),
			)

			if pkg.Installed && m.mode == modeInstall {
				line += " " + installedBadge.Render("[installed]")
			}

			if lipgloss.Width(line) > contentWidth-4 {
				line = line[:contentWidth-7] + "..."
			}

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
		Width(contentWidth - 2).
		Height(resultsHeight).
		Render(results.String())

	inputLine := ""
	if m.mode == modeInstall || m.mode == modeUninstall || m.mode == modeUpdate {
		inputLine = m.textInput.View()
	} else {
		inputLine = statusStyle.Render(m.statusMessage)
	}

	statusLine := statusStyle.Render(m.statusMessage)

	bottomContent := lipgloss.JoinVertical(
		lipgloss.Left,
		resultsBox,
		"",
		statusLine,
		inputLine,
	)

	bottomPanel := borderStyle.
		Width(contentWidth).
		Height(bottomHeight).
		Render(bottomContent)

	helpWidth := lipgloss.Width(helpText)
	padding := contentWidth - helpWidth
	if padding < 0 {
		padding = 0
	}
	footer := strings.Repeat(" ", padding) + helpText

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		infoPanel,
		bottomPanel,
		footer,
	)

	if len(m.markedPackages) > 0 {
		content = m.overlaySelectionsPanel(content, contentWidth)
	}

	return content
}

// overlaySelectionsPanel renders a selection panel on the bottom right of the screen
func (m model) overlaySelectionsPanel(content string, contentWidth int) string {

	borderColor := lipgloss.Color("205")
	if m.selectionPanelFocused {
		borderColor = lipgloss.Color("213")
	}
	panelStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 1)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205"))

	itemStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	selectedItemStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("213")).
		Bold(true)

	keyHintStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Bold(true)

	// Build the selections list with * hint in title
	var selectionsList strings.Builder

	titleText := titleStyle.Render(fmt.Sprintf("Selected (%d) ", len(m.markedPackages))) + keyHintStyle.Render("[*]")
	selectionsList.WriteString(titleText)

	// Collect and sort package names for consistent display
	var pkgNames []string
	for name := range m.markedPackages {
		pkgNames = append(pkgNames, name)
	}
	sort.Strings(pkgNames)

	maxDisplay := 20
	minPanelWidth := 12
	maxPanelWidth := 32

	maxContentWidth := lipgloss.Width(titleText)
	visibleCount := maxDisplay
	if len(pkgNames) < visibleCount {
		visibleCount = len(pkgNames)
	}
	for i := 0; i < visibleCount; i++ {

		nameWidth := lipgloss.Width(pkgNames[i]) + 2
		if nameWidth > maxContentWidth {
			maxContentWidth = nameWidth
		}
	}
	if len(pkgNames) > maxDisplay {
		moreStr := itemStyle.Render(fmt.Sprintf("... +%d more", len(pkgNames)-maxDisplay))
		if w := lipgloss.Width(moreStr); w > maxContentWidth {
			maxContentWidth = w
		}
	}

	desiredPanelWidth := maxContentWidth + 4
	if desiredPanelWidth < minPanelWidth {
		desiredPanelWidth = minPanelWidth
	}
	if desiredPanelWidth > maxPanelWidth {
		desiredPanelWidth = maxPanelWidth
	}
	panelWidth := desiredPanelWidth

	for i, name := range pkgNames {
		if i >= maxDisplay {
			selectionsList.WriteString("\n")
			selectionsList.WriteString(itemStyle.Render(fmt.Sprintf("... +%d more", len(pkgNames)-maxDisplay)))
			break
		}

		innerWidth := panelWidth - 4
		nameMaxWidth := innerWidth - 2
		if nameMaxWidth < 1 {
			nameMaxWidth = 1
		}

		displayName := name
		if lipgloss.Width(displayName) > nameMaxWidth {

			runes := []rune(displayName)
			truncWidth := nameMaxWidth - 3
			if truncWidth < 1 {
				truncWidth = 1
			}
			var truncated string
			for j := 1; j <= len(runes); j++ {
				s := string(runes[:j])
				if lipgloss.Width(s) > truncWidth {
					truncated = string(runes[:j-1]) + "..."
					break
				}
				if j == len(runes) {
					truncated = s
				}
			}
			displayName = truncated
		}

		selectionsList.WriteString("\n")

		if m.selectionPanelFocused && i == m.selectionPanelIndex {
			selectionsList.WriteString(selectedItemStyle.Render("> " + displayName))
		} else {
			selectionsList.WriteString(itemStyle.Render("  " + displayName))
		}
	}

	panel := panelStyle.Width(panelWidth).Render(selectionsList.String())
	panelHeight := strings.Count(panel, "\n") + 1

	lines := strings.Split(content, "\n")

	panelActualWidth := lipgloss.Width(panel)

	startRow := 1
	startCol := contentWidth - panelActualWidth + 2
	if startCol < 0 {
		startCol = 0
	}

	// Build new content with overlay
	var result strings.Builder
	panelLines := strings.Split(panel, "\n")

	for i, line := range lines {
		if i >= startRow && i < startRow+panelHeight {
			panelLineIdx := i - startRow
			if panelLineIdx < len(panelLines) {

				lineWidth := lipgloss.Width(line)
				if lineWidth < startCol {

					line = line + strings.Repeat(" ", startCol-lineWidth)
				} else if lineWidth > startCol {

					line = truncateWithAnsi(line, startCol)
				}
				line = line + panelLines[panelLineIdx]
			}
		}
		result.WriteString(line)
		if i < len(lines)-1 {
			result.WriteString("\n")
		}
	}

	return result.String()
}

// renderConfirmationDialog renders a centered confirmation dialog for install/uninstall/update
func (m model) renderConfirmationDialog(contentWidth, contentHeight int, activeColor lipgloss.Color) string {

	dialogWidth := contentWidth - 20
	if dialogWidth < 50 {
		dialogWidth = 50
	}
	if dialogWidth > 80 {
		dialogWidth = 80
	}

	// Determine packages to display and title
	var packages []Package
	var title string
	var actionDesc string
	var simpleConfirm bool // For confirmations without package lists

	switch m.confirmType {
	case confirmInstall:
		title = "📦 Confirm Installation"
		actionDesc = "install"
		for _, name := range m.confirmPackages {
			packages = append(packages, Package{Name: name})
		}
	case confirmUninstall:
		title = "🗑️  Confirm Removal"
		actionDesc = "remove"
		for _, name := range m.confirmPackages {
			packages = append(packages, Package{Name: name})
		}
	case confirmUpdate:
		title = "🔄 Confirm System Update"
		actionDesc = "update"
		packages = m.pendingUpdates
	case confirmCleanCache:
		title = "🧹 Confirm Cache Cleaning"
		actionDesc = "clean"
		simpleConfirm = true
	case confirmRemoveOrphans:
		title = "🗑️  Confirm Orphan Removal"
		actionDesc = "remove"
		for _, name := range m.confirmPackages {
			packages = append(packages, Package{Name: name})
		}
	}

	dialogBorderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(activeColor).
		Padding(1, 2)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(activeColor).
		MarginBottom(1)

	packageNameStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("39"))

	packageVersionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))

	sourceStyle := func(source string) lipgloss.Style {
		if color, ok := sourceColors[source]; ok {
			return lipgloss.NewStyle().Foreground(color)
		}
		return lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	}

	countStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("214")).
		Bold(true)

	promptStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		MarginTop(1)

	keyStyle := lipgloss.NewStyle().
		Foreground(activeColor).
		Bold(true)

	scrollHintStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))

	// Build dialog content
	var content strings.Builder

	content.WriteString(titleStyle.Render(title))
	content.WriteString("\n\n")

	if simpleConfirm {
		if m.confirmType == confirmCleanCache {
			content.WriteString("This will remove cached packages that are no longer installed.\n\n")

			content.WriteString(packageNameStyle.Render("Pacman Cache (system):\n"))
			content.WriteString(fmt.Sprintf("  Path: %s\n", scrollHintStyle.Render(m.dashboard.PacmanCachePath)))
			content.WriteString(fmt.Sprintf("  Size: %s\n\n", countStyle.Render(m.dashboard.PacmanCacheSize)))

			content.WriteString(packageNameStyle.Render("Paru Cache (user):\n"))
			content.WriteString(fmt.Sprintf("  Path: %s\n", scrollHintStyle.Render(m.dashboard.ParuCachePath)))
			content.WriteString(fmt.Sprintf("  Size: %s\n\n", countStyle.Render(m.dashboard.ParuCacheSize)))

			content.WriteString(fmt.Sprintf("Total cache size: %s\n",
				lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214")).Render(m.dashboard.CleanerSize)))
		}
	} else {

		if m.confirmType == confirmUpdate {
			if len(packages) == 1 {
				content.WriteString("The following update is available:\n\n")
			} else {
				content.WriteString(fmt.Sprintf("The following %s updates are available:\n\n",
					countStyle.Render(fmt.Sprintf("%d", len(packages)))))
			}
			if actionDesc == "update" {
				if len(packages) == 1 {
					content.WriteString("The following update is available:\n\n")
				} else {
					content.WriteString(fmt.Sprintf("The following %s updates are available:\n\n",
						countStyle.Render(fmt.Sprintf("%d", len(packages)))))
				}
			} else {
				if len(packages) == 1 {
					content.WriteString(fmt.Sprintf("The following package will be %sd:\n\n", actionDesc))
				} else {
					content.WriteString(fmt.Sprintf("The following %s packages will be %sd:\n\n",
						countStyle.Render(fmt.Sprintf("%d", len(packages))), actionDesc))
				}
			}
		}

		maxVisible := 10
		startIdx := m.confirmScrollOffset
		endIdx := startIdx + maxVisible
		if endIdx > len(packages) {
			endIdx = len(packages)
		}
		visibleCount := endIdx - startIdx

		innerWidth := dialogWidth - 6
		if innerWidth < 10 {
			innerWidth = 10
		}

		showScrollbar := len(packages) > maxVisible

		// Build list entries
		type listEntry struct {
			text  string
			isPkg bool
		}
		var entries []listEntry

		if showScrollbar {

			topHintText := ""
			if startIdx > 0 {
				topHintText = fmt.Sprintf("  ↑ %d more above", startIdx)
			}
			entries = append(entries, listEntry{text: topHintText})

			for i := startIdx; i < endIdx; i++ {
				pkg := packages[i]
				var line string
				if m.confirmType == confirmUpdate {
					sourceBadge := sourceStyle(pkg.Source).Render(fmt.Sprintf("[%s]", pkg.Source))
					line = fmt.Sprintf("  • %s %s %s",
						sourceBadge,
						packageNameStyle.Render(pkg.Name),
						packageVersionStyle.Render(pkg.Version))
				} else {
					line = fmt.Sprintf("  • %s", packageNameStyle.Render(pkg.Name))
				}
				entries = append(entries, listEntry{text: line, isPkg: true})
			}

			bottomHintText := ""
			remaining := len(packages) - endIdx
			if remaining > 0 {
				bottomHintText = fmt.Sprintf("  ↓ %d more below", remaining)
			}
			entries = append(entries, listEntry{text: bottomHintText})
		} else {

			for i := 0; i < len(packages); i++ {
				pkg := packages[i]
				var line string
				if m.confirmType == confirmUpdate {
					sourceBadge := sourceStyle(pkg.Source).Render(fmt.Sprintf("[%s]", pkg.Source))
					line = fmt.Sprintf("  • %s %s %s",
						sourceBadge,
						packageNameStyle.Render(pkg.Name),
						packageVersionStyle.Render(pkg.Version))
				} else {
					line = fmt.Sprintf("  • %s", packageNameStyle.Render(pkg.Name))
				}
				entries = append(entries, listEntry{text: line, isPkg: true})
			}
		}

		// Calculate scrollbar metrics if needed
		var thumbSize, thumbTop int

		trackHeight := maxVisible
		if showScrollbar && len(packages) > maxVisible {

			thumbSize = (visibleCount * trackHeight) / len(packages)
			if thumbSize < 1 {
				thumbSize = 1
			}
			if thumbSize > trackHeight {
				thumbSize = trackHeight
			}

			if len(packages) > maxVisible {
				thumbTop = 1 + startIdx*(trackHeight-thumbSize)/(len(packages)-maxVisible)
			}
		}

		scrollbarTrackStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("236"))
		scrollbarThumbStyle := lipgloss.NewStyle().Foreground(activeColor).Bold(true)

		for i, entry := range entries {
			contentWidth := innerWidth
			var sbChar string
			if showScrollbar {
				contentWidth = innerWidth - 1
				if entry.isPkg {
					if i >= thumbTop && i < thumbTop+thumbSize {
						sbChar = scrollbarThumbStyle.Render("█")
					} else {
						sbChar = scrollbarTrackStyle.Render("│")
					}
				} else {
					sbChar = " "
				}
			}
			line := lipgloss.NewStyle().Width(contentWidth).Render(entry.text)
			if sbChar != "" {
				line = line + sbChar
			}
			content.WriteString(line + "\n")
		}

		if len(packages) > maxVisible {

			content.WriteString("\n")
			hint := scrollHintStyle.Render("  Use [↑/↓] or [j/k] to scroll")

			if lipgloss.Width(hint) > innerWidth {
				hint = lipgloss.NewStyle().Width(innerWidth).Render(hint)
			}
			content.WriteString(hint)
		}
	}

	content.WriteString("\n\n")
	var promptLine string
	promptLine = fmt.Sprintf("Proceed? %ses  %so",
		keyStyle.Render("[y]"),
		keyStyle.Render("[n]"))
	content.WriteString(promptStyle.Render(promptLine))

	dialogContent := content.String()
	dialog := dialogBorderStyle.Width(dialogWidth).Render(dialogContent)

	dialogHeight := strings.Count(dialog, "\n") + 1

	vertPadding := (contentHeight - dialogHeight) / 2
	if vertPadding < 0 {
		vertPadding = 0
	}
	horizPadding := (contentWidth - lipgloss.Width(dialog)) / 2
	if horizPadding < 0 {
		horizPadding = 0
	}

	// Build final output with centering
	var output strings.Builder

	for i := 0; i < vertPadding; i++ {
		output.WriteString("\n")
	}

	for _, line := range strings.Split(dialog, "\n") {
		output.WriteString(strings.Repeat(" ", horizPadding))
		output.WriteString(line)
		output.WriteString("\n")
	}

	return output.String()
}

// renderErrorOverlay renders a centered error overlay dialog
func (m model) renderErrorOverlay(contentWidth, contentHeight int) string {

	errorColor := lipgloss.Color("#FF5555")

	dialogWidth := contentWidth - 20
	if dialogWidth < 50 {
		dialogWidth = 50
	}
	if dialogWidth > 80 {
		dialogWidth = 80
	}

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(errorColor).
		Width(dialogWidth - 4).
		Align(lipgloss.Center)

	messageStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Width(dialogWidth - 4).
		Align(lipgloss.Center)

	detailsStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#999999")).
		Width(dialogWidth-4).
		Padding(1, 0)

	hintStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666")).
		Width(dialogWidth - 4).
		Align(lipgloss.Center)

	dialogBorderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(errorColor).
		Padding(1, 2)

	// Build content
	var content strings.Builder

	content.WriteString(titleStyle.Render("⚠  " + m.errorTitle + "  ⚠"))
	content.WriteString("\n\n")

	content.WriteString(messageStyle.Render(m.errorMessage))
	content.WriteString("\n")

	if m.errorDetails != "" {
		content.WriteString(detailsStyle.Render(m.errorDetails))
		content.WriteString("\n")
	}

	content.WriteString(hintStyle.Render("Press [esc], [enter], or [q] to dismiss"))

	dialogContent := content.String()
	dialog := dialogBorderStyle.Width(dialogWidth).Render(dialogContent)

	dialogHeight := strings.Count(dialog, "\n") + 1

	vertPadding := (contentHeight - dialogHeight) / 2
	if vertPadding < 0 {
		vertPadding = 0
	}
	horizPadding := (contentWidth - lipgloss.Width(dialog)) / 2
	if horizPadding < 0 {
		horizPadding = 0
	}

	// Build final output with centering
	var output strings.Builder

	for i := 0; i < vertPadding; i++ {
		output.WriteString("\n")
	}

	for _, line := range strings.Split(dialog, "\n") {
		output.WriteString(strings.Repeat(" ", horizPadding))
		output.WriteString(line)
		output.WriteString("\n")
	}

	return output.String()
}
