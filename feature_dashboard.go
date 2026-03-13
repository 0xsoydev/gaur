package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
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

	renderCountLine := func(char, label string, count int, color lipgloss.Color) string {
		shortcut := fmt.Sprintf("[%s]", char)
		sStyle := lipgloss.NewStyle().Bold(true)
		vStyle := lipgloss.NewStyle().Bold(true)
		if count > 0 {
			sStyle = sStyle.Foreground(color)
			vStyle = vStyle.Foreground(color)
		}
		// Prefix is exactly 3 chars "[x]", align labels to start at same column
		return fmt.Sprintf("  %s%-7s │ %s", sStyle.Render(shortcut), label, vStyle.Render(fmt.Sprintf("%d", count)))
	}

	countsLines := []string{
		renderCountLine("t", "otal", m.dashboard.TotalPackages, cyanColor),
		renderCountLine("e", "xplicit", m.dashboard.ExplicitlyInstalled, greenColor),
		renderCountLine("f", "oreign", m.dashboard.ForeignPackages, yellowColor),
	}

	orphanStyle := lipgloss.NewStyle().Bold(true)
	if m.dashboard.Orphans > 0 {
		orphanStyle = orphanStyle.Foreground(redColor)
	}
	orphanLine := fmt.Sprintf("  %s%-7s │ %s",
		orphanStyle.Render("[o]"),
		"rphans",
		orphanStyle.Render(fmt.Sprintf("%d", m.dashboard.Orphans)))
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

	boxWidth := (innerWidth - 6) / 2
	if boxWidth < 30 {
		boxWidth = 30
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
	// Bar Layout Constants - ensures all bars align perfectly
	// ═══════════════════════════════════════════════════════
	const barLeftMargin = 2     // Spaces before bar
	const barSuffixReserve = 55 // Increased to allow full names on the right
	barStartCol := barLeftMargin
	availableBarWidth := innerWidth - barStartCol - barSuffixReserve
	if availableBarWidth < 20 {
		availableBarWidth = 20
	}

	renderBarLine := func(bar string, suffix string) string {
		return fmt.Sprintf("%*s%s %s", barLeftMargin, "", bar, suffix)
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
		Render("\ueea8  Explicit vs Dependencies")
	ratioSuffix := fmt.Sprintf("%d/%d (%.0f%% explicit)", m.dashboard.ExplicitlyInstalled, dependencies, explicitRatio*100)
	ratioBar := renderBarLine(filledBar+emptyBar, ratioSuffix)

	dashboard.WriteString(ratioTitle + "\n")
	dashboard.WriteString(ratioBar + "\n")

	// Centered Labels below bar
	labels := []string{fmt.Sprintf("%d", m.dashboard.ExplicitlyInstalled), fmt.Sprintf("%d", dependencies)}
	widths := []int{filledWidth, availableBarWidth - filledWidth}
	labelColors := []lipgloss.Color{greenColor, dimColor}
	dashboard.WriteString(fmt.Sprintf("%*s%s\n\n", barStartCol, "", renderCenteredLabels(widths, labels, labelColors)))

	// ═══════════════════════════════════════════════════════
	// Disk Usage Analysis (Combined Stacked Bar)
	// ═══════════════════════════════════════════════════════
	diskTitle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229")).
		Render("\U000f02ca  Disk Usage Analysis")
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

	pkgBar := lipgloss.NewStyle().Background(cyanColor).Render(strings.Repeat(" ", pkgWidth))
	cacheBar := lipgloss.NewStyle().Background(orangeColor).Render(strings.Repeat(" ", cacheWidth))
	otherBar := lipgloss.NewStyle().Background(lipgloss.Color("240")).Render(strings.Repeat(" ", otherWidth))
	freeBar := lipgloss.NewStyle().Background(lipgloss.Color("236")).Render(strings.Repeat(" ", freeWidth))

	diskSuffix := fmt.Sprintf("%s/%s (%.0f%% used)", m.dashboard.DiskUsed, m.dashboard.DiskTotal, m.dashboard.DiskUsedPercent*100)
	dashboard.WriteString(renderBarLine(pkgBar+cacheBar+otherBar+freeBar, diskSuffix) + "\n")

	// Legend for the disk bar (aligned with suffix)
	pkgLegend := lipgloss.NewStyle().Foreground(cyanColor).Render("Packages ")
	cacheLegend := lipgloss.NewStyle().Foreground(orangeColor).Render("Cache ")
	otherLegend := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("Other ")
	freeLegend := lipgloss.NewStyle().Foreground(lipgloss.Color("236")).Render("Free")
	legendContent := pkgLegend + cacheLegend + otherLegend + freeLegend

	// Centered sizes below bar
	pkgSize := m.dashboard.TotalSize
	cacheSize := m.dashboard.CleanerSize
	otherSize := formatBytes(otherUsedBytes)
	freeBytes := totalDiskBytes - usedDiskBytes
	if freeBytes < 0 {
		freeBytes = 0
	}
	freeSize := formatBytes(int64(freeBytes))

	diskSizeLabels := []string{pkgSize, cacheSize, otherSize, freeSize}
	diskWidths := []int{pkgWidth, cacheWidth, otherWidth, freeWidth}
	diskColors := []lipgloss.Color{cyanColor, orangeColor, lipgloss.Color("240"), lipgloss.Color("236")}
	
	sizesLine := renderCenteredLabels(diskWidths, diskSizeLabels, diskColors)
	
	// Write the sizes line followed by the legend aligned with suffix
	dashboard.WriteString(fmt.Sprintf("%*s%s %s\n\n", barStartCol, "", sizesLine, legendContent))

	// Repo Distribution Bar
	repoTitle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229")).
		Render("\ueb9c  Repository Distribution")
	dashboard.WriteString(repoTitle + "\n")

	if m.dashboard.TotalPackages > 0 {
		type segment struct {
			name  string
			count int
			width int
			color lipgloss.Color
		}
		var segments []segment
		totalWidthUsed := 0

		// Handle standard and custom repos
		for repo, count := range m.dashboard.RepoDistribution {
			if count > 0 {
				w := int(float64(count) / float64(m.dashboard.TotalPackages) * float64(availableBarWidth))
				color, ok := sourceColors[repo]
				if !ok {
					color = lipgloss.Color("246") // Grey for unknown repos
				}
				segments = append(segments, segment{repo, count, w, color})
				totalWidthUsed += w
			}
		}

		// Handle Foreign (AUR)
		if m.dashboard.ForeignPackages > 0 {
			w := int(float64(m.dashboard.ForeignPackages) / float64(m.dashboard.TotalPackages) * float64(availableBarWidth))
			segments = append(segments, segment{"aur", m.dashboard.ForeignPackages, w, sourceColors["aur"]})
			totalWidthUsed += w
		}

		// Distribute remainder to the largest segment to fill 100%
		remainder := availableBarWidth - totalWidthUsed
		if remainder > 0 && len(segments) > 0 {
			largestIdx := 0
			for i := 1; i < len(segments); i++ {
				if segments[i].count > segments[largestIdx].count {
					largestIdx = i
				}
			}
			segments[largestIdx].width += remainder
		}

		// Sort segments for consistent display (core, extra, multilib, then others)
		sort.Slice(segments, func(i, j int) bool {
			order := map[string]int{"core": 0, "extra": 1, "multilib": 2, "aur": 4}
			oi, oki := order[segments[i].name]
			oj, okj := order[segments[j].name]
			if !oki { oi = 3 }
			if !okj { oj = 3 }
			return oi < oj
		})

		var repoBar strings.Builder
		for _, seg := range segments {
			if seg.width > 0 {
				repoBar.WriteString(lipgloss.NewStyle().Background(seg.color).Render(strings.Repeat(" ", seg.width)))
			}
		}

		repoSuffix := fmt.Sprintf("%s %s %s %s",
			lipgloss.NewStyle().Foreground(sourceColors["core"]).Render(fmt.Sprintf("Core(%d)", m.dashboard.RepoDistribution["core"])),
			lipgloss.NewStyle().Foreground(sourceColors["extra"]).Render(fmt.Sprintf("Extra(%d)", m.dashboard.RepoDistribution["extra"])),
			lipgloss.NewStyle().Foreground(sourceColors["multilib"]).Render(fmt.Sprintf("Multilib(%d)", m.dashboard.RepoDistribution["multilib"])),
			lipgloss.NewStyle().Foreground(sourceColors["aur"]).Render(fmt.Sprintf("AUR(%d)", m.dashboard.ForeignPackages)))
		dashboard.WriteString(renderBarLine(repoBar.String(), repoSuffix) + "\n\n")
	}

	if len(m.dashboard.TopPackages) > 0 {
		topTitle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229")).
			Render("\ueddf  Top by Weight")
		dashboard.WriteString(topTitle + "\n")

		for i, pkg := range m.dashboard.TopPackages {
			rankStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
			nameStyle := lipgloss.NewStyle().Foreground(cyanColor)

			// Color-coded size
			pkgSizeBytes := parseSizeToBytes(pkg.Size)
			sizeColor := cyanColor
			if pkgSizeBytes > 1024*1024*1024 { // > 1GiB
				sizeColor = redColor
			} else if pkgSizeBytes > 500*1024*1024 { // > 500MiB
				sizeColor = orangeColor
			}
			sizeStyle := lipgloss.NewStyle().Foreground(sizeColor)

			dashboard.WriteString(fmt.Sprintf("  %s %s %s\n",
				rankStyle.Render(fmt.Sprintf("%2d.", i+1)),
				nameStyle.Render(fmt.Sprintf("%-30s", pkg.Name)),
				sizeStyle.Render(pkg.Size)))
		}
		dashboard.WriteString("\n")
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
