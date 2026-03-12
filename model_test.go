package main

import (
	"testing"
)

func TestInitialModel(t *testing.T) {
	tests := []struct {
		name        string
		mode        viewMode
		placeholder string
		status      string
	}{
		{
			name:        "install mode",
			mode:        modeInstall,
			placeholder: "Search packages...",
			status:      "Loading package database...",
		},
		{
			name:        "uninstall mode",
			mode:        modeUninstall,
			placeholder: "Filter installed packages...",
			status:      "Loading installed packages...",
		},
		{
			name:        "update mode",
			mode:        modeUpdate,
			placeholder: "Checking for updates...",
			status:      "Checking for updates...",
		},
		{
			name:        "installed mode",
			mode:        modeInstalled,
			placeholder: "View installed packages",
			status:      "Loading installed packages...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := initialModel(tt.mode)
			if m.mode != tt.mode {
				t.Errorf("initialModel(%v).mode = %v, want %v", tt.mode, m.mode, tt.mode)
			}
			if m.textInput.Placeholder != tt.placeholder {
				t.Errorf("initialModel(%v).textInput.Placeholder = %q, want %q", tt.mode, m.textInput.Placeholder, tt.placeholder)
			}
			if m.statusMessage != tt.status {
				t.Errorf("initialModel(%v).statusMessage = %q, want %q", tt.mode, m.statusMessage, tt.status)
			}
			if !m.loading {
				t.Errorf("initialModel(%v).loading = %v, want true", tt.mode, m.loading)
			}
		})
	}
}

func TestCurrentPackageList(t *testing.T) {
	m := model{
		mode:     modeInstall,
		filtered: []Package{{Name: "pkg1"}, {Name: "pkg2"}},
	}
	list := m.currentPackageList()
	if len(list) != 2 {
		t.Errorf("currentPackageList (install) length = %d, want 2", len(list))
	}

	m.mode = modeUninstall
	m.filteredInstalled = []Package{{Name: "pkg3"}}
	list = m.currentPackageList()
	if len(list) != 1 {
		t.Errorf("currentPackageList (uninstall) length = %d, want 1", len(list))
	}

	m.mode = modeUpdate
	list = m.currentPackageList()
	if list != nil {
		t.Errorf("currentPackageList (update) should be nil, got %v", list)
	}
}

func TestMaxSelectableIndex(t *testing.T) {
	m := model{
		mode:     modeInstall,
		filtered: []Package{{Name: "pkg1"}, {Name: "pkg2"}},
	}
	if m.maxSelectableIndex() != 1 {
		t.Errorf("maxSelectableIndex = %d, want 1", m.maxSelectableIndex())
	}

	m.filtered = []Package{}
	if m.maxSelectableIndex() != 0 {
		t.Errorf("maxSelectableIndex (empty) = %d, want 0", m.maxSelectableIndex())
	}
}

func TestSelectedPackage(t *testing.T) {
	pkg1 := Package{Name: "pkg1"}
	pkg2 := Package{Name: "pkg2"}
	m := model{
		mode:          modeInstall,
		filtered:      []Package{pkg1, pkg2},
		selectedIndex: 1,
	}
	selected := m.selectedPackage()
	if selected == nil || selected.Name != "pkg2" {
		t.Errorf("selectedPackage = %v, want pkg2", selected)
	}

	m.selectedIndex = 5
	selected = m.selectedPackage()
	if selected != nil {
		t.Errorf("selectedPackage (out of bounds) should be nil, got %v", selected)
	}
}
