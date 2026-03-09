package main

import (
	"fmt"
	"time"
)

// View modes for the TUI application
type viewMode int

const (
	modeInstall viewMode = iota
	modeInstalled
	modeUninstall
	modeUpdate // Viewing available updates
)

// Confirmation operation types
type confirmationType int

const (
	confirmInstall confirmationType = iota
	confirmUninstall
	confirmUpdate
	confirmCleanCache
	confirmRemoveOrphans
)

// UI configuration constants
const (
	minSearchQueryLen       = 2
	textInputCharLimit      = 100
	textInputDefaultWidth   = 50
	packageInfoDebounceTime = 150 * time.Millisecond
)

// Package represents a package with its source and name
type Package struct {
	Source      string // core, extra, multilib, aur
	Name        string
	Version     string
	Description string
	Installed   bool
	Explicit    bool // Explicitly installed (not a dependency)
	Orphan      bool // Orphan package (no longer required)
}

func (p Package) String() string {
	return fmt.Sprintf("%s/%s", p.Source, p.Name)
}

// Messages
type repoPackagesMsg struct {
	packages []Package
	err      error
}

type aurSearchMsg struct {
	packages []Package
	query    string
	err      error
}

type packageInfoMsg struct {
	info        string
	packageName string
	err         error
}

type installedPackagesMsg struct {
	packages []Package
	err      error
}

type actionCompleteMsg struct {
	message string
	err     error
}

type updateOutputMsg struct {
	output string
	done   bool
	err    error
}

type updateCheckMsg struct {
	packages []Package
	err      error
}

type execCompleteMsg struct {
	operation confirmationType
	packages  []string
	err       error
}

type dashboardMsg struct {
	data DashboardData
	err  error
}

// debounceTickMsg is sent after debounce timer expires to trigger package info fetch
type debounceTickMsg struct {
	packageName string
}

// DashboardData holds system package statistics
type DashboardData struct {
	TotalPackages        int
	ExplicitlyInstalled  int
	ForeignPackages      int
	TotalSize            string
	TotalSizeBytes       int64 // For comparison
	CleanerSize          string
	CleanerSizeBytes     int64 // For comparison and coloring
	PacmanCacheSize      string
	PacmanCacheSizeBytes int64
	PacmanCachePath      string
	ParuCacheSize        string
	ParuCacheSizeBytes   int64
	ParuCachePath        string
	Orphans              int
	MissingFromAUR       int
	TopPackages          []PackageSize // Top 10 packages by size
}

// PackageSize holds package name and its installed size
type PackageSize struct {
	Name string
	Size string
}

// Dashboard action messages
type cleanCacheMsg struct {
	output string
	err    error
}

type removeOrphansMsg struct {
	output string
	err    error
}

type syncRepositoriesMsg struct {
	err error
}
