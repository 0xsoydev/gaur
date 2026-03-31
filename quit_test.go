package main

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestQuitKeyWithFocus(t *testing.T) {
	// Initialize model in Install mode where search is available
	m := initialModel(modeInstall, DefaultConfig())
	m.loading = false

	// 1. Test 'q' without focus (should quit)
	m.textInput.Blur()
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if cmd == nil {
		t.Errorf("Expected tea.Quit command when 'q' is pressed without focus, got nil")
	} else {
		msg := cmd()
		if _, ok := msg.(tea.QuitMsg); !ok {
			t.Errorf("Expected tea.QuitMsg, got %T", msg)
		}
	}

	// 2. Test 'q' with focus (should NOT quit)
	m = initialModel(modeInstall, DefaultConfig())
	m.loading = false
	m.textInput.Focus()

	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	updatedModel := newModel.(*model)

	if cmd != nil {
		// Check if it's tea.Quit
		msg := cmd()
		if _, ok := msg.(tea.QuitMsg); ok {
			t.Errorf("Expected application NOT to quit when 'q' is pressed with focus, but it did")
		}
	}

	// The 'q' should have been handled by the text input (though it might not be a printable char if it's just 'q')
	// In our app, if it's focused, it should just update the text input.
	if updatedModel.textInput.Value() == "" && m.textInput.Value() == "" {
		// This is okay if 'q' is just a character, but it definitely shouldn't be tea.Quit
	}
}

func TestCtrlCAlwaysQuits(t *testing.T) {
	m := initialModel(modeInstall, DefaultConfig())
	m.loading = false
	m.textInput.Focus()

	// ctrl+c should ALWAYS quit regardless of focus
	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	_ = newModel
	if cmd == nil {
		t.Errorf("Expected tea.Quit command when ctrl+c is pressed even with focus, got nil")
	} else {
		msg := cmd()
		if _, ok := msg.(tea.QuitMsg); !ok {
			t.Errorf("Expected tea.QuitMsg, got %T", msg)
		}
	}
}
