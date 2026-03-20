package main

import (
	"fmt"
	"strings"
	"testing"

)

// TestLayoutRobustness covers dimensions, borders, footers, search bar positioning, 
// and internal content integrity (like unit wrapping).
func TestLayoutRobustness(t *testing.T) {
	config := DefaultConfig()
	
	terminalSizes := []struct {
		w, h int
	}{
		{80, 24},  // Standard
		{120, 40}, // Large
		{60, 15},  // Small/Tight
	}

	modes := []viewMode{
		modeInstall,
		modeDashboard,
		modeUninstall,
		modeUpdate,
		modeUpdateSelective,
		modeCacheMenu,
		modeCacheSelective,
		modeSettings,
	}

	modeNames := map[viewMode]string{
		modeInstall:         "Install",
		modeDashboard:       "Dashboard",
		modeUninstall:       "Uninstall",
		modeUpdate:          "Update",
		modeUpdateSelective: "UpdateSelective",
		modeCacheMenu:       "CacheMenu",
		modeCacheSelective:  "CacheSelective",
		modeSettings:        "Settings",
	}

	for _, size := range terminalSizes {
		for _, mode := range modes {
			t.Run(fmt.Sprintf("%s_%dx%d", modeNames[mode], size.w, size.h), func(t *testing.T) {
				m := initialModel(mode, config)
				m.width = size.w
				m.height = size.h
				
				// Inline Mock Data Setup
				m.loading = false
				m.packages = []Package{
					{Name: "git", Source: "core", Version: "2.44.0", Size: "10.00 MiB"},
					{Name: "vim", Source: "extra", Version: "9.1.0", Size: "20.00 MiB"},
					{Name: "gaur-git", Source: "aur", Version: "1.0.0", Size: "1.50 MiB"},
				}
				m.filtered = m.packages
				m.installed = m.packages
				m.filteredInstalled = m.packages
				m.pendingUpdates = m.packages
				m.dashboard = DashboardData{
					TotalPackages: 1000,
					TotalSize: "5.20 GiB",
					DiskTotal: "500.00 GiB",
					DiskUsed: "200.00 GiB",
					DiskUsedPercent: 0.4,
					CleanerSize: "2.10 GiB",
					RepoDistribution: map[string]int{"core": 200, "extra": 500, "multilib": 50, "aur": 250},
					TopPackages: []PackageSize{{Name: "test-heavy", Size: "1.20 GiB"}},
					AllCacheHogs: []PackageSize{{Name: "hog1", Size: "50.00 MiB", SizeBytes: 50*1024*1024}},
				}
				
				if mode == modeSettings {
					m.previousMode = modeDashboard
				}

				view := m.View()
				lines := strings.Split(view, "\n")

				// 1. DIMENSIONS
				if len(lines) != size.h {
					t.Errorf("Height mismatch: expected %d, got %d", size.h, len(lines))
				}

				// 2. SEARCH BAR POSITIONING
				if mode == modeInstall || mode == modeUninstall || mode == modeUpdateSelective || mode == modeCacheSelective {
					if !strings.Contains(view, "> ") {
						t.Errorf("Search bar prompt ('> ') not found in %s mode", modeNames[mode])
					}
				}

				// 3. UNIT WRAPPING
				for i, line := range lines {
					plain := strings.TrimSpace(stripAnsiLoc(line))
					if plain == "MiB" || plain == "GiB" || plain == "B" {
						t.Errorf("Unit %q wrapped at L%d", plain, i)
					}
				}

				// 4. FOOTER
				lastLine := stripAnsiLoc(lines[len(lines)-1])
				keywords := []string{"quit", "search", "install", "update", "dash"}
				foundKeyword := false
				for _, kw := range keywords {
					if strings.Contains(strings.ToLower(lastLine), kw) {
						foundKeyword = true
						break
					}
				}
				if !foundKeyword {
					t.Errorf("Footer missing on last line. Got: %q", lastLine)
				}
			})
		}
	}
}

func stripAnsiLoc(s string) string {
	var result strings.Builder
	inEscape := false
	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEscape = false
			}
			continue
		}
		result.WriteRune(r)
	}
	return result.String()
}
