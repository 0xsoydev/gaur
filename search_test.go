package main

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

func TestSearchProgressIndicator(t *testing.T) {
	// Initialize model in install mode
	m := initialModel(modeInstall, DefaultConfig())
	
	// Initially no search
	if m.searchingAUR {
		t.Error("Initially searchingAUR should be false")
	}
	if m.searchStatus != "" {
		t.Errorf("Initially searchStatus should be empty, got %q", m.searchStatus)
	}

	// 1. Simulate user typing a search query
	// The minSearchQueryLen is 2
	m.textInput.SetValue("vim")
	m.performFiltering()

	if !m.searchingAUR {
		t.Error("After typing query >= minSearchQueryLen, searchingAUR should be true")
	}
	if !strings.Contains(m.searchStatus, "Searching AUR for \"vim\"") {
		t.Errorf("Unexpected search status: %q", m.searchStatus)
	}

	// 2. Verify that while searchingAUR is true, typing more doesn't trigger new search immediately
	// but updates the searchStatus text (the searchTerm)
	m.textInput.SetValue("vim-git")
	m.performFiltering()
	
	if !strings.Contains(m.searchStatus, "Searching AUR for \"vim-git\"") {
		t.Errorf("Search status should update to new query: %q", m.searchStatus)
	}
	
	// Set it back to "vim" to test completion for the current query
	m.textInput.SetValue("vim")
	m.performFiltering()

	// 3. Simulate AUR search results arriving for the query "vim"
	// Ensure we match what's in m.lastAURQuery
	msg := aurSearchMsg{
		packages: []Package{
			{Source: "aur", Name: "vim-git", Version: "1.0"},
		},
		query:     "vim",
		timeTaken: 1200 * time.Millisecond,
		err:       nil,
	}

	resModel, _ := m.Update(msg)
	m = resModel.(*model)

	if m.searchingAUR {
		t.Error("After aurSearchMsg with matching query, searchingAUR should be false")
	}
	if !strings.Contains(m.searchStatus, "AUR search complete") {
		t.Errorf("Status should indicate completion: %q", m.searchStatus)
	}
	if !strings.Contains(m.searchStatus, "1.20 seconds") {
		t.Errorf("Status should include time taken: %q", m.searchStatus)
	}

	// 4. Test search error state
	m.searchingAUR = true // manually set to simulate start
	// Actually msg.err is an error interface
	msgErr := aurSearchMsg{
		err:   &mockError{msg: "network error"},
		query: "vim",
	}

	resModel, _ = m.Update(msgErr)
	m = resModel.(*model)

	if m.searchingAUR {
		t.Error("After error, searchingAUR should be false")
	}
	if !m.searchError {
		t.Error("searchError should be true after error")
	}
	if !strings.Contains(m.searchStatus, "AUR search failed") {
		t.Errorf("Status should indicate failure: %q", m.searchStatus)
	}
}

type mockError struct {
	msg string
}

func (e *mockError) Error() string {
	return e.msg
}

func TestSearchStateIsolationOnModeSwitch(t *testing.T) {
	m := initialModel(modeInstall, DefaultConfig())
	m.textInput.SetValue("vim")
	m.performFiltering()

	if !m.searchingAUR {
		t.Fatal("Should be searching AUR")
	}

	// 1. Simulate switching to Uninstall mode via key press
	msgUninstall := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("u")} // Assuming 'u' is UninstallMode in default keys
	// Wait, I should use the actual key binding
	m.keys.UninstallMode = key.NewBinding(key.WithKeys("u"))
	
	resModel, _ := m.Update(msgUninstall)
	m = resModel.(*model)

	if m.mode != modeUninstall {
		t.Fatalf("Expected modeUninstall, got %v", m.mode)
	}
	if m.searchingAUR {
		t.Error("searchingAUR should be false after mode switch")
	}
	if m.searchStatus != "" {
		t.Errorf("searchStatus should be empty after mode switch, got %q", m.searchStatus)
	}

	// 2. Switch back to Install mode
	m.keys.InstallMode = key.NewBinding(key.WithKeys("i"))
	msgInstall := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("i")}
	
	resModel, _ = m.Update(msgInstall)
	m = resModel.(*model)

	if m.mode != modeInstall {
		t.Fatalf("Expected modeInstall, got %v", m.mode)
	}
	if m.searchingAUR {
		t.Error("searchingAUR should still be false until filtering is performed")
	}
}

func TestSpinnerIntegration(t *testing.T) {
	m := initialModel(modeInstall, DefaultConfig())
	m.searchingAUR = true
	m.searchStatus = "Searching..."

	// Test that spinner tick updates the model's spinner
	tick := spinner.TickMsg{Time: time.Now()}
	resModel, cmd := m.Update(tick)
	m = resModel.(*model)

	if cmd == nil {
		t.Error("Spinner tick should return a new tick command")
	}
}
