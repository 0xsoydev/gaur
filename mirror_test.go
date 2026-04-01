package main

import (
	"strings"
	"testing"
)

func TestDefaultMirrorConfig(t *testing.T) {
	cfg := DefaultMirrorConfig()

	if cfg.SortBy != 0 {
		t.Errorf("Expected SortBy=0 (Rate), got %d", cfg.SortBy)
	}
	if cfg.CountryIndex != 0 {
		t.Errorf("Expected CountryIndex=0 (Worldwide), got %d", cfg.CountryIndex)
	}
	if cfg.Latest != 20 {
		t.Errorf("Expected Latest=20, got %d", cfg.Latest)
	}
	if cfg.Protocol != 0 {
		t.Errorf("Expected Protocol=0 (HTTPS), got %d", cfg.Protocol)
	}
	if !cfg.Save {
		t.Error("Expected Save=true")
	}
}

func TestBuildReflectorCommand(t *testing.T) {
	tests := []struct {
		name     string
		cfg      MirrorConfig
		expected []string
		contains []string
	}{
		{
			name: "default config",
			cfg:  DefaultMirrorConfig(),
			expected: []string{
				"--latest", "20",
				"--sort", "rate",
				"--protocol", "https",
				"--save", "/etc/pacman.d/mirrorlist",
			},
		},
		{
			name: "with country filter",
			cfg: MirrorConfig{
				SortBy:       0,
				CountryIndex: 30, // United States
				Latest:       10,
				Protocol:     0,
				Save:         true,
			},
			contains: []string{"--country", "US"},
		},
		{
			name: "sort by age",
			cfg: MirrorConfig{
				SortBy:       1, // Age
				CountryIndex: 0,
				Latest:       5,
				Protocol:     0,
				Save:         true,
			},
			contains: []string{"--sort", "age"},
		},
		{
			name: "http protocol",
			cfg: MirrorConfig{
				SortBy:       0,
				CountryIndex: 0,
				Latest:       20,
				Protocol:     1, // HTTP
				Save:         true,
			},
			contains: []string{"--protocol", "http"},
		},
		{
			name: "both protocols (no filter)",
			cfg: MirrorConfig{
				SortBy:       0,
				CountryIndex: 0,
				Latest:       20,
				Protocol:     2, // Both
				Save:         true,
			},
		},
		{
			name: "no save",
			cfg: MirrorConfig{
				SortBy:       0,
				CountryIndex: 0,
				Latest:       20,
				Protocol:     0,
				Save:         false,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			args := BuildReflectorCommand(tc.cfg)
			argsStr := strings.Join(args, " ")

			// Check expected args
			for i := 0; i < len(tc.expected); i += 2 {
				if i+1 < len(tc.expected) {
					if !strings.Contains(argsStr, tc.expected[i]+" "+tc.expected[i+1]) {
						t.Errorf("Expected args to contain '%s %s', got: %v", tc.expected[i], tc.expected[i+1], args)
					}
				}
			}

			// Check contains
			for _, s := range tc.contains {
				if !strings.Contains(argsStr, s) {
					t.Errorf("Expected args to contain '%s', got: %v", s, args)
				}
			}

			// Check no save
			if !tc.cfg.Save {
				if strings.Contains(argsStr, "--save") {
					t.Error("Should not contain --save when Save is false")
				}
			}

			// Check both protocols (no --protocol flag)
			if tc.cfg.Protocol == 2 {
				if strings.Contains(argsStr, "--protocol") {
					t.Error("Should not contain --protocol when Both is selected")
				}
			}
		})
	}
}

func TestValidateMirrorConfig(t *testing.T) {
	tests := []struct {
		name string
		cfg  MirrorConfig
		want MirrorConfig
	}{
		{
			name: "valid config unchanged",
			cfg: MirrorConfig{
				SortBy:       2,
				CountryIndex: 5,
				Latest:       15,
				Protocol:     1,
			},
			want: MirrorConfig{
				SortBy:       2,
				CountryIndex: 5,
				Latest:       15,
				Protocol:     1,
			},
		},
		{
			name: "negative SortBy clamped",
			cfg: MirrorConfig{
				SortBy:       -1,
				CountryIndex: 0,
				Latest:       20,
				Protocol:     0,
			},
			want: MirrorConfig{
				SortBy:       0,
				CountryIndex: 0,
				Latest:       20,
				Protocol:     0,
			},
		},
		{
			name: "SortBy too high clamped",
			cfg: MirrorConfig{
				SortBy:       100,
				CountryIndex: 0,
				Latest:       20,
				Protocol:     0,
			},
			want: MirrorConfig{
				SortBy:       0,
				CountryIndex: 0,
				Latest:       20,
				Protocol:     0,
			},
		},
		{
			name: "negative CountryIndex clamped",
			cfg: MirrorConfig{
				SortBy:       0,
				CountryIndex: -5,
				Latest:       20,
				Protocol:     0,
			},
			want: MirrorConfig{
				SortBy:       0,
				CountryIndex: 0,
				Latest:       20,
				Protocol:     0,
			},
		},
		{
			name: "CountryIndex too high clamped",
			cfg: MirrorConfig{
				SortBy:       0,
				CountryIndex: 1000,
				Latest:       20,
				Protocol:     0,
			},
			want: MirrorConfig{
				SortBy:       0,
				CountryIndex: 0,
				Latest:       20,
				Protocol:     0,
			},
		},
		{
			name: "Latest too low clamped to 1",
			cfg: MirrorConfig{
				SortBy:       0,
				CountryIndex: 0,
				Latest:       0,
				Protocol:     0,
			},
			want: MirrorConfig{
				SortBy:       0,
				CountryIndex: 0,
				Latest:       1,
				Protocol:     0,
			},
		},
		{
			name: "Latest negative clamped to 1",
			cfg: MirrorConfig{
				SortBy:       0,
				CountryIndex: 0,
				Latest:       -10,
				Protocol:     0,
			},
			want: MirrorConfig{
				SortBy:       0,
				CountryIndex: 0,
				Latest:       1,
				Protocol:     0,
			},
		},
		{
			name: "Latest too high clamped to 100",
			cfg: MirrorConfig{
				SortBy:       0,
				CountryIndex: 0,
				Latest:       200,
				Protocol:     0,
			},
			want: MirrorConfig{
				SortBy:       0,
				CountryIndex: 0,
				Latest:       100,
				Protocol:     0,
			},
		},
		{
			name: "negative Protocol clamped",
			cfg: MirrorConfig{
				SortBy:       0,
				CountryIndex: 0,
				Latest:       20,
				Protocol:     -1,
			},
			want: MirrorConfig{
				SortBy:       0,
				CountryIndex: 0,
				Latest:       20,
				Protocol:     0,
			},
		},
		{
			name: "Protocol too high clamped",
			cfg: MirrorConfig{
				SortBy:       0,
				CountryIndex: 0,
				Latest:       20,
				Protocol:     100,
			},
			want: MirrorConfig{
				SortBy:       0,
				CountryIndex: 0,
				Latest:       20,
				Protocol:     0,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := tc.cfg
			ValidateMirrorConfig(&cfg)

			if cfg.SortBy != tc.want.SortBy {
				t.Errorf("SortBy: got %d, want %d", cfg.SortBy, tc.want.SortBy)
			}
			if cfg.CountryIndex != tc.want.CountryIndex {
				t.Errorf("CountryIndex: got %d, want %d", cfg.CountryIndex, tc.want.CountryIndex)
			}
			if cfg.Latest != tc.want.Latest {
				t.Errorf("Latest: got %d, want %d", cfg.Latest, tc.want.Latest)
			}
			if cfg.Protocol != tc.want.Protocol {
				t.Errorf("Protocol: got %d, want %d", cfg.Protocol, tc.want.Protocol)
			}
		})
	}
}

func TestGetMirrorSortOptions(t *testing.T) {
	options := GetMirrorSortOptions()

	if len(options) != len(MirrorSortOptions) {
		t.Errorf("Expected %d sort options, got %d", len(MirrorSortOptions), len(options))
	}

	expectedOptions := []string{"Rate", "Age", "Score", "Country", "Delay"}
	for i, expected := range expectedOptions {
		if options[i] != expected {
			t.Errorf("Option %d: expected %s, got %s", i, expected, options[i])
		}
	}
}

func TestGetMirrorCountryNames(t *testing.T) {
	names := GetMirrorCountryNames()

	if len(names) != len(MirrorCountries) {
		t.Errorf("Expected %d country names, got %d", len(MirrorCountries), len(names))
	}

	// First should be Worldwide
	if names[0] != "Worldwide" {
		t.Errorf("First country should be Worldwide, got %s", names[0])
	}

	// Check some known countries
	found := make(map[string]bool)
	for _, name := range names {
		found[name] = true
	}

	expectedCountries := []string{"United States", "Germany", "Japan", "Australia"}
	for _, country := range expectedCountries {
		if !found[country] {
			t.Errorf("Expected to find %s in country list", country)
		}
	}
}

func TestGetMirrorProtocolNames(t *testing.T) {
	protocols := GetMirrorProtocolNames()

	if len(protocols) != len(MirrorProtocols) {
		t.Errorf("Expected %d protocol names, got %d", len(MirrorProtocols), len(protocols))
	}

	expectedProtocols := []string{"HTTPS", "HTTP", "Both"}
	for i, expected := range expectedProtocols {
		if protocols[i] != expected {
			t.Errorf("Protocol %d: expected %s, got %s", i, expected, protocols[i])
		}
	}
}

func TestGetReflectorCommandPreview(t *testing.T) {
	cfg := DefaultMirrorConfig()
	preview := GetReflectorCommandPreview(cfg)

	if !strings.HasPrefix(preview, "sudo reflector") {
		t.Errorf("Preview should start with 'sudo reflector', got: %s", preview)
	}

	if !strings.Contains(preview, "--latest 20") {
		t.Error("Preview should contain '--latest 20'")
	}

	if !strings.Contains(preview, "--sort rate") {
		t.Error("Preview should contain '--sort rate'")
	}
}

func TestSortCountriesByName(t *testing.T) {
	sorted := SortCountriesByName()

	// First should still be Worldwide
	if sorted[0].Name != "Worldwide" {
		t.Errorf("First country should be Worldwide, got %s", sorted[0].Name)
	}

	// Rest should be sorted alphabetically
	for i := 2; i < len(sorted); i++ {
		if sorted[i-1].Name > sorted[i].Name {
			t.Errorf("Countries not sorted: %s > %s", sorted[i-1].Name, sorted[i].Name)
		}
	}
}

func TestFindCountryByCode(t *testing.T) {
	// Test Worldwide (empty code)
	if FindCountryByCode("") != 0 {
		t.Error("Empty code should return 0 (Worldwide)")
	}

	// Test unknown code defaults to Worldwide
	if FindCountryByCode("XX") != 0 {
		t.Error("Unknown code should return 0 (Worldwide)")
	}
	if FindCountryByCode("invalid") != 0 {
		t.Error("Invalid code should return 0 (Worldwide)")
	}

	// Test known countries return correct indices
	usIdx := FindCountryByCode("US")
	if usIdx <= 0 || MirrorCountries[usIdx].Code != "US" {
		t.Errorf("FindCountryByCode(US) returned wrong index %d", usIdx)
	}

	deIdx := FindCountryByCode("DE")
	if deIdx <= 0 || MirrorCountries[deIdx].Code != "DE" {
		t.Errorf("FindCountryByCode(DE) returned wrong index %d", deIdx)
	}

	jpIdx := FindCountryByCode("JP")
	if jpIdx <= 0 || MirrorCountries[jpIdx].Code != "JP" {
		t.Errorf("FindCountryByCode(JP) returned wrong index %d", jpIdx)
	}
}

func TestFindCountryByName(t *testing.T) {
	// Test Worldwide
	if FindCountryByName("Worldwide") != 0 {
		t.Error("Worldwide should return 0")
	}

	// Test case insensitivity
	usIdx := FindCountryByName("United States")
	usIdxLower := FindCountryByName("united states")
	usIdxUpper := FindCountryByName("UNITED STATES")

	if usIdx <= 0 || usIdxLower != usIdx || usIdxUpper != usIdx {
		t.Errorf("Case insensitive search failed for United States")
	}

	// Test unknown country defaults to Worldwide
	if FindCountryByName("Unknown Country") != 0 {
		t.Error("Unknown country should return 0 (Worldwide)")
	}

	// Verify returned index is correct
	if MirrorCountries[usIdx].Name != "United States" {
		t.Errorf("FindCountryByName(United States) returned wrong index")
	}
}

func TestMirrorSortOptions(t *testing.T) {
	// Verify all sort options have valid flags
	for _, opt := range MirrorSortOptions {
		if opt.Name == "" {
			t.Error("Sort option has empty name")
		}
		if opt.Flag == "" {
			t.Errorf("Sort option %s has empty flag", opt.Name)
		}
		if opt.Description == "" {
			t.Errorf("Sort option %s has empty description", opt.Name)
		}
	}
}

func TestMirrorProtocols(t *testing.T) {
	// Verify protocol options
	if len(MirrorProtocols) < 3 {
		t.Error("Expected at least 3 protocol options")
	}

	// HTTPS should have a flag
	if MirrorProtocols[0].Flag != "https" {
		t.Errorf("HTTPS protocol flag should be 'https', got %s", MirrorProtocols[0].Flag)
	}

	// HTTP should have a flag
	if MirrorProtocols[1].Flag != "http" {
		t.Errorf("HTTP protocol flag should be 'http', got %s", MirrorProtocols[1].Flag)
	}

	// Both should have empty flag (no filter)
	if MirrorProtocols[2].Flag != "" {
		t.Errorf("Both protocol flag should be empty, got %s", MirrorProtocols[2].Flag)
	}
}

func TestMirrorCountries(t *testing.T) {
	// Verify first country is Worldwide with empty code
	if MirrorCountries[0].Name != "Worldwide" {
		t.Errorf("First country should be Worldwide, got %s", MirrorCountries[0].Name)
	}
	if MirrorCountries[0].Code != "" {
		t.Errorf("Worldwide should have empty code, got %s", MirrorCountries[0].Code)
	}

	// Verify all other countries have non-empty codes
	for i := 1; i < len(MirrorCountries); i++ {
		if MirrorCountries[i].Code == "" {
			t.Errorf("Country %s should have a code", MirrorCountries[i].Name)
		}
		if MirrorCountries[i].Name == "" {
			t.Errorf("Country at index %d has empty name", i)
		}
	}
}

func TestMirrorOverlayItemConstants(t *testing.T) {
	// Verify item constants are sequential
	if mirrorItemSortBy != 0 {
		t.Errorf("mirrorItemSortBy should be 0, got %d", mirrorItemSortBy)
	}
	if mirrorItemCountry != 1 {
		t.Errorf("mirrorItemCountry should be 1, got %d", mirrorItemCountry)
	}
	if mirrorItemLatest != 2 {
		t.Errorf("mirrorItemLatest should be 2, got %d", mirrorItemLatest)
	}
	if mirrorItemProtocol != 3 {
		t.Errorf("mirrorItemProtocol should be 3, got %d", mirrorItemProtocol)
	}
	if mirrorItemCount != 4 {
		t.Errorf("mirrorItemCount should be 4, got %d", mirrorItemCount)
	}
}

func TestBuildReflectorCommandAllSortOptions(t *testing.T) {
	for i, opt := range MirrorSortOptions {
		cfg := MirrorConfig{
			SortBy:       i,
			CountryIndex: 0,
			Latest:       20,
			Protocol:     0,
			Save:         false,
		}

		args := BuildReflectorCommand(cfg)
		argsStr := strings.Join(args, " ")

		if !strings.Contains(argsStr, "--sort "+opt.Flag) {
			t.Errorf("Sort option %s: expected --sort %s in args, got: %s", opt.Name, opt.Flag, argsStr)
		}
	}
}

func TestBuildReflectorCommandAllCountries(t *testing.T) {
	for i, country := range MirrorCountries {
		cfg := MirrorConfig{
			SortBy:       0,
			CountryIndex: i,
			Latest:       20,
			Protocol:     0,
			Save:         false,
		}

		args := BuildReflectorCommand(cfg)
		argsStr := strings.Join(args, " ")

		if i == 0 {
			// Worldwide should not have --country flag
			if strings.Contains(argsStr, "--country") {
				t.Errorf("Worldwide should not have --country flag, got: %s", argsStr)
			}
		} else {
			// All others should have --country flag with code
			if !strings.Contains(argsStr, "--country "+country.Code) {
				t.Errorf("Country %s: expected --country %s in args, got: %s", country.Name, country.Code, argsStr)
			}
		}
	}
}
