package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func getDashboardData() tea.Cmd {
	return func() tea.Msg {
		var data DashboardData
		var errs []error

		cmd := exec.Command("paru", "-Q")
		var out bytes.Buffer
		cmd.Stdout = &out
		if err := cmd.Run(); err == nil {
			data.TotalPackages = countLines(out.String())
		} else {
			errs = append(errs, fmt.Errorf("total packages: %w", err))
		}

		out.Reset()
		cmd = exec.Command("paru", "-Qe")
		cmd.Stdout = &out
		if err := cmd.Run(); err == nil {
			data.ExplicitlyInstalled = countLines(out.String())
		} else {
			errs = append(errs, fmt.Errorf("explicitly installed: %w", err))
		}

		out.Reset()
		cmd = exec.Command("paru", "-Qm")
		cmd.Stdout = &out
		if err := cmd.Run(); err == nil {
			data.ForeignPackages = countLines(out.String())
		} else {
			errs = append(errs, fmt.Errorf("foreign packages: %w", err))
		}

		out.Reset()
		cmd = exec.Command("paru", "-Qdt")
		cmd.Stdout = &out
		if err := cmd.Run(); err == nil {
			data.Orphans = countLines(out.String())
		} else {
			errs = append(errs, fmt.Errorf("orphans: %w", err))
		}

		out.Reset()
		cmd = exec.Command("paru", "-Ps")
		cmd.Stdout = &out
		if err := cmd.Run(); err == nil {
			data.TotalSize, data.TotalSizeBytes, data.MissingFromAUR, data.TopPackages = parseParuStats(out.String())
		} else {
			errs = append(errs, fmt.Errorf("package statistics: %w", err))
		}

		pacmanCachePath := "/var/cache/pacman/pkg"
		pacmanCacheSize := calculateDirSize(pacmanCachePath)

		homeDir, _ := os.UserHomeDir()
		paruCachePath := filepath.Join(homeDir, ".cache", "paru")
		paruCacheSize := calculateDirSize(paruCachePath)

		data.PacmanCachePath = pacmanCachePath
		data.PacmanCacheSizeBytes = pacmanCacheSize
		data.PacmanCacheSize = formatBytes(pacmanCacheSize)
		data.ParuCachePath = paruCachePath
		data.ParuCacheSizeBytes = paruCacheSize
		data.ParuCacheSize = formatBytes(paruCacheSize)

		totalCacheBytes := pacmanCacheSize + paruCacheSize

		data.CleanerSizeBytes = totalCacheBytes
		data.CleanerSize = formatBytes(totalCacheBytes)

		if len(errs) > 0 {
			return dashboardMsg{data: data, err: fmt.Errorf("errors loading dashboard: %v", errs)}
		}
		return dashboardMsg{data: data}
	}
}

// parseParuStats extracts total installed size, missing AUR package count,
// and top 10 biggest packages from paru -Ps output.
func parseParuStats(output string) (totalSize string, totalSizeBytes int64, missingAUR int, topPackages []PackageSize) {
	lines := strings.Split(output, "\n")
	inTopPackages := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.Contains(line, "biggest packages") {
			inTopPackages = true
			continue
		}

		if inTopPackages && (strings.HasPrefix(line, "===") || line == "") {
			inTopPackages = false
			continue
		}

		if inTopPackages {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				topPackages = append(topPackages, PackageSize{
					Name: strings.TrimSpace(parts[0]),
					Size: strings.TrimSpace(parts[1]),
				})
			}
			continue
		}

		if strings.Contains(line, "Total Size occupied") || strings.Contains(line, "Total Installed Size") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				totalSize = strings.TrimSpace(parts[1])
				totalSizeBytes = parseSizeToBytes(totalSize)
			}
		}
		if strings.Contains(line, "Missing") && strings.Contains(line, "AUR") {

			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				_, _ = fmt.Sscanf(strings.TrimSpace(parts[1]), "%d", &missingAUR)
			}
		}
	}
	return
}

// cleanCache runs paru -Sc to clean package cache
func cleanCache() tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("paru", "-Sc", "--noconfirm")
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out
		err := cmd.Run()
		return cleanCacheMsg{output: out.String(), err: err}
	}
}

// removeOrphans runs paru -Rns to remove orphan packages
func removeOrphans() tea.Cmd {
	return func() tea.Msg {

		cmd := exec.Command("paru", "-Qdtq")
		var orphanList bytes.Buffer
		cmd.Stdout = &orphanList
		if err := cmd.Run(); err != nil || orphanList.Len() == 0 {
			return removeOrphansMsg{output: "No orphans to remove", err: nil}
		}

		orphans := strings.Fields(orphanList.String())
		validOrphans, _ := sanitizePackageNames(orphans)
		if len(validOrphans) == 0 {
			return removeOrphansMsg{output: "No valid orphans to remove", err: nil}
		}

		args := append([]string{"-Rns", "--noconfirm"}, validOrphans...)
		cmd = exec.Command("paru", args...)
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out
		err := cmd.Run()
		return removeOrphansMsg{output: out.String(), err: err}
	}
}

func (m model) renderDashboard(helpText string, contentWidth, contentHeight int) string {

	activeColor := modeColors[m.mode]
	if activeColor == "" {
		activeColor = defaultBorderColor
	}
	borderStyle := baseBorderStyle.BorderForeground(activeColor)

	helpWidth := lipgloss.Width(helpText)
	padding := contentWidth - helpWidth
	if padding < 0 {
		padding = 0
	}
	footerLine := strings.Repeat(" ", padding) + helpText

	if m.loading {
		loadingBox := borderStyle.
			Width(contentWidth).
			Height(contentHeight - 1).
			Render(lipgloss.Place(contentWidth-2, contentHeight-3, lipgloss.Center, lipgloss.Center, "Loading system statistics..."))
		return lipgloss.JoinVertical(lipgloss.Left, loadingBox, footerLine)
	}

	var dashboard strings.Builder

	greenColor := lipgloss.Color("42")
	redColor := lipgloss.Color("196")
	yellowColor := lipgloss.Color("214")
	orangeColor := lipgloss.Color("208")
	cyanColor := lipgloss.Color("51")
	dimColor := lipgloss.Color("240")

	boxTitleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("229"))

	shortcutStyle := lipgloss.NewStyle().Foreground(dimColor)

	countsLines := []string{
		fmt.Sprintf("  %s Total    │ %s",
			shortcutStyle.Render("[t]"),
			lipgloss.NewStyle().Bold(true).Foreground(cyanColor).Render(fmt.Sprintf("%d", m.dashboard.TotalPackages))),
		fmt.Sprintf("  %s Explicit │ %s",
			shortcutStyle.Render("[e]"),
			lipgloss.NewStyle().Bold(true).Foreground(greenColor).Render(fmt.Sprintf("%d", m.dashboard.ExplicitlyInstalled))),
		fmt.Sprintf("  %s Foreign  │ %s",
			shortcutStyle.Render("[f]"),
			lipgloss.NewStyle().Bold(true).Foreground(yellowColor).Render(fmt.Sprintf("%d", m.dashboard.ForeignPackages))),
	}

	orphanStyle := lipgloss.NewStyle().Bold(true).Foreground(greenColor)
	if m.dashboard.Orphans > 0 {
		orphanStyle = lipgloss.NewStyle().Bold(true).Foreground(redColor)
	}
	orphanLine := fmt.Sprintf("  %s Orphans  │ %s",
		shortcutStyle.Render("[o]"),
		orphanStyle.Render(fmt.Sprintf("%d", m.dashboard.Orphans)))
	if m.dashboard.Orphans > 0 {
		orphanLine += shortcutStyle.Render(" [R]remove")
	}
	countsLines = append(countsLines, orphanLine)

	cacheStyle := lipgloss.NewStyle().Bold(true).Foreground(greenColor)
	const tenGiB = 10 * 1024 * 1024 * 1024
	if m.dashboard.CleanerSizeBytes > tenGiB {
		cacheStyle = lipgloss.NewStyle().Bold(true).Foreground(orangeColor)
	}
	if m.dashboard.CleanerSizeBytes > tenGiB*2 {
		cacheStyle = lipgloss.NewStyle().Bold(true).Foreground(redColor)
	}

	missingStyle := lipgloss.NewStyle().Bold(true).Foreground(greenColor)
	if m.dashboard.MissingFromAUR > 0 {
		missingStyle = lipgloss.NewStyle().Bold(true).Foreground(redColor)
	}

	storageLines := []string{
		fmt.Sprintf("  System  │ %s",
			lipgloss.NewStyle().Bold(true).Foreground(cyanColor).Render(m.dashboard.TotalSize)),
		fmt.Sprintf("  Cache   │ %s %s",
			cacheStyle.Render(m.dashboard.CleanerSize),
			shortcutStyle.Render("[c]lean")),
		fmt.Sprintf("  Missing │ %s",
			missingStyle.Render(fmt.Sprintf("%d AUR", m.dashboard.MissingFromAUR))),
		"",
	}

	borderColor := lipgloss.NewStyle().Foreground(activeColor)

	renderBox := func(title string, lines []string, width int) string {
		var b strings.Builder

		innerWidth := width - 4
		if innerWidth < 20 {
			innerWidth = 20
		}

		titleLen := lipgloss.Width(title)
		topLeft := borderColor.Render("╭─")
		topRight := borderColor.Render("─╮")
		topPadding := innerWidth - titleLen
		if topPadding < 0 {
			topPadding = 0
		}
		b.WriteString(topLeft + title + borderColor.Render(strings.Repeat("─", topPadding)) + topRight + "\n")

		leftBorder := borderColor.Render("│ ")
		rightBorder := borderColor.Render(" │")
		for _, line := range lines {

			lineWidth := lipgloss.Width(line)
			padding := innerWidth - lineWidth
			if padding < 0 {
				padding = 0
			}
			b.WriteString(leftBorder + line + strings.Repeat(" ", padding) + rightBorder + "\n")
		}

		b.WriteString(borderColor.Render("╰" + strings.Repeat("─", innerWidth+2) + "╯"))

		return b.String()
	}

	boxWidth := (contentWidth - 6) / 2
	if boxWidth < 30 {
		boxWidth = 30
	}

	countsBox := renderBox(boxTitleStyle.Render(" 📦 Package Counts "), countsLines, boxWidth)
	storageBox := renderBox(boxTitleStyle.Render(" 💾 Storage "), storageLines, boxWidth)

	countsBoxLines := strings.Split(countsBox, "\n")
	storageBoxLines := strings.Split(storageBox, "\n")

	maxLines := len(countsBoxLines)
	if len(storageBoxLines) > maxLines {
		maxLines = len(storageBoxLines)
	}
	for len(countsBoxLines) < maxLines {
		countsBoxLines = append(countsBoxLines, strings.Repeat(" ", boxWidth))
	}
	for len(storageBoxLines) < maxLines {
		storageBoxLines = append(storageBoxLines, strings.Repeat(" ", boxWidth))
	}

	for i := 0; i < maxLines; i++ {
		dashboard.WriteString(countsBoxLines[i] + "  " + storageBoxLines[i] + "\n")
	}
	dashboard.WriteString("\n")

	// ═══════════════════════════════════════════════════════
	// Bar Layout Constants - ensures all bars align perfectly
	// ═══════════════════════════════════════════════════════
	const barLeftMargin = 2     // Spaces before label
	const barLabelWidth = 8     // Fixed width for labels (e.g., "System", "Cache")
	const barSeparator = "│"    // Separator between label and bar
	const barSuffixReserve = 30 // Reserve space for suffix text (e.g., "1234/5678 (100% explicit)")
	barStartCol := barLeftMargin + barLabelWidth + len(barSeparator)
	availableBarWidth := contentWidth - barStartCol - barSuffixReserve
	if availableBarWidth < 20 {
		availableBarWidth = 20
	}

	renderBarLine := func(label string, bar string, suffix string) string {
		paddedLabel := fmt.Sprintf("%*s%-*s%s", barLeftMargin, "", barLabelWidth, label, barSeparator)
		return paddedLabel + bar + " " + suffix
	}

	dependencies := m.dashboard.TotalPackages - m.dashboard.ExplicitlyInstalled
	explicitRatio := float64(m.dashboard.ExplicitlyInstalled) / float64(m.dashboard.TotalPackages)
	if m.dashboard.TotalPackages == 0 {
		explicitRatio = 0
	}

	filledWidth := int(explicitRatio * float64(availableBarWidth))
	if filledWidth > availableBarWidth {
		filledWidth = availableBarWidth
	}

	filledBar := lipgloss.NewStyle().Background(greenColor).Foreground(lipgloss.Color("0")).
		Render(strings.Repeat(" ", filledWidth))
	emptyBar := lipgloss.NewStyle().Background(lipgloss.Color("238")).
		Render(strings.Repeat(" ", availableBarWidth-filledWidth))

	ratioTitle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229")).
		Render("📊 Explicit vs Dependencies")
	ratioSuffix := fmt.Sprintf("%d/%d (%.0f%% explicit)", m.dashboard.ExplicitlyInstalled, dependencies, explicitRatio*100)
	ratioBar := renderBarLine("", filledBar+emptyBar, ratioSuffix)

	dashboard.WriteString(ratioTitle + "\n")
	dashboard.WriteString(ratioBar + "\n\n")

	chartTitle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229")).
		Render("📈 Size Comparison")
	dashboard.WriteString(chartTitle + "\n")

	maxSize := m.dashboard.TotalSizeBytes
	if m.dashboard.CleanerSizeBytes > maxSize {
		maxSize = m.dashboard.CleanerSizeBytes
	}
	if maxSize == 0 {
		maxSize = 1
	}

	systemBarWidth := int(float64(m.dashboard.TotalSizeBytes) / float64(maxSize) * float64(availableBarWidth))
	cacheBarWidth := int(float64(m.dashboard.CleanerSizeBytes) / float64(maxSize) * float64(availableBarWidth))
	if systemBarWidth < 1 {
		systemBarWidth = 1
	}
	if cacheBarWidth < 1 && m.dashboard.CleanerSizeBytes > 0 {
		cacheBarWidth = 1
	}

	systemBar := lipgloss.NewStyle().Background(cyanColor).Render(strings.Repeat(" ", systemBarWidth))
	cacheBar := lipgloss.NewStyle().Background(orangeColor).Render(strings.Repeat(" ", cacheBarWidth))

	dashboard.WriteString(renderBarLine("System", systemBar, m.dashboard.TotalSize) + "\n")
	dashboard.WriteString(renderBarLine("Cache", cacheBar, m.dashboard.CleanerSize) + "\n\n")

	if len(m.dashboard.TopPackages) > 0 {
		topTitle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229")).
			Render("🏆 Top 10 Packages by Size")
		dashboard.WriteString(topTitle + "\n")

		for i, pkg := range m.dashboard.TopPackages {
			rankStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
			nameStyle := lipgloss.NewStyle().Foreground(cyanColor)
			sizeStyle := lipgloss.NewStyle().Foreground(yellowColor)

			dashboard.WriteString(fmt.Sprintf("  %s %s %s\n",
				rankStyle.Render(fmt.Sprintf("%2d.", i+1)),
				nameStyle.Render(fmt.Sprintf("%-30s", pkg.Name)),
				sizeStyle.Render(pkg.Size)))
		}
		dashboard.WriteString("\n")
	}

	dashContent := lipgloss.NewStyle().
		Width(contentWidth-2).
		Height(contentHeight-3).
		Padding(0, 1).
		Render(dashboard.String())

	dashPanel := borderStyle.
		Width(contentWidth).
		Height(contentHeight - 1).
		Render(dashContent)

	return lipgloss.JoinVertical(lipgloss.Left, dashPanel, footerLine)
}
