package main

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
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
		loadRepoPackages(),
		getInstalledPackages(),
		func() tea.Msg {
			switch m.mode {
			case modeInstalled:
				return getDashboardData()()
			case modeUpdate:
				return checkUpdates()()
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
