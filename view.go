package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderHelpText creates the help menu with the active mode highlighted
func (m *model) renderHelpText(activeColor lipgloss.Color) string {
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

	if m.mode == modeUpdate || m.mode == modeUpdateSelective {
		parts = append(parts, activeStyle.Render("[u]pdate"))
	} else {
		parts = append(parts, dimStyle.Render("[u]pdate"))
	}
	parts = append(parts, dimStyle.Render("  "))

	parts = append(parts, dimStyle.Render("[q]uit"))

	return strings.Join(parts, "")
}

func (m *model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	innerWidth := m.width
	innerHeight := m.height // Use full terminal height for base calculations

	activeColor := modeColors[m.mode]
	if activeColor == "" {
		activeColor = defaultBorderColor
	}

	helpText := m.renderHelpText(activeColor)

	var content string

	if m.showConfirmation {
		content = m.renderConfirmationDialog(innerWidth, innerHeight, activeColor)
	} else if m.showErrorOverlay {
		content = m.renderErrorOverlay(innerWidth, innerHeight)
	} else if m.mode == modeInstalled {
		content = m.renderDashboard(helpText, innerWidth, innerHeight)
	} else if m.mode == modeUpdate {
		content = m.renderSimpleUpdateView(helpText, innerWidth, innerHeight, activeColor)
	} else if m.mode == modeUpdateSelective {
		content = m.renderUpdateSelectiveView(helpText, innerWidth, innerHeight, activeColor)
	} else {
		// Handle modeInstall and modeUninstall
		helpWidth := lipgloss.Width(helpText)
		padding := innerWidth - helpWidth
		if padding < 0 {
			padding = 0
		}
		footer := strings.Repeat(" ", padding) + helpText
		if lipgloss.Width(footer) > innerWidth {
			footer = truncateWithAnsi(footer, innerWidth)
		}
		content = m.renderPackageListLayout(innerWidth, innerHeight, activeColor, "", footer)
	}

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Top,
		content,
	)
}

// renderUpdateSelectiveView renders the selective update overlay on top of the simple update view
func (m *model) renderUpdateSelectiveView(helpText string, innerWidth, innerHeight int, activeColor lipgloss.Color) string {
	overlayWidth := int(float64(innerWidth) * 0.75)
	overlayHeight := int(float64(innerHeight) * 0.75)

	if overlayWidth < 60 {
		overlayWidth = 60
	}
	if overlayHeight < 20 {
		overlayHeight = 20
	}
	if overlayWidth > innerWidth {
		overlayWidth = innerWidth
	}
	if overlayHeight > innerHeight {
		overlayHeight = innerHeight
	}

	paddingX := (innerWidth - overlayWidth) / 2
	paddingY := (innerHeight - overlayHeight) / 2

	warningSymbol := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("⚠")
	warningText := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(" Selective updates can break system dependencies")
	warningBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("196")).
		Padding(0, 1).
		Render(warningSymbol + warningText)

	warningOverlay := lipgloss.PlaceHorizontal(overlayWidth, lipgloss.Center, warningBox)
	warningHeight := lipgloss.Height(warningOverlay)

	// Subtract space for the actual warning overlay height
	overlayInnerHeight := overlayHeight - warningHeight
	if overlayInnerHeight < 5 {
		overlayInnerHeight = 5
	}

	paneContent := m.renderVerticalSplitLayout(overlayWidth, overlayInnerHeight, activeColor)
	
	// Composite panels and warning
	// Ensure warning overlay has a fixed height that we accounted for
	warningBoxWrapped := lipgloss.NewStyle().
		Height(warningHeight).
		Render(warningOverlay)
		
	paneContent = lipgloss.JoinVertical(lipgloss.Left, paneContent, warningBoxWrapped)

	// Enforce strict rectangle bounds - this ensures we exactly match overlayHeight
	paneContent = lipgloss.Place(overlayWidth, overlayHeight, lipgloss.Center, lipgloss.Center, paneContent)

	bg := m.renderSimpleUpdateView(helpText, innerWidth, innerHeight, activeColor)
	bgLines := strings.Split(bg, "\n")
	paneLines := strings.Split(paneContent, "\n")

	var output strings.Builder
	for i, bgLine := range bgLines {
		if i >= paddingY && i < paddingY+overlayHeight {
			paneLineIdx := i - paddingY
			if paneLineIdx < len(paneLines) {
				paneLine := paneLines[paneLineIdx]

				// Draw background left
				leftStr := ""
				if paddingX > 0 {
					leftStr = truncateWithAnsi(bgLine, paddingX)
				}

				rightStr := ""
				rightStart := paddingX + overlayWidth
				if rightStart < lipgloss.Width(bgLine) {
					rightStr = substringAnsi(bgLine, rightStart)
				}

				output.WriteString(leftStr)
				output.WriteString(paneLine)
				output.WriteString(rightStr)
				if i < len(bgLines)-1 {
					output.WriteString("\n")
				}
				continue
			}
		}
		output.WriteString(bgLine)
		if i < len(bgLines)-1 {
			output.WriteString("\n")
		}
	}

	return output.String()
}

// renderVerticalSplitLayout renders a side-by-side view (list on left, details on right)
func (m *model) renderVerticalSplitLayout(innerWidth, innerHeight int, activeColor lipgloss.Color) string {
	borderStyle := baseBorderStyle.BorderForeground(activeColor)

	// Width distribution: 40% list, 60% details
	listWidth := int(float64(innerWidth) * 0.4)
	detailsWidth := innerWidth - listWidth

	if listWidth < 25 {
		listWidth = 25
		detailsWidth = innerWidth - listWidth
	}
	if detailsWidth < 20 {
		detailsWidth = 20
		listWidth = innerWidth - detailsWidth
	}

	// 1. Render List Side (Left)
	var results strings.Builder
	pkgList := m.filtered

	// Calculate results height for the list
	// InnerHeight - 2 for borders - 3 for search input and labels
	resultsHeight := innerHeight - 5 
	if resultsHeight < 1 {
		resultsHeight = 1
	}

	if m.loading {
		results.WriteString("  Loading...")
	} else if len(pkgList) == 0 {
		results.WriteString("  No matches")
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

			var displayPkgStr string
			if indices, ok := m.matchIndices[i]; ok {
				displayPkgStr = highlightMatchesWithSourceColor(pkg, indices)
			} else {
				displayPkgStr = sourceStyle.Render(pkg.Source) + "/" + pkg.Name
			}

			line := fmt.Sprintf("%s%s", prefix, displayPkgStr)
			
			// Truncate to fit listWidth
			if lipgloss.Width(line) > listWidth-2 {
				line = truncateWithAnsi(line, listWidth-5) + "..."
			}

			if i == m.selectedIndex {
				line = selectedStyle.Render(line)
			}
			lines = append(lines, line)
		}
		// List is rendered bottom-to-top near input field
		for i := len(lines) - 1; i >= 0; i-- {
			results.WriteString(lines[i])
			if i > 0 {
				results.WriteString("\n")
			}
		}
	}

	listPanel := borderStyle.
		Width(listWidth - 2).
		Height(innerHeight - 2).
		Render(lipgloss.JoinVertical(lipgloss.Left,
			results.String(),
			lipgloss.NewStyle().Height(resultsHeight - lipgloss.Height(results.String())).Render(""), // filler
			m.textInput.View(),
		))

	// 2. Render Details Side (Right)
	infoContent := ""
	if m.loadingInfo {
		infoContent = fmt.Sprintf("Loading details for %s...", m.infoForPackage)
	} else if m.packageInfo != "" {
		infoContent = m.packageInfo
	} else {
		infoContent = "Select an update to see details"
	}

	infoInnerHeight := innerHeight - 2
	infoInnerWidth := detailsWidth - 4 // padding

	wrappedText := lipgloss.NewStyle().Width(infoInnerWidth).Render(infoContent)
	infoLines := strings.Split(wrappedText, "\n")

	totalLines := len(infoLines)
	if totalLines > infoInnerHeight {
		maxScroll := totalLines - infoInnerHeight
		m.maxInfoScroll = maxScroll
		if m.infoScrollOffset > maxScroll {
			m.infoScrollOffset = maxScroll
		}
		infoLines = infoLines[m.infoScrollOffset : m.infoScrollOffset+infoInnerHeight]
	} else {
		m.maxInfoScroll = 0
		m.infoScrollOffset = 0
	}
	infoContent = strings.Join(infoLines, "\n")

	detailsBox := lipgloss.NewStyle().
		Width(infoInnerWidth).
		Height(infoInnerHeight).
		Padding(0, 1).
		Render(infoContent)

	if len(m.markedPackages) > 0 {
		selectionPanel := m.renderSelectionBox(detailsWidth - 6)
		panelLines := strings.Split(selectionPanel, "\n")
		panelHeight := len(panelLines)
		panelWidth := lipgloss.Width(panelLines[0])

		bgLines := strings.Split(detailsBox, "\n")
		
		// Overlay selectionPanel on bottom right of detailsBox
		startRow := infoInnerHeight - panelHeight
		startCol := infoInnerWidth + 2 - panelWidth
		
		if startRow < 0 { startRow = 0 }
		if startCol < 0 { startCol = 0 }

		var result strings.Builder
		for i, line := range bgLines {
			if i >= startRow && i < startRow+panelHeight {
				panelLineIdx := i - startRow
				lineWidth := lipgloss.Width(line)
				
				leftStr := ""
				if startCol > 0 {
					if lineWidth < startCol {
						leftStr = line + strings.Repeat(" ", startCol-lineWidth)
					} else {
						leftStr = truncateWithAnsi(line, startCol)
					}
				}
				
				line = leftStr + panelLines[panelLineIdx]
			}
			result.WriteString(line)
			if i < len(bgLines)-1 {
				result.WriteString("\n")
			}
		}
		detailsBox = result.String()
	}

	detailsPanel := borderStyle.
		Width(detailsWidth - 2).
		Height(innerHeight - 2).
		Render(detailsBox)

	return lipgloss.JoinHorizontal(lipgloss.Top, listPanel, detailsPanel)
}

// renderSelectionBox builds the selection panel string
func (m *model) renderSelectionBox(maxWidth int) string {
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

	var selectionsList strings.Builder
	titleText := titleStyle.Render(fmt.Sprintf("Selected (%d) ", len(m.markedPackages))) + keyHintStyle.Render("[*]")
	selectionsList.WriteString(titleText)

	var pkgNames []string
	for name := range m.markedPackages {
		pkgNames = append(pkgNames, name)
	}
	sort.Strings(pkgNames)

	maxDisplay := 10 // Reduced for overlay
	if m.selectionPanelFocused {
		maxDisplay = 15
	}
	
	startIdx := m.selectionScrollOffset
	endIdx := startIdx + maxDisplay
	if endIdx > len(pkgNames) {
		endIdx = len(pkgNames)
	}

	visibleNames := pkgNames[startIdx:endIdx]
	maxContentWidth := lipgloss.Width(titleText)
	for _, name := range visibleNames {
		nameWidth := lipgloss.Width(name) + 2
		if nameWidth > maxContentWidth {
			maxContentWidth = nameWidth
		}
	}

	desiredPanelWidth := maxContentWidth + 4
	if desiredPanelWidth > maxWidth {
		desiredPanelWidth = maxWidth
	}
	panelWidth := desiredPanelWidth

	if startIdx > 0 {
		selectionsList.WriteString("\n")
		selectionsList.WriteString(itemStyle.Render(fmt.Sprintf("... %d above", startIdx)))
	}

	for i := startIdx; i < endIdx; i++ {
		name := pkgNames[i]
		innerWidth := panelWidth - 4
		nameMaxWidth := innerWidth - 2
		if nameMaxWidth < 1 { nameMaxWidth = 1 }

		displayName := name
		if lipgloss.Width(displayName) > nameMaxWidth {
			displayName = truncateWithAnsi(displayName, nameMaxWidth-3) + "..."
		}

		selectionsList.WriteString("\n")
		if m.selectionPanelFocused && i == m.selectionPanelIndex {
			selectionsList.WriteString(selectedItemStyle.Render("> " + displayName))
		} else {
			selectionsList.WriteString(itemStyle.Render("  " + displayName))
		}
	}

	if endIdx < len(pkgNames) {
		selectionsList.WriteString("\n")
		selectionsList.WriteString(itemStyle.Render(fmt.Sprintf("... %d more below", len(pkgNames)-endIdx)))
	}

	return panelStyle.Width(panelWidth).Render(selectionsList.String())
}

// renderPackageListLayout renders the standard split-pane list view for install, uninstall, and selective update
func (m *model) renderPackageListLayout(innerWidth, innerHeight int, activeColor lipgloss.Color, header, footer string) string {
	borderStyle := baseBorderStyle.BorderForeground(activeColor)

	headerHeight := 0
	if header != "" {
		headerHeight = 1
	}
	footerHeight := 0
	if footer != "" {
		footerHeight = 1
	}

	availableHeight := innerHeight - headerHeight - footerHeight
	if availableHeight < 6 {
		availableHeight = 6
	}

	targetInfoPanelHeight := availableHeight / 2
	targetBottomPanelHeight := availableHeight - targetInfoPanelHeight

	infoInnerHeight := targetInfoPanelHeight - 2
	bottomInnerHeight := targetBottomPanelHeight - 2

	resultsHeight := bottomInnerHeight - 3
	if resultsHeight < 1 {
		resultsHeight = 1
		bottomInnerHeight = 4
		targetBottomPanelHeight = 6
		targetInfoPanelHeight = availableHeight - targetBottomPanelHeight
		infoInnerHeight = targetInfoPanelHeight - 2
		if infoInnerHeight < 1 {
			infoInnerHeight = 1
		}
	}

	infoContent := ""
	if m.mode == modeUpdateSelective {
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

	// FIX: Use innerWidth - 2 for inner content width (accounts for panel borders)
	contentWidth := innerWidth - 2
	if contentWidth < 10 {
		contentWidth = 10
	}

	// Render the text with a width limit first so Lipgloss wraps it
	wrappedText := lipgloss.NewStyle().Width(innerWidth - 2).Render(infoContent)
	infoLines := strings.Split(wrappedText, "\n")

	totalLines := len(infoLines)
	if totalLines > infoInnerHeight {
		maxScroll := totalLines - infoInnerHeight
		m.maxInfoScroll = maxScroll

		// Clamp offset
		if m.infoScrollOffset > maxScroll {
			m.infoScrollOffset = maxScroll
		} else if m.infoScrollOffset < 0 {
			m.infoScrollOffset = 0
		}

		infoLines = infoLines[m.infoScrollOffset : m.infoScrollOffset+infoInnerHeight]
	} else {
		m.maxInfoScroll = 0
		m.infoScrollOffset = 0
	}
	infoContent = strings.Join(infoLines, "\n")

	infoBox := lipgloss.NewStyle().
		Width(contentWidth).
		Height(infoInnerHeight).
		Padding(0, 1).
		Render(infoContent)

	infoPanel := borderStyle.
		Width(innerWidth - 2).
		Height(targetInfoPanelHeight - 2).
		Render(infoBox)

	// Build results list
	var results strings.Builder
	var pkgList []Package
	if m.mode == modeInstall {
		pkgList = m.filtered
	} else if m.mode == modeUninstall {
		pkgList = m.filteredInstalled
	} else if m.mode == modeUpdateSelective {
		pkgList = m.filtered
	}

	if m.loading {
		results.WriteString("  Loading...")
	} else if m.mode == modeUpdateSelective && len(pkgList) == 0 && !m.loading {
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
		} else if m.mode == modeUpdateSelective {
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

			if lipgloss.Width(line) > innerWidth-4 {
				line = line[:innerWidth-7] + "..."
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
		Width(contentWidth).
		Height(resultsHeight).
		Render(results.String())

	inputLine := ""
	if m.mode == modeInstall || m.mode == modeUninstall || m.mode == modeUpdateSelective {
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
		Width(innerWidth - 2).
		Height(targetBottomPanelHeight - 2).
		Render(bottomContent)

	var layoutSections []string
	if header != "" {
		layoutSections = append(layoutSections, header)
	}
	layoutSections = append(layoutSections, infoPanel, bottomPanel)
	if footer != "" {
		layoutSections = append(layoutSections, footer)
	}

	content := lipgloss.JoinVertical(lipgloss.Left, layoutSections...)

	if len(m.markedPackages) > 0 {
		content = m.overlaySelectionsPanel(content, innerWidth, headerHeight)
	}

	return content
}

// overlaySelectionsPanel renders a selection panel on the bottom right of the screen
func (m *model) overlaySelectionsPanel(content string, innerWidth int, headerHeight int) string {
	panel := m.renderSelectionBox(32)
	panelLines := strings.Split(panel, "\n")
	panelHeight := len(panelLines)
	panelWidth := lipgloss.Width(panelLines[0])

	lines := strings.Split(content, "\n")

	startRow := headerHeight // Anchor exactly on the top border
	startCol := innerWidth - panelWidth
	if startCol < 0 {
		startCol = 0
	}

	// Build new content with overlay
	var result strings.Builder
	for i, line := range lines {
		if i >= startRow && i < startRow+panelHeight {
			panelLineIdx := i - startRow
			
			lineWidth := lipgloss.Width(line)
			leftStr := ""
			if startCol > 0 {
				if lineWidth < startCol {
					leftStr = line + strings.Repeat(" ", startCol-lineWidth)
				} else {
					leftStr = truncateWithAnsi(line, startCol)
				}
			}

			rightStr := ""
			rightStart := startCol + panelWidth
			if rightStart < lineWidth {
				rightStr = substringAnsi(line, rightStart)
			}

			line = leftStr + panelLines[panelLineIdx] + rightStr
		}
		result.WriteString(line)
		if i < len(lines)-1 {
			result.WriteString("\n")
		}
	}

	return result.String()
}

// renderConfirmationDialog renders a centered confirmation dialog for install/uninstall/update
func (m *model) renderConfirmationDialog(innerWidth, innerHeight int, activeColor lipgloss.Color) string {

	dialogWidth := innerWidth - 20
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
	case confirmSelectiveUpdate:
		title = "🔄 Confirm Selective Update"
		actionDesc = "update"
		for _, name := range m.confirmPackages {
			packages = append(packages, Package{Name: name})
		}
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

		maxScroll := len(packages) - maxVisible
		if maxScroll < 0 {
			maxScroll = 0
		}
		m.maxConfirmScroll = maxScroll
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
			innerWidth := innerWidth
			var sbChar string
			if showScrollbar {
				innerWidth = innerWidth - 1
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
			line := lipgloss.NewStyle().Width(innerWidth).Render(entry.text)
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

	content.WriteString("\n")
	var promptLine string
	promptLine = fmt.Sprintf("Proceed? %ses  %so",
		keyStyle.Render("[y]"),
		keyStyle.Render("[n]"))
	content.WriteString(promptStyle.Render(promptLine))

	dialogContent := content.String()
	dialog := dialogBorderStyle.Width(dialogWidth).Render(dialogContent)

	// Use lipgloss.Place for reliable centering instead of manual string padding
	return lipgloss.Place(
		innerWidth,
		innerHeight,
		lipgloss.Center,
		lipgloss.Center,
		dialog,
	)
}

// renderErrorOverlay renders a centered error overlay dialog
func (m *model) renderErrorOverlay(innerWidth, innerHeight int) string {

	errorColor := lipgloss.Color("#FF5555")

	dialogWidth := innerWidth - 20
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

	vertPadding := (innerHeight - dialogHeight) / 2
	if vertPadding < 0 {
		vertPadding = 0
	}
	horizPadding := (innerWidth - lipgloss.Width(dialog)) / 2
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

// renderSimpleUpdateView renders the simple overview page for pending updates
func (m *model) renderSimpleUpdateView(helpText string, innerWidth, innerHeight int, activeColor lipgloss.Color) string {
	borderStyle := baseBorderStyle.BorderForeground(activeColor)

	helpWidth := lipgloss.Width(helpText)
	padding := innerWidth - helpWidth
	if padding < 0 {
		padding = 0
	}
	footerLine := strings.Repeat(" ", padding) + helpText
	if lipgloss.Width(footerLine) > innerWidth {
		footerLine = truncateWithAnsi(footerLine, innerWidth)
	}

	footerHeight := 1
	panelHeight := innerHeight - footerHeight

	var content strings.Builder
	innerContentHeight := 0

	if m.loading {
		content.WriteString("\n  Checking for updates...")
		innerContentHeight = 2
	} else if len(m.pendingUpdates) == 0 {
		content.WriteString("\n  System is up to date!")
		innerContentHeight = 2
	} else {
		countStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
		content.WriteString(fmt.Sprintf("  The following %s system updates are available:\n\n", countStyle.Render(fmt.Sprintf("%d", len(m.pendingUpdates)))))
		innerContentHeight += 2

		innerFooterHeight := 1
		availableLinesForList := panelHeight - innerContentHeight - innerFooterHeight - 2 // -2 for borders
		if availableLinesForList < 1 {
			availableLinesForList = 1
		}

		displayCount := availableLinesForList

		if m.updateScrollOffset > 0 {
			displayCount -= 2
		}
		if m.updateScrollOffset+displayCount < len(m.pendingUpdates) {
			displayCount -= 2
		}
		if displayCount < 1 {
			displayCount = 1
		}

		if displayCount > len(m.pendingUpdates)-m.updateScrollOffset {
			displayCount = len(m.pendingUpdates) - m.updateScrollOffset
		}

		m.maxUpdateScroll = len(m.pendingUpdates) - 1
		if m.maxUpdateScroll < 0 {
			m.maxUpdateScroll = 0
		}

		if m.updateScrollOffset > 0 {
			content.WriteString(fmt.Sprintf("    ... and %d more above.\n\n", m.updateScrollOffset))
			innerContentHeight += 2
		}

		for i := 0; i < displayCount; i++ {
			pkgIndex := i + m.updateScrollOffset
			if pkgIndex >= len(m.pendingUpdates) {
				break
			}
			pkg := m.pendingUpdates[pkgIndex]
			sourceBadge := ""
			if color, ok := sourceColors[pkg.Source]; ok {
				sourceBadge = lipgloss.NewStyle().Foreground(color).Render(fmt.Sprintf("[%s]", pkg.Source))
			} else {
				sourceBadge = fmt.Sprintf("[%s]", pkg.Source)
			}
			content.WriteString(fmt.Sprintf("    • %s %s %s\n", sourceBadge, lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Render(pkg.Name), lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(pkg.Version)))
			innerContentHeight++
		}

		if m.updateScrollOffset+displayCount < len(m.pendingUpdates) {
			content.WriteString(fmt.Sprintf("\n    ... and %d more below.", len(m.pendingUpdates)-(m.updateScrollOffset+displayCount)))
			innerContentHeight += 2
		}
	}

	// Build main list content
	listContent := content.String()

	// Build buttons separately to pin them to the bottom right
	var buttonsContent string
	if !m.loading && len(m.pendingUpdates) > 0 {
		buttonStyle := lipgloss.NewStyle().
			Background(lipgloss.Color("238")).
			Foreground(lipgloss.Color("255")).
			Padding(0, 1).
			Bold(true)

		buttonStyleRed := lipgloss.NewStyle().
			Background(lipgloss.Color("196")).
			Foreground(lipgloss.Color("255")).
			Padding(0, 1).
			Bold(true)

		btnUpdate := buttonStyle.Render("[enter] update all")
		btnSelective := buttonStyleRed.Render("[s]elective")

		buttons := btnUpdate + "   " + btnSelective
		
		// Use lipgloss.Place to push buttons to the bottom-right corner
		// Total inner width is innerWidth - 4 (padding+borders)
		buttonsContent = lipgloss.Place(innerWidth-4, 1, lipgloss.Right, lipgloss.Bottom, buttons)
	}

	// Join list and buttons, ensuring buttons are at the bottom
	// Total available inner height is panelHeight - 2
	innerPanelContent := lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().Height(panelHeight-2-1).Render(listContent), // List takes all but the last line
		buttonsContent, // Buttons take exactly the last line
	)

	panelContent := lipgloss.NewStyle().
		Width(innerWidth - 2).
		Height(panelHeight - 2).
		Padding(0, 1).
		Render(innerPanelContent)

	mainPanel := borderStyle.
		Width(innerWidth - 2).
		Height(panelHeight - 2).
		Render(panelContent)

	return lipgloss.JoinVertical(lipgloss.Left, mainPanel, footerLine)
}
