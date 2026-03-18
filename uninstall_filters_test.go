package main

import (
	"testing"
)

func TestUninstallFilters(t *testing.T) {
	installed := []Package{
		{Source: "core", Name: "linux", Explicit: true, Orphan: false},
		{Source: "extra", Name: "vim", Explicit: true, Orphan: false},
		{Source: "aur", Name: "google-chrome", Explicit: true, Orphan: false},
		{Source: "extra", Name: "libpng", Explicit: false, Orphan: false},
		{Source: "aur", Name: "yay-git", Explicit: false, Orphan: true},
	}

	m := initialModel(modeUninstall, DefaultConfig())
	m.installed = installed

	tests := []struct {
		name          string
		query         string
		expectedCount int
	}{
		{
			name:          "Orphan filter",
			query:         "o:",
			expectedCount: 1,
		},
		{
			name:          "Foreign/AUR filter (f:)",
			query:         "f:",
			expectedCount: 2,
		},
		{
			name:          "Foreign/AUR filter (a:)",
			query:         "a:",
			expectedCount: 2,
		},
		{
			name:          "Explicit filter (e:)",
			query:         "e:",
			expectedCount: 3,
		},
		{
			name:          "Explicit filter (l:)",
			query:         "l:",
			expectedCount: 3,
		},
		{
			name:          "Combined Orphan Foreign",
			query:         "of:",
			expectedCount: 2,
		},
		{
			name:          "Filter + Search",
			query:         "e:linux",
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m.filterInstalledPackages(tt.query)
			if len(m.filteredInstalled) != tt.expectedCount {
				t.Errorf("expected %d packages, got %d for query %q", tt.expectedCount, len(m.filteredInstalled), tt.query)
			}
		})
	}
}
