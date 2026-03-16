package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.MouseMsg:
		return m.handleMouse(msg)

	case tea.KeyMsg:
		// 1. Global Intercepts (Highest Priority)
		if key.Matches(msg, m.keys.Quit) {
			m.saveSettingsToDisk()
			return m, tea.Quit
		}
		if msg.String() == "ctrl+r" && m.mode == modeInstalled {
			m.loading = true
			m.statusMessage = "Refreshing dashboard..."
			return m, getDashboardData()
		}

		// 2. Overlays & Panel Intercepts
		if m.mode == modeSettings {
			switch {
			case key.Matches(msg, m.keys.Cancel) || key.Matches(msg, m.keys.Settings):
				m.saveSettingsToDisk()
				m.mode = m.previousMode
				return m, nil
			case msg.String() == "up" || msg.String() == "k":
				if m.settingsIndex > 0 {
					m.settingsIndex--
				}
			case msg.String() == "down" || msg.String() == "j":
				if m.settingsIndex < len(m.settingsItems)-1 {
					m.settingsIndex++
				}
			case msg.String() == "left" || msg.String() == "h":
				item := &m.settingsItems[m.settingsIndex]
				item.ActiveIndex--
				if item.ActiveIndex < 0 {
					item.ActiveIndex = len(item.Options) - 1
				}
				m.updateConfigFromSettings()
			case msg.String() == "right" || msg.String() == "l":
				item := &m.settingsItems[m.settingsIndex]
				item.ActiveIndex++
				if item.ActiveIndex >= len(item.Options) {
					item.ActiveIndex = 0
				}
				m.updateConfigFromSettings()
			}
			return m, nil
		}

		if m.showErrorOverlay {
			if key.Matches(msg, m.keys.Cancel) || key.Matches(msg, m.keys.Confirm) || key.Matches(msg, m.keys.Quit) {
				m.showErrorOverlay = false
			}
			return m, nil
		}
		if m.showConfirmation {
			return m.handleConfirmationKey(msg)
		}
		if m.selectionPanelFocused {
			return m.handleSelectionPanelKey(msg)
		}

		// 3. Navigation (Arrows, Page Keys, and JK when not focused)
		isNav := false
		switch msg.Type {
		case tea.KeyUp, tea.KeyDown, tea.KeyPgUp, tea.KeyPgDown:
			isNav = true
		}
		if !m.textInput.Focused() && (msg.String() == "j" || msg.String() == "k") {
			isNav = true
		}

		if isNav {
			return m.handleNavigation(msg)
		}

		// 4. Input Focus Mode
		if m.textInput.Focused() {
			if key.Matches(msg, m.keys.Cancel) {
				if m.mode == modeUpdateSelective {
					m.mode = modeUpdate
					m.markedPackages = make(map[string]bool)
					m.textInput.SetValue("")
					m.lastQuery = ""
					m.packageInfo = ""
					m.infoForPackage = ""
					m.infoScrollOffset = 0
				}
				m.textInput.Blur()
				return m, nil
			}
			if key.Matches(msg, m.keys.Confirm) {
				return m.handleActionTrigger()
			}
			if key.Matches(msg, m.keys.Mark) || msg.String() == "m" {
				return m.handleMarking()
			}

			var cmd tea.Cmd
			m.textInput, cmd = m.textInput.Update(msg)
			filterCmd := m.performFiltering()
			return m, tea.Batch(cmd, filterCmd)
		}

		// 5. General Mode Keys (Unfocused)
		switch {
		case key.Matches(msg, m.keys.Cancel):
			if m.mode == modeCacheSelective {
				m.mode = modeCacheMenu
				m.markedPackages = make(map[string]bool)
				m.cacheToFree = 0
				m.textInput.SetValue("")
				m.lastQuery = ""
				m.packageInfo = ""
				m.infoForPackage = ""
				m.infoScrollOffset = 0
				m.statusMessage = "Selective cache cleaning cancelled"
				return m, nil
			}
			if m.mode == modeCacheMenu {
				m.mode = modeInstalled
				m.statusMessage = "Cache menu cancelled"
				m.packageInfo = ""
				m.infoForPackage = ""
				m.infoScrollOffset = 0
				return m, nil
			}
			if len(m.markedPackages) > 0 {
				m.markedPackages = make(map[string]bool)
				m.statusMessage = "Selections cleared"
				return m, nil
			}
			return m, nil
		case key.Matches(msg, m.keys.Settings):
			m.previousMode = m.mode
			m.mode = modeSettings
			return m, nil
		case key.Matches(msg, m.keys.Search):
			m.textInput.Focus()
			return m, nil
		case msg.String() == "c":
			if m.mode == modeInstalled && !m.loading {
				m.mode = modeCacheMenu
				m.cacheMenuIndex = 0
			}
		case msg.String() == "R":
			if m.mode == modeInstalled && !m.loading && m.dashboard.Orphans > 0 {
				orphanList, _ := runner.Run(m.config.Commands.AurHelper, "-Qdtq")
				m.confirmPackages = strings.Fields(string(orphanList))
				m.showConfirmation = true
				m.confirmType = confirmRemoveOrphans
			}
		case msg.String() == "t", msg.String() == "e", msg.String() == "f", msg.String() == "o":
			if m.mode == modeInstalled && !m.loading {
				m.mode = modeUninstall
				m.loading = true
				m.selectedIndex = 0
				m.textInput.SetValue(msg.String() + ":")
				m.lastQuery = ""
				m.packageInfo = ""
				m.infoForPackage = ""
				m.infoScrollOffset = 0
				return m, getInstalledPackages()
			}
		case key.Matches(msg, m.keys.DashboardMode):
			m.mode = modeInstalled
			m.loading = true
			m.textInput.SetValue("")
			m.lastQuery = ""
			m.packageInfo = ""
			m.infoForPackage = ""
			m.infoScrollOffset = 0
			return m, getDashboardData()
		case key.Matches(msg, m.keys.UninstallMode):
			if m.mode != modeUninstall {
				m.mode = modeUninstall
				m.loading = true
				m.selectedIndex = 0
				m.textInput.SetValue("")
				m.lastQuery = ""
				m.packageInfo = ""
				m.infoForPackage = ""
				m.infoScrollOffset = 0
				return m, getInstalledPackages()
			}
		case key.Matches(msg, m.keys.UpdateMode):
			m.mode = modeUpdate
			m.textInput.SetValue("")
			m.lastQuery = ""
			m.packageInfo = ""
			m.infoForPackage = ""
			m.infoScrollOffset = 0
			return m, syncRepositoriesInTerminal(m)
		case key.Matches(msg, m.keys.Selective):
			if m.mode == modeUpdate {
				m.mode = modeUpdateSelective
				m.selectedIndex = 0
				m.textInput.SetValue("")
				m.lastQuery = ""
				m.textInput.Focus()
				if len(m.pendingUpdates) > 0 {
					m.filtered = m.pendingUpdates
					m.loadingInfo = true
					m.infoForPackage = m.filtered[0].Name
					return m, getPackageInfo(m, m.filtered[0])
				}
			}
		case key.Matches(msg, m.keys.Confirm):
			return m.handleActionTrigger()
		case msg.String() == "y" || msg.String() == "Y" || msg.String() == "a":
			if m.mode == modeUpdate && !m.loading && len(m.pendingUpdates) > 0 {
				m.statusMessage = "Running system update..."
				return m, executeUpdateInTerminal(m)
			}
		case key.Matches(msg, m.keys.InstallMode):
			if m.mode != modeInstall {
				m.mode = modeInstall
				m.selectedIndex = 0
				m.filtered = []Package{}
				m.textInput.SetValue("")
				m.lastQuery = ""
				m.packageInfo = ""
				m.infoForPackage = ""
				m.infoScrollOffset = 0
				m.textInput.Focus()
			}
		case key.Matches(msg, m.keys.Mark) || msg.String() == "m":
			return m.handleMarking()
		case msg.String() == "*":
			if len(m.markedPackages) > 0 {
				m.selectionPanelFocused = true
				m.selectionPanelIndex = 0
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.textInput.Width = msg.Width - 6

	case repoPackagesMsg:
		m.loading = false
		if msg.err == nil {
			m.repoPackages = msg.packages
			m.statusMessage = fmt.Sprintf("Loaded %d packages", len(msg.packages))
			return m, m.performFiltering()
		} else {
			m.statusMessage = "Failed to load packages"
		}

	case syncRepositoriesMsg:
		if msg.err == nil {
			m.loading = true
			return m, checkUpdates()
		}
		m.loading = false
		m.statusMessage = "Sync failed"
		m.showErrorOverlay = true
		m.errorTitle = "Repository Sync Failed"
		m.errorMessage = msg.err.Error()

	case updateCheckMsg:
		m.loading = false
		if msg.err == nil {
			m.pendingUpdates = msg.packages
			m.updatableAll = msg.packages
			if len(msg.packages) == 0 {
				m.statusMessage = "System is up to date"
			} else {
				m.statusMessage = fmt.Sprintf("%d updates available", len(msg.packages))
			}
		} else {
			m.statusMessage = "Failed to check for updates"
			m.showErrorOverlay = true
			m.errorTitle = "Update Check Error"
			m.errorMessage = msg.err.Error()
		}

	case aurSearchMsg:
		m.searchingAUR = false
		if msg.err == nil {
			m.aurPackages = msg.packages
			return m, m.performFiltering()
		}

	case packageInfoMsg:
		if msg.packageName == m.infoForPackage {
			m.loadingInfo = false
			m.packageInfo = msg.info
			m.infoCache[msg.packageName] = msg.info
		}

	case debounceTickMsg:
		if msg.packageName == m.pendingInfoPackage {
			m.infoForPackage = msg.packageName
			pkg := m.getPackageByName(msg.packageName)
			if pkg != nil {
				m.infoScrollOffset = 0
				return m, getPackageInfo(m, *pkg)
			}
		}

	case installedPackagesMsg:
		m.loading = false
		if msg.err == nil {
			m.installed = msg.packages
			m.statusMessage = fmt.Sprintf("Loaded %d installed packages", len(msg.packages))
			return m, m.performFiltering()
		} else {
			m.statusMessage = "Failed to load installed packages"
		}

	case dashboardMsg:
		m.loading = false
		if msg.err == nil { m.dashboard = msg.data }

	case actionCompleteMsg:
		m.loading = false
		m.statusMessage = msg.message
		if msg.err != nil {
			m.showErrorOverlay = true
			m.errorTitle = "Action Failed"
			m.errorMessage = msg.err.Error()
		}
		if m.mode == modeInstall { cmds = append(cmds, loadRepoPackages()) }
		if m.mode == modeUninstall { cmds = append(cmds, getInstalledPackages()) }

	case updateOutputMsg:
		if msg.done {
			m.loading = false
			if msg.err != nil {
				m.showErrorOverlay = true
				m.errorTitle = "Update Failed"
				m.errorMessage = msg.err.Error()
			} else {
				m.statusMessage = "Update completed successfully"
			}
			return m, checkUpdates()
		}

	case execCompleteMsg:
		return m.handleExecComplete(msg)
	}

	return m, tea.Batch(cmds...)
}

// --- Internal Helper Methods ---

func (m *model) handleNavigation(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := strings.ToLower(msg.String())
	
	// 1. Cache Menu Navigation
	if m.mode == modeCacheMenu {
		switch key {
		case "up", "k": if m.cacheMenuIndex > 0 { m.cacheMenuIndex-- }
		case "down", "j": if m.cacheMenuIndex < 4 { m.cacheMenuIndex++ }
		case "pgup": m.cacheMenuIndex = 0
		case "pgdown": m.cacheMenuIndex = 4
		}
		return m, nil
	}

	// 2. Mode Update Scroll
	if m.mode == modeUpdate {
		switch key {
		case "up", "k": if m.updateScrollOffset > 0 { m.updateScrollOffset-- }
		case "down", "j": if m.updateScrollOffset < len(m.pendingUpdates)-1 { m.updateScrollOffset++ }
		case "pgup": m.updateScrollOffset = 0
		case "pgdown": m.updateScrollOffset = len(m.pendingUpdates) - 1
		}
		return m, nil
	}

	// 3. Selection List Navigation
	maxIndex := 0
	if m.mode == modeUninstall {
		maxIndex = len(m.filteredInstalled) - 1
	} else {
		maxIndex = len(m.filtered) - 1
	}

	oldIdx := m.selectedIndex
	jump := 1
	if key == "pgup" || key == "pgdown" { jump = 10 }

	if key == "up" || key == "k" || key == "pgup" {
		m.selectedIndex += jump
	} else {
		m.selectedIndex -= jump
	}

	// Clamp
	if m.selectedIndex > maxIndex { m.selectedIndex = maxIndex }
	if m.selectedIndex < 0 { m.selectedIndex = 0 }

	if m.selectedIndex != oldIdx {
		m.infoScrollOffset = 0
		pkg := m.getSelectedPkg()
		if pkg != nil && m.mode != modeCacheSelective {
			m.loadingInfo = true
			m.pendingInfoPackage = pkg.Name
			return m, debouncePackageInfo(m, m.pendingInfoPackage)
		}
	}
	return m, nil
}

func (m *model) handleMarking() (tea.Model, tea.Cmd) {
	pkg := m.getSelectedPkg()
	if pkg == nil { return m, nil }
	if m.mode == modeCacheSelective {
		if m.markedPackages[pkg.Name] {
			m.cacheToFree -= pkg.SizeBytes
			delete(m.markedPackages, pkg.Name)
		} else {
			m.cacheToFree += pkg.SizeBytes
			m.markedPackages[pkg.Name] = true
		}
	} else {
		if m.markedPackages[pkg.Name] {
			delete(m.markedPackages, pkg.Name)
		} else {
			m.markedPackages[pkg.Name] = true
		}
	}
	return m, nil
}

func (m *model) handleActionTrigger() (tea.Model, tea.Cmd) {
	switch m.mode {
	case modeInstall, modeUninstall, modeUpdateSelective:
		var pkgs []string
		for n := range m.markedPackages { pkgs = append(pkgs, n) }
		if len(pkgs) == 0 {
			p := m.getSelectedPkg()
			if p != nil { pkgs = []string{p.Name} }
		}
		if len(pkgs) > 0 {
			sort.Strings(pkgs); m.showConfirmation = true; m.confirmPackages = pkgs
			if m.mode == modeInstall { m.confirmType = confirmInstall }
			if m.mode == modeUninstall { m.confirmType = confirmUninstall }
			if m.mode == modeUpdateSelective { m.confirmType = confirmSelectiveUpdate }
		}
	case modeCacheMenu:
		switch m.cacheMenuIndex {
		case 4:
			m.mode = modeCacheSelective; m.selectedIndex = 0; m.markedPackages = make(map[string]bool); m.cacheToFree = 0
			m.textInput.SetValue(""); m.lastQuery = ""
			m.packageInfo = ""
			m.infoForPackage = ""
			m.infoScrollOffset = 0
			m.filtered = make([]Package, len(m.dashboard.AllCacheHogs))
			for i, h := range m.dashboard.AllCacheHogs { m.filtered[i] = Package{Name: h.Name, Size: h.Size, SizeBytes: h.SizeBytes} }
		default:
			m.showConfirmation = true
			types := []confirmationType{confirmCleanKeep3, confirmCleanKeep1, confirmCleanUninstalled, confirmCleanNuke}
			m.confirmType = types[m.cacheMenuIndex]
		}
	case modeCacheSelective:
		if len(m.markedPackages) > 0 {
			var pkgs []string
			for n := range m.markedPackages { pkgs = append(pkgs, n) }
			sort.Strings(pkgs); m.showConfirmation = true; m.confirmType = confirmCleanSelective; m.confirmPackages = pkgs
		}
	case modeUpdate:
		if len(m.pendingUpdates) > 0 {
			var pkgs []string
			for _, p := range m.pendingUpdates { pkgs = append(pkgs, p.Name) }
			m.showConfirmation = true; m.confirmType = confirmUpdate; m.confirmPackages = pkgs
			m.statusMessage = fmt.Sprintf("Confirm update for %d packages", len(m.pendingUpdates))
		}
	}
	return m, nil
}

func (m *model) handleConfirmationKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, m.keys.Confirm) || msg.String() == "y" || msg.String() == "Y" {
		m.showConfirmation = false
		switch m.confirmType {
		case confirmInstall: return m, executeInstallInTerminal(m, m.confirmPackages)
		case confirmUninstall: return m, executeUninstallInTerminal(m, m.confirmPackages)
		case confirmUpdate: return m, executeUpdateInTerminal(m)
		case confirmSelectiveUpdate: return m, executeSelectiveUpdateInTerminal(m, m.confirmPackages)
		case confirmCleanKeep3: return m, executeCleanCache(m, confirmCleanKeep3, 3, false)
		case confirmCleanKeep1: return m, executeCleanCache(m, confirmCleanKeep1, 1, false)
		case confirmCleanUninstalled: return m, executeCleanCache(m, confirmCleanUninstalled, 0, true)
		case confirmCleanNuke: return m, executeCleanCache(m, confirmCleanNuke, 0, false)
		case confirmCleanSelective: return m, executeSelectiveClean(m.confirmPackages, m.dashboard.PacmanCachePath, m.dashboard.ParuCachePath)
		case confirmRemoveOrphans: return m, executeRemoveOrphansInTerminal(m, m.confirmPackages)
		}
	} else if key.Matches(msg, m.keys.Cancel) || msg.String() == "n" || msg.String() == "N" {
		m.showConfirmation = false
	}
	
	// Scrolling in confirmation
	key := strings.ToLower(msg.String())
	if key == "up" || key == "k" { if m.confirmScrollOffset > 0 { m.confirmScrollOffset-- } }
	if key == "down" || key == "j" { if m.confirmScrollOffset < m.maxConfirmScroll { m.confirmScrollOffset++ } }
	if key == "pgup" { m.confirmScrollOffset = 0 }
	if key == "pgdown" { m.confirmScrollOffset = m.maxConfirmScroll }

	return m, nil
}

func (m *model) handleSelectionPanelKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var names []string
	for n := range m.markedPackages { names = append(names, n) }
	sort.Strings(names)
	
	switch {
	case key.Matches(msg, m.keys.Cancel):
		m.selectionPanelFocused = false
	case msg.String() == "up", msg.String() == "k":
		if m.selectionPanelIndex > 0 { m.selectionPanelIndex-- }
	case msg.String() == "down", msg.String() == "j":
		if m.selectionPanelIndex < len(names)-1 { m.selectionPanelIndex++ }
	case key.Matches(msg, m.keys.Mark) || msg.String() == "m":
		if m.selectionPanelIndex < len(names) {
			delete(m.markedPackages, names[m.selectionPanelIndex])
			if len(m.markedPackages) == 0 { m.selectionPanelFocused = false }
		}
	case key.Matches(msg, m.keys.Confirm):
		m.selectionPanelFocused = false
		return m.handleActionTrigger()
	}
	return m, nil
}

func (m *model) performFiltering() tea.Cmd {
	query := m.textInput.Value()
	if query == m.lastQuery {
		return nil
	}
	m.lastQuery = query
	m.selectedIndex = 0

	if m.mode == modeInstall { m.filterAllPackages(query) }
	if m.mode == modeUninstall {
		if query == "" {
			m.filteredInstalled = m.installed
		} else {
			m.filteredInstalled = fuzzyFilter(m.installed, query)
		}
	}
	if m.mode == modeUpdateSelective {
		if query == "" {
			m.filtered = m.updatableAll
		} else {
			m.filtered = fuzzyFilter(m.updatableAll, query)
		}
	}
	if m.mode == modeCacheSelective {
		// Convert AllCacheHogs to Package slice for fuzzyFilter
		var allPkgs []Package
		for _, h := range m.dashboard.AllCacheHogs {
			allPkgs = append(allPkgs, Package{Name: h.Name, Size: h.Size, SizeBytes: h.SizeBytes})
		}
		if query == "" {
			m.filtered = allPkgs
			m.matchIndices = nil
		} else {
			m.filtered = fuzzyFilter(allPkgs, query)
			m.matchIndices = computeAllMatchIndices(m.filtered, query)
		}
	}

	// Fetch info for the first item automatically if list is not empty
	pkg := m.getSelectedPkg()
	if pkg != nil && m.mode != modeCacheSelective {
		m.loadingInfo = true
		m.pendingInfoPackage = pkg.Name
		return debouncePackageInfo(m, m.pendingInfoPackage)
	} else {
		m.packageInfo = ""
		m.infoForPackage = ""
	}
	return nil
}

func (m *model) getSelectedPkg() *Package {
	list := m.filtered
	if m.mode == modeUninstall { list = m.filteredInstalled }
	if m.selectedIndex >= 0 && m.selectedIndex < len(list) { return &list[m.selectedIndex] }
	return nil
}

func (m *model) getPackageByName(name string) *Package {
	list := m.filtered
	if m.mode == modeUninstall { list = m.filteredInstalled }
	for i := range list { if list[i].Name == name { return &list[i] } }
	return nil
}

func (m *model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if m.showConfirmation {
		if msg.Type == tea.MouseWheelUp { if m.confirmScrollOffset > 0 { m.confirmScrollOffset-- } }
		if msg.Type == tea.MouseWheelDown { if m.confirmScrollOffset < m.maxConfirmScroll { m.confirmScrollOffset++ } }
		return m, nil
	}

	if msg.Type != tea.MouseWheelUp && msg.Type != tea.MouseWheelDown { return m, nil }
	
	// Details pane scroll check
	if (m.mode == modeInstall || m.mode == modeUninstall) && msg.Y < m.height/2 {
		if msg.Type == tea.MouseWheelUp { if m.infoScrollOffset > 0 { m.infoScrollOffset-- } } else { if m.infoScrollOffset < m.maxInfoScroll { m.infoScrollOffset++ } }
		return m, nil
	}
	if m.mode == modeUpdateSelective && msg.X >= m.width/2 {
		if msg.Type == tea.MouseWheelUp { if m.infoScrollOffset > 0 { m.infoScrollOffset-- } } else { if m.infoScrollOffset < m.maxInfoScroll { m.infoScrollOffset++ } }
		return m, nil
	}

	fake := tea.KeyMsg{Type: tea.KeyUp}
	if msg.Type == tea.MouseWheelDown { fake.Type = tea.KeyDown }
	return m.handleNavigation(fake)
}

func (m *model) handleExecComplete(msg execCompleteMsg) (tea.Model, tea.Cmd) {
	m.loading = false
	if msg.err != nil {
		m.showErrorOverlay = true
		m.errorTitle = "Operation Failed"
		m.errorMessage = msg.err.Error()
		return m, nil
	}
	if m.mode == modeInstall { return m, loadRepoPackages() }
	if m.mode == modeUninstall { return m, getInstalledPackages() }
	if m.mode == modeUpdate || m.mode == modeUpdateSelective {
		m.loading = true
		m.statusMessage = "Refreshing update list..."
		return m, checkUpdates()
	}
	return m, getDashboardData()
}
