package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
)

// isValidPackageName checks if a package name contains only safe characters.
// Valid package names contain only alphanumeric, @, ., _, +, and - characters.
// This prevents command injection through malicious package names.
func isValidPackageName(name string) bool {
	if name == "" {
		return false
	}
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '@' || r == '.' ||
			r == '_' || r == '+' || r == '-') {
			return false
		}
	}
	return true
}

// sanitizePackageNames filters a list of package names to only include valid ones.
// Returns the filtered list and a boolean indicating if all names were valid.
func sanitizePackageNames(names []string) ([]string, bool) {
	var valid []string
	allValid := true
	for _, name := range names {
		if isValidPackageName(name) {
			valid = append(valid, name)
		} else {
			allValid = false
		}
	}
	return valid, allValid
}

// fuzzyFilter filters packages using fzf for fuzzy matching.
// Returns filtered packages sorted by fzf's relevance ranking.
func fuzzyFilter(packages []Package, query string) []Package {
	if query == "" || len(packages) == 0 {
		return packages
	}

	// Build input for fzf: one package name per line with index
	var input strings.Builder
	for i, pkg := range packages {
		input.WriteString(fmt.Sprintf("%d\t%s\n", i, pkg.Name))
	}

	stdout, _ := runner.RunWithInput(input.String(), "fzf", "--filter", query, "-d", "\t", "-n2", "--tiebreak=begin,length")

	// Parse output and rebuild package list
	var result []Package
	lines := strings.Split(strings.TrimSpace(string(stdout)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) >= 1 {
			var idx int
			if _, err := fmt.Sscanf(parts[0], "%d", &idx); err == nil && idx >= 0 && idx < len(packages) {
				result = append(result, packages[idx])
			}
		}
	}

	if len(result) == 0 {
		queryLower := strings.ToLower(query)
		for _, pkg := range packages {
			if strings.Contains(strings.ToLower(pkg.Name), queryLower) {
				result = append(result, pkg)
			}
		}
	}

	return result
}

// computeMatchIndices finds the character indices in the package string (source/name)
// that match the query using case-insensitive fuzzy matching.
// Returns indices relative to the full "source/name" string.
func computeMatchIndices(pkg Package, query string) []int {
	if query == "" {
		return nil
	}

	// Determine the string being displayed
	var pkgStr string
	if pkg.Source != "" {
		pkgStr = pkg.Source + "/" + pkg.Name
	} else {
		pkgStr = pkg.Name
	}

	pkgLower := strings.ToLower(pkgStr)
	queryLower := strings.ToLower(query)

	var indices []int

	// Priority 1: Exact substring match
	if idx := strings.Index(pkgLower, queryLower); idx != -1 {
		for i := 0; i < len(queryLower); i++ {
			indices = append(indices, idx+i)
		}
		return indices
	}

	// Priority 2: Fuzzy match
	pkgRunes := []rune(pkgLower)
	queryRunes := []rune(queryLower)
	
	// Simple greedy forward match
	currentPkgIdx := 0
	for _, qr := range queryRunes {
		found := false
		for currentPkgIdx < len(pkgRunes) {
			if pkgRunes[currentPkgIdx] == qr {
				indices = append(indices, currentPkgIdx)
				currentPkgIdx++
				found = true
				break
			}
			currentPkgIdx++
		}
		if !found {
			return nil
		}
	}

	return indices
}

// computeAllMatchIndices computes match indices for all packages in the filtered list.
// Returns a map from package index to matched character indices.
func computeAllMatchIndices(packages []Package, query string) map[int][]int {
	if query == "" || len(packages) == 0 {
		return nil
	}

	result := make(map[int][]int, len(packages))
	for i, pkg := range packages {
		indices := computeMatchIndices(pkg, query)
		if len(indices) > 0 {
			result[i] = indices
		}
	}
	return result
}

// highlightMatches renders a string with matched character indices highlighted.
// It uses a map for O(1) lookup of matched positions and handles Unicode correctly.
func highlightMatches(s string, matchedIndices []int) string {
	if len(matchedIndices) == 0 {
		return s
	}

	matchSet := make(map[int]struct{}, len(matchedIndices))
	for _, idx := range matchedIndices {
		matchSet[idx] = struct{}{}
	}

	var result strings.Builder
	result.Grow(len(s) * 2)
	runes := []rune(s)
	for i, r := range runes {
		if _, matched := matchSet[i]; matched {
			result.WriteString(matchHighlightStyle.Render(string(r)))
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// highlightMatchesWithSourceColor renders a package string (source/name) with:
// - Source portion colored by repository
// - Matched characters highlighted with matchHighlightStyle
// - Non-matched characters in normal text color
func highlightMatchesWithSourceColor(pkg Package, matchedIndices []int) string {
	pkgStr := pkg.Source + "/" + pkg.Name

	sourceColor, hasSourceColor := sourceColors[pkg.Source]

	if len(matchedIndices) == 0 {
		if hasSourceColor {
			return lipgloss.NewStyle().Foreground(sourceColor).Render(pkg.Source) + "/" + pkg.Name
		}
		return pkgStr
	}

	matchSet := make(map[int]struct{}, len(matchedIndices))
	for _, idx := range matchedIndices {
		matchSet[idx] = struct{}{}
	}

	slashIdx := len(pkg.Source)

	var result strings.Builder
	result.Grow(len(pkgStr) * 2)
	runes := []rune(pkgStr)

	for i, r := range runes {
		if _, matched := matchSet[i]; matched {

			result.WriteString(matchHighlightStyle.Render(string(r)))
		} else if i < slashIdx && hasSourceColor {

			result.WriteString(lipgloss.NewStyle().Foreground(sourceColor).Render(string(r)))
		} else {

			result.WriteRune(r)
		}
	}
	return result.String()
}

// Repo filter character mappings
var repoFilterChars = map[rune]string{
	'c': "core",
	'e': "extra",
	'm': "multilib",
	'a': "aur",
}

// uninstallFilterChars maps single characters to package filter types for uninstall mode
var uninstallFilterChars = map[rune]string{
	't': "total",
	'e': "explicit",
	'f': "foreign",
	'o': "orphan",
}

// parseRepoFilter extracts repo filters and search query from input
// Supports combined filters like "ae:", "cem:", "aem:" in any order
// Returns (repoFilters, searchQuery) where repoFilters is empty if no filter specified
func parseRepoFilter(input string) (map[string]bool, string) {
	input = strings.TrimSpace(input)

	colonIdx := strings.Index(input, ":")
	if colonIdx == -1 {
		return nil, input
	}

	prefix := strings.ToLower(input[:colonIdx])
	searchQuery := strings.TrimSpace(input[colonIdx+1:])

	repoFilters := make(map[string]bool)
	for _, ch := range prefix {
		if repo, ok := repoFilterChars[ch]; ok {
			repoFilters[repo] = true
		}
	}

	if len(repoFilters) == 0 {
		return nil, input
	}

	return repoFilters, searchQuery
}

// formatRepoFilters returns a human-readable string of active repo filters
func formatRepoFilters(filters map[string]bool) string {
	if len(filters) == 0 {
		return ""
	}
	var repos []string

	for _, repo := range []string{"core", "extra", "multilib", "aur"} {
		if filters[repo] {
			repos = append(repos, repo)
		}
	}
	return strings.Join(repos, "+")
}

// parseUninstallFilter extracts source filters and search query from input for uninstall mode
// Supports 'a:' for AUR/foreign packages and 'l:' for local/official packages
func parseUninstallFilter(input string) (map[string]bool, string) {
	input = strings.TrimSpace(input)

	colonIdx := strings.Index(input, ":")
	if colonIdx == -1 {
		return nil, input
	}

	prefix := strings.ToLower(input[:colonIdx])
	searchQuery := strings.TrimSpace(input[colonIdx+1:])

	sourceFilters := make(map[string]bool)
	for _, ch := range prefix {
		if source, ok := uninstallFilterChars[ch]; ok {
			sourceFilters[source] = true
		}
	}

	if len(sourceFilters) == 0 {
		return nil, input
	}

	return sourceFilters, searchQuery
}

// formatUninstallFilters returns a human-readable string of active uninstall filters
func formatUninstallFilters(filters map[string]bool) string {
	if len(filters) == 0 {
		return ""
	}
	var names []string
	if filters["total"] {
		names = append(names, "total")
	}
	if filters["explicit"] {
		names = append(names, "explicit")
	}
	if filters["foreign"] {
		names = append(names, "foreign")
	}
	if filters["orphan"] {
		names = append(names, "orphan")
	}
	return strings.Join(names, "+")
}

// parseAUROutput parses paru -Ss output for AUR packages
func parseAUROutput(output string) []Package {
	var packages []Package
	lines := strings.Split(output, "\n")

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		if line == "" || strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		repoPkg := strings.SplitN(parts[0], "/", 2)
		if len(repoPkg) != 2 {
			continue
		}

		pkg := Package{
			Source:    repoPkg[0],
			Name:      repoPkg[1],
			Version:   parts[1],
			Installed: strings.Contains(line, "[Installed"),
		}

		if i+1 < len(lines) && (strings.HasPrefix(lines[i+1], " ") || strings.HasPrefix(lines[i+1], "\t")) {
			pkg.Description = strings.TrimSpace(lines[i+1])
		}

		packages = append(packages, pkg)
	}

	return packages
}

func parseSearchOutput(output string) []Package {
	var packages []Package
	lines := strings.Split(output, "\n")

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		if strings.Contains(line, "/") && !strings.HasPrefix(line, " ") || (len(line) > 0 && line[0] >= '0' && line[0] <= '9') {

			fields := strings.Fields(line)
			if len(fields) == 0 {
				continue
			}

			pkgField := ""
			pkgFieldIdx := 0
			for idx, f := range fields {
				if strings.Contains(f, "/") {
					pkgField = f
					pkgFieldIdx = idx
					break
				}
			}
			if pkgField == "" {
				continue
			}

			parts := strings.SplitN(pkgField, "/", 2)
			if len(parts) < 2 {
				continue
			}

			source := parts[0]
			name := parts[1]

			version := ""
			if pkgFieldIdx+1 < len(fields) {
				version = fields[pkgFieldIdx+1]
			}

			installed := strings.Contains(line, "[Installed")

			description := ""
			if i+1 < len(lines) {
				description = strings.TrimSpace(lines[i+1])
				i++
			}

			packages = append(packages, Package{
				Source:      source,
				Name:        name,
				Version:     version,
				Description: description,
				Installed:   installed,
			})
		}
	}

	return packages
}

func countLines(output string) int {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return 0
	}
	return len(lines)
}

// parseSizeToBytes converts a human-readable size (e.g., "10.5 GiB") to bytes
func parseSizeToBytes(size string) int64 {
	size = strings.TrimSpace(size)
	var value float64
	var unit string
	_, _ = fmt.Sscanf(size, "%f %s", &value, &unit)

	unit = strings.ToLower(unit)
	switch {
	case strings.HasPrefix(unit, "kib") || strings.HasPrefix(unit, "kb"):
		return int64(value * 1024)
	case strings.HasPrefix(unit, "mib") || strings.HasPrefix(unit, "mb"):
		return int64(value * 1024 * 1024)
	case strings.HasPrefix(unit, "gib") || strings.HasPrefix(unit, "gb"):
		return int64(value * 1024 * 1024 * 1024)
	case strings.HasPrefix(unit, "tib") || strings.HasPrefix(unit, "tb"):
		return int64(value * 1024 * 1024 * 1024 * 1024)
	default:
		return int64(value)
	}
}

// calculateDirSize walks a directory and returns the total size of all files in bytes.
// It gracefully handles permission errors by skipping inaccessible files.
func calculateDirSize(path string) int64 {
	var size int64
	err := filepath.WalkDir(path, func(_ string, d os.DirEntry, err error) error {
		if err != nil {

			return nil
		}
		if !d.IsDir() {
			info, err := d.Info()
			if err == nil {
				size += info.Size()
			}
		}
		return nil
	})

	if err != nil {
		return 0
	}
	return size
}

// formatBytes converts bytes to human-readable format
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// truncateWithAnsi truncates a string to a visual width, preserving ANSI codes
func truncateWithAnsi(s string, maxWidth int) string {
	var result strings.Builder
	width := 0
	inEscape := false

	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			result.WriteRune(r)
			continue
		}
		if inEscape {
			result.WriteRune(r)
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEscape = false
			}
			continue
		}
		if width >= maxWidth {
			break
		}
		result.WriteRune(r)
		width++
	}

	result.WriteString("\x1b[0m")
	return result.String()
}

// substringAnsi skips the first skipWidth visual characters, preserving ANSI codes
func substringAnsi(s string, skipWidth int) string {
	var result strings.Builder
	width := 0
	inEscape := false

	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			result.WriteRune(r)
			continue
		}
		if inEscape {
			result.WriteRune(r)
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEscape = false
			}
			continue
		}
		if width >= skipWidth {
			result.WriteRune(r)
		}
		width++
	}

	return result.String()
}

// getKeyDisplay returns a string representing the primary key for a binding
func getKeyDisplay(b key.Binding) string {
	keys := b.Keys()
	if len(keys) == 0 {
		return ""
	}
	k := keys[0]
	// Special cases for display
	switch k {
	case "ctrl+c":
		return "^c"
	case "enter":
		return "enter"
	case "esc":
		return "esc"
	case "tab":
		return "tab"
	case " ":
		return "space"
	}
	return k
}

// renderKeyHint formats a label with a keybinding hint.
// If the key exists in the label, it highlights it in [].
// Otherwise, it prepends the key in [].
func renderKeyHint(label string, b key.Binding, style lipgloss.Style) string {
	k := getKeyDisplay(b)
	if k == "" {
		return style.Render(label)
	}

	// Create a segment style that preserves colors/bolding but strips layout AND padding
	// to prevent each segment from adding its own internal gaps.
	segStyle := style.Copy().UnsetWidth().UnsetAlign().Padding(0, 0)

	// Only try to embed if it's a single character
	if len(k) == 1 {
		idx := strings.Index(strings.ToLower(label), strings.ToLower(k))
		if idx != -1 {
			// Found the letter in the word
			before := label[:idx]
			letter := label[idx : idx+1]
			after := label[idx+1:]
			
			// We only render non-empty parts to avoid any potential zero-width rendering artifacts
			var result string
			if before != "" {
				result += segStyle.Render(before)
			}
			result += segStyle.Render("["+letter+"]")
			if after != "" {
				result += segStyle.Render(after)
			}
			return result
		}
	}

	// Not found or not single char, prepend with colon.
	// The colon must be styled to match the background of the segments.
	return segStyle.Render("["+k+"]") + segStyle.Render(":") + segStyle.Render(label)
}

// maintainBackground replaces ANSI reset codes with a sequence that resets but then re-applies the given background color.
func maintainBackground(s string, bgColor lipgloss.Color) string {
	// Get the ANSI sequence for the background color
	bgStyle := lipgloss.NewStyle().Background(bgColor)
	bgSeq := bgStyle.Render(" ")
	// Extract just the escape sequence (everything before the space)
	idx := strings.Index(bgSeq, " ")
	if idx == -1 {
		return s
	}
	bgSeq = bgSeq[:idx]

	// Replace \x1b[0m with \x1b[0m + bgSeq
	return strings.ReplaceAll(s, "\x1b[0m", "\x1b[0m"+bgSeq)
}

// renderScrollbar generates a vertical scrollbar indicator
func renderScrollbar(total, offset, visibleHeight int, activeColor lipgloss.Color, reversed bool) string {
	if total <= visibleHeight || visibleHeight <= 0 {
		return ""
	}

	scrollbarHeight := visibleHeight
	trackStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("236"))
	thumbStyle := lipgloss.NewStyle().Foreground(activeColor)

	// Calculate thumb size and position
	thumbHeight := int(float64(visibleHeight) * (float64(visibleHeight) / float64(total)))
	if thumbHeight < 1 {
		thumbHeight = 1
	}

	maxOffset := total - visibleHeight
	if maxOffset <= 0 {
		return ""
	}

	// Clamp offset to ensure it stays within valid bounds [0, maxOffset]
	if offset > maxOffset {
		offset = maxOffset
	}
	if offset < 0 {
		offset = 0
	}

	thumbStart := int(float64(offset) / float64(maxOffset) * float64(visibleHeight-thumbHeight))

	// Invert for bottom-to-top menus
	if reversed {
		thumbStart = visibleHeight - thumbHeight - thumbStart
	}

	var scrollbar strings.Builder
	for i := 0; i < scrollbarHeight; i++ {
		if i >= thumbStart && i < thumbStart+thumbHeight {
			scrollbar.WriteString(thumbStyle.Render("┃"))
		} else {
			scrollbar.WriteString(trackStyle.Render("│"))
		}
		if i < scrollbarHeight-1 {
			scrollbar.WriteString("\n")
		}
	}

	return scrollbar.String()
}
