package main

import (
	"testing"
)

func TestParsePaccacheDryRun(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected string
	}{
		{
			name:     "no candidates",
			output:   "==> no candidate packages found for pruning",
			expected: "0 B",
		},
		{
			name:     "success mb",
			output:   "\n==> finished dry run: 78 candidates (disk space saved: 782.43 MiB)\n",
			expected: "782.43 MiB",
		},
		{
			name:     "success kb",
			output:   "\n==> finished dry run: 1 candidates (disk space saved: 141.83 KiB)\n",
			expected: "141.83 KiB",
		},
		{
			name:     "empty output",
			output:   "",
			expected: "0 B",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := parsePaccacheDryRun(tt.output)
			if actual != tt.expected {
				t.Errorf("parsePaccacheDryRun() = %v, want %v", actual, tt.expected)
			}
		})
	}
}
