package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func getDashboardData() tea.Cmd {
	return func() tea.Msg {
		var data DashboardData
		var errs []error
		var dataMu sync.Mutex

		addErr := func(msg string, err error) {
			dataMu.Lock()
			defer dataMu.Unlock()
			errs = append(errs, fmt.Errorf("%s: %w", msg, err))
		}

		var wg sync.WaitGroup

		// Total Packages
		wg.Add(1)
		go func() {
			defer wg.Done()
			out, err := runner.Run("paru", "-Q")
			if err == nil {
				val := countLines(string(out))
				dataMu.Lock()
				data.TotalPackages = val
				dataMu.Unlock()
			} else {
				addErr("total packages", err)
			}
		}()

		// Explicitly Installed
		wg.Add(1)
		go func() {
			defer wg.Done()
			out, err := runner.Run("paru", "-Qe")
			if err == nil {
				val := countLines(string(out))
				dataMu.Lock()
				data.ExplicitlyInstalled = val
				dataMu.Unlock()
			} else {
				addErr("explicitly installed", err)
			}
		}()

		// Foreign Packages
		wg.Add(1)
		go func() {
			defer wg.Done()
			out, err := runner.Run("paru", "-Qm")
			if err == nil {
				val := countLines(string(out))
				dataMu.Lock()
				data.ForeignPackages = val
				dataMu.Unlock()
			} else {
				addErr("foreign packages", err)
			}
		}()

		// Orphans (with ExitCode 1 fix)
		wg.Add(1)
		go func() {
			defer wg.Done()
			out, err := runner.Run("paru", "-Qdt")
			if err == nil {
				val := countLines(string(out))
				dataMu.Lock()
				data.Orphans = val
				dataMu.Unlock()
			} else {
				if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 && len(out) == 0 {
					dataMu.Lock()
					data.Orphans = 0
					dataMu.Unlock()
				} else {
					addErr("orphans", err)
				}
			}
		}()

		// Repository Distribution
		wg.Add(1)
		go func() {
			defer wg.Done()
			out, err := runner.Run("pacman", "-Sl")
			if err == nil {
				dist := make(map[string]int)
				lines := strings.Split(strings.TrimSpace(string(out)), "\n")
				for _, line := range lines {
					if strings.Contains(line, "[installed") {
						parts := strings.Fields(line)
						if len(parts) > 0 {
							dist[parts[0]]++
						}
					}
				}
				dataMu.Lock()
				data.RepoDistribution = dist
				dataMu.Unlock()
			}
		}()

		// Package Statistics
		wg.Add(1)
		go func() {
			defer wg.Done()
			out, err := runner.Run("paru", "-Ps")
			if err == nil {
				ts, tsb, miss, top := parseParuStats(string(out))
				dataMu.Lock()
				data.TotalSize = ts
				data.TotalSizeBytes = tsb
				data.MissingFromAUR = miss
				data.TopPackages = top
				dataMu.Unlock()
			} else {
				addErr("package statistics", err)
			}
		}()

		// Recently Installed
		wg.Add(1)
		go func() {
			defer wg.Done()
			out, err := runner.Run("grep", " installed ", "/var/log/pacman.log")
			if err == nil {
				var recent []RecentPackage
				lines := strings.Split(strings.TrimSpace(string(out)), "\n")
				seen := make(map[string]bool)
				for i := len(lines) - 1; i >= 0 && len(recent) < 5; i-- {
					parts := strings.Fields(lines[i])
					if len(parts) >= 4 {
						pkg := parts[3]
						if !seen[pkg] {
							ts := parts[0]
							if len(ts) > 17 {
								ts = ts[1:11] + " " + ts[12:17]
							}
							recent = append(recent, RecentPackage{
								Name:      pkg,
								Timestamp: ts,
							})
							seen[pkg] = true
						}
					}
				}
				dataMu.Lock()
				data.RecentlyInstalled = recent
				dataMu.Unlock()
			}
		}()

		// Caches
		wg.Add(1)
		go func() {
			defer wg.Done()
			pacmanCachePath := "/var/cache/pacman/pkg"
			pacmanSize := calculateDirSize(pacmanCachePath)

			homeDir, _ := os.UserHomeDir()
			paruBase := filepath.Join(homeDir, ".cache", "paru")
			paruClonePath := filepath.Join(paruBase, "clone")

			cacheHogs := make(map[string]int64)

			if entries, err := os.ReadDir(pacmanCachePath); err == nil {
				for _, entry := range entries {
					if entry.IsDir() {
						continue
					}
					name := entry.Name()
					if !strings.HasSuffix(name, ".pkg.tar.zst") && !strings.HasSuffix(name, ".pkg.tar.xz") {
						continue
					}
					parts := strings.Split(name, "-")
					if len(parts) > 3 {
						baseName := strings.Join(parts[:len(parts)-3], "-")
						info, err := entry.Info()
						if err == nil {
							cacheHogs[baseName] += info.Size()
						}
					}
				}
			}

			filepath.WalkDir(paruClonePath, func(path string, d os.DirEntry, err error) error {
				if err != nil || d.IsDir() {
					return nil
				}
				name := d.Name()
				if !strings.HasSuffix(name, ".pkg.tar.zst") && !strings.HasSuffix(name, ".pkg.tar.xz") {
					return nil
				}
				if info, err := d.Info(); err == nil {
					parts := strings.Split(name, "-")
					if len(parts) > 3 {
						baseName := strings.Join(parts[:len(parts)-3], "-")
						cacheHogs[baseName] += info.Size()
					}
				}
				return nil
			})

			type cacheEntry struct {
				name string
				size int64
			}
			var sortedHogs []cacheEntry
			for k, v := range cacheHogs {
				sortedHogs = append(sortedHogs, cacheEntry{k, v})
			}
			sort.Slice(sortedHogs, func(i, j int) bool {
				return sortedHogs[i].size > sortedHogs[j].size
			})
			var allHogs []PackageSize
			for i := 0; i < len(sortedHogs); i++ {
				allHogs = append(allHogs, PackageSize{
					Name:      sortedHogs[i].name,
					Size:      formatBytes(sortedHogs[i].size),
					SizeBytes: sortedHogs[i].size,
				})
			}

			var topHogs []PackageSize
			for i := 0; i < 5 && i < len(allHogs); i++ {
				topHogs = append(topHogs, allHogs[i])
			}

			paruSize := calculateDirSize(paruBase)
			totalCache := pacmanSize + paruSize

			// Estimates for paccache (only for system cache)
			estimates := make(map[confirmationType]string)
			fetchEstimate := func(ct confirmationType, args ...string) {
				out, err := runner.Run("paccache", append([]string{"-d"}, args...)...)
				if err == nil {
					estimates[ct] = parsePaccacheDryRun(string(out))
				} else {
					estimates[ct] = "0 B"
				}
			}

			fetchEstimate(confirmCleanKeep3, "-k3")
			fetchEstimate(confirmCleanKeep1, "-k1")
			fetchEstimate(confirmCleanUninstalled, "-uk0")
			estimates[confirmCleanNuke] = formatBytes(pacmanSize) // Nuke is everything

			dataMu.Lock()
			data.TopCacheHogs = topHogs
			data.AllCacheHogs = allHogs
			data.CacheFreedEstimates = estimates
			data.PacmanCachePath = pacmanCachePath
			data.PacmanCacheSizeBytes = pacmanSize
			data.PacmanCacheSize = formatBytes(pacmanSize)
			data.ParuCachePath = paruBase
			data.ParuCacheSizeBytes = paruSize
			data.ParuCacheSize = formatBytes(paruSize)
			data.CleanerSizeBytes = totalCache
			data.CleanerSize = formatBytes(totalCache)
			dataMu.Unlock()
		}()
		// Disk Usage
		wg.Add(1)
		go func() {
			defer wg.Done()
			var stat syscall.Statfs_t
			if err := syscall.Statfs("/", &stat); err == nil {
				total := stat.Blocks * uint64(stat.Bsize)
				free := stat.Bfree * uint64(stat.Bsize)
				used := total - free

				dataMu.Lock()
				data.DiskTotal = formatBytes(int64(total))
				data.DiskFree = formatBytes(int64(free))
				data.DiskUsed = formatBytes(int64(used))
				if total > 0 {
					data.DiskUsedPercent = float64(used) / float64(total)
				}
				dataMu.Unlock()
			}
		}()

		wg.Wait()

		dataMu.Lock()
		defer dataMu.Unlock()
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

// parsePaccacheDryRun extracts the "disk space saved" string from paccache -d output
func parsePaccacheDryRun(output string) string {
	if strings.Contains(output, "no candidate packages found") {
		return "0 B"
	}
	// Format: ==> finished dry run: 78 candidates (disk space saved: 782.43 MiB)
	parts := strings.Split(output, "disk space saved: ")
	if len(parts) > 1 {
		// Split by space or ")" to get the size part
		resParts := strings.Fields(parts[1])
		if len(resParts) >= 2 {
			return resParts[0] + " " + strings.TrimSuffix(resParts[1], ")")
		}
	}
	return "0 B"
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
	// availableContentWidth accounts for dashPanel borders (-2) and dashContent padding (-4 total)
	safeWidth := innerWidth - 6
	if safeWidth < 60 {
		safeWidth = 60
	}

	barStartCol := 0
	availableBarWidth := safeWidth
	if availableBarWidth < 20 {
		availableBarWidth = 20
	}

	renderBarLine := func(bar string, suffix string) string {
		if suffix == "" {
			return bar
		}
		return bar + " " + suffix
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

	topBoxWidth := (safeWidth - 2) / 2
	if topBoxWidth < 30 {
		topBoxWidth = 30
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
		// Use Width(12) and Width(10) for symmetry with Disk Space box
		labelPart := lipgloss.NewStyle().Width(12).Render(sStyle.Render(fullLabel))
		valuePart := lipgloss.NewStyle().Width(10).Render(vStyle.Render(fmt.Sprintf("%d", count)))

		internalBoxWidth := currentBoxWidth - 4
		const textWidth = 26 // 12 + 10 + 4
		dynamicBarWidth := internalBoxWidth - textWidth - 2
		if dynamicBarWidth < 5 {
			dynamicBarWidth = 5
		}

		bar := ""
		if count > 0 && m.dashboard.TotalPackages > 0 {
			filled := int(float64(count) / float64(m.dashboard.TotalPackages) * float64(dynamicBarWidth))
			if filled < 1 && count > 0 {
				filled = 1
			}
			if filled > dynamicBarWidth {
				filled = dynamicBarWidth
			}
			bar = lipgloss.NewStyle().Background(color).Render(strings.Repeat(" ", filled))
		}

		return "  " + labelPart + " │ " + valuePart + " " + bar
	}

	countsLines := []string{
		renderCountLine("t", "otal", m.dashboard.TotalPackages, cyanColor, topBoxWidth),
		renderCountLine("e", "xplicit", m.dashboard.ExplicitlyInstalled, greenColor, topBoxWidth),
		renderCountLine("f", "oreign", m.dashboard.ForeignPackages, yellowColor, topBoxWidth),
		renderCountLine("m", "issing", m.dashboard.MissingFromAUR, redColor, topBoxWidth),
	}

	orphanStyle := lipgloss.NewStyle().Bold(true)

	// Calculate orphan layout consistently with renderCountLine
	oInternalBoxWidth := topBoxWidth - 4
	const oTextWidth = 26
	oBarW := oInternalBoxWidth - oTextWidth - 2
	if oBarW < 5 {
		oBarW = 5
	}

	orphanFullLabel := "[o]rphans"
	if m.dashboard.Orphans > 0 {
		orphanStyle = orphanStyle.Foreground(redColor)
	}

	oLabelPart := lipgloss.NewStyle().Width(12).Render(orphanStyle.Render(orphanFullLabel))
	oValuePart := lipgloss.NewStyle().Width(10).Render(orphanStyle.Render(fmt.Sprintf("%d", m.dashboard.Orphans)))

	orphanBar := ""
	if m.dashboard.Orphans > 0 {
		filled := int(float64(m.dashboard.Orphans) / float64(m.dashboard.TotalPackages) * float64(oBarW))
		if filled < 1 {
			filled = 1
		}
		orphanBar = lipgloss.NewStyle().Background(redColor).Render(strings.Repeat(" ", filled))
	}

	orphanLine := "  " + oLabelPart + " │ " + oValuePart + " " + orphanBar
	countsLines = append(countsLines, orphanLine)

	// Calculate Disk Usage Breakdown for Disk Space Box
	totalDiskBytes := parseSizeToBytes(m.dashboard.DiskTotal)
	if totalDiskBytes == 0 {
		totalDiskBytes = 1
	}

	usedDiskBytes := parseSizeToBytes(m.dashboard.DiskUsed)
	otherUsedBytes := usedDiskBytes - m.dashboard.TotalSizeBytes - m.dashboard.CleanerSizeBytes
	if otherUsedBytes < 0 {
		otherUsedBytes = 0
	}

	freeBytes := totalDiskBytes - usedDiskBytes
	if freeBytes < 0 {
		freeBytes = 0
	}

	otherColor := lipgloss.Color("135") // Purple for "Other" files

	renderDiskLine := func(char, label, size string, bytes int64, color lipgloss.Color, currentBoxWidth int) string {
		prefix := ""
		if char != "" {
			prefix = fmt.Sprintf("[%s]", char)
		}
		sStyle := lipgloss.NewStyle().Bold(true).Foreground(color)
		vStyle := lipgloss.NewStyle().Bold(true).Foreground(color)

		fullLabel := prefix + label
		labelPart := lipgloss.NewStyle().Width(12).Render(sStyle.Render(fullLabel))
		valuePart := lipgloss.NewStyle().Width(10).Render(vStyle.Render(size))

		internalBoxWidth := currentBoxWidth - 4
		const textWidth = 26
		dynamicBarWidth := internalBoxWidth - textWidth - 2
		if dynamicBarWidth < 5 {
			dynamicBarWidth = 5
		}

		bar := ""
		if bytes > 0 && totalDiskBytes > 0 {
			filled := int(float64(bytes) / float64(totalDiskBytes) * float64(dynamicBarWidth))
			if filled < 1 && bytes > 0 {
				filled = 1
			}
			if filled > dynamicBarWidth {
				filled = dynamicBarWidth
			}
			bar = lipgloss.NewStyle().Background(color).Render(strings.Repeat(" ", filled))
		}

		return "  " + labelPart + " │ " + valuePart + " " + bar
	}

	storageLines := []string{
		renderDiskLine("", "total", m.dashboard.DiskTotal, totalDiskBytes, lipgloss.Color("229"), topBoxWidth),
		renderDiskLine("", "packages", m.dashboard.TotalSize, m.dashboard.TotalSizeBytes, cyanColor, topBoxWidth),
		renderDiskLine("c", "ache", m.dashboard.CleanerSize, m.dashboard.CleanerSizeBytes, orangeColor, topBoxWidth),
		renderDiskLine("", "other", formatBytes(otherUsedBytes), otherUsedBytes, otherColor, topBoxWidth),
		renderDiskLine("", "free", formatBytes(freeBytes), freeBytes, lipgloss.Color("246"), topBoxWidth),
	}

	borderColor := lipgloss.NewStyle().Foreground(activeColor)

	renderBox := func(title string, lines []string, width int) string {
		var b strings.Builder

		innerWidth := width - 4
		if innerWidth < 10 {
			innerWidth = 10
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
			// Ensure no internal newlines break the box formatting
			cleanLine := strings.ReplaceAll(line, "\n", " ")
			lineWidth := lipgloss.Width(cleanLine)
			padding := innerWidth - lineWidth
			if padding < 0 {
				cleanLine = truncateWithAnsi(cleanLine, innerWidth)
				padding = 0
			}
			b.WriteString(leftBorder + cleanLine + strings.Repeat(" ", padding) + rightBorder + "\n")
		}

		b.WriteString(borderColor.Render("╰" + strings.Repeat("─", innerWidth+2) + "╯"))

		return b.String()
	}

	// ═══════════════════════════════════════════════════════
	// Top Row: Package Counts & Disk Space
	// ═══════════════════════════════════════════════════════
	countsBox := renderBox(boxTitleStyle.Render(" \U000f03d7  Package Counts "), countsLines, topBoxWidth)
	diskSpaceTitle := fmt.Sprintf(" \uf0c7  Disk Space (/:%.0f%%) ", m.dashboard.DiskUsedPercent*100)
	storageBox := renderBox(boxTitleStyle.Render(diskSpaceTitle), storageLines, topBoxWidth)

	topRow := lipgloss.JoinHorizontal(lipgloss.Top, countsBox, "  ", storageBox)
	dashboard.WriteString(topRow + "\n\n")

	// Repo Distribution Bar
	repoTitle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229")).
		Render("\ueb9c  Repository Distribution")
	dashboard.WriteString(repoTitle + "\n")

	if m.dashboard.TotalPackages > 0 {
		var repoBar strings.Builder

		// Identify all repositories and group non-core/extra/multilib into "Other"
		type repoStat struct {
			name  string
			count int
			color lipgloss.Color
		}

		coreCount := m.dashboard.RepoDistribution["core"]
		extraCount := m.dashboard.RepoDistribution["extra"]
		multilibCount := m.dashboard.RepoDistribution["multilib"]
		aurCount := m.dashboard.ForeignPackages

		// Calculate "Other" packages (any repo not in core/extra/multilib, or manually installed)
		accounted := coreCount + extraCount + multilibCount + aurCount
		otherCount := m.dashboard.TotalPackages - accounted
		if otherCount < 0 {
			otherCount = 0
		}

		allStats := []repoStat{
			{"Core", coreCount, sourceColors["core"]},
			{"Extra", extraCount, sourceColors["extra"]},
			{"Multilib", multilibCount, sourceColors["multilib"]},
			{"AUR", aurCount, sourceColors["aur"]},
		}
		if otherCount > 0 {
			allStats = append(allStats, repoStat{"Other", otherCount, lipgloss.Color("244")})
		}

		// Calculate initial widths using floor
		widths := make([]int, len(allStats))
		totalWUsed := 0
		maxWIdx := 0
		for i, stat := range allStats {
			if stat.count > 0 {
				w := int(float64(stat.count) / float64(m.dashboard.TotalPackages) * float64(availableBarWidth))
				if w < 1 {
					w = 1
				}
				widths[i] = w
				totalWUsed += w
				if widths[i] > widths[maxWIdx] {
					maxWIdx = i
				}
			}
		}

		// Distribute remainder to the largest segment to fill bar perfectly
		if totalWUsed < availableBarWidth {
			widths[maxWIdx] += (availableBarWidth - totalWUsed)
		} else if totalWUsed > availableBarWidth {
			// This could happen due to the "at least 1 cell" rule
			widths[maxWIdx] -= (totalWUsed - availableBarWidth)
			if widths[maxWIdx] < 1 {
				widths[maxWIdx] = 1
			}
		}

		// Build the bar
		for i, stat := range allStats {
			if widths[i] > 0 {
				repoBar.WriteString(lipgloss.NewStyle().Background(stat.color).Render(strings.Repeat(" ", widths[i])))
			}
		}

		dashboard.WriteString(renderBarLine(repoBar.String(), "") + "\n\n")

		// Legend blocks below bar
		renderBlock := func(label string, count int, color lipgloss.Color) string {
			cStr := fmt.Sprintf("%d", count)
			if len(label)%2 != len(cStr)%2 {
				cStr = " " + cStr
			}
			w := len(label)
			if len(cStr) > w {
				w = len(cStr)
			}
			w += 4
			style := lipgloss.NewStyle().Background(color).Foreground(lipgloss.Color("16")).Bold(true).Width(w).Align(lipgloss.Center)
			return lipgloss.JoinVertical(lipgloss.Center, style.Render(label), style.Render(cStr))
		}

		var legendBlocks []string
		for _, stat := range allStats {
			if stat.count > 0 {
				legendBlocks = append(legendBlocks, renderBlock(stat.name, stat.count, stat.color))
			}
		}

		repoLegend := lipgloss.JoinHorizontal(lipgloss.Top, legendBlocks...)
		repoLegendWidth := lipgloss.Width(repoLegend)
		repoPadding := (availableBarWidth - repoLegendWidth) / 2
		if repoPadding < 0 {
			repoPadding = 0
		}

		legendLines := strings.Split(repoLegend, "\n")
		for _, line := range legendLines {
			dashboard.WriteString(fmt.Sprintf("%*s%*s%s\n", barStartCol, "", repoPadding, "", line))
		}
		dashboard.WriteString("\n")
	}

	// ═══════════════════════════════════════════════════════
	// Bottom Row: Top by Weight, Cache Hogs & Recently Installed
	// ═══════════════════════════════════════════════════════
	bottomVizWidth := (safeWidth - 4) / 3 // 2 spaces between boxes
	if bottomVizWidth < 30 {
		bottomVizWidth = 30
	}
	boxInnerW := bottomVizWidth - 4
	if boxInnerW < 10 {
		boxInnerW = 10
	}

	var topWeightLines []string
	if len(m.dashboard.TopPackages) > 0 {
		count := len(m.dashboard.TopPackages)
		if count > 5 {
			count = 5
		} // Cut down to 5

		nameW := boxInnerW - 13 // rank(3) + space(1) + space(1) + size(~8)
		if nameW < 5 {
			nameW = 5
		}

		for i := 0; i < count; i++ {
			pkg := m.dashboard.TopPackages[i]
			rankStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
			nameStyle := lipgloss.NewStyle().Foreground(cyanColor)

			pkgSizeBytes := parseSizeToBytes(pkg.Size)
			sizeColor := cyanColor
			if pkgSizeBytes > 1024*1024*1024 {
				sizeColor = redColor
			} else if pkgSizeBytes > 500*1024*1024 {
				sizeColor = orangeColor
			}
			sizeStyle := lipgloss.NewStyle().Foreground(sizeColor)

			line := fmt.Sprintf("%s %s %s",
				rankStyle.Render(fmt.Sprintf("%2d.", i+1)),
				lipgloss.NewStyle().Width(nameW).Render(nameStyle.Render(truncateWithAnsi(pkg.Name, nameW))),
				sizeStyle.Render(fmt.Sprintf("%8s", pkg.Size)))
			topWeightLines = append(topWeightLines, line)
		}
	}
	topWeightBox := renderBox(boxTitleStyle.Render(" \ueddf  Top by Weight "), topWeightLines, bottomVizWidth)

	var topCacheLines []string
	if len(m.dashboard.TopCacheHogs) > 0 {
		count := len(m.dashboard.TopCacheHogs)
		if count > 5 {
			count = 5
		}

		nameW := boxInnerW - 13 // rank(3) + space(1) + space(1) + size(~8)
		if nameW < 5 {
			nameW = 5
		}

		for i := 0; i < count; i++ {
			pkg := m.dashboard.TopCacheHogs[i]
			rankStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
			nameStyle := lipgloss.NewStyle().Foreground(orangeColor)
			sizeStyle := lipgloss.NewStyle().Foreground(yellowColor)

			line := fmt.Sprintf("%s %s %s",
				rankStyle.Render(fmt.Sprintf("%2d.", i+1)),
				lipgloss.NewStyle().Width(nameW).Render(nameStyle.Render(truncateWithAnsi(pkg.Name, nameW))),
				sizeStyle.Render(fmt.Sprintf("%8s", pkg.Size)))
			topCacheLines = append(topCacheLines, line)
		}
	}
	cacheHogsBox := renderBox(boxTitleStyle.Render(" \uf0c7  Top Cache Hogs "), topCacheLines, bottomVizWidth)

	var recentLines []string
	for i, pkg := range m.dashboard.RecentlyInstalled {
		nameW := boxInnerW - 22 // rank(3) + space(1) + time(16) + space(1) + extra
		if nameW < 5 {
			nameW = 5
		}

		rankStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		nameStyle := lipgloss.NewStyle().Foreground(greenColor)
		timeStyle := lipgloss.NewStyle().Foreground(dimColor)

		line := fmt.Sprintf("%s %s %s",
			rankStyle.Render(fmt.Sprintf("%2d.", i+1)),
			lipgloss.NewStyle().Width(nameW).Render(nameStyle.Render(truncateWithAnsi(pkg.Name, nameW))),
			timeStyle.Render(pkg.Timestamp))
		recentLines = append(recentLines, line)
	}
	recentBox := renderBox(boxTitleStyle.Render(" \uf1da  Recently Installed "), recentLines, bottomVizWidth)

	bottomRow := lipgloss.JoinHorizontal(lipgloss.Top, topWeightBox, "  ", cacheHogsBox, "  ", recentBox)
	dashboard.WriteString(bottomRow)

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
