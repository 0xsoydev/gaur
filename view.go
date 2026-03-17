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

	parts = append(parts, renderKeyHint("search", m.keys.Search, dimStyle))
	parts = append(parts, dimStyle.Render("  "))
	parts = append(parts, renderKeyHint("mark", m.keys.Mark, dimStyle))
	parts = append(parts, dimStyle.Render("  "))

	installStyle := dimStyle
	if m.mode == modeInstall {
		installStyle = activeStyle
	}
	parts = append(parts, renderKeyHint("install", m.keys.InstallMode, installStyle))
	parts = append(parts, dimStyle.Render("  "))

	infoStyle := dimStyle
	if m.mode == modeInstalled {
		infoStyle = activeStyle
	}
	parts = append(parts, renderKeyHint("info", m.keys.DashboardMode, infoStyle))
	parts = append(parts, dimStyle.Render("  "))

	removeStyle := dimStyle
	if m.mode == modeUninstall {
		removeStyle = activeStyle
	}
	parts = append(parts, renderKeyHint("remove", m.keys.UninstallMode, removeStyle))
	parts = append(parts, dimStyle.Render("  "))

	updateStyle := dimStyle
	if m.mode == modeUpdate || m.mode == modeUpdateSelective {
		updateStyle = activeStyle
	}
	parts = append(parts, renderKeyHint("update", m.keys.UpdateMode, updateStyle))
	parts = append(parts, dimStyle.Render("  "))

	settingsStyle := dimStyle
	if m.mode == modeSettings {
		settingsStyle = activeStyle
	}
	parts = append(parts, renderKeyHint("settings", m.keys.Settings, settingsStyle))
	parts = append(parts, dimStyle.Render("  "))

	parts = append(parts, renderKeyHint("quit", m.keys.Quit, dimStyle))

	return strings.Join(parts, "")
}

func (m *model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	innerWidth := m.width
	innerHeight := m.height // Use full terminal height for base calculations

	// Determine effective mode for rendering background elements
	effectiveMode := m.mode
	if m.mode == modeSettings {
		effectiveMode = m.previousMode
	}

	// Apply configured border style
	baseBorderStyle = baseBorderStyle.Border(m.getBorderStyle())

	activeColor := modeColors[effectiveMode]
	if activeColor == "" {
		activeColor = defaultBorderColor
	}

	helpText := m.renderHelpText(activeColor)

	var content string
	
	if m.showConfirmation {
		content = m.renderConfirmationDialog(innerWidth, innerHeight, activeColor)
	} else if m.showErrorOverlay {
		content = m.renderErrorOverlay(innerWidth, innerHeight)
	} else if effectiveMode == modeCacheMenu {
		content = m.renderCacheMenu(helpText, innerWidth, innerHeight)
	} else if effectiveMode == modeCacheSelective {
		content = m.renderSelectiveCacheView(helpText, innerWidth, innerHeight, activeColor)
	} else if effectiveMode == modeInstalled {
		content = m.renderDashboard(helpText, innerWidth, innerHeight)
	} else if effectiveMode == modeUpdate {
		content = m.renderSimpleUpdateView(helpText, innerWidth, innerHeight, activeColor)
	} else if effectiveMode == modeUpdateSelective {
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

	// If settings are active, overlay them on top of the rendered content
	if m.mode == modeSettings {
		settingsOverlay := m.renderSettings(innerWidth, innerHeight)
		// Since true terminal layering is complex, we use lipgloss.Place 
		// but let it handle the background by NOT filling with whitespace if we want transparency.
		// However, most TUI overlays just clear their area.
		content = m.overlaySettings(content, settingsOverlay, innerWidth, innerHeight)
	}

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Top,
		content,
	)
}

// overlaySettings manually layers the settings menu on top of base content
func (m *model) overlaySettings(base, overlay string, width, height int) string {
	baseLines := strings.Split(base, "\n")
	overlayLines := strings.Split(overlay, "\n")
	
	overlayHeight := len(overlayLines)
	if overlayHeight == 0 {
		return base
	}
	
	overlayWidth := lipgloss.Width(overlayLines[0])

	// Ensure base has enough lines to match terminal height
	for len(baseLines) < height {
		baseLines = append(baseLines, strings.Repeat(" ", width))
	}

	startY := (height - overlayHeight) / 2
	startX := (width - overlayWidth) / 2

	result := make([]string, len(baseLines))
	copy(result, baseLines)

	for y := 0; y < overlayHeight; y++ {
		targetY := startY + y
		if targetY >= 0 && targetY < len(result) {
			bgLine := result[targetY]
			
			// Reconstruct the line using precise slicing
			left := truncateWithAnsi(bgLine, startX)
			// Ensure left part is exactly startX wide by padding if needed
			leftWidth := lipgloss.Width(left)
			if leftWidth < startX {
				left += strings.Repeat(" ", startX - leftWidth)
			}
			
			right := substringAnsi(bgLine, startX + overlayWidth)
			
			// Composite the line: [Base Left] [Overlay Content] [Base Right]
			result[targetY] = left + overlayLines[y] + right
		}
	}

	return strings.Join(result, "\n")
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
		Border(m.getBorderStyle()).
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
	var resultsStr string
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

		resultsStr = results.String()
		if !m.loading && len(pkgList) > resultsHeight {
			scrollbar := renderScrollbar(len(pkgList), startIdx, resultsHeight, activeColor, true)
			resultsStr = lipgloss.JoinHorizontal(lipgloss.Top,
				lipgloss.NewStyle().Width(listWidth-4).Render(resultsStr),
				lipgloss.NewStyle().MarginLeft(1).Render(scrollbar))
		}
	}

	resultsContainer := lipgloss.NewStyle().
		Height(resultsHeight).
		Align(lipgloss.Left, lipgloss.Bottom).
		Render(resultsStr)

	listPanel := borderStyle.
		Width(listWidth - 2).
		Height(innerHeight - 2).
		Render(lipgloss.JoinVertical(lipgloss.Left,
			resultsContainer,
			"", // spacing
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
		Border(m.getBorderStyle()).
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

	var listBuilder strings.Builder
	for i := startIdx; i < endIdx; i++ {
		name := pkgNames[i]
		innerWidth := panelWidth - 4
		nameMaxWidth := innerWidth - 2
		if nameMaxWidth < 1 { nameMaxWidth = 1 }

		displayName := name
		if lipgloss.Width(displayName) > nameMaxWidth {
			displayName = truncateWithAnsi(displayName, nameMaxWidth-3) + "..."
		}

		if i > startIdx {
			listBuilder.WriteString("\n")
		}
		if m.selectionPanelFocused && i == m.selectionPanelIndex {
			listBuilder.WriteString(selectedItemStyle.Render("> " + displayName))
		} else {
			listBuilder.WriteString(itemStyle.Render("  " + displayName))
		}
	}

	listStr := listBuilder.String()
	if len(pkgNames) > (endIdx - startIdx) {
		// Add scrollbar for selection box (Top-down)
		scrollbar := renderScrollbar(len(pkgNames), startIdx, (endIdx - startIdx), lipgloss.Color("205"), false)
		listStr = lipgloss.JoinHorizontal(lipgloss.Top,
			lipgloss.NewStyle().Width(panelWidth - 4).Render(listStr),
			lipgloss.NewStyle().MarginLeft(1).Render(scrollbar))
	}

	return panelStyle.Width(panelWidth).Render(titleText + "\n" + listStr)
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
	var resultsStr string
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

		resultsStr = results.String()
		if !m.loading && len(pkgList) > resultsHeight {
			scrollbar := renderScrollbar(len(pkgList), startIdx, resultsHeight, activeColor, true)
			resultsStr = lipgloss.JoinHorizontal(lipgloss.Top,
				lipgloss.NewStyle().Width(contentWidth-2).Render(resultsStr),
				lipgloss.NewStyle().MarginLeft(1).Render(scrollbar))
		}
	}

	resultsBox := lipgloss.NewStyle().
		Width(contentWidth).
		Height(resultsHeight).
		Align(lipgloss.Left, lipgloss.Bottom).
		Render(resultsStr)

	inputLine := ""
	
	// Determine what mode's input line to show
	displayMode := m.mode
	if m.mode == modeSettings {
		displayMode = m.previousMode
	}

	if displayMode == modeInstall || displayMode == modeUninstall || displayMode == modeUpdateSelective {
		inputLine = m.textInput.View()
	} else {
		inputLine = statusStyle.Render(m.statusMessage)
	}

	bottomContent := lipgloss.JoinVertical(
		lipgloss.Left,
		resultsBox,
		"",
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
	var packages []Package
	var title string
	var actionDesc string
	var simpleConfirm bool

	sourceStyle := func(source string) lipgloss.Style {
		if color, ok := sourceColors[source]; ok {
			return lipgloss.NewStyle().Foreground(color)
		}
		return lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	}

	switch m.confirmType {
	case confirmInstall:
		title = "📦 Confirm Installation"
		actionDesc = "install"
		simpleConfirm = false
		for _, name := range m.confirmPackages {
			pkg := m.getPackageByName(name)
			if pkg != nil {
				packages = append(packages, *pkg)
			} else {
				packages = append(packages, Package{Name: name, Source: "aur"})
			}
		}
	case confirmUninstall:
		title = "🗑 Confirm Removal"
		actionDesc = "uninstall"
		simpleConfirm = false
		for _, name := range m.confirmPackages {
			pkg := m.getPackageByName(name)
			if pkg != nil {
				packages = append(packages, *pkg)
			} else {
				packages = append(packages, Package{Name: name, Source: "core"})
			}
		}
	case confirmUpdate:
		title = "🔄 Confirm System Update"
		actionDesc = "update"
		simpleConfirm = false
		packages = m.pendingUpdates
	case confirmSelectiveUpdate:
		title = "🔄 Confirm Selective Update"
		actionDesc = "update"
		simpleConfirm = false
		for _, name := range m.confirmPackages {
			pkg := m.getPackageByName(name)
			if pkg != nil {
				packages = append(packages, *pkg)
			}
		}
	case confirmRemoveOrphans:
		title = "🧹 Confirm Orphan Removal"
		actionDesc = "remove orphans"
		simpleConfirm = false
		for _, name := range m.confirmPackages {
			packages = append(packages, Package{Name: name, Source: "core"})
		}
	case confirmCleanKeep3, confirmCleanKeep1, confirmCleanNuke:
		title = "🧹 Confirm Cache Cleaning"
		actionDesc = "clean"
		simpleConfirm = true
	case confirmCleanUninstalled:
		title = "🧹 Confirm Orphaned Cache Clean"
		actionDesc = "clean orphaned packages from cache"
		simpleConfirm = false
		if len(m.dashboard.UninstalledPacmanCache) > 0 {
			packages = append(packages, Package{Name: "pacman:", Version: "HEADER"})
			for _, p := range m.dashboard.UninstalledPacmanCache {
				packages = append(packages, Package{Name: p.Name, Size: p.Size})
			}
		}
		if len(m.dashboard.UninstalledParuCache) > 0 {
			packages = append(packages, Package{Name: m.config.Commands.AurHelper + ":", Version: "HEADER"})
			for _, p := range m.dashboard.UninstalledParuCache {
				packages = append(packages, Package{Name: p.Name, Size: p.Size})
			}
		}
	case confirmCleanSelective:
		title = "🧹 Confirm Selective Clean"
		actionDesc = "clean selected packages from cache"
		simpleConfirm = false
		for _, name := range m.confirmPackages {
			packages = append(packages, Package{Name: name})
		}
	}

	dialogWidth := innerWidth - 10
	if dialogWidth < 60 { dialogWidth = 60 }
	if dialogWidth > 90 { dialogWidth = 90 }

	activeBorderColor := activeColor
	switch m.confirmType {
	case confirmCleanKeep3:
		activeBorderColor = lipgloss.Color("39")
	case confirmCleanKeep1:
		activeBorderColor = lipgloss.Color("208")
	case confirmCleanUninstalled:
		activeBorderColor = lipgloss.Color("42")
	case confirmCleanNuke:
		activeBorderColor = lipgloss.Color("196")
	case confirmCleanSelective:
		activeBorderColor = lipgloss.Color("135")
	}

	dialogBorderStyle := lipgloss.NewStyle().

		Border(m.getBorderStyle()).
		BorderForeground(activeBorderColor).
		Padding(1, 2).
		Align(lipgloss.Left)

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(activeBorderColor)
	packageNameStyle := lipgloss.NewStyle().Foreground(activeBorderColor).Bold(true)
	packageVersionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("246"))
	countStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
	promptStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252")).MarginTop(1)
	keyStyle := lipgloss.NewStyle().Foreground(activeBorderColor).Bold(true)
	scrollHintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	var lines []string
	
	// Center Title
	lines = append(lines, lipgloss.PlaceHorizontal(dialogWidth-4, lipgloss.Center, titleStyle.Render(title)))
	lines = append(lines, "")

	// Warning/Description
	descStyle := lipgloss.NewStyle().Width(dialogWidth - 4)
	if m.confirmType == confirmCleanNuke {
		warningStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true).Width(dialogWidth - 4).Align(lipgloss.Center)
		lines = append(lines, warningStyle.Render("WARNING: This will completely empty the package cache."))
		lines = append(lines, "")
	} else if simpleConfirm {
		if m.confirmType == confirmCleanKeep3 {
			lines = append(lines, descStyle.Render("This will remove all but the 3 most recent cached versions of packages."))
		} else if m.confirmType == confirmCleanKeep1 {
			lines = append(lines, descStyle.Render("This will aggressively remove all but the currently installed cached versions."))
		}
		lines = append(lines, "")
	}

	if simpleConfirm {
		if m.confirmType == confirmCleanNuke {
			labelStyle := lipgloss.NewStyle().Width(8).Foreground(lipgloss.Color("241"))
			lines = append(lines, packageNameStyle.Render("System Cache:"))
			lines = append(lines, fmt.Sprintf("  %s %s", labelStyle.Render("Path:"), scrollHintStyle.Render(m.dashboard.PacmanCachePath)))
			lines = append(lines, packageNameStyle.Render("User Cache:"))
			lines = append(lines, fmt.Sprintf("  %s %s", labelStyle.Render("Path:"), scrollHintStyle.Render(m.dashboard.ParuCachePath)))
			lines = append(lines, "")
		}

		breakdownHeaderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Bold(true)
		lines = append(lines, breakdownHeaderStyle.Render("Breakdown:"))
		
		pacmanEstimate := m.dashboard.CacheFreedPacman[m.confirmType]
		paruEstimate := m.dashboard.CacheFreedParu[m.confirmType]
		if m.confirmType == confirmCleanNuke {
			pacmanEstimate = m.dashboard.PacmanCacheSize
			paruEstimate = m.dashboard.ParuCacheSize
		}
		if pacmanEstimate == "" { pacmanEstimate = "calculating..." }
		if paruEstimate == "" { paruEstimate = "calculating..." }

		valStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
		pacmanLabel := sourceStyle("core").Render("  pacman:")
		helperLabel := m.config.Commands.AurHelper + ":"
		if len(helperLabel) < 7 {
			helperLabel += strings.Repeat(" ", 7-len(helperLabel))
		}
		paruLabel := sourceStyle("aur").Render("  " + helperLabel)

		lines = append(lines, fmt.Sprintf("%s %s", pacmanLabel, valStyle.Render(pacmanEstimate)))
		lines = append(lines, fmt.Sprintf("%s %s", paruLabel, valStyle.Render(paruEstimate)))
		lines = append(lines, "")

		estimate := m.dashboard.CacheFreedEstimates[m.confirmType]
		if estimate == "" { estimate = "calculating..." }
		lines = append(lines, fmt.Sprintf("Estimated space to be freed: %s", lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214")).Render(estimate)))
	} else {
		// List-based Confirmations
		if m.confirmType == confirmUpdate {
			lines = append(lines, fmt.Sprintf("The following %s updates are available:", countStyle.Render(fmt.Sprintf("%d", len(packages)))))
		} else if m.confirmType == confirmCleanUninstalled {
			lines = append(lines, "This will remove cached packages that are no longer installed.")
		} else {
			lines = append(lines, fmt.Sprintf("The following %s packages will be %sd:", countStyle.Render(fmt.Sprintf("%d", len(packages))), actionDesc))
		}
		lines = append(lines, "")

		maxVisible := 10
		startIdx := m.confirmScrollOffset
		endIdx := startIdx + maxVisible
		if endIdx > len(packages) { endIdx = len(packages) }
		m.maxConfirmScroll = len(packages) - maxVisible
		if m.maxConfirmScroll < 0 { m.maxConfirmScroll = 0 }

		for i := startIdx; i < endIdx; i++ {
			pkg := packages[i]
			var line string
			if m.confirmType == confirmUpdate {
				line = fmt.Sprintf("  • %s %s %s", sourceStyle(pkg.Source).Render(fmt.Sprintf("[%s]", pkg.Source)), packageNameStyle.Render(pkg.Name), packageVersionStyle.Render(pkg.Version))
			} else if pkg.Version == "HEADER" {
				line = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229")).Render(pkg.Name)
			} else if m.confirmType == confirmCleanUninstalled || m.confirmType == confirmCleanSelective {
				namePart := "  • " + packageNameStyle.Render(pkg.Name)
				sizePart := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(pkg.Size)
				spacing := (dialogWidth - 10) - lipgloss.Width(namePart) - lipgloss.Width(sizePart)
				if spacing < 1 { spacing = 1 }
				line = namePart + strings.Repeat(" ", spacing) + sizePart
			} else {
				line = fmt.Sprintf("  • %s", packageNameStyle.Render(pkg.Name))
			}
			lines = append(lines, line)
		}

		if len(packages) > maxVisible {
			lines = append(lines, "")
			lines = append(lines, scrollHintStyle.Render("  Use [↑/↓] or [j/k] to scroll"))
		}

		if m.confirmType == confirmCleanUninstalled {
			lines = append(lines, "")
			breakdownHeaderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Bold(true)
			lines = append(lines, breakdownHeaderStyle.Render("Breakdown:"))
			pacmanEst := m.dashboard.CacheFreedPacman[m.confirmType]
			paruEst := m.dashboard.CacheFreedParu[m.confirmType]
			if pacmanEst == "" { pacmanEst = "calculating..." }
			if paruEst == "" { paruEst = "calculating..." }
			helperLabel := m.config.Commands.AurHelper + ":"
			if len(helperLabel) < 7 {
				helperLabel += strings.Repeat(" ", 7-len(helperLabel))
			}
			lines = append(lines, fmt.Sprintf("%s %s", sourceStyle("core").Render("  pacman:"), lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Render(pacmanEst)))
			lines = append(lines, fmt.Sprintf("%s %s", sourceStyle("aur").Render("  "+helperLabel), lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Render(paruEst)))
		}

		if m.confirmType == confirmCleanUninstalled || m.confirmType == confirmCleanSelective {
			lines = append(lines, "")
			est := m.dashboard.CacheFreedEstimates[m.confirmType]
			if m.confirmType == confirmCleanSelective { est = formatBytes(m.cacheToFree) }
			if est == "" { est = "calculating..." }
			lines = append(lines, fmt.Sprintf("Estimated space to be freed: %s", lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214")).Render(est)))
		}
	}

	lines = append(lines, "")
	lines = append(lines, promptStyle.Render(fmt.Sprintf("Proceed? %s  %s", 
		renderKeyHint("yes", m.keys.Confirm, keyStyle),
		renderKeyHint("no", m.keys.Cancel, keyStyle))))

	dialog := dialogBorderStyle.Width(dialogWidth).Render(strings.Join(lines, "\n"))
	return lipgloss.Place(innerWidth, innerHeight, lipgloss.Center, lipgloss.Center, dialog)
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
		Border(m.getBorderStyle()).
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

	content.WriteString(hintStyle.Render(fmt.Sprintf("Press %s, %s, or %s to dismiss",
		renderKeyHint("esc", m.keys.Cancel, hintStyle),
		renderKeyHint("enter", m.keys.Confirm, hintStyle),
		renderKeyHint("quit", m.keys.Quit, hintStyle))))

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
		if displayCount > len(m.pendingUpdates) {
			displayCount = len(m.pendingUpdates)
		}

		m.maxUpdateScroll = len(m.pendingUpdates) - displayCount
		if m.maxUpdateScroll < 0 {
			m.maxUpdateScroll = 0
		}

		// Clamp current offset
		if m.updateScrollOffset > m.maxUpdateScroll {
			m.updateScrollOffset = m.maxUpdateScroll
		}

		var listBuilder strings.Builder
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
			listBuilder.WriteString(fmt.Sprintf("    • %s %s %s\n", sourceBadge, lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Render(pkg.Name), lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(pkg.Version)))
		}

		listStr := listBuilder.String()
		if len(m.pendingUpdates) > displayCount {
			// Add scrollbar (Top-down)
			scrollbar := renderScrollbar(len(m.pendingUpdates), m.updateScrollOffset, displayCount, activeColor, false)
			// Join list and scrollbar
			listStr = lipgloss.JoinHorizontal(lipgloss.Top, 
				lipgloss.NewStyle().Width(innerWidth - 6).Render(listStr),
				lipgloss.NewStyle().MarginLeft(1).Render(scrollbar))
		}
		content.WriteString(listStr)
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

		btnUpdate := buttonStyle.Render(renderKeyHint("update", m.keys.Confirm, buttonStyle))
		btnSelective := buttonStyleRed.Render(renderKeyHint("select", m.keys.Selective, buttonStyleRed))

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
