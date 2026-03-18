package main

import (
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Model
type model struct {
	config                Config
	keys                  KeyMap
	textInput             textinput.Model
	repoPackages          []Package       // All repo packages from local cache
	aurPackages           []Package       // AUR packages from last search
	installedSet          map[string]bool // Quick lookup for installed packages
	packages              []Package
	filtered              []Package
	installed             []Package
	filteredInstalled     []Package
	matchIndices          map[int][]int // Maps package index to matched character indices
	installedMatchIndices map[int][]int
	selectedIndex         int
	markedPackages        map[string]bool // Packages marked for batch operation
	selectionPanelFocused bool            // Whether selection panel is focused
	selectionPanelIndex   int             // Selected index within selection panel
	selectionScrollOffset int             // Scroll offset for the selection panel
	packageInfo           string
	infoCache             map[string]string // Cache for fetched package info
	infoForPackage        string
	pendingInfoPackage    string // Package waiting for debounce to complete
	infoScrollOffset      int    // Scroll offset for the info/details pane
	maxInfoScroll         int    // Maximum allowed scroll for info pane
	loadingInfo           bool
	mode                  viewMode
	width                 int
	height                int
	loading               bool
	statusMessage         string
	updateOutput          string
	lastQuery             string
	lastAURQuery          string // Last query sent to AUR search
	searchingAUR          bool   // Whether AUR search is in progress
	searchTerm            string // Current search term for status line
	searchStatus          string // "Searching..." or "Search complete..."
	searchError           bool   // Whether the last search failed
	searchStartTime       time.Time
	lastSearchDuration    time.Duration
	spinner               spinner.Model
	dashboard             DashboardData
	// Confirmation dialog state
	showConfirmation    bool
	confirmType         confirmationType
	confirmPackages     []string  // Package names to operate on
	pendingUpdates      []Package // Updates available (for update confirmation)
	confirmScrollOffset int       // Scroll offset for confirmation package list
	maxConfirmScroll    int       // Max scroll for confirmation list
	lastCompletedOp     string    // Description of last completed operation
	// Update selection state
	updatableAll []Package // All packages available for update (before selection)
	updateScrollOffset int   // Scroll offset for the simple update view
	maxUpdateScroll    int   // Max scroll for update view
	// Error overlay state
	showErrorOverlay bool
	errorTitle       string
	errorMessage     string
	errorDetails     string
	// Sync logic
	checkAfterUpdate bool // Whether we should perform a sync+check if updates are zero
	// Cache cleaning state
	cacheMenuIndex int
	cacheToFree    int64
	// Settings state
	settingsItems []SettingItem
	settingsIndex int
	previousMode  viewMode
}

func initialModel(initialMode viewMode, cfg Config) *model {
	ti := textinput.New()
	ti.CharLimit = textInputCharLimit
	ti.Width = textInputDefaultWidth

	statusMsg := "Loading package database..."
	placeholder := "Search packages..."
	switch initialMode {
	case modeUninstall:
		placeholder = "Filter installed packages..."
		statusMsg = "Loading installed packages..."
	case modeUpdate:
		placeholder = "Checking for updates..."
		statusMsg = "Checking for updates..."
	case modeInstalled:
		placeholder = "View installed packages"
		statusMsg = "Loading installed packages..."
	}
	ti.Placeholder = placeholder

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	m := &model{
		config:         cfg,
		keys:           NewKeyMap(cfg.Keys),
		textInput:      ti,
		repoPackages:   []Package{},
		installedSet:   make(map[string]bool),
		packages:       []Package{},
		filtered:       []Package{},
		installed:      []Package{},
		markedPackages: make(map[string]bool),
		infoCache:      make(map[string]string),
		selectedIndex:  0,
		mode:           initialMode,
		loading:        true,
		statusMessage:  statusMsg,
		spinner:        s,
	}

	if initialMode == modeUninstall {
		m.loadingInfo = true
		m.infoForPackage = "..."
	}

	m.initSettings()
	return m
}

func (m *model) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		m.spinner.Tick,
		loadRepoPackages(),
		getInstalledPackages(),
		func() tea.Msg {
			switch m.mode {
			case modeInstalled:
				return getDashboardData(&m.config)()
			case modeUpdate:
				return checkUpdates(&m.config)()
			}
			return nil
		},
	)
}

// currentPackageList returns the appropriate package list based on current mode.
func (m *model) currentPackageList() []Package {
	switch m.mode {
	case modeInstall:
		return m.filtered
	case modeUninstall:
		return m.filteredInstalled
	default:
		return nil
	}
}

// maxSelectableIndex returns the maximum valid index for the current package list.
func (m *model) maxSelectableIndex() int {
	pkgList := m.currentPackageList()
	if len(pkgList) == 0 {
		return 0
	}
	return len(pkgList) - 1
}

// selectedPackage returns the currently selected package, or nil if none.
func (m *model) selectedPackage() *Package {
	pkgList := m.currentPackageList()
	if m.selectedIndex >= 0 && m.selectedIndex < len(pkgList) {
		return &pkgList[m.selectedIndex]
	}
	return nil
}

// refreshAll triggers a full refresh of all system data
func (m *model) refreshAll() tea.Cmd {
	m.loading = true
	m.pendingUpdates = nil
	return tea.Batch(
		getDashboardData(&m.config),
		loadRepoPackages(),
		getInstalledPackages(),
		checkUpdates(&m.config),
	)
}

// resetState clears common state fields like search progress and package selections
func (m *model) resetState() {
	m.searchingAUR = false
	m.lastAURQuery = ""
	m.searchStatus = ""
	m.searchError = false
	m.searchTerm = ""
	m.markedPackages = make(map[string]bool)
	m.selectionPanelFocused = false
	m.selectionPanelIndex = 0
	m.selectionScrollOffset = 0
	m.cacheToFree = 0
}
