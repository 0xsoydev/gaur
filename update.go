package main

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.MouseMsg:
		return m.handleMouse(msg)

	case tea.KeyMsg:
		// 1. Global Intercepts
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if msg.String() == "ctrl+r" && m.mode == modeInstalled {
			m.loading = true
			m.statusMessage = "Refreshing dashboard..."
			return m, getDashboardData()
		}

		// 2. Overlays & Panel Intercepts
		if m.showErrorOverlay {
			if msg.String() == "esc" || msg.String() == "enter" || msg.String() == "q" {
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
			switch msg.String() {
			case "esc":
				if m.mode == modeUpdateSelective {
					m.mode = modeUpdate
					m.markedPackages = make(map[string]bool)
				}
				m.textInput.Blur()
				return m, nil
			case "enter":
				return m.handleActionTrigger()
			case "tab", "m":
				return m.handleMarking()
			}

			var cmd tea.Cmd
			m.textInput, cmd = m.textInput.Update(msg)
			m.performFiltering()
			return m, cmd
		}

		// 5. General Mode Keys (Unfocused)
		switch msg.String() {
		case "q":
			return m, tea.Quit
		case "/":
			m.textInput.Focus()
			return m, nil
		case "c":
			if m.mode == modeInstalled && !m.loading {
				m.mode = modeCacheMenu
				m.cacheMenuIndex = 0
			}
		case "R":
			if m.mode == modeInstalled && !m.loading && m.dashboard.Orphans > 0 {
				orphanList, _ := runner.Run("paru", "-Qdtq")
				m.confirmPackages = strings.Fields(string(orphanList))
				m.showConfirmation = true
				m.confirmType = confirmRemoveOrphans
			}
		case "t", "e", "f", "o":
			if m.mode == modeInstalled && !m.loading {
				m.mode = modeUninstall
				m.loading = true
				m.selectedIndex = 0
				m.textInput.SetValue(msg.String() + ":")
				return m, getInstalledPackages()
			}
		case "n":
			m.mode = modeInstalled
			m.loading = true
			return m, getDashboardData()
		case "r":
			if m.mode != modeUninstall {
				m.mode = modeUninstall
				m.loading = true
				m.selectedIndex = 0
				m.textInput.SetValue("")
				return m, getInstalledPackages()
			}
		case "u":
			m.mode = modeUpdate
			return m, syncRepositoriesInTerminal()
		case "s":
			if m.mode == modeUpdate {
				m.mode = modeUpdateSelective
				m.selectedIndex = 0
				m.textInput.Focus()
				if len(m.pendingUpdates) > 0 {
					m.filtered = m.pendingUpdates
					m.loadingInfo = true
					m.infoForPackage = m.filtered[0].Name
					return m, getPackageInfo(m.filtered[0])
				}
			}
		case "a", "y", "Y":
			if m.mode == modeUpdate && !m.loading && len(m.pendingUpdates) > 0 {
				m.statusMessage = "Running system update..."
				return m, executeUpdateInTerminal()
			}
		case "i":
			if m.mode != modeInstall {
				m.mode = modeInstall
				m.selectedIndex = 0
				m.filtered = []Package{}
				m.textInput.SetValue("")
				m.textInput.Focus()
			}
		case "tab", "m":
			return m.handleMarking()
		case "enter":
			return m.handleActionTrigger()
		case "*":
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
			m.performFiltering()
		}

	case syncRepositoriesMsg:
		if msg.err == nil {
			m.loading = true
			return m, checkUpdates()
		}

	case aurSearchMsg:
		m.searchingAUR = false
		if msg.err == nil {
			m.aurPackages = msg.packages
			m.performFiltering()
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
				return m, getPackageInfo(*pkg)
			}
		}

	case installedPackagesMsg:
		m.loading = false
		if msg.err == nil {
			m.installed = msg.packages
			m.performFiltering()
		}

	case dashboardMsg:
		m.loading = false
		if msg.err == nil { m.dashboard = msg.data }

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
			return m, debouncePackageInfo(m.pendingInfoPackage)
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
	switch msg.String() {
	case "y", "Y", "enter":
		m.showConfirmation = false
		switch m.confirmType {
		case confirmInstall: return m, executeInstallInTerminal(m.confirmPackages)
		case confirmUninstall: return m, executeUninstallInTerminal(m.confirmPackages)
		case confirmUpdate: return m, executeUpdateInTerminal()
		case confirmSelectiveUpdate: return m, executeSelectiveUpdateInTerminal(m.confirmPackages)
		case confirmCleanKeep3: return m, executeCleanCache(confirmCleanKeep3, 3, false)
		case confirmCleanKeep1: return m, executeCleanCache(confirmCleanKeep1, 1, false)
		case confirmCleanUninstalled: return m, executeCleanCache(confirmCleanUninstalled, 0, true)
		case confirmCleanNuke: return m, executeCleanCache(confirmCleanNuke, 0, false)
		case confirmCleanSelective: return m, executeSelectiveClean(m.confirmPackages, m.dashboard.PacmanCachePath, m.dashboard.ParuCachePath)
		case confirmRemoveOrphans: return m, executeRemoveOrphansInTerminal(m.confirmPackages)
		}
	case "n", "esc": m.showConfirmation = false
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
	
	key := strings.ToLower(msg.String())
	switch key {
	case "esc": m.selectionPanelFocused = false
	case "up", "k": if m.selectionPanelIndex > 0 { m.selectionPanelIndex-- }
	case "down", "j": if m.selectionPanelIndex < len(names)-1 { m.selectionPanelIndex++ }
	case "tab", "m":
		if m.selectionPanelIndex < len(names) {
			delete(m.markedPackages, names[m.selectionPanelIndex])
			if len(m.markedPackages) == 0 { m.selectionPanelFocused = false }
		}
	case "enter": m.selectionPanelFocused = false; return m.handleActionTrigger()
	}
	return m, nil
}

func (m *model) performFiltering() {
	query := m.textInput.Value()
	if m.mode == modeInstall { m.filterAllPackages(query) }
	if m.mode == modeUninstall { m.filteredInstalled = fuzzyFilter(m.installed, query) }
	if m.mode == modeUpdateSelective { m.filtered = fuzzyFilter(m.updatableAll, query) }
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
	return m, getDashboardData()
}
