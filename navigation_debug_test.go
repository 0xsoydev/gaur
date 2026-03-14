package main

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNavigationExhaustive(t *testing.T) {
	modes := []viewMode{modeInstall, modeUninstall, modeUpdateSelective, modeCacheSelective}
	
	for _, mode := range modes {
		t.Run(fmt.Sprintf("Mode_%v", mode), func(t *testing.T) {
			m := initialModel(mode)
			m.loading = false
			
			// Mock data
			pkgs := []Package{{Name: "p0"}, {Name: "p1"}, {Name: "p2"}, {Name: "p3"}, {Name: "p4"}, 
			                {Name: "p5"}, {Name: "p6"}, {Name: "p7"}, {Name: "p8"}, {Name: "p9"}, {Name: "p10"}}
			
			if mode == modeUninstall {
				m.installed = pkgs
				m.filteredInstalled = pkgs
			} else {
				m.filtered = pkgs
			}
			
			m.selectedIndex = 5 // Start in the middle
			
			// Test Cases
			tests := []struct {
				key      string
				expected int
				focus    bool
			}{
				{"up", 6, false},
				{"k", 7, false},
				{"down", 6, false},
				{"j", 5, false},
				{"pgup", 10, false}, // 5 + 10 = 15 -> clamped to 10
				{"pgdown", 0, false}, // 10 - 10 = 0
				
				// Focused tests
				{"up", 1, true},
				{"down", 0, true},
				{"pgup", 10, true},
				{"pgdown", 0, true},
			}

			for _, tt := range tests {
				if tt.focus {
					m.textInput.Focus()
				} else {
					m.textInput.Blur()
				}
				
				// Apply key
				msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)}
				if tt.key == "up" { msg = tea.KeyMsg{Type: tea.KeyUp} }
				if tt.key == "down" { msg = tea.KeyMsg{Type: tea.KeyDown} }
				if tt.key == "pgup" { msg = tea.KeyMsg{Type: tea.KeyPgUp} }
				if tt.key == "pgdown" { msg = tea.KeyMsg{Type: tea.KeyPgDown} }
				
				newModel, _ := m.Update(msg)
				m = newModel.(*model)
				
				if m.selectedIndex != tt.expected {
					t.Errorf("Key %s (Focus:%v): selectedIndex = %d, want %d", tt.key, tt.focus, m.selectedIndex, tt.expected)
				}
			}
		})
	}
}
