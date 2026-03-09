package main

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestIsValidPackageName tests the package name validation function
func TestIsValidPackageName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"valid simple", "vim", true},
		{"valid with hyphen", "my-package", true},
		{"valid with underscore", "my_package", true},
		{"valid with plus", "gcc11", true}, // doesn't contain plus but still valid
		{"valid with dot", "my.package", true},
		{"valid with at", "host@example", true},
		{"valid mixed", "my-package_123.test", true},
		{"empty string", "", false},
		{"space", "my package", false},
		{"special char", "my$package", false},
		{"bracket", "my[package]", false},
		{"backtick", "my`package`", false},
		{"newline", "my\npackage", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidPackageName(tt.input)
			if result != tt.expected {
				t.Errorf("isValidPackageName(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// TestSanitizePackageNames tests the package name sanitization
func TestSanitizePackageNames(t *testing.T) {
	tests := []struct {
		name         string
		input        []string
		wantValid    []string
		wantAllValid bool
	}{
		{
			name:         "all valid",
			input:        []string{"vim", "neovim", "htop"},
			wantValid:    []string{"vim", "neovim", "htop"},
			wantAllValid: true,
		},
		{
			name:         "some invalid",
			input:        []string{"vim", "bad name", "neovim"},
			wantValid:    []string{"vim", "neovim"},
			wantAllValid: false,
		},
		{
			name:         "all invalid",
			input:        []string{"", "bad name", "test test"},
			wantValid:    nil, // no valid packages returns nil slice
			wantAllValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, allValid := sanitizePackageNames(tt.input)
			if !reflect.DeepEqual(valid, tt.wantValid) {
				t.Errorf("sanitizePackageNames(%v) returned %v, want %v", tt.input, valid, tt.wantValid)
			}
			if allValid != tt.wantAllValid {
				t.Errorf("sanitizePackageNames(%v) allValid = %v, want %v", tt.input, allValid, tt.wantAllValid)
			}
		})
	}
}

// TestParseRepoFilter tests the repository filter parsing
func TestParseRepoFilter(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantFilters map[string]bool
		wantQuery   string
	}{
		{
			name:        "no filter",
			input:       "vim",
			wantFilters: nil,
			wantQuery:   "vim",
		},
		{
			name:        "single repo filter",
			input:       "a:vim",
			wantFilters: map[string]bool{"aur": true},
			wantQuery:   "vim",
		},
		{
			name:        "multiple repo filters",
			input:       "ae:firefox",
			wantFilters: map[string]bool{"aur": true, "extra": true},
			wantQuery:   "firefox",
		},
		{
			name:        "all repos filter",
			input:       "acem:package",
			wantFilters: map[string]bool{"aur": true, "core": true, "extra": true, "multilib": true},
			wantQuery:   "package",
		},
		{
			name:        "filter with spaces in query",
			input:       "a:  my package  ",
			wantFilters: map[string]bool{"aur": true},
			wantQuery:   "my package",
		},
		{
			name:        "filter with no query",
			input:       "a:",
			wantFilters: map[string]bool{"aur": true},
			wantQuery:   "",
		},
		{
			name:        "invalid filter char ignored",
			input:       "z:vim",
			wantFilters: nil,
			wantQuery:   "z:vim",
		},
		{
			name:        "mixed invalid and valid",
			input:       "za:vim",
			wantFilters: map[string]bool{"aur": true},
			wantQuery:   "vim",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filters, query := parseRepoFilter(tt.input)
			if !mapsEqual(filters, tt.wantFilters) {
				t.Errorf("parseRepoFilter(%q) returned filters %v, want %v", tt.input, filters, tt.wantFilters)
			}
			if query != tt.wantQuery {
				t.Errorf("parseRepoFilter(%q) returned query %q, want %q", tt.input, query, tt.wantQuery)
			}
		})
	}
}

// TestParseUninstallFilter tests the uninstall filter parsing
func TestParseUninstallFilter(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantFilters map[string]bool
		wantQuery   string
	}{
		{
			name:        "no filter",
			input:       "vim",
			wantFilters: nil,
			wantQuery:   "vim",
		},
		{
			name:        "total filter",
			input:       "t:",
			wantFilters: map[string]bool{"total": true},
			wantQuery:   "",
		},
		{
			name:        "explicit filter",
			input:       "e:firefox",
			wantFilters: map[string]bool{"explicit": true},
			wantQuery:   "firefox",
		},
		{
			name:        "multiple filters",
			input:       "et:package",
			wantFilters: map[string]bool{"total": true, "explicit": true},
			wantQuery:   "package",
		},
		{
			name:        "orphan filter",
			input:       "o:",
			wantFilters: map[string]bool{"orphan": true},
			wantQuery:   "",
		},
		{
			name:        "invalid filter char",
			input:       "z:vim",
			wantFilters: nil,
			wantQuery:   "z:vim",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filters, query := parseUninstallFilter(tt.input)
			if !mapsEqual(filters, tt.wantFilters) {
				t.Errorf("parseUninstallFilter(%q) returned filters %v, want %v", tt.input, filters, tt.wantFilters)
			}
			if query != tt.wantQuery {
				t.Errorf("parseUninstallFilter(%q) returned query %q, want %q", tt.input, query, tt.wantQuery)
			}
		})
	}
}

// TestFormatRepoFilters tests the repo filter formatting
func TestFormatRepoFilters(t *testing.T) {
	tests := []struct {
		name     string
		filters  map[string]bool
		expected string
	}{
		{"empty", nil, ""},
		{"single", map[string]bool{"aur": true}, "aur"},
		{"multiple", map[string]bool{"core": true, "extra": true, "aur": true}, "core+extra+aur"},
		{"all", map[string]bool{"core": true, "extra": true, "multilib": true, "aur": true}, "core+extra+multilib+aur"},
		{"unordered", map[string]bool{"aur": true, "core": true}, "core+aur"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatRepoFilters(tt.filters)
			if result != tt.expected {
				t.Errorf("formatRepoFilters(%v) = %q, want %q", tt.filters, result, tt.expected)
			}
		})
	}
}

// TestFormatUninstallFilters tests the uninstall filter formatting
func TestFormatUninstallFilters(t *testing.T) {
	tests := []struct {
		name     string
		filters  map[string]bool
		expected string
	}{
		{"empty", nil, ""},
		{"total", map[string]bool{"total": true}, "total"},
		{"explicit", map[string]bool{"explicit": true}, "explicit"},
		{"multiple", map[string]bool{"total": true, "orphan": true}, "total+orphan"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatUninstallFilters(tt.filters)
			if result != tt.expected {
				t.Errorf("formatUninstallFilters(%v) = %q, want %q", tt.filters, result, tt.expected)
			}
		})
	}
}

// TestParseSizeToBytes tests size string parsing
func TestParseSizeToBytes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int64
	}{
		{"bytes", "100 B", 100},
		{"kilobytes", "1.5 KB", 1536}, // 1.5 * 1024
		{"megabytes", "2 MiB", 2 * 1024 * 1024},
		{"gigabytes", "1.5 GiB", int64(1.5 * 1024 * 1024 * 1024)},
		{"terabytes", "1 TiB", 1024 * 1024 * 1024 * 1024},
		{"lowercase kb", "10 kb", 10 * 1024},
		{"lowercase mb", "5 mb", 5 * 1024 * 1024},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseSizeToBytes(tt.input)
			if result != tt.expected {
				t.Errorf("parseSizeToBytes(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

// TestCountLines tests line counting
func TestCountLines(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"empty", "", 0},
		{"single line", "hello", 1},
		{"multiple lines", "line1\nline2\nline3", 3},
		{"trailing newline", "line1\nline2\n", 2},
		{"blank lines", "\n\n\n", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := countLines(tt.input)
			if result != tt.expected {
				t.Errorf("countLines(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

// TestFormatBytes tests byte formatting
func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{"0 bytes", 0, "0 B"},
		{"500 bytes", 500, "500 B"},
		{"1 KB", 1024, "1.0 KiB"},
		{"1.5 KB", 1536, "1.5 KiB"},
		{"1 MB", 1024 * 1024, "1.0 MiB"},
		{"1 GB", 1024 * 1024 * 1024, "1.0 GiB"},
		{"1 TB", 1024 * 1024 * 1024 * 1024, "1.0 TiB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatBytes(tt.bytes)
			if result != tt.expected {
				t.Errorf("formatBytes(%d) = %q, want %q", tt.bytes, result, tt.expected)
			}
		})
	}
}

// TestComputeMatchIndices tests fuzzy match index calculation
func TestComputeMatchIndices(t *testing.T) {
	tests := []struct {
		name        string
		pkg         Package
		query       string
		wantIndices []int
	}{
		{
			name:        "exact match at start",
			pkg:         Package{Source: "aur", Name: "vim"},
			query:       "vim",
			wantIndices: []int{4, 5, 6}, // "aur/vim" - indices 4,5,6 match
		},
		{
			name:        "consecutive substring",
			pkg:         Package{Source: "extra", Name: "firefox"},
			query:       "fire",
			wantIndices: []int{6, 7, 8, 9}, // "extra/firefox" - "fire" at positions
		},
		{
			name:        "fuzzy match",
			pkg:         Package{Source: "core", Name: "bash"},
			query:       "bsh",
			wantIndices: []int{5, 7, 8}, // "core/bash": b(5), s(7), h(8)
		},
		{
			name:        "no match",
			pkg:         Package{Source: "aur", Name: "vim"},
			query:       "xyz",
			wantIndices: []int{},
		},
		{
			name:        "case insensitive",
			pkg:         Package{Source: "aur", Name: "VIM"},
			query:       "vim",
			wantIndices: []int{4, 5, 6}, // should match case-insensitively
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := computeMatchIndices(tt.pkg, tt.query)
			if !intsEqual(result, tt.wantIndices) {
				t.Errorf("computeMatchIndices(%v, %q) = %v, want %v", tt.pkg, tt.query, result, tt.wantIndices)
			}
		})
	}
}

// TestComputeAllMatchIndices tests the batch match index computation
func TestComputeAllMatchIndices(t *testing.T) {
	packages := []Package{
		{Source: "aur", Name: "vim"},
		{Source: "extra", Name: "firefox"},
		{Source: "core", Name: "bash"},
	}
	query := "v"

	result := computeAllMatchIndices(packages, query)

	// Should have entries for packages that match
	if len(result) == 0 {
		t.Error("Expected some matches for query 'v'")
	}

	// Check that indices are within bounds
	for idx, indices := range result {
		if idx < 0 || idx >= len(packages) {
			t.Errorf("Invalid package index: %d", idx)
		}
		for _, charIdx := range indices {
			pkgStr := packages[idx].Source + "/" + packages[idx].Name
			if charIdx < 0 || charIdx >= len(pkgStr) {
				t.Errorf("Index %d out of range for package string %q", charIdx, pkgStr)
			}
		}
	}
}

// TestParseAUROutput tests AUR output parsing
func TestParseAUROutput(t *testing.T) {
	sampleOutput := `aur/yay 12.5.7-1
 AUR helper written in Go
aur/paru 1.9.3-1
 Another AUR helper
`

	result := parseAUROutput(sampleOutput)

	if len(result) != 2 {
		t.Errorf("Expected 2 AUR packages, got %d", len(result))
	}

	if len(result) > 0 {
		if result[0].Source != "aur" || result[0].Name != "yay" {
			t.Errorf("First package incorrect: %+v", result[0])
		}
		if result[0].Description != "AUR helper written in Go" {
			t.Errorf("Description mismatch: %q", result[0].Description)
		}
	}
}

// TestParseSearchOutput tests search output parsing
func TestParseSearchOutput(t *testing.T) {
	sampleOutput := `1 aur/yay 12.5.7-1 [+2480 ~39.12]
 AUR helper written in Go
2 extra/firefox 120.0-1
 Popular web browser
core/linux 6.5.3-1 [+12] [-]
 Linux kernel
`

	result := parseSearchOutput(sampleOutput)

	// Should parse all three packages
	if len(result) != 3 {
		t.Errorf("Expected 3 packages, got %d", len(result))
	}

	// Check first package
	if len(result) > 0 {
		if result[0].Source != "aur" || result[0].Name != "yay" {
			t.Errorf("First package incorrect: %+v", result[0])
		}
		if result[0].Version != "12.5.7-1" {
			t.Errorf("Version mismatch: %q", result[0].Version)
		}
	}
}

// TestGetThemeByName tests theme lookup
func TestGetThemeByName(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		wantOK bool
	}{
		{"exact match", "Catppuccin Mocha", true},
		{"lowercase", "catppuccin mocha", true},
		{"no spaces", "catppuccinmocha", true},
		{"hyphens", "catppuccin-mocha", true},
		{"unknown", "Nonexistent Theme", false},
		{"partial", "cat", false}, // too short, not a theme
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tp, ok := getThemeByName(tt.input)
			if ok != tt.wantOK {
				t.Errorf("getThemeByName(%q) returned ok=%v, want %v", tt.input, ok, tt.wantOK)
				return
			}
			if ok && tp != themeCatppuccinMocha && !strings.Contains(strings.ToLower(tt.input), "catppuccin") {
				// For non-catppuccin themes, just check we got some theme
				if tp < themeBasic || tp > themeTokyonightStorm {
					t.Errorf("getThemeByName returned invalid theme type: %d", tp)
				}
			}
		})
	}
}

// TestListThemes tests theme listing
func TestListThemes(t *testing.T) {
	themes := listThemes()
	if len(themes) == 0 {
		t.Fatal("listThemes returned empty list")
	}

	// Should have all 11 themes
	expectedCount := 11
	if len(themes) != expectedCount {
		t.Logf("Expected %d themes, got %d. Themes: %v", expectedCount, len(themes), themes)
	}

	// Check some known themes
	knownThemes := []string{"Catppuccin Frappe", "Dracula", "Solarized Dark"}
	for _, kt := range knownThemes {
		found := false
		for _, theme := range themes {
			if theme == kt {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Known theme %q not found in list", kt)
		}
	}
}

// Helper function to compare maps
func mapsEqual(a, b map[string]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if bv, ok := b[k]; !ok || bv != v {
			return false
		}
	}
	return true
}

// Helper function to compare int slices
func intsEqual(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if b[i] != v {
			return false
		}
	}
	return true
}

// Benchmark Fuzzy Filter (optional performance test)
func BenchmarkFuzzyFilter(b *testing.B) {
	packages := []Package{
		{Source: "aur", Name: "vim"},
		{Source: "extra", Name: "firefox"},
		{Source: "core", Name: "bash"},
		// ... could add more
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = fuzzyFilter(packages, "vim")
	}
}

// ============================================================================
// Test helpers for bubbletea model tests
// ============================================================================

// keyMsg creates a tea.KeyMsg for a single rune key (e.g., 'y', 's', 'a')
func keyMsg(key string) tea.KeyMsg {
	switch key {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
	}
}

// newTestModelUpdate creates a model in modeUpdate state with the given packages loaded
func newTestModelUpdate(packages []Package) model {
	m := initialModel(modeUpdate)
	m.loading = false
	m.width = 120
	m.height = 40
	m.pendingUpdates = packages
	m.updatableAll = packages
	m.confirmScrollOffset = 0
	m.statusMessage = fmt.Sprintf("%d updates available", len(packages))
	return m
}

// testPackages returns a set of test packages for use in tests
func testPackages() []Package {
	return []Package{
		{Source: "core", Name: "linux", Version: "6.5.3-1"},
		{Source: "core", Name: "glibc", Version: "2.38-1"},
		{Source: "extra", Name: "firefox", Version: "120.0-1"},
		{Source: "extra", Name: "vlc", Version: "3.0.18-1"},
		{Source: "aur", Name: "yay", Version: "12.5.7-1"},
	}
}

// ============================================================================
// modeUpdate key handling tests
// ============================================================================

// TestUpdateModeYKeyExecutesUpdate tests that pressing 'y' in modeUpdate
// with pending updates triggers the update execution.
func TestUpdateModeYKeyExecutesUpdate(t *testing.T) {
	m := newTestModelUpdate(testPackages())

	result, cmd := m.Update(keyMsg("y"))
	resultModel := result.(model)

	if resultModel.statusMessage != "Running system update..." {
		t.Errorf("Expected status 'Running system update...', got %q", resultModel.statusMessage)
	}
	if cmd == nil {
		t.Error("Expected a command to be returned for update execution")
	}
}

// TestUpdateModeUpperYKeyExecutesUpdate tests that pressing 'Y' in modeUpdate
// also triggers the update execution.
func TestUpdateModeUpperYKeyExecutesUpdate(t *testing.T) {
	m := newTestModelUpdate(testPackages())

	result, cmd := m.Update(keyMsg("Y"))
	resultModel := result.(model)

	if resultModel.statusMessage != "Running system update..." {
		t.Errorf("Expected status 'Running system update...', got %q", resultModel.statusMessage)
	}
	if cmd == nil {
		t.Error("Expected a command to be returned for update execution")
	}
}

// TestUpdateModeEnterKeyExecutesUpdate tests that pressing enter in modeUpdate
// with pending updates triggers the update directly (no confirmation dialog).
func TestUpdateModeEnterKeyExecutesUpdate(t *testing.T) {
	m := newTestModelUpdate(testPackages())

	result, cmd := m.Update(keyMsg("enter"))
	resultModel := result.(model)

	if resultModel.statusMessage != "Running system update..." {
		t.Errorf("Expected status 'Running system update...', got %q", resultModel.statusMessage)
	}
	if cmd == nil {
		t.Error("Expected a command to be returned for update execution")
	}
	// Should NOT show confirmation dialog
	if resultModel.showConfirmation {
		t.Error("Should not show confirmation dialog in modeUpdate; update should execute directly")
	}
}

// TestUpdateModeAKeyExecutesUpdate tests that pressing 'a' in modeUpdate
// triggers the update directly.
func TestUpdateModeAKeyExecutesUpdate(t *testing.T) {
	m := newTestModelUpdate(testPackages())

	result, cmd := m.Update(keyMsg("a"))
	resultModel := result.(model)

	if resultModel.statusMessage != "Running system update..." {
		t.Errorf("Expected status 'Running system update...', got %q", resultModel.statusMessage)
	}
	if cmd == nil {
		t.Error("Expected a command to be returned for update execution")
	}
}

// TestUpdateModeSKeyEntersSelectionMode tests that pressing 's' in modeUpdate
// switches to modeUpdateSelect.
func TestUpdateModeSKeyEntersSelectionMode(t *testing.T) {
	m := newTestModelUpdate(testPackages())

	result, cmd := m.Update(keyMsg("s"))
	resultModel := result.(model)

	if resultModel.mode != modeUpdateSelect {
		t.Errorf("Expected mode modeUpdateSelect, got %d", resultModel.mode)
	}
	if len(resultModel.filtered) != len(testPackages()) {
		t.Errorf("Expected filtered to have %d packages, got %d", len(testPackages()), len(resultModel.filtered))
	}
	if resultModel.selectedIndex != 0 {
		t.Errorf("Expected selectedIndex 0, got %d", resultModel.selectedIndex)
	}
	// Should trigger package info loading for first item
	if !resultModel.loadingInfo {
		t.Error("Expected loadingInfo to be true after entering selection mode")
	}
	if resultModel.pendingInfoPackage != "linux" {
		t.Errorf("Expected pendingInfoPackage 'linux', got %q", resultModel.pendingInfoPackage)
	}
	if cmd == nil {
		t.Error("Expected a debounce command for package info loading")
	}
}

// TestUpdateModeYKeyNoEffectWhenLoading tests that y/enter/a do nothing
// when the model is still loading.
func TestUpdateModeYKeyNoEffectWhenLoading(t *testing.T) {
	m := newTestModelUpdate(testPackages())
	m.loading = true

	result, _ := m.Update(keyMsg("y"))
	resultModel := result.(model)

	// Status should not change since the key should have no effect
	if resultModel.statusMessage == "Running system update..." {
		t.Error("Should not trigger update while loading")
	}
}

// TestUpdateModeYKeyNoEffectWhenNoUpdates tests that y/enter/a do nothing
// when there are no pending updates.
func TestUpdateModeYKeyNoEffectWhenNoUpdates(t *testing.T) {
	m := newTestModelUpdate(nil) // No updates
	m.statusMessage = "System is up to date!"

	result, _ := m.Update(keyMsg("y"))
	resultModel := result.(model)

	if resultModel.statusMessage == "Running system update..." {
		t.Error("Should not trigger update when no pending updates")
	}
}

// TestUpdateModeSKeyNoEffectWhenLoading tests that s key does nothing
// when the model is still loading.
func TestUpdateModeSKeyNoEffectWhenLoading(t *testing.T) {
	m := newTestModelUpdate(testPackages())
	m.loading = true

	result, _ := m.Update(keyMsg("s"))
	resultModel := result.(model)

	if resultModel.mode != modeUpdate {
		t.Errorf("Should remain in modeUpdate when loading, got mode %d", resultModel.mode)
	}
}

// ============================================================================
// modeUpdate scroll tests
// ============================================================================

// TestUpdateModeDownKeyScrolls tests that down/j key scrolls the package list
// in modeUpdate when packages are loaded.
func TestUpdateModeDownKeyScrolls(t *testing.T) {
	m := newTestModelUpdate(testPackages())
	m.height = 12 // small enough so maxVisible < 5
	m.confirmScrollOffset = 0

	result, cmd := m.Update(keyMsg("down"))
	resultModel := result.(model)

	if resultModel.confirmScrollOffset != 1 {
		t.Errorf("Expected confirmScrollOffset 1 after scrolling down, got %d", resultModel.confirmScrollOffset)
	}
	if cmd != nil {
		t.Error("Expected no command from scrolling")
	}
}

// TestUpdateModeJKeyScrolls tests that j key also scrolls down.
func TestUpdateModeJKeyScrolls(t *testing.T) {
	m := newTestModelUpdate(testPackages())
	m.height = 12
	m.confirmScrollOffset = 0

	result, cmd := m.Update(keyMsg("j"))
	resultModel := result.(model)

	if resultModel.confirmScrollOffset != 1 {
		t.Errorf("Expected confirmScrollOffset 1 after j key, got %d", resultModel.confirmScrollOffset)
	}
	if cmd != nil {
		t.Error("Expected no command from scrolling")
	}
}

// TestUpdateModeUpKeyScrolls tests that up/k key scrolls the package list up
// in modeUpdate.
func TestUpdateModeUpKeyScrolls(t *testing.T) {
	m := newTestModelUpdate(testPackages())
	m.height = 12
	m.confirmScrollOffset = 3

	result, cmd := m.Update(keyMsg("up"))
	resultModel := result.(model)

	if resultModel.confirmScrollOffset != 2 {
		t.Errorf("Expected confirmScrollOffset 2 after scrolling up, got %d", resultModel.confirmScrollOffset)
	}
	if cmd != nil {
		t.Error("Expected no command from scrolling")
	}
}

// TestUpdateModeKKeyScrolls tests that k key also scrolls up.
func TestUpdateModeKKeyScrolls(t *testing.T) {
	m := newTestModelUpdate(testPackages())
	m.height = 12
	m.confirmScrollOffset = 3

	result, cmd := m.Update(keyMsg("k"))
	resultModel := result.(model)

	if resultModel.confirmScrollOffset != 2 {
		t.Errorf("Expected confirmScrollOffset 2 after k key, got %d", resultModel.confirmScrollOffset)
	}
	if cmd != nil {
		t.Error("Expected no command from scrolling")
	}
}

// TestUpdateModeUpKeyAtTopDoesNothing tests that scrolling up when already
// at the top doesn't go negative.
func TestUpdateModeUpKeyAtTopDoesNothing(t *testing.T) {
	m := newTestModelUpdate(testPackages())
	m.confirmScrollOffset = 0

	result, _ := m.Update(keyMsg("up"))
	resultModel := result.(model)

	if resultModel.confirmScrollOffset != 0 {
		t.Errorf("Expected confirmScrollOffset to stay 0, got %d", resultModel.confirmScrollOffset)
	}
}

// TestUpdateModeScrollDoesNothingWhenLoading tests that scroll keys do nothing
// for modeUpdate when loading.
func TestUpdateModeScrollDoesNothingWhenLoading(t *testing.T) {
	m := newTestModelUpdate(testPackages())
	m.loading = true
	m.confirmScrollOffset = 0

	result, _ := m.Update(keyMsg("down"))
	resultModel := result.(model)

	// When loading, the modeUpdate scroll guard doesn't match, so it falls
	// through to the default down handler which doesn't change confirmScrollOffset
	if resultModel.confirmScrollOffset != 0 {
		t.Errorf("Expected confirmScrollOffset to stay 0 when loading, got %d", resultModel.confirmScrollOffset)
	}
}

// ============================================================================
// updateCheckMsg handler tests
// ============================================================================

// TestUpdateCheckMsgStoresPackagesWithoutConfirmation tests that receiving
// an updateCheckMsg stores packages but does NOT show the confirmation dialog.
func TestUpdateCheckMsgStoresPackagesWithoutConfirmation(t *testing.T) {
	m := initialModel(modeUpdate)
	m.width = 120
	m.height = 40

	pkgs := testPackages()
	result, _ := m.Update(updateCheckMsg{packages: pkgs})
	resultModel := result.(model)

	if resultModel.loading {
		t.Error("Expected loading to be false after receiving packages")
	}
	if len(resultModel.pendingUpdates) != len(pkgs) {
		t.Errorf("Expected %d pending updates, got %d", len(pkgs), len(resultModel.pendingUpdates))
	}
	if len(resultModel.updatableAll) != len(pkgs) {
		t.Errorf("Expected %d updatable packages, got %d", len(pkgs), len(resultModel.updatableAll))
	}
	// Key assertion: should NOT show confirmation dialog
	if resultModel.showConfirmation {
		t.Error("updateCheckMsg should NOT trigger confirmation dialog; packages should be shown inline")
	}
	if resultModel.confirmScrollOffset != 0 {
		t.Errorf("Expected confirmScrollOffset 0, got %d", resultModel.confirmScrollOffset)
	}
	if !strings.Contains(resultModel.statusMessage, "5 updates available") {
		t.Errorf("Expected status to contain '5 updates available', got %q", resultModel.statusMessage)
	}
}

// TestUpdateCheckMsgNoUpdates tests the case when no updates are available.
func TestUpdateCheckMsgNoUpdates(t *testing.T) {
	m := initialModel(modeUpdate)
	m.width = 120
	m.height = 40

	result, _ := m.Update(updateCheckMsg{packages: []Package{}})
	resultModel := result.(model)

	if resultModel.loading {
		t.Error("Expected loading to be false")
	}
	if resultModel.statusMessage != "System is up to date!" {
		t.Errorf("Expected 'System is up to date!', got %q", resultModel.statusMessage)
	}
	if resultModel.updateOutput != "No updates available." {
		t.Errorf("Expected 'No updates available.', got %q", resultModel.updateOutput)
	}
}

// TestUpdateCheckMsgError tests the case when checking for updates fails.
func TestUpdateCheckMsgError(t *testing.T) {
	m := initialModel(modeUpdate)
	m.width = 120
	m.height = 40

	result, _ := m.Update(updateCheckMsg{err: fmt.Errorf("network error")})
	resultModel := result.(model)

	if resultModel.loading {
		t.Error("Expected loading to be false")
	}
	if !strings.Contains(resultModel.statusMessage, "Error checking updates") {
		t.Errorf("Expected error status, got %q", resultModel.statusMessage)
	}
}

// ============================================================================
// modeUpdate View rendering tests
// ============================================================================

// TestUpdateViewShowsLoadingMessage tests that View() shows loading message
// during update check.
func TestUpdateViewShowsLoadingMessage(t *testing.T) {
	m := initialModel(modeUpdate)
	m.width = 120
	m.height = 40

	view := m.View()

	if !strings.Contains(view, "Checking for updates...") {
		t.Error("View should show 'Checking for updates...' when loading")
	}
}

// TestUpdateViewShowsPackageList tests that View() renders the package list
// in the top pane when updates are available.
func TestUpdateViewShowsPackageList(t *testing.T) {
	m := newTestModelUpdate(testPackages())

	view := m.View()

	// Should show package count
	if !strings.Contains(view, "5") {
		t.Error("View should show package count '5'")
	}
	if !strings.Contains(view, "updates are available") {
		t.Error("View should show 'updates are available'")
	}

	// Should show package names
	for _, pkg := range testPackages() {
		if !strings.Contains(view, pkg.Name) {
			t.Errorf("View should show package name %q", pkg.Name)
		}
	}
}

// TestUpdateViewShowsNoUpdatesMessage tests that View() shows appropriate
// message when system is up to date.
func TestUpdateViewShowsNoUpdatesMessage(t *testing.T) {
	m := newTestModelUpdate(nil)
	m.statusMessage = "System is up to date!"

	view := m.View()

	if !strings.Contains(view, "up to date") {
		t.Error("View should show 'up to date' message when no updates")
	}
}

// TestUpdateViewShowsFooterMenu tests that the footer with menu items
// is visible in modeUpdate (not hidden like the old full-screen dialog).
func TestUpdateViewShowsFooterMenu(t *testing.T) {
	m := newTestModelUpdate(testPackages())

	view := m.View()

	// Footer should contain mode shortcuts
	if !strings.Contains(view, "nstall") {
		t.Error("Footer should contain '[i]nstall' text")
	}
	if !strings.Contains(view, "pdate") {
		t.Error("Footer should contain '[u]pdate' text")
	}
	if !strings.Contains(view, "uit") {
		t.Error("Footer should contain '[q]uit' text")
	}
}

// TestUpdateViewDoesNotShowConfirmTitle tests that the "Confirm System Update"
// title is no longer shown.
func TestUpdateViewDoesNotShowConfirmTitle(t *testing.T) {
	m := newTestModelUpdate(testPackages())

	view := m.View()

	if strings.Contains(view, "Confirm System Update") {
		t.Error("View should NOT contain 'Confirm System Update' title; it was removed")
	}
}

// TestUpdateViewShowsActionPrompts tests that action prompts are shown
// in the input line area.
func TestUpdateViewShowsActionPrompts(t *testing.T) {
	m := newTestModelUpdate(testPackages())

	view := m.View()

	if !strings.Contains(view, "update all") {
		t.Error("View should show 'update all' action prompt")
	}
	if !strings.Contains(view, "select") {
		t.Error("View should show 'select' action prompt")
	}
	if !strings.Contains(view, "refresh") {
		t.Error("View should show 'refresh' action prompt")
	}
}

// ============================================================================
// modeUpdateSelect info display tests
// ============================================================================

// TestUpdateSelectShowsPackageInfo tests that modeUpdateSelect displays
// package details in the top pane when info is available.
func TestUpdateSelectShowsPackageInfo(t *testing.T) {
	m := newTestModelUpdate(testPackages())
	m.mode = modeUpdateSelect
	m.filtered = testPackages()
	m.packageInfo = "Name            : linux\nVersion         : 6.5.3-1\nDescription     : The Linux kernel"

	view := m.View()

	if !strings.Contains(view, "Name") {
		t.Error("View should show package info 'Name' field in top pane")
	}
	if !strings.Contains(view, "Description") {
		t.Error("View should show package info 'Description' field in top pane")
	}
}

// TestUpdateSelectShowsLoadingInfo tests that modeUpdateSelect shows
// loading message while package info is being fetched.
func TestUpdateSelectShowsLoadingInfo(t *testing.T) {
	m := newTestModelUpdate(testPackages())
	m.mode = modeUpdateSelect
	m.filtered = testPackages()
	m.loadingInfo = true
	m.infoForPackage = "firefox"

	view := m.View()

	if !strings.Contains(view, "Loading details for firefox") {
		t.Error("View should show 'Loading details for firefox...' in top pane")
	}
}

// TestUpdateSelectShowsDefaultMessage tests that modeUpdateSelect shows
// default message when no package info is loaded yet.
func TestUpdateSelectShowsDefaultMessage(t *testing.T) {
	m := newTestModelUpdate(testPackages())
	m.mode = modeUpdateSelect
	m.filtered = testPackages()
	m.packageInfo = ""
	m.loadingInfo = false

	view := m.View()

	if !strings.Contains(view, "Select a package to see details") {
		t.Error("View should show 'Select a package to see details' in top pane")
	}
}

// TestUpdateSelectShowsTextInput tests that modeUpdateSelect shows
// the text input field for filtering.
func TestUpdateSelectShowsTextInput(t *testing.T) {
	m := newTestModelUpdate(testPackages())
	m.mode = modeUpdateSelect
	m.filtered = testPackages()
	m.textInput.Placeholder = "Filter updates to select..."

	view := m.View()

	// The text input placeholder should be visible
	if !strings.Contains(view, "Filter updates") {
		t.Error("View should show text input placeholder in modeUpdateSelect")
	}
}

// ============================================================================
// Confirmation dialog behavior tests (from modeUpdateSelect)
// ============================================================================

// TestUpdateSelectConfirmationSKeyLoadsInfo tests that pressing 's' in the
// updateConfirmation dialog enters selection mode and loads package info.
func TestUpdateSelectConfirmationSKeyLoadsInfo(t *testing.T) {
	m := newTestModelUpdate(testPackages())
	m.showConfirmation = true
	m.confirmType = confirmUpdate

	result, cmd := m.Update(keyMsg("s"))
	resultModel := result.(model)

	if resultModel.showConfirmation {
		t.Error("Confirmation dialog should be dismissed after pressing 's'")
	}
	if resultModel.mode != modeUpdateSelect {
		t.Errorf("Expected modeUpdateSelect, got %d", resultModel.mode)
	}
	if !resultModel.loadingInfo {
		t.Error("Expected loadingInfo to be true after entering selection from confirmation")
	}
	if cmd == nil {
		t.Error("Expected debounce command for initial package info load")
	}
}

// TestUpdateSelectEnterFromSelectionShowsConfirmation tests that pressing
// enter in modeUpdateSelect with a selected package shows a confirmation
// dialog (not a direct update).
func TestUpdateSelectEnterFromSelectionShowsConfirmation(t *testing.T) {
	m := newTestModelUpdate(testPackages())
	m.mode = modeUpdateSelect
	m.filtered = testPackages()
	m.textInput.Focus()

	result, _ := m.Update(keyMsg("enter"))
	resultModel := result.(model)

	// Should show confirmation dialog for the selected package
	if !resultModel.showConfirmation {
		t.Error("Expected confirmation dialog when pressing enter from modeUpdateSelect")
	}
	if resultModel.confirmType != confirmUpdate {
		t.Errorf("Expected confirmUpdate type, got %d", resultModel.confirmType)
	}
}

// TestUpdateSelectTabMarksPackage tests that tab key marks a package
// in modeUpdateSelect.
func TestUpdateSelectTabMarksPackage(t *testing.T) {
	m := newTestModelUpdate(testPackages())
	m.mode = modeUpdateSelect
	m.filtered = testPackages()
	m.textInput.Focus()

	result, _ := m.Update(keyMsg("tab"))
	resultModel := result.(model)

	if len(resultModel.markedPackages) != 1 {
		t.Errorf("Expected 1 marked package, got %d", len(resultModel.markedPackages))
	}
	// The selected package should be marked (selectedIndex=0 → "linux")
	if !resultModel.markedPackages["linux"] {
		t.Error("Expected 'linux' to be marked")
	}
}

// ============================================================================
// initialModel tests
// ============================================================================

// TestInitialModelUpdateMode tests that initialModel(modeUpdate) creates
// a properly initialized model for update mode.
func TestInitialModelUpdateMode(t *testing.T) {
	m := initialModel(modeUpdate)

	if m.mode != modeUpdate {
		t.Errorf("Expected modeUpdate, got %d", m.mode)
	}
	if !m.loading {
		t.Error("Expected loading to be true initially")
	}
	if m.statusMessage != "Checking for updates..." {
		t.Errorf("Expected status 'Checking for updates...', got %q", m.statusMessage)
	}
	if m.textInput.Placeholder != "Checking for updates..." {
		t.Errorf("Expected placeholder 'Checking for updates...', got %q", m.textInput.Placeholder)
	}
}

