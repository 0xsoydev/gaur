package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

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

		// Repository distribution
		data.RepoDistribution = make(map[string]int)
		out.Reset()
		cmd = exec.Command("pacman", "-Sl")
		cmd.Stdout = &out
		if err := cmd.Run(); err == nil {
			lines := strings.Split(strings.TrimSpace(out.String()), "\n")
			for _, line := range lines {
				if strings.Contains(line, "[installed]") {
					parts := strings.Fields(line)
					if len(parts) > 0 {
						data.RepoDistribution[parts[0]]++
					}
				}
			}
		}

		out.Reset()
		cmd = exec.Command("paru", "-Ps")
		cmd.Stdout = &out
		if err := cmd.Run(); err == nil {
			data.TotalSize, data.TotalSizeBytes, data.MissingFromAUR, data.TopPackages = parseParuStats(out.String())
		} else {
			errs = append(errs, fmt.Errorf("package statistics: %w", err))
		}

		// Recently installed (last 5 unique)
		// Format in log: [2024-03-12T10:00:00+0000] [ALPM] installed pkgname (version)
		cmd = exec.Command("grep", " installed ", "/var/log/pacman.log")
		var logOut bytes.Buffer
		cmd.Stdout = &logOut
		if err := cmd.Run(); err == nil {
			lines := strings.Split(strings.TrimSpace(logOut.String()), "\n")
			seen := make(map[string]bool)
			for i := len(lines) - 1; i >= 0 && len(data.RecentlyInstalled) < 5; i-- {
				parts := strings.Fields(lines[i])
				if len(parts) >= 4 {
					pkg := parts[3]
					if !seen[pkg] {
						// Extract date/time from [2024-03-12T10:00:00+0000]
						ts := parts[0]
						if len(ts) > 17 {
							ts = ts[1:11] + " " + ts[12:17] // "2024-03-12 10:00"
						}
						data.RecentlyInstalled = append(data.RecentlyInstalled, RecentPackage{
							Name:      pkg,
							Timestamp: ts,
						})
						seen[pkg] = true
					}
				}
			}
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

		// Disk usage info
		var stat syscall.Statfs_t
		if err := syscall.Statfs("/", &stat); err == nil {
			total := stat.Blocks * uint64(stat.Bsize)
			free := stat.Bfree * uint64(stat.Bsize)
			used := total - free
			data.DiskTotal = formatBytes(int64(total))
			data.DiskFree = formatBytes(int64(free))
			data.DiskUsed = formatBytes(int64(used))
			if total > 0 {
				data.DiskUsedPercent = float64(used) / float64(total)
			}
		}

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

		if inTopPackages && line == "" {
			inTopPackages = false
			continue
		}

		if inTopPackages && strings.HasPrefix(line, "===") {
			continue
		}

		if inTopPackages {
			if len(topPackages) >= 10 {
				inTopPackages = false
				continue
			}
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				name := strings.TrimSpace(parts[0])
				size := strings.TrimSpace(parts[1])
				if name != "" {
					topPackages = append(topPackages, PackageSize{
						Name: name,
						Size: size,
					})
				}
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

// renderCenteredLabels returns a string with labels centered under segments of given widths
func renderCenteredLabels(widths []int, labels []string, colors []lipgloss.Color) string {
	var result strings.Builder
	for i, w := range widths {
		if i >= len(labels) {
			break
		}
		label := labels[i]
		if w <= 0 {
			continue
		}

		style := lipgloss.NewStyle()
		if i < len(colors) {
			style = style.Foreground(colors[i])
		}

		if len(label) > w {
			label = label[:w]
		}

		padding := (w - len(label)) / 2
		extra := (w - len(label)) % 2
		result.WriteString(strings.Repeat(" ", padding))
		result.WriteString(style.Render(label))
		result.WriteString(strings.Repeat(" ", padding+extra))
	}
	return result.String()
}

func (m *model) renderDashboard(helpText string, innerWidth, innerHeight int) string {

	activeColor := modeColors[m.mode]
	if activeColor == "" {
		activeColor = defaultBorderColor
	}
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

	if m.loading {
		loadingBox := borderStyle.
			Width(innerWidth - 2).
			Height(innerHeight - 1 - 2). // innerContentHeight = totalHeight - footer - borders
			Render(lipgloss.Place(innerWidth-2, innerHeight-1-2, lipgloss.Center, lipgloss.Center, "Loading system statistics..."))
		return lipgloss.JoinVertical(lipgloss.Left, loadingBox, footerLine)
	}

	// ═══════════════════════════════════════════════════════
	// Bar Layout Constants - ensures all bars align perfectly
	// ═══════════════════════════════════════════════════════
	const barLeftMargin = 2    // Spaces before bar
	barStartCol := barLeftMargin
	availableBarWidth := innerWidth - barStartCol - 2 // -2 for right safety margin
	if availableBarWidth < 20 {
		availableBarWidth = 20
	}

	renderBarLine := func(bar string, suffix string) string {
		if suffix == "" {
			return fmt.Sprintf("%*s%s", barLeftMargin, "", bar)
		}
		return fmt.Sprintf("%*s%s %s", barLeftMargin, "", bar, suffix)
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

	boxWidth := (innerWidth - 6) / 2
	if boxWidth < 30 {
		boxWidth = 30
	}

	renderCountLine := func(char, label string, count int, color lipgloss.Color, currentBoxWidth int) string {
		shortcut := fmt.Sprintf("[%s]", char)
		sStyle := lipgloss.NewStyle().Bold(true)
		vStyle := lipgloss.NewStyle().Bold(true)
		
		if count > 0 {
			sStyle = sStyle.Foreground(color)
			vStyle = vStyle.Foreground(color)
		}

		fullLabel := shortcut + label
		// Use Lipgloss to pad correctly regardless of color codes
		labelPart := lipgloss.NewStyle().Width(10).Render(sStyle.Render(fullLabel))
		valuePart := lipgloss.NewStyle().Width(6).Render(vStyle.Render(fmt.Sprintf("%d", count)))
		
		internalBoxWidth := currentBoxWidth - 4
		const textWidth = 20
		dynamicBarWidth := internalBoxWidth - textWidth - 2
		if dynamicBarWidth < 5 { dynamicBarWidth = 5 }

		bar := ""
		if count > 0 && m.dashboard.TotalPackages > 0 {
			filled := int(float64(count) / float64(m.dashboard.TotalPackages) * float64(dynamicBarWidth))
			if filled < 1 && count > 0 { filled = 1 }
			if filled > dynamicBarWidth { filled = dynamicBarWidth }
			bar = lipgloss.NewStyle().Background(color).Render(strings.Repeat(" ", filled))
		}

		return "  " + labelPart + " │ " + valuePart + " " + bar
	}

	countsLines := []string{
		renderCountLine("t", "otal", m.dashboard.TotalPackages, cyanColor, boxWidth),
		renderCountLine("e", "xplicit", m.dashboard.ExplicitlyInstalled, greenColor, boxWidth),
		renderCountLine("f", "oreign", m.dashboard.ForeignPackages, yellowColor, boxWidth),
		renderCountLine("m", "issing", m.dashboard.MissingFromAUR, redColor, boxWidth),
	}

	orphanStyle := lipgloss.NewStyle().Bold(true)
	
	// Calculate orphan layout consistently with renderCountLine
	internalBoxWidth := boxWidth - 4
	const textWidth = 20
	oBarW := internalBoxWidth - textWidth - 2
	if oBarW < 5 { oBarW = 5 }

	orphanFullLabel := "[o]rphans"
	if m.dashboard.Orphans > 0 {
		orphanStyle = orphanStyle.Foreground(redColor)
	}
	
	labelPart := lipgloss.NewStyle().Width(10).Render(orphanStyle.Render(orphanFullLabel))
	valuePart := lipgloss.NewStyle().Width(6).Render(orphanStyle.Render(fmt.Sprintf("%d", m.dashboard.Orphans)))

	orphanBar := ""
	if m.dashboard.Orphans > 0 {
		filled := int(float64(m.dashboard.Orphans) / float64(m.dashboard.TotalPackages) * float64(oBarW))
		if filled < 1 { filled = 1 }
		orphanBar = lipgloss.NewStyle().Background(redColor).Render(strings.Repeat(" ", filled))
	}
	
	orphanLine := "  " + labelPart + " │ " + valuePart + " " + orphanBar
	countsLines = append(countsLines, orphanLine)

	cacheStyle := lipgloss.NewStyle().Bold(true).Foreground(greenColor)
	const tenGiB = 10 * 1024 * 1024 * 1024
	if m.dashboard.CleanerSizeBytes > tenGiB {
		cacheStyle = lipgloss.NewStyle().Bold(true).Foreground(orangeColor)
	}
	if m.dashboard.CleanerSizeBytes > tenGiB*2 {
		cacheStyle = lipgloss.NewStyle().Bold(true).Foreground(redColor)
	}

	storageLines := []string{
		fmt.Sprintf("  %s │ %s",
			lipgloss.NewStyle().Width(10).Render("System"),
			lipgloss.NewStyle().Bold(true).Foreground(cyanColor).Render(m.dashboard.TotalSize)),
		fmt.Sprintf("  %s │ %s %s",
			lipgloss.NewStyle().Width(10).Render("Cache"),
			cacheStyle.Render(m.dashboard.CleanerSize),
			shortcutStyle.Render("[c]lean")),
		"",
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

	countsBox := renderBox(boxTitleStyle.Render(" \U000f03d7  Package Counts "), countsLines, boxWidth)
	storageBox := renderBox(boxTitleStyle.Render(" \uf0c7  Storage "), storageLines, boxWidth)

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
	// Disk Usage Analysis (Combined Stacked Bar)
	// ═══════════════════════════════════════════════════════
	diskTitle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229")).
		Render(fmt.Sprintf("\U000f02ca  Disk Usage Analysis (%.0f%% used)", m.dashboard.DiskUsedPercent*100))
	dashboard.WriteString(diskTitle + "\n")

	// Calculate breakdown
	totalDiskBytes := parseSizeToBytes(m.dashboard.DiskTotal)
	if totalDiskBytes == 0 {
		totalDiskBytes = 1
	}

	pkgWidth := int(float64(m.dashboard.TotalSizeBytes) / float64(totalDiskBytes) * float64(availableBarWidth))
	cacheWidth := int(float64(m.dashboard.CleanerSizeBytes) / float64(totalDiskBytes) * float64(availableBarWidth))

	usedDiskBytes := parseSizeToBytes(m.dashboard.DiskUsed)
	otherUsedBytes := usedDiskBytes - m.dashboard.TotalSizeBytes - m.dashboard.CleanerSizeBytes
	if otherUsedBytes < 0 {
		otherUsedBytes = 0
	}
	otherWidth := int(float64(otherUsedBytes) / float64(totalDiskBytes) * float64(availableBarWidth))

	// Ensure segments don't exceed total width
	if pkgWidth+cacheWidth+otherWidth > availableBarWidth {
		// Adjust proportional to avoid overflow
		sum := pkgWidth + cacheWidth + otherWidth
		pkgWidth = pkgWidth * availableBarWidth / sum
		cacheWidth = cacheWidth * availableBarWidth / sum
		otherWidth = otherWidth * availableBarWidth / sum
	}

	freeWidth := availableBarWidth - pkgWidth - cacheWidth - otherWidth
	if freeWidth < 0 {
		freeWidth = 0
	}

	otherColor := lipgloss.Color("135") // Purple for "Other" files
	freeColor := lipgloss.Color("244")  // Dim grey for "Free" space
	pkgBar := lipgloss.NewStyle().Background(cyanColor).Render(strings.Repeat(" ", pkgWidth))
	cacheBar := lipgloss.NewStyle().Background(orangeColor).Render(strings.Repeat(" ", cacheWidth))
	otherBar := lipgloss.NewStyle().Background(otherColor).Render(strings.Repeat(" ", otherWidth))
	freeBar := lipgloss.NewStyle().Background(freeColor).Render(strings.Repeat(" ", freeWidth))

	dashboard.WriteString(renderBarLine(pkgBar+cacheBar+otherBar+freeBar, "") + "\n")

	// Centered Labels & Sizes below bar
	diskLabels := []string{"Packages", "Cache", "Other", "Free"}
	pkgSize := m.dashboard.TotalSize
	cacheSize := m.dashboard.CleanerSize
	otherSize := formatBytes(otherUsedBytes)
	freeBytes := totalDiskBytes - usedDiskBytes
	if freeBytes < 0 { freeBytes = 0 }
	freeSize := formatBytes(int64(freeBytes))
	diskSizes := []string{pkgSize, cacheSize, otherSize, freeSize}

	diskWidths := []int{pkgWidth, cacheWidth, otherWidth, freeWidth}
	diskColors := []lipgloss.Color{cyanColor, orangeColor, otherColor, freeColor}
	
	labelsLine := renderCenteredLabels(diskWidths, diskLabels, diskColors)
	sizesLine := renderCenteredLabels(diskWidths, diskSizes, diskColors)
	
	dashboard.WriteString(fmt.Sprintf("%*s%s\n", barStartCol, "", labelsLine))
	dashboard.WriteString(fmt.Sprintf("%*s%s\n\n", barStartCol, "", sizesLine))

	// Repo Distribution Bar
	repoTitle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229")).
		Render("\ueb9c  Repository Distribution")
	dashboard.WriteString(repoTitle + "\n")

	if m.dashboard.TotalPackages > 0 {
		var repoBar strings.Builder
		totalWUsed := 0
		
		// 1. Core
		if count := m.dashboard.RepoDistribution["core"]; count > 0 {
			w := int(float64(count) / float64(m.dashboard.TotalPackages) * float64(availableBarWidth))
			if w < 1 { w = 1 }
			repoBar.WriteString(lipgloss.NewStyle().Background(sourceColors["core"]).Render(strings.Repeat(" ", w)))
			totalWUsed += w
		}
		// 2. Extra
		if count := m.dashboard.RepoDistribution["extra"]; count > 0 {
			w := int(float64(count) / float64(m.dashboard.TotalPackages) * float64(availableBarWidth))
			if w < 1 { w = 1 }
			repoBar.WriteString(lipgloss.NewStyle().Background(sourceColors["extra"]).Render(strings.Repeat(" ", w)))
			totalWUsed += w
		}
		// 3. Multilib
		if count := m.dashboard.RepoDistribution["multilib"]; count > 0 {
			w := int(float64(count) / float64(m.dashboard.TotalPackages) * float64(availableBarWidth))
			if w < 1 { w = 1 }
			repoBar.WriteString(lipgloss.NewStyle().Background(sourceColors["multilib"]).Render(strings.Repeat(" ", w)))
			totalWUsed += w
		}
		// 4. Foreign (AUR)
		if count := m.dashboard.ForeignPackages; count > 0 {
			w := int(float64(count) / float64(m.dashboard.TotalPackages) * float64(availableBarWidth))
			if w < 1 { w = 1 }
			repoBar.WriteString(lipgloss.NewStyle().Background(sourceColors["aur"]).Render(strings.Repeat(" ", w)))
			totalWUsed += w
		}

		// Fill remaining with background
		if totalWUsed < availableBarWidth {
			repoBar.WriteString(lipgloss.NewStyle().Background(lipgloss.Color("238")).Render(strings.Repeat(" ", availableBarWidth-totalWUsed)))
		}

		dashboard.WriteString(renderBarLine(repoBar.String(), "") + "\n")

		// Centered Repo Suffix below bar
		repoSuffix := fmt.Sprintf("%s %s %s %s",
			lipgloss.NewStyle().Foreground(sourceColors["core"]).Render(fmt.Sprintf("Core(%d)", m.dashboard.RepoDistribution["core"])),
			lipgloss.NewStyle().Foreground(sourceColors["extra"]).Render(fmt.Sprintf("Extra(%d)", m.dashboard.RepoDistribution["extra"])),
			lipgloss.NewStyle().Foreground(sourceColors["multilib"]).Render(fmt.Sprintf("Multilib(%d)", m.dashboard.RepoDistribution["multilib"])),
			lipgloss.NewStyle().Foreground(sourceColors["aur"]).Render(fmt.Sprintf("AUR(%d)", m.dashboard.ForeignPackages)))
		
		repoSuffixWidth := lipgloss.Width(repoSuffix)
		// availableBarWidth is the visual width of the bar itself
		repoPadding := (availableBarWidth - repoSuffixWidth) / 2
		if repoPadding < 0 { repoPadding = 0 }
		
		// Use renderBarLine style or simple padding to align with bar
		dashboard.WriteString(fmt.Sprintf("%*s%*s%s\n\n", barLeftMargin, "", repoPadding, "", repoSuffix))
	}

	// ═══════════════════════════════════════════════════════
	// Bottom Row: Top by Weight & Recently Installed
	// ═══════════════════════════════════════════════════════
	bottomVizWidth := (innerWidth - 6) / 2
	if bottomVizWidth < 30 {
		bottomVizWidth = 30
	}

	var topWeightLines []string
	if len(m.dashboard.TopPackages) > 0 {
		count := len(m.dashboard.TopPackages)
		if count > 5 { count = 5 } // Cut down to 5
		
		for i := 0; i < count; i++ {
			pkg := m.dashboard.TopPackages[i]
			rankStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
			nameStyle := lipgloss.NewStyle().Foreground(cyanColor)

			pkgSizeBytes := parseSizeToBytes(pkg.Size)
			sizeColor := cyanColor
			if pkgSizeBytes > 1024*1024*1024 { sizeColor = redColor } else if pkgSizeBytes > 500*1024*1024 { sizeColor = orangeColor }
			sizeStyle := lipgloss.NewStyle().Foreground(sizeColor)

			line := fmt.Sprintf("%s %s %s",
				rankStyle.Render(fmt.Sprintf("%d.", i+1)),
				lipgloss.NewStyle().Width(25).Render(nameStyle.Render(truncateWithAnsi(pkg.Name, 25))),
				sizeStyle.Render(pkg.Size))
			topWeightLines = append(topWeightLines, line)
		}
	}
	topWeightBox := renderBox(boxTitleStyle.Render(" \ueddf  Top by Weight "), topWeightLines, bottomVizWidth)

	var recentLines []string
	for i, pkg := range m.dashboard.RecentlyInstalled {
		rankStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		nameStyle := lipgloss.NewStyle().Foreground(greenColor)
		timeStyle := lipgloss.NewStyle().Foreground(dimColor)
		
		line := fmt.Sprintf("%s %s %s",
			rankStyle.Render(fmt.Sprintf("%d.", i+1)),
			lipgloss.NewStyle().Width(18).Render(nameStyle.Render(truncateWithAnsi(pkg.Name, 18))),
			timeStyle.Render(pkg.Timestamp))
		recentLines = append(recentLines, line)
	}
	recentBox := renderBox(boxTitleStyle.Render(" \uf1da  Recently Installed "), recentLines, bottomVizWidth)

	// Join side-by-side
	twLines := strings.Split(topWeightBox, "\n")
	reLines := strings.Split(recentBox, "\n")
	maxBL := len(twLines)
	if len(reLines) > maxBL { maxBL = len(reLines) }
	
	for i := 0; i < maxBL; i++ {
		l := ""; if i < len(twLines) { l = twLines[i] } else { l = strings.Repeat(" ", bottomVizWidth) }
		r := ""; if i < len(reLines) { r = reLines[i] } else { r = strings.Repeat(" ", bottomVizWidth) }
		dashboard.WriteString(l + "  " + r + "\n")
	}

	dashContent := lipgloss.NewStyle().
		Width(innerWidth - 2).
		Height(innerHeight - 1 - 2). // inner content height (subtract footer and borders)
		Padding(0, 1).
		Render(dashboard.String())

	dashPanel := borderStyle.
		Width(innerWidth - 2).
		Height(innerHeight - 1 - 2). // Total height of panel = (innerHeight-1-2) + 2 = innerHeight - 1
		Render(dashContent)

	return lipgloss.JoinVertical(lipgloss.Left, dashPanel, footerLine)
}
