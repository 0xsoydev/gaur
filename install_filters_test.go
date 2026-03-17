package main

import (
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
)

func TestInstallRepoFilters(t *testing.T) {
	// Setup mock packages
	repos := []Package{
		{Source: "core", Name: "linux", Version: "6.6.1"},
		{Source: "extra", Name: "vim", Version: "9.0.1"},
		{Source: "extra", Name: "neovim", Version: "0.9.4"},
		{Source: "multilib", Name: "lib32-gcc-libs", Version: "13.2.1"},
	}
	aur := []Package{
		{Source: "aur", Name: "visual-studio-code-bin", Version: "1.84.2"},
		{Source: "aur", Name: "yay-git", Version: "12.2.0"},
	}

	m := initialModel(modeInstall, DefaultConfig())
	m.repoPackages = repos
	m.aurPackages = aur
	m.textInput = textinput.New()

	tests := []struct {
		name          string
		query         string
		expectedCount int
		checkSources  []string
	}{
		{
			name:          "Core filter",
			query:         "c:",
			expectedCount: 1,
			checkSources:  []string{"core"},
		},
		{
			name:          "Extra filter",
			query:         "e:",
			expectedCount: 2,
			checkSources:  []string{"extra"},
		},
		{
			name:          "AUR filter",
			query:         "a:",
			expectedCount: 2,
			checkSources:  []string{"aur"},
		},
		{
			name:          "Combined filter (AUR + Extra)",
			query:         "ae:",
			expectedCount: 4,
			checkSources:  []string{"aur", "extra"},
		},
		{
			name:          "Filter + Search (Extra/vim)",
			query:         "e:vim",
			expectedCount: 2, // vim and neovim both contain "vim"
			checkSources:  []string{"extra"},
		},
		{
			name:          "Filter + Search (AUR/yay)",
			query:         "a:yay",
			expectedCount: 1,
			checkSources:  []string{"aur"},
		},
		{
			name:          "Empty query",
			query:         "",
			expectedCount: 0,
		},
		{
			name:          "No matching filter",
			query:         "m:linux",
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m.textInput.SetValue(tt.query)
			m.filterAllPackages(tt.query)

			if len(m.filtered) != tt.expectedCount {
				t.Errorf("expected %d packages, got %d for query %q", tt.expectedCount, len(m.filtered), tt.query)
			}

			// Verify all returned packages belong to the filtered sources
			if len(tt.checkSources) > 0 {
				for _, pkg := range m.filtered {
					found := false
					for _, src := range tt.checkSources {
						if pkg.Source == src {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("package %s with source %s should not be in results for query %q", pkg.Name, pkg.Source, tt.query)
					}
				}
			}
		})
	}
}
