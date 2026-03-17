package main

import (
	"reflect"
	"testing"
)

func TestBuildAURCommand(t *testing.T) {
	tests := []struct {
		name     string
		helper   string
		action   string
		packages []string
		expected []string
	}{
		{
			name:     "paru install single",
			helper:   "paru",
			action:   "install",
			packages: []string{"vim"},
			expected: []string{"paru", "-S", "--noconfirm", "vim"},
		},
		{
			name:     "yay install multiple",
			helper:   "yay",
			action:   "install",
			packages: []string{"vim", "neovim"},
			expected: []string{"yay", "-S", "--noconfirm", "vim", "neovim"},
		},
		{
			name:     "paru remove",
			helper:   "paru",
			action:   "remove",
			packages: []string{"nano"},
			expected: []string{"paru", "-Rns", "nano"},
		},
		{
			name:     "yay remove",
			helper:   "yay",
			action:   "remove",
			packages: []string{"nano"},
			expected: []string{"yay", "-Rns", "nano"},
		},
		{
			name:     "paru update",
			helper:   "paru",
			action:   "update",
			packages: nil,
			expected: []string{"paru", "-Qu"},
		},
		{
			name:     "yay update",
			helper:   "yay",
			action:   "update",
			packages: nil,
			expected: []string{"yay", "-Qu"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				Commands: CommandConfig{
					AurHelper: tt.helper,
				},
			}
			// This function will be implemented later
			got := BuildAURCommand(config, tt.action, tt.packages...)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("BuildAURCommand() = %v, want %v", got, tt.expected)
			}
		})
	}
}
