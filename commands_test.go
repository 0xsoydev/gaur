package main

import (
	"reflect"
	"testing"
)

func TestBuildAURCommand(t *testing.T) {
	tests := []struct {
		name           string
		helper         string
		action         string
		packages       []string
		installFlags   string
		uninstallFlags string
		expected       []string
	}{
		{
			name:     "paru install single",
			helper:   "paru",
			action:   "install",
			packages: []string{"vim"},
			expected: []string{"paru", "-S", "vim"},
		},
		{
			name:     "yay install multiple",
			helper:   "yay",
			action:   "install",
			packages: []string{"vim", "neovim"},
			expected: []string{"yay", "-S", "vim", "neovim"},
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
		{
			name:           "custom install flags",
			helper:         "yay",
			action:         "install",
			packages:       []string{"vim"},
			installFlags:   "--needed --noconfirm",
			expected:       []string{"yay", "-S", "--needed", "--noconfirm", "vim"},
		},
		{
			name:           "custom remove flags",
			helper:         "paru",
			action:         "remove",
			packages:       []string{"vim"},
			uninstallFlags: "-Rcns",
			expected:       []string{"paru", "-Rcns", "vim"},
		},
		{
			name:     "paru sync",
			helper:   "paru",
			action:   "sync",
			expected: []string{"paru", "-Sy"},
		},
		{
			name:     "yay full-update",
			helper:   "yay",
			action:   "full-update",
			expected: []string{"yay", "-Syu"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				Commands: CommandConfig{
					AurHelper:      tt.helper,
					InstallFlags:   tt.installFlags,
					UninstallFlags: tt.uninstallFlags,
				},
			}
			got := BuildAURCommand(config, tt.action, tt.packages...)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("%s: BuildAURCommand() = %v, want %v", tt.name, got, tt.expected)
			}
		})
	}
}
