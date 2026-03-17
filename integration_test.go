package main

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestLoadRepoPackagesWithMock(t *testing.T) {
	// Save original runner and restore after test
	oldRunner := runner
	defer func() { runner = oldRunner }()

	mock := &MockCommandRunner{
		RunFunc: func(name string, args ...string) ([]byte, error) {
			if name == "pacman" && args[0] == "-Sl" {
				return []byte("core pkg1 1.0-1\nextra pkg2 2.0-1\n"), nil
			}
			if name == "pacman" && args[0] == "-Qq" {
				return []byte("pkg1\n"), nil
			}
			return nil, nil
		},
	}
	runner = mock

	cmd := loadRepoPackages()
	msg := cmd()

	repoMsg, ok := msg.(repoPackagesMsg)
	if !ok {
		t.Fatalf("expected repoPackagesMsg, got %T", msg)
	}

	if len(repoMsg.packages) != 2 {
		t.Errorf("expected 2 packages, got %d", len(repoMsg.packages))
	}

	if !repoMsg.packages[0].Installed {
		t.Errorf("expected pkg1 to be installed")
	}
	if repoMsg.packages[1].Installed {
		t.Errorf("expected pkg2 to NOT be installed")
	}
}

func TestSearchAURWithMock(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()

	mock := &MockCommandRunner{
		RunFunc: func(name string, args ...string) ([]byte, error) {
			if name == "paru" && args[0] == "-Ss" {
				return []byte("aur/pkg-aur 1.0-1 (10)\n    Description of pkg-aur\n"), nil
			}
			return nil, nil
		},
	}
	runner = mock

	cfg := DefaultConfig()
	cmd := searchAUR(&cfg, "pkg")
	msg := cmd()

	aurMsg, ok := msg.(aurSearchMsg)
	if !ok {
		t.Fatalf("expected aurSearchMsg, got %T", msg)
	}

	if len(aurMsg.packages) != 1 {
		t.Errorf("expected 1 package, got %d", len(aurMsg.packages))
	}

	if aurMsg.packages[0].Name != "pkg-aur" {
		t.Errorf("expected pkg-aur, got %s", aurMsg.packages[0].Name)
	}
}

func TestSearchAURFailure(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()

	expectedError := "network timeout: could not connect to AUR"
	mock := &MockCommandRunner{
		RunFunc: func(name string, args ...string) ([]byte, error) {
			// Simulate a command failure with output (stderr)
			return []byte(expectedError), fmt.Errorf("exit status 1")
		},
	}
	runner = mock

	cfg := DefaultConfig()
	cmd := searchAUR(&cfg, "pkg")
	msg := cmd()

	aurMsg, ok := msg.(aurSearchMsg)
	if !ok {
		t.Fatalf("expected aurSearchMsg, got %T", msg)
	}

	if aurMsg.err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(aurMsg.err.Error(), expectedError) {
		t.Errorf("expected error to contain %q, got %q", expectedError, aurMsg.err.Error())
	}
}

func TestSearchAURNoResults(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()

	mock := &MockCommandRunner{
		RunFunc: func(name string, args ...string) ([]byte, error) {
			// Simulate no packages found (empty output, success)
			return []byte(""), nil
		},
	}
	runner = mock

	cfg := DefaultConfig()
	cmd := searchAUR(&cfg, "nonexistent-pkg")
	msg := cmd()

	aurMsg, ok := msg.(aurSearchMsg)
	if !ok {
		t.Fatalf("expected aurSearchMsg, got %T", msg)
	}

	if aurMsg.err != nil {
		t.Errorf("expected no error, got %v", aurMsg.err)
	}

	if len(aurMsg.packages) != 0 {
		t.Errorf("expected 0 packages, got %d", len(aurMsg.packages))
	}
}

func TestSearchAURNoResultsExit1(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()

	mock := &MockCommandRunner{
		RunFunc: func(name string, args ...string) ([]byte, error) {
			// Simulate no packages found with exit 1 (common in some helpers)
			return []byte(""), fmt.Errorf("exit status 1")
		},
	}
	runner = mock

	cfg := DefaultConfig()
	cmd := searchAUR(&cfg, "nonexistent-pkg")
	msg := cmd()

	aurMsg, ok := msg.(aurSearchMsg)
	if !ok {
		t.Fatalf("expected aurSearchMsg, got %T", msg)
	}

	// Currently it probably returns an error because of exit status 1
	// We want to verify this behavior
	if aurMsg.err == nil {
		t.Log("AUR helper returned success for no matches (exit 0)")
	} else {
		t.Logf("AUR helper returned error for no matches (exit 1): %v", aurMsg.err)
	}
}

func TestUpdateKeyboardNavigation(t *testing.T) {
	m := initialModel(modeInstall, DefaultConfig())
	m.filtered = make([]Package, 20)
	for i := range m.filtered {
		m.filtered[i] = Package{Name: fmt.Sprintf("pkg%d", i)}
	}
	m.loading = false
	m.selectedIndex = 0

	// In this app, "up" (k) increases index, "down" (j) decreases index
	
	// Test 'k' (Up) -> index 1
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	m = newModel.(*model)
	if m.selectedIndex != 1 {
		t.Errorf("selectedIndex = %d, want 1 after 'k'", m.selectedIndex)
	}

	// Test 'j' (Down) -> index 0
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m = newModel.(*model)
	if m.selectedIndex != 0 {
		t.Errorf("selectedIndex = %d, want 0 after 'j'", m.selectedIndex)
	}

	// Test PgUp -> jumps 10 UP (index increases)
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	m = newModel.(*model)
	if m.selectedIndex != 10 {
		t.Errorf("selectedIndex = %d, want 10 after PgUp", m.selectedIndex)
	}

	// Test PgDown -> jumps 10 DOWN (index decreases)
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	m = newModel.(*model)
	if m.selectedIndex != 0 {
		t.Errorf("selectedIndex = %d, want 0 after PgDown", m.selectedIndex)
	}
}

func TestGetDashboardDataWithMock(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()

	mock := &MockCommandRunner{
		RunFunc: func(name string, args ...string) ([]byte, error) {
			switch name {
			case "paru":
				if args[0] == "-Qq" {
					return []byte("pkg1\npkg2\n"), nil
				}
				if args[0] == "-Ps" {
					return []byte("Total Size: 100 MiB\nMissing from AUR: 0\n"), nil
				}
			case "pacman":
				if args[0] == "-Sl" {
					return []byte("core pkg1 1.0-1 [installed]\n"), nil
				}
			case "grep":
				return []byte(""), nil
			}
			return []byte(""), nil
		},
	}
	runner = mock

	cfg := DefaultConfig()
	cmd := getDashboardData(&cfg)
	msg := cmd()

	dashMsg, ok := msg.(dashboardMsg)
	if !ok {
		t.Fatalf("expected dashboardMsg, got %T", msg)
	}

	if dashMsg.data.TotalPackages != 2 {
		t.Errorf("expected 2 packages, got %d", dashMsg.data.TotalPackages)
	}
}

func TestModeSwitchingShortcuts(t *testing.T) {
	m := initialModel(modeInstall, DefaultConfig())
	m.loading = false

	// 'r' -> Uninstall
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	m = newModel.(*model)
	if m.mode != modeUninstall {
		t.Errorf("mode = %v, want modeUninstall", m.mode)
	}

	// 'u' -> Update
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("u")})
	m = newModel.(*model)
	if m.mode != modeUpdate {
		t.Errorf("mode = %v, want modeUpdate", m.mode)
	}

	// 'n' -> Info/Dashboard
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	m = newModel.(*model)
	if m.mode != modeInstalled {
		t.Errorf("mode = %v, want modeInstalled", m.mode)
	}

	// 'i' -> Install
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("i")})
	m = newModel.(*model)
	if m.mode != modeInstall {
		t.Errorf("mode = %v, want modeInstall", m.mode)
	}
}

func TestMarkingPackages(t *testing.T) {
	m := initialModel(modeInstall, DefaultConfig())
	m.filtered = []Package{{Name: "pkg1"}, {Name: "pkg2"}}
	m.loading = false
	m.selectedIndex = 0

	// Test 'tab' marks pkg1
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = newModel.(*model)
	if !m.markedPackages["pkg1"] {
		t.Errorf("pkg1 should be marked after tab")
	}

	// Test 'tab' on pkg2
	m.selectedIndex = 1
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = newModel.(*model)
	if !m.markedPackages["pkg2"] {
		t.Errorf("pkg2 should be marked after tab")
	}

	// Test 'tab' unmarks pkg1
	m.selectedIndex = 0
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = newModel.(*model)
	if m.markedPackages["pkg1"] {
		t.Errorf("pkg1 should be unmarked after tab")
	}
}

func TestConfirmationFlow(t *testing.T) {
	m := initialModel(modeInstall, DefaultConfig())
	m.filtered = []Package{{Name: "pkg1"}}
	m.loading = false
	m.selectedIndex = 0

	// Enter to show confirmation
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(*model)
	if !m.showConfirmation {
		t.Errorf("confirmation dialog should be shown")
	}
	if m.confirmType != confirmInstall {
		t.Errorf("confirmType = %v, want confirmInstall", m.confirmType)
	}

	// 'n' or 'esc' to cancel
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	m = newModel.(*model)
	if m.showConfirmation {
		t.Errorf("confirmation dialog should be hidden after 'n'")
	}

	// Enter again
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(*model)
	
	// 'y' to confirm
	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	m = newModel.(*model)
	if cmd == nil {
		t.Errorf("expected command after confirming with 'y'")
	}
}

func TestAURHelperIntegration(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()

	tests := []struct {
		helper string
	}{
		{"paru"},
		{"yay"},
	}

	for _, tt := range tests {
		t.Run(tt.helper, func(t *testing.T) {
			var capturedName string
			mock := &MockCommandRunner{
				RunFunc: func(name string, args ...string) ([]byte, error) {
					capturedName = name
					return []byte(""), nil
				},
			}
			runner = mock

			cfg := DefaultConfig()
			cfg.Commands.AurHelper = tt.helper

			// 1. Test search uses helper
			cmd := searchAUR(&cfg, "pkg")
			_ = cmd() // Execute the command
			if capturedName != tt.helper {
				t.Errorf("expected AUR search to use %q, got %q", tt.helper, capturedName)
			}

			// 2. Test update check uses helper
			cmd = checkUpdates(&cfg)
			_ = cmd()
			if capturedName != tt.helper {
				t.Errorf("expected update check to use %q, got %q", tt.helper, capturedName)
			}
		})
	}
}
