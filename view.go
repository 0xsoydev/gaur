package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func sourceStyle(source string) lipgloss.Style {
	if color, ok := sourceColors[source]; ok {
		return lipgloss.NewStyle().Foreground(color)
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	}

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
		var pkgList []Package
		if effectiveMode == modeInstall {
			pkgList = m.filtered
		} else if effectiveMode == modeUninstall {
			pkgList = m.filteredInstalled
		} else if effectiveMode == modeUpdateSelective {
			pkgList = m.filtered
		}

		repoSummary := m.renderRepoSummary(pkgList)
		if repoSummary != "" {
			repoSummary = " " + repoSummary + " "
		}

		helpWidth := lipgloss.Width(helpText)
		summaryWidth := lipgloss.Width(repoSummary)
		padding := innerWidth - helpWidth - summaryWidth
		if padding < 0 {
			padding = 0
		}
		footer := repoSummary + strings.Repeat(" ", padding) + helpText
		if lipgloss.Width(footer) > innerWidth {
			footer = truncateWithAnsi(footer, innerWidth)
		}
		content = m.renderPackageListLayout(innerWidth, innerHeight, activeColor, "", footer)
	}

	// If settings are active, overlay them on top of the rendered content
	if m.mode == modeSettings {
		settingsOverlay := m.renderSettings(innerWidth, innerHeight)
		content = m.overlaySettings(content, settingsOverlay, innerWidth, innerHeight)
	}

	return content
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
	if len(baseLines) < height {
		for len(baseLines) < height {
			baseLines = append(baseLines, strings.Repeat(" ", width))
		}
	}

	startY := (height - overlayHeight) / 2
	startCol := (width - overlayWidth) / 2

	result := make([]string, len(baseLines))
	copy(result, baseLines)

	for y := 0; y < overlayHeight; y++ {
		targetY := startY + y
		if targetY >= 0 && targetY < len(result) {
			bgLine := result[targetY]
			
			// Ensure bgLine is at least width chars wide (considering ANSI)
			bgWidth := lipgloss.Width(bgLine)
			if bgWidth < width {
				bgLine += strings.Repeat(" ", width - bgWidth)
			}
			
			// Reconstruct the line using precise slicing
			left := truncateWithAnsi(bgLine, startCol)
			// Ensure left part is exactly startCol wide by padding if needed
			leftWidth := lipgloss.Width(left)
			if leftWidth < startCol {
				left += strings.Repeat(" ", startCol - leftWidth)
			}
			
			right := substringAnsi(bgLine, startCol + overlayWidth)
			// Ensure right part doesn't exceed the remaining width
			right = truncateWithAnsi(right, width - (startCol + overlayWidth))
			
			// Composite the line: [Base Left] [Overlay Content] [Base Right]
			result[targetY] = left + overlayLines[y] + right
		}
	}

	return SafeJoinVertical(width, height, "", []string{strings.Join(result, "\n")}, "")
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

	// Ensure we don't exceed terminal dimensions
	if overlayWidth > innerWidth {
		overlayWidth = innerWidth
	}

	// In all terminals, we must leave at least one line for the footer
	maxOverlayHeight := innerHeight - 1
	if overlayHeight > maxOverlayHeight {
		overlayHeight = maxOverlayHeight
	}
	// Sanity check for extremely small terminals
	if overlayHeight < 5 && innerHeight >= 6 {
		overlayHeight = 5
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

	// Subtract space for the actual warning overlay height AND the JoinVertical separator
	overlayInnerHeight := overlayHeight - warningHeight - 1
	if overlayInnerHeight < 5 {
		overlayInnerHeight = 5
	}

	paneContent := m.renderVerticalSplitLayout(overlayWidth, overlayInnerHeight, activeColor)
	
	// Composite panels and warning
	// Ensure warning overlay has a fixed height that we accounted for
	warningBoxWrapped := lipgloss.NewStyle().
		Height(warningHeight).
		Render(strings.TrimSuffix(warningOverlay, "\n"))
		
	paneContent = lipgloss.JoinVertical(lipgloss.Left, strings.TrimSuffix(paneContent, "\n"), strings.TrimSuffix(warningBoxWrapped, "\n"))

	// Enforce strict rectangle bounds - this ensures we exactly match overlayHeight
	paneContent = lipgloss.Place(overlayWidth, overlayHeight, lipgloss.Center, lipgloss.Center, paneContent)

	bg := strings.TrimSuffix(m.renderSimpleUpdateView(helpText, innerWidth, innerHeight, activeColor), "\n")
	bgLines := strings.Split(bg, "\n")
	paneLines := strings.Split(paneContent, "\n")

	// Ensure bgLines has exactly innerHeight lines
	if len(bgLines) > innerHeight {
		bgLines = bgLines[:innerHeight]
	} else if len(bgLines) < innerHeight {
		for len(bgLines) < innerHeight {
			bgLines = append(bgLines, strings.Repeat(" ", innerWidth))
		}
	}

	var output strings.Builder
	for i, bgLine := range bgLines {
		if i >= paddingY && i < paddingY+overlayHeight {
			paneLineIdx := i - paddingY
			if paneLineIdx < len(paneLines) {
				paneLine := paneLines[paneLineIdx]

				// Ensure paneLine is exactly overlayWidth wide
				pw := lipgloss.Width(paneLine)
				if pw < overlayWidth {
					paneLine += strings.Repeat(" ", overlayWidth-pw)
				} else if pw > overlayWidth {
					paneLine = truncateWithAnsi(paneLine, overlayWidth)
				}

				// Detect if this paneLine is a horizontal border line to avoid "border leakage"
				// which occurs when background content (like bullets) is placed next to many border chars.
				isBorderLine := strings.Count(paneLine, "─") > overlayWidth/2
				
				// Get the background left and right parts
				bgWidth := lipgloss.Width(bgLine)
				
				leftStr := ""
				if paddingX > 0 {
					leftStr = truncateWithAnsi(bgLine, paddingX)
					// If it's a border line, we want to clear any background content
					if isBorderLine {
						leftStr = strings.Repeat(" ", paddingX)
					} else {
						lw := lipgloss.Width(leftStr)
						if lw < paddingX {
							leftStr += strings.Repeat(" ", paddingX-lw)
						}
					}
				}

				rightStr := ""
				rightStart := paddingX + overlayWidth
				if rightStart < bgWidth {
					rightStr = substringAnsi(bgLine, rightStart)
					// If it's a border line, clear background
					if isBorderLine {
						rightStr = strings.Repeat(" ", innerWidth-rightStart)
					} else {
						rw := lipgloss.Width(rightStr)
						if rightStart + rw < innerWidth {
							rightStr += strings.Repeat(" ", innerWidth - (rightStart + rw))
						}
					}
				} else if paddingX + overlayWidth < innerWidth {
					rightStr = strings.Repeat(" ", innerWidth - (paddingX + overlayWidth))
				}

				line := leftStr + paneLine + rightStr
				output.WriteString(truncateWithAnsi(line, innerWidth))
				if i < len(bgLines)-1 {
					output.WriteString("\n")
				}
				continue
			}
		}
		output.WriteString(truncateWithAnsi(bgLine, innerWidth))
		if i < len(bgLines)-1 {
			output.WriteString("\n")
		}
	}

	return SafeJoinVertical(innerWidth, innerHeight, "", []string{output.String()}, "")
	}

// renderVerticalSplitLayout renders a side-by-side view (list on left, details on right)
func (m *model) renderVerticalSplitLayout(innerWidth, innerHeight int, activeColor lipgloss.Color) string {
	borderStyle := baseBorderStyle.BorderForeground(activeColor)

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
	var pkgList []Package
	if m.mode == modeUpdateSelective {
		pkgList = m.filtered
	}

	// Calculate results height for the list
	// InnerHeight - 2 for borders - 2 for search input and separator - 1 for bottom padding
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
			
			// Truncate to fit listWidth-6 (accounting for scrollbar space)
			if lipgloss.Width(line) > listWidth-6 {
				line = truncateWithAnsi(line, listWidth-9) + "..."
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
				lipgloss.NewStyle().Width(listWidth-6).Render(resultsStr),
				lipgloss.NewStyle().MarginLeft(1).Render(scrollbar))
		}
	}

	resultsContainer := lipgloss.NewStyle().
		Height(resultsHeight).
		Width(listWidth - 4).
		Align(lipgloss.Left, lipgloss.Bottom).
		Render(strings.TrimSuffix(resultsStr, "\n"))

	listPanel := borderStyle.
		Width(listWidth - 2).
		Height(innerHeight - 2).
		Align(lipgloss.Left, lipgloss.Bottom).
		Render(strings.TrimSuffix(truncateHeight(lipgloss.JoinVertical(lipgloss.Left,
			resultsContainer,
			"", // spacing separator
			"", // bottom truncation gap
			strings.TrimSuffix(m.textInput.View(), "\n"),
		), innerHeight-2), "\n"))

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
	infoInnerWidth := detailsWidth - 6 // 2 chars padding on each side

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
		Width(detailsWidth - 2).
		Height(infoInnerHeight).
		Padding(0, 2).
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
					// Ensure left part is exactly startCol wide
					lw := lipgloss.Width(leftStr)
					if lw < startCol {
						leftStr += strings.Repeat(" ", startCol - lw)
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
		Render(strings.TrimSuffix(truncateHeight(detailsBox, innerHeight-2), "\n"))

	return lipgloss.JoinHorizontal(lipgloss.Top, strings.TrimSuffix(listPanel, "\n"), strings.TrimSuffix(detailsPanel, "\n"))
	}

// renderSelectionBox renders a box containing currently marked packages
func (m *model) renderSelectionBox(maxWidth int) string {
	if len(m.markedPackages) == 0 {
		return ""
	}

	var pkgNames []string
	for name := range m.markedPackages {
		pkgNames = append(pkgNames, name)
	}
	sort.Strings(pkgNames)

	panelWidth := maxWidth
	if panelWidth < 20 {
		panelWidth = 20
	}

	maxVisible := 8
	startIdx := m.selectionScrollOffset
	if m.selectionPanelFocused {
		if m.selectionPanelIndex >= startIdx+maxVisible {
			startIdx = m.selectionPanelIndex - maxVisible + 1
		} else if m.selectionPanelIndex < startIdx {
			startIdx = m.selectionPanelIndex
		}
	}
	m.selectionScrollOffset = startIdx

	endIdx := startIdx + maxVisible
	if endIdx > len(pkgNames) {
		endIdx = len(pkgNames)
	}

	panelStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("205")).
		Padding(0, 1)

	if m.selectionPanelFocused {
		panelStyle = panelStyle.BorderForeground(lipgloss.Color("214"))
	}

	titleText := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")).Render(fmt.Sprintf(" Selected (%d) ", len(pkgNames)))
	
	itemStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	selectedItemStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)

	var listBuilder strings.Builder
	nameMaxWidth := panelWidth - 6

	for i := startIdx; i < endIdx; i++ {
		name := pkgNames[i]
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
// renderRepoSummary creates a color-coded summary of packages by repository
func (m *model) renderRepoSummary(pkgList []Package) string {
	if len(pkgList) == 0 {
		return ""
	}

	counts := make(map[string]int)
	for _, p := range pkgList {
		counts[p.Source]++
	}

	// Ordered repos for consistent display
	standardRepos := []string{"core", "extra", "multilib", "aur"}
	var parts []string
	for _, r := range standardRepos {
		if c, ok := counts[r]; ok && c > 0 {
			style := sourceStyle(r)
			parts = append(parts, fmt.Sprintf("%d %s", c, style.Render(r)))
		}
	}

	// Add any "other" repos
	var others []string
	for r := range counts {
		isStandard := false
		for _, sr := range standardRepos {
			if r == sr {
				isStandard = true
				break
			}
		}
		if !isStandard {
			others = append(others, r)
		}
	}
	sort.Strings(others)
	for _, r := range others {
		style := sourceStyle(r)
		parts = append(parts, fmt.Sprintf("%d %s", counts[r], style.Render(r)))
	}

	return strings.Join(parts, ", ")
	}

func (m *model) renderPackageListLayout(innerWidth, innerHeight int, activeColor lipgloss.Color, header, footer string) string {
	borderStyle := baseBorderStyle.BorderForeground(activeColor)

	// Use manual height calculation to avoid trailing zero-width lines issues
	calcHeight := func(s string) int {
		if s == "" {
			return 0
		}
		lines := strings.Split(s, "\n")
		// Trust the string's internal line count, just trim the trailing newline from Render()
		// if lipgloss.Width(lines[len(lines)-1]) == 0 {
		// 	return len(lines) - 1
		// }
		return len(lines)
	}

	headerHeight := calcHeight(header)
	footerHeight := calcHeight(footer)

	availableHeight := innerHeight - headerHeight - footerHeight
	if availableHeight < 6 {
		availableHeight = 6
	}

	targetInfoPanelHeight := availableHeight / 2
	targetBottomPanelHeight := availableHeight - targetInfoPanelHeight

	infoInnerHeight := targetInfoPanelHeight - 2
	bottomInnerHeight := targetBottomPanelHeight - 2

	if bottomInnerHeight < 5 {
		bottomInnerHeight = 5
		targetBottomPanelHeight = bottomInnerHeight + 2
		targetInfoPanelHeight = availableHeight - targetBottomPanelHeight
		infoInnerHeight = targetInfoPanelHeight - 2
		if infoInnerHeight < 1 {
			infoInnerHeight = 1
		}
	}

	resultsHeight := bottomInnerHeight - 4
	if resultsHeight < 1 {
		resultsHeight = 1
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
	} else {
		if m.loadingInfo {
			infoContent = fmt.Sprintf("Loading details for %s...", m.infoForPackage)
		} else if m.packageInfo != "" {
			infoContent = m.packageInfo
		} else {
			infoContent = "Select a package to see details"
		}
	}

	// InnerWidth is the total terminal width
	// infoPanel Total Width = innerWidth
	// infoPanel Inner Width = innerWidth - 2
	// infoBox Total Width (including padding) = innerWidth - 4 (1 char margin on each side)
	// infoBox Content Width = innerWidth - 6
	contentWidth := innerWidth - 6
	if contentWidth < 10 {
		contentWidth = 10
	}

	// Render the text with a width limit first so Lipgloss wraps it
	wrappedText := lipgloss.NewStyle().Width(contentWidth).Render(infoContent)
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
		Width(innerWidth - 2).
		Padding(0, 2).
		Render(truncateHeight(infoContent, infoInnerHeight))

	infoPanel := borderStyle.
		Width(innerWidth - 2).
		Height(max(0, targetInfoPanelHeight-2)).
		Render(truncateHeight(infoBox, max(0, targetInfoPanelHeight-2)))

	inputLine := ""
	statusLine := ""
	
	// Determine what mode's input line to show
	displayMode := m.mode
	if m.mode == modeSettings {
		displayMode = m.previousMode
	}

	if displayMode == modeInstall || displayMode == modeUninstall || displayMode == modeUpdateSelective {
		inputLine = m.textInput.View()
		
		if m.searchStatus != "" {
			style := lipgloss.NewStyle().
				Foreground(lipgloss.Color("244")).
				Italic(true)
			
			if m.searchError {
				style = style.Foreground(lipgloss.Color("#FF5555"))
			}
			
			renderedStatus := style.Render(m.searchStatus)
			
			if m.searchingAUR {
				spinnerStr := m.spinner.View()
				sw := lipgloss.Width(spinnerStr)
				if sw >= 2 {
					statusLine = spinnerStr + renderedStatus
				} else {
					statusLine = spinnerStr + " " + renderedStatus
				}
			} else {
				statusLine = "  " + renderedStatus
			}
		}
	} else {
		inputLine = statusStyle.Render(m.statusMessage)
	}

	// Calculate resultsHeight precisely to fill available space
	// Overhead: separator(1) + inputLine(1)
	overhead := 2
	if statusLine != "" {
		overhead++
	}
	resultsHeight = bottomInnerHeight - overhead
	if resultsHeight < 1 {
		resultsHeight = 1
	}

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

			// Truncate to fit innerWidth-6 (accounting for scrollbar space)
			if lipgloss.Width(line) > innerWidth-6 {
				line = truncateWithAnsi(line, innerWidth-9) + "..."
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
				lipgloss.NewStyle().Width(innerWidth-6).Render(resultsStr),
				lipgloss.NewStyle().MarginLeft(1).Render(scrollbar))
		}
	}

	resultsBox := lipgloss.NewStyle().
		Width(innerWidth - 4).
		Height(resultsHeight).
		Align(lipgloss.Left, lipgloss.Bottom).
		Render(resultsStr)

	bottomParts := []string{
		resultsBox,
		lipgloss.NewStyle().Foreground(lipgloss.Color("236")).Render(strings.Repeat("─", innerWidth-4)),
		inputLine,
	}
	if statusLine != "" {
		bottomParts = append(bottomParts, statusLine)
	}
	bottomContent := strings.Join(bottomParts, "\n")

	bottomPanel := borderStyle.
		Width(innerWidth - 2).
		Height(max(0, targetBottomPanelHeight-2)).
		Align(lipgloss.Left, lipgloss.Bottom).
		Render(truncateHeight(bottomContent, max(0, targetBottomPanelHeight-2)))

	content := SafeJoinVertical(innerWidth, innerHeight, header, []string{infoPanel, bottomPanel}, footer)

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
				// Ensure left part is exactly startCol wide
				lw := lipgloss.Width(leftStr)
				if lw < startCol {
					leftStr += strings.Repeat(" ", startCol-lw)
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
	var title string
	var packages []Package
	var actionDesc string
	var simpleConfirm bool

	switch m.confirmType {
	case confirmInstall:
		title = "📦 Confirm Installation"
		actionDesc = "install"
		for name := range m.markedPackages {
			pkg := m.getPackageByName(name)
			if pkg != nil { packages = append(packages, *pkg) }
		}
	case confirmUninstall:
		title = "🗑 Confirm Removal"
		actionDesc = "uninstall"
		for name := range m.markedPackages {
			pkg := m.getPackageByName(name)
			if pkg != nil { packages = append(packages, *pkg) }
		}
	case confirmUpdate:
		title = "🔄 Confirm System Update"
		actionDesc = "update"
		packages = m.pendingUpdates
	case confirmSelectiveUpdate:
		title = "🔄 Confirm Selective Update"
		actionDesc = "update"
		for name := range m.markedPackages {
			pkg := m.getPackageByName(name)
			if pkg != nil { packages = append(packages, *pkg) }
		}
	case confirmRemoveOrphans:
		title = "🧹 Confirm Orphan Removal"
		actionDesc = "remove orphans"
		for _, name := range m.confirmPackages {
			packages = append(packages, Package{Name: name})
		}
	case confirmCleanKeep3, confirmCleanKeep1, confirmCleanNuke:
		title = "🧹 Confirm Cache Cleaning"
		actionDesc = "clean"
		simpleConfirm = true
	case confirmCleanUninstalled:
		title = "🧹 Confirm Orphaned Cache Clean"
		actionDesc = "clean orphaned packages from cache"
		if len(m.dashboard.UninstalledPacmanCache) > 0 {
			for _, p := range m.dashboard.UninstalledPacmanCache {
				packages = append(packages, Package{Name: p.Name, Size: p.Size})
			}
		}
		if len(m.dashboard.UninstalledParuCache) > 0 {
			for _, p := range m.dashboard.UninstalledParuCache {
				packages = append(packages, Package{Name: p.Name, Size: p.Size})
			}
		}
	case confirmCleanSelective:
		title = "🧹 Confirm Selective Clean"
		actionDesc = "clean selected packages from cache"
		for name := range m.markedPackages {
			packages = append(packages, Package{Name: name, Size: ""}) // Size filled later if available
		}
	}

	dialogWidth := innerWidth - 10
	if dialogWidth < 60 { dialogWidth = 60 }
	if dialogWidth > 90 { dialogWidth = 90 }

	activeBorderColor := activeColor
	switch m.confirmType {
	case confirmInstall:
		activeBorderColor = lipgloss.Color("39")
	case confirmUninstall:
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
		Align(lipgloss.Left)

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(activeBorderColor)
	packageNameStyle := lipgloss.NewStyle().Foreground(activeBorderColor).Bold(true)
	packageVersionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("246"))
	countStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
	promptStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252")).MarginTop(1)
	keyStyle := lipgloss.NewStyle().Foreground(activeBorderColor).Bold(true)
	scrollHintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	var dialogContent []string
	contentWidth := dialogWidth - 4

	// Title
	dialogContent = append(dialogContent, lipgloss.PlaceHorizontal(contentWidth, lipgloss.Center, titleStyle.Render(title)))
	dialogContent = append(dialogContent, "")

	// Warning/Description
	descStyle := lipgloss.NewStyle().Width(contentWidth).Align(lipgloss.Center)
	if simpleConfirm {
		if m.confirmType == confirmCleanNuke {
			warningStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true).Width(contentWidth).Align(lipgloss.Center)
			dialogContent = append(dialogContent, warningStyle.Render("WARNING: This will completely empty the package cache."))
			dialogContent = append(dialogContent, "")
		} else {
			if m.confirmType == confirmCleanKeep3 {
				dialogContent = append(dialogContent, descStyle.Render("This will remove all but the 3 most recent cached versions of packages."))
			} else {
				dialogContent = append(dialogContent, descStyle.Render("This will aggressively remove all but the currently installed cached versions."))
			}
			dialogContent = append(dialogContent, "")
		}

		if m.confirmType != confirmCleanNuke {
			labelStyle := lipgloss.NewStyle().Width(8).Foreground(lipgloss.Color("241"))
			dialogContent = append(dialogContent, packageNameStyle.Render("System Cache:"))
			dialogContent = append(dialogContent, fmt.Sprintf("  %s %s", labelStyle.Render("Path:"), scrollHintStyle.Render(m.dashboard.PacmanCachePath)))
			dialogContent = append(dialogContent, packageNameStyle.Render("User Cache:"))
			dialogContent = append(dialogContent, fmt.Sprintf("  %s %s", labelStyle.Render("Path:"), scrollHintStyle.Render(m.dashboard.ParuCachePath)))
			dialogContent = append(dialogContent, "")
		}

		breakdownHeaderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Bold(true).Width(contentWidth).Align(lipgloss.Center)
		dialogContent = append(dialogContent, breakdownHeaderStyle.Render("Breakdown:"))

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

		dialogContent = append(dialogContent, lipgloss.PlaceHorizontal(contentWidth, lipgloss.Center, fmt.Sprintf("%s %s", pacmanLabel, valStyle.Render(pacmanEstimate))))
		dialogContent = append(dialogContent, lipgloss.PlaceHorizontal(contentWidth, lipgloss.Center, fmt.Sprintf("%s %s", paruLabel, valStyle.Render(paruEstimate))))
		dialogContent = append(dialogContent, "")

		estStyle := lipgloss.NewStyle().Width(contentWidth).Align(lipgloss.Center)
		estimate := m.dashboard.CacheFreedEstimates[m.confirmType]
		if estimate == "" { estimate = "calculating..." }
		dialogContent = append(dialogContent, estStyle.Render(fmt.Sprintf("Estimated space to be freed: %s", lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214")).Render(estimate))))
	} else {
		// List-based Confirmations
		listTitleStyle := lipgloss.NewStyle().Width(contentWidth).Align(lipgloss.Center)
		if m.confirmType == confirmUpdate {
			dialogContent = append(dialogContent, listTitleStyle.Render(fmt.Sprintf("The following %s updates are available:", countStyle.Render(fmt.Sprintf("%d", len(packages))))))
		} else if m.confirmType == confirmCleanUninstalled {
			dialogContent = append(dialogContent, listTitleStyle.Render("This will remove cached packages that are no longer installed."))
		} else {
			dialogContent = append(dialogContent, listTitleStyle.Render(fmt.Sprintf("The following %s packages will be %sd:", countStyle.Render(fmt.Sprintf("%d", len(packages))), actionDesc)))
		}
		dialogContent = append(dialogContent, "")

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
			dialogContent = append(dialogContent, line)
		}

		if len(packages) > maxVisible {
			dialogContent = append(dialogContent, "")
			dialogContent = append(dialogContent, lipgloss.PlaceHorizontal(contentWidth, lipgloss.Center, scrollHintStyle.Render("  Use [↑/↓] or [j/k] to scroll")))
		}

		if m.confirmType == confirmCleanUninstalled {
			dialogContent = append(dialogContent, "")
			breakdownHeaderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Bold(true).Width(contentWidth).Align(lipgloss.Center)
			dialogContent = append(dialogContent, breakdownHeaderStyle.Render("Breakdown:"))
			pacmanEst := m.dashboard.CacheFreedPacman[m.confirmType]
			paruEst := m.dashboard.CacheFreedParu[m.confirmType]
			if pacmanEst == "" { pacmanEst = "calculating..." }
			if paruEst == "" { paruEst = "calculating..." }

			helperLabel := m.config.Commands.AurHelper + ":"
			if len(helperLabel) < 7 {
				helperLabel += strings.Repeat(" ", 7-len(helperLabel))
			}
			dialogContent = append(dialogContent, lipgloss.PlaceHorizontal(contentWidth, lipgloss.Center, fmt.Sprintf("%s %s", sourceStyle("core").Render("  pacman:"), lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Render(pacmanEst))))
			dialogContent = append(dialogContent, lipgloss.PlaceHorizontal(contentWidth, lipgloss.Center, fmt.Sprintf("%s %s", sourceStyle("aur").Render("  "+helperLabel), lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Render(paruEst))))
		}

		if m.confirmType == confirmCleanUninstalled || m.confirmType == confirmCleanSelective {
			dialogContent = append(dialogContent, "")
			est := m.dashboard.CacheFreedEstimates[m.confirmType]
			if m.confirmType == confirmCleanSelective { est = formatBytes(m.cacheToFree) }
			if est == "" { est = "calculating..." }
			dialogContent = append(dialogContent, lipgloss.PlaceHorizontal(contentWidth, lipgloss.Center, fmt.Sprintf("Estimated space to be freed: %s", lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214")).Render(est))))
		}
	}

	dialogContent = append(dialogContent, "")
	dialogContent = append(dialogContent, lipgloss.PlaceHorizontal(contentWidth, lipgloss.Center, promptStyle.Render(fmt.Sprintf("Proceed? %s  %s", 
		renderKeyHint("yes", m.keys.Confirm, keyStyle),
		renderKeyHint("no", m.keys.Cancel, keyStyle)))))

	dialog := dialogBorderStyle.Width(dialogWidth).Render(strings.Join(dialogContent, "\n"))

	return SafeJoinVertical(innerWidth, innerHeight, "", []string{lipgloss.Place(innerWidth, innerHeight, lipgloss.Center, lipgloss.Center, dialog)}, "")
	}

// renderErrorOverlay renders a centered error overlay dialog
func (m *model) renderErrorOverlay(innerWidth, innerHeight int) string {
	dialogWidth := innerWidth - 20
	if dialogWidth < 50 {
		dialogWidth = 50
	}
	if dialogWidth > 100 {
		dialogWidth = 100
	}

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Width(dialogWidth - 4).
		Align(lipgloss.Center)

	hintStyle := lipgloss.NewStyle().
		Width(dialogWidth - 4).
		Align(lipgloss.Center)

	dialogBorderStyle := lipgloss.NewStyle().
		Border(m.getBorderStyle()).
		BorderForeground(lipgloss.Color("196")).
		Align(lipgloss.Center)

	// Build content pieces
	title := titleStyle.Render("⚠  " + m.errorTitle + "  ⚠")
	
	// Ensure each line of the error message is individually centered
	// We wrap the text manually then center each resulting line
	msgWidth := dialogWidth - 4
	wrappedMessage := lipgloss.NewStyle().Width(msgWidth).Render(m.errorMessage)
	var messageLines []string
	for _, line := range strings.Split(wrappedMessage, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			messageLines = append(messageLines, lipgloss.NewStyle().Width(msgWidth).Align(lipgloss.Center).Render(trimmed))
		}
	}
	message := lipgloss.JoinVertical(lipgloss.Center, messageLines...)
	
	var details string
	if m.errorDetails != "" {
		wrappedDetails := lipgloss.NewStyle().Width(msgWidth).Render(m.errorDetails)
		var detailsLines []string
		for _, line := range strings.Split(wrappedDetails, "\n") {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" {
				detailsLines = append(detailsLines, lipgloss.NewStyle().Width(msgWidth).Align(lipgloss.Center).Render(trimmed))
			}
		}
		details = "\n" + lipgloss.JoinVertical(lipgloss.Center, detailsLines...)
	}

	hint := "\n" + hintStyle.Render(fmt.Sprintf("Press %s, %s, or %s to dismiss",
		renderKeyHint("esc", m.keys.Cancel, hintStyle),
		renderKeyHint("enter", m.keys.Confirm, hintStyle),
		renderKeyHint("quit", m.keys.Quit, hintStyle)))

	dialogContent := lipgloss.JoinVertical(lipgloss.Center, title, "", message, details, hint)
	dialog := dialogBorderStyle.Width(dialogWidth).Render(dialogContent)

	return SafeJoinVertical(innerWidth, innerHeight, "", []string{lipgloss.Place(innerWidth, innerHeight, lipgloss.Center, lipgloss.Center, dialog)}, "")
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

	footerHeight := 0
	if footerLine != "" {
		footerHeight = 1
	}
	
	// The panel must fit in the remaining height
	panelHeight := innerHeight - footerHeight
	if panelHeight < 5 {
		panelHeight = 5
	}

	var content strings.Builder
	innerContentHeight := 0

	if m.loading || m.pendingUpdates == nil {
		content.WriteString("\n  Checking for updates...")
		innerContentHeight = 2
	} else if len(m.pendingUpdates) == 0 {
		content.WriteString("\n  System is up to date!")
		innerContentHeight = 2
	} else {
		countStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
		content.WriteString(fmt.Sprintf("  The following %s system updates are available:\n\n", countStyle.Render(fmt.Sprintf("%d", len(m.pendingUpdates)))))
		innerContentHeight += 2

		// Calculate available lines for the list
		// panelHeight is TOTAL height of the panel including borders
		// Inner height is panelHeight - 2
		// Buttons take 1 line
		// A blank gap is needed above the buttons to truncate the list by 1 line
		// Header takes 2 lines ("The following... \n\n")
		// availableLinesForList = (innerHeightOfPanel) - buttons - gap - header
		availableLinesForList := (panelHeight - 2) - 1 - 1 - 2
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
			
			line := fmt.Sprintf("    • %s %s %s", sourceBadge, lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Render(pkg.Name), lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(pkg.Version))
			// Truncate to fit innerWidth-8 (accounting for scrollbar space)
			if lipgloss.Width(line) > innerWidth - 8 {
				line = truncateWithAnsi(line, innerWidth - 11) + "..."
			}
			listBuilder.WriteString(line + "\n")
		}

		listStr := strings.TrimSuffix(listBuilder.String(), "\n")
		if len(m.pendingUpdates) > displayCount {
			// Add scrollbar (Top-down)
			scrollbar := renderScrollbar(len(m.pendingUpdates), m.updateScrollOffset, displayCount, activeColor, false)
			// Join list and scrollbar
			listStr = lipgloss.JoinHorizontal(lipgloss.Top, 
				lipgloss.NewStyle().Width(innerWidth - 8).Render(listStr),
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
		
		// Total inner width is innerWidth - 2 (borders) - 2 (padding) = innerWidth - 4
		buttonsContent = lipgloss.PlaceHorizontal(innerWidth-4, lipgloss.Right, buttons)
	}

	// Total available inner height is panelHeight - 2
	innerHeightOfPanel := panelHeight - 2
	
	// We use Height() on the list container to push buttons to the bottom
	// -1 for buttons, -1 for truncation gap
	listHeight := innerHeightOfPanel - 1 - 1
	if listHeight < 1 { listHeight = 1 }

	innerPanelContent := lipgloss.JoinVertical(lipgloss.Left,
		truncateHeight(listContent, listHeight),
		"", // bottom truncation gap
		buttonsContent,
	)

	mainPanel := borderStyle.
		Width(innerWidth - 2).
		Height(max(0, panelHeight-2)).
		Padding(0, 1).
		Align(lipgloss.Left, lipgloss.Bottom).
		Render(truncateHeight(innerPanelContent, max(0, panelHeight-2)))

	return SafeJoinVertical(innerWidth, innerHeight, "", []string{mainPanel}, footerLine)
	}
