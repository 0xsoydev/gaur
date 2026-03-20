package main

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// Config represents the TOML configuration structure
type Config struct {
	Startup  StartupConfig  `toml:"startup"`
	UI       UIConfig       `toml:"ui"`
	Commands CommandConfig  `toml:"commands"`
	Advanced AdvancedConfig `toml:"advanced"`
	Keys     KeyConfig      `toml:"keys"`
}

type StartupConfig struct {
	DefaultMode string `toml:"default_mode"`
}

type UIConfig struct {
	Theme      string `toml:"theme"`
	BorderType string `toml:"border_type"`
}

type CommandConfig struct {
	AurHelper      string `toml:"aur_helper"`
	InstallFlags   string `toml:"install_flags"`
	UninstallFlags string `toml:"uninstall_flags"`
	CacheTool      string `toml:"cache_tool"`
}

type AdvancedConfig struct {
	DebounceMs int    `toml:"debounce_ms"`
	CacheDir   string `toml:"cache_dir"`
}

type KeyConfig struct {
	Quit           []string `toml:"quit"`
	InstallMode    string   `toml:"install_mode"`
	UninstallMode  string   `toml:"uninstall_mode"`
	UpdateMode     string   `toml:"update_mode"`
	DashboardMode  string   `toml:"dashboard_mode"`
	Search         string   `toml:"search"`
	Mark           string   `toml:"mark"`
	Selective      string   `toml:"selective"`
	Settings       string   `toml:"settings"`
	Confirm        string   `toml:"confirm"`
	Cancel         string   `toml:"cancel"`
}

// KeyMap defines the application's keybindings using charmbracelet/bubbles/key
type KeyMap struct {
	Quit           key.Binding
	InstallMode    key.Binding
	UninstallMode  key.Binding
	UpdateMode     key.Binding
	DashboardMode  key.Binding
	Search         key.Binding
	Mark           key.Binding
	Selective      key.Binding
	Settings       key.Binding
	Confirm        key.Binding
	Cancel         key.Binding
}

// CommandRunner defines an interface for executing shell commands.
type CommandRunner interface {
	Run(name string, args ...string) ([]byte, error)
	RunWithInput(input string, name string, args ...string) ([]byte, error)
	Interactive(onExit func(error) tea.Msg, name string, args ...string) tea.Cmd
}

// RealCommandRunner implements CommandRunner using os/exec.
type RealCommandRunner struct{}

func (r RealCommandRunner) Run(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).CombinedOutput()
}

func (r RealCommandRunner) RunWithInput(input string, name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	cmd.Stdin = strings.NewReader(input)
	return cmd.CombinedOutput()
}

func (r RealCommandRunner) Interactive(onExit func(error) tea.Msg, name string, args ...string) tea.Cmd {
	return tea.ExecProcess(exec.Command(name, args...), onExit)
}

var runner CommandRunner = RealCommandRunner{}

// View modes for the TUI application
type viewMode int

const (
	modeInstall viewMode = iota
	modeInstalled
	modeUninstall
	modeUpdate          // Viewing available updates
	modeUpdateSelective // Selecting specific updates
	modeCacheMenu       // Menu for selecting cache clearing strategy
	modeCacheSelective  // Selecting specific packages to clear from cache
	modeSettings        // In-app settings overlay
)

// SettingItem represents a single configurable setting in the carousel
type SettingItem struct {
	Label       string
	ConfigKey   string // Path to config like "ui.theme"
	Options     []string
	ActiveIndex int
}

// Confirmation operation types
type confirmationType int

const (
	confirmInstall confirmationType = iota
	confirmUninstall
	confirmUpdate
	confirmSelectiveUpdate
	confirmCleanKeep3       // paccache -r
	confirmCleanKeep1       // paccache -rk1
	confirmCleanUninstalled // paccache -ruk0
	confirmCleanNuke        // paccache -rk0
	confirmCleanSelective   // Custom selective clean
	confirmRemoveOrphans
)

// UI configuration constants
const (
	minSearchQueryLen = 2
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
	Size        string // For formatted sizes, like in cache hogs
	SizeBytes   int64  // Raw size
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
	packages  []Package
	query     string
	timeTaken time.Duration
	err       error
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
	RepoDistribution     map[string]int // core, extra, multilib, etc.
	TotalSize            string
	TotalSizeBytes       int64 // For comparison
	CleanerSize          string
	CleanerSizeBytes     int64 // For comparison and coloring
	PacmanCacheSize      string
	PacmanCacheSizeBytes int64
	PacmanCachePath      string
	ParuCacheSize        string `toml:"-"` // Deprecated: use AurCacheSize
	AurCacheSize         string
	AurCacheSizeBytes    int64
	AurCachePath         string
	Orphans              int
	MissingFromAUR       int
	TopPackages          []PackageSize // Top 10 packages by size
	RecentlyInstalled    []RecentPackage // Details of 5 recently installed packages
	TopCacheHogs         []PackageSize // Top 5 packages taking up cache space
	AllCacheHogs         []PackageSize // All packages taking up cache space
	UninstalledPacmanCache []PackageSize // Uninstalled packages in pacman cache
	UninstalledAurCache    []PackageSize // Uninstalled packages in AUR helper cache
	CacheFreedPacman     map[confirmationType]string // Estimated savings for pacman
	CacheFreedAur        map[confirmationType]string // Estimated savings for AUR helper
	CacheFreedEstimates  map[confirmationType]string // Total estimated savings
	// Disk usage info
	DiskTotal      string
	DiskUsed       string
	DiskFree       string
	DiskUsedPercent float64
}

// PackageSize holds package name and its installed size
type PackageSize struct {
	Name      string
	Size      string
	SizeBytes int64
}

// RecentPackage holds details about a recently installed package
type RecentPackage struct {
	Name      string
	Timestamp string // e.g. "2024-03-12 10:00"
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
