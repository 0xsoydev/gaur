package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.MouseMsg:
		if m.showConfirmation {
			switch msg.Type {
			case tea.MouseWheelUp:
				if m.confirmScrollOffset > 0 {
					m.confirmScrollOffset--
				}
			case tea.MouseWheelDown:
				if m.confirmScrollOffset < m.maxConfirmScroll {
					m.confirmScrollOffset++
				}
			}
			return m, nil
		}

		if msg.Type == tea.MouseWheelUp || msg.Type == tea.MouseWheelDown {
			// Determine if we should scroll the details pane or the list
			isSelectiveUpdate := m.mode == modeUpdateSelective
			isOtherTwoPane := m.mode == modeInstall || m.mode == modeUninstall

			// Selective Update uses vertical split: Left side list, Right side details
			// Other two-pane modes use horizontal split: Top side details, Bottom side list
			
			shouldScrollDetails := false
			if isSelectiveUpdate {
				// Vertical split threshold
				shouldScrollDetails = msg.X >= m.width/2
			} else if isOtherTwoPane {
				// Horizontal split threshold
				shouldScrollDetails = msg.Y < m.height/2
			}

			if (isSelectiveUpdate || isOtherTwoPane) && shouldScrollDetails {
				// Scroll Details Pane
				if msg.Type == tea.MouseWheelUp {
					if m.infoScrollOffset > 0 {
						m.infoScrollOffset--
					}
				} else {
					if m.infoScrollOffset < m.maxInfoScroll {
						m.infoScrollOffset++
					}
				}
				return m, nil
			}

			// Scroll List Pane
			if m.mode == modeUpdate {
				if msg.Type == tea.MouseWheelUp {
					if m.updateScrollOffset > 0 {
						m.updateScrollOffset--
					}
				} else {
					if m.updateScrollOffset < m.maxUpdateScroll {
						m.updateScrollOffset++
					}
				}
				return m, nil
			}

			if (isSelectiveUpdate || isOtherTwoPane) && !shouldScrollDetails {
				// List Navigation
				maxIndex := 0
				if m.mode == modeInstall || m.mode == modeUpdateSelective {
					maxIndex = len(m.filtered) - 1
				} else if m.mode == modeUninstall {
					maxIndex = len(m.filteredInstalled) - 1
				}

				if msg.Type == tea.MouseWheelUp {
					if m.selectedIndex < maxIndex {
						m.infoScrollOffset = 0
						m.selectedIndex++
						name := ""
						if m.mode == modeUninstall {
							name = m.filteredInstalled[m.selectedIndex].Name
						} else {
							name = m.filtered[m.selectedIndex].Name
						}
						m.loadingInfo = true
						m.pendingInfoPackage = name
						return m, debouncePackageInfo(m.pendingInfoPackage)
					}
				} else {
					if m.selectedIndex > 0 {
						m.infoScrollOffset = 0
						m.selectedIndex--
						name := ""
						if m.mode == modeUninstall {
							name = m.filteredInstalled[m.selectedIndex].Name
						} else {
							name = m.filtered[m.selectedIndex].Name
						}
						m.loadingInfo = true
						m.pendingInfoPackage = name
						return m, debouncePackageInfo(m.pendingInfoPackage)
					}
				}
				return m, nil
			}
		}
		return m, nil
	case tea.KeyMsg:

		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

		if msg.String() == "ctrl+r" {
			if m.mode == modeInstalled {
				m.loading = true
				m.statusMessage = "Refreshing dashboard..."
				return m, getDashboardData()
			}
			return m, nil
		}

		if m.showErrorOverlay {
			if msg.String() == "esc" || msg.String() == "enter" || msg.String() == "q" {
				m.showErrorOverlay = false
				m.errorTitle = ""
				m.errorMessage = ""
				m.errorDetails = ""
				return m, nil
			}
			return m, nil
		}

		if m.showConfirmation {
			switch msg.String() {
			case "y", "Y", "enter":
				m.showConfirmation = false
				m.confirmScrollOffset = 0
				switch m.confirmType {
				case confirmInstall:
					m.statusMessage = fmt.Sprintf("Installing %d package(s)...", len(m.confirmPackages))
					return m, executeInstallInTerminal(m.confirmPackages)
				case confirmUninstall:
					m.statusMessage = fmt.Sprintf("Removing %d package(s)...", len(m.confirmPackages))
					return m, executeUninstallInTerminal(m.confirmPackages)
				case confirmUpdate:
					m.statusMessage = "Running system update..."
					return m, executeUpdateInTerminal()
				case confirmSelectiveUpdate:
					m.statusMessage = fmt.Sprintf("Updating %d package(s)...", len(m.confirmPackages))
					return m, executeSelectiveUpdateInTerminal(m.confirmPackages)
				case confirmCleanCache:
					m.statusMessage = "Cleaning package cache..."
					return m, executeCleanCacheInTerminal()
				case confirmRemoveOrphans:
					m.statusMessage = fmt.Sprintf("Removing %d orphan package(s)...", len(m.confirmPackages))
					orphans := m.confirmPackages
					m.confirmPackages = nil
					return m, executeRemoveOrphansInTerminal(orphans)
				}
			case "s":
				return m, nil
			case "down", "j":

				maxVisible := 10
				var total int
				if m.confirmType == confirmUpdate {
					total = len(m.pendingUpdates)
				} else {
					total = len(m.confirmPackages)
				}
				maxScroll := total - maxVisible
				if maxScroll < 0 {
					maxScroll = 0
				}
				if m.confirmScrollOffset < maxScroll {
					m.confirmScrollOffset++
				}
				return m, nil
			case "up", "k":

				if m.confirmScrollOffset > 0 {
					m.confirmScrollOffset--
				}
				return m, nil
			case "n", "esc":
				m.showConfirmation = false
				m.confirmScrollOffset = 0
				if m.mode == modeUpdateSelective {
					m.statusMessage = fmt.Sprintf("%d packages marked", len(m.markedPackages))
				} else if m.mode == modeUpdate {
					m.statusMessage = fmt.Sprintf("%d updates available", len(m.updatableAll))
				}
				return m, nil
			}
			return m, nil
		}

		if msg.String() == "*" {
			if len(m.markedPackages) > 0 {
				m.selectionPanelFocused = !m.selectionPanelFocused
				if m.selectionPanelFocused {
					m.textInput.Blur()
					m.selectionPanelIndex = 0
					m.selectionScrollOffset = 0
					m.statusMessage = "Selection panel: [↑↓] navigate  [tab] deselect  [enter] install  [*] close"
				} else {
					m.statusMessage = fmt.Sprintf("%d packages marked", len(m.markedPackages))
				}
			}
			return m, nil
		}

		if m.selectionPanelFocused {
			// Get sorted package names (same order as displayed)
			var pkgNames []string
			for name := range m.markedPackages {
				pkgNames = append(pkgNames, name)
			}
			sort.Strings(pkgNames)
			maxIdx := len(pkgNames) - 1
			maxVisible := 20

			switch msg.String() {
			case "esc", "*":
				m.selectionPanelFocused = false
				m.statusMessage = fmt.Sprintf("%d packages marked", len(m.markedPackages))
				return m, nil
			case "up", "k":
				if m.selectionPanelIndex > 0 {
					m.selectionPanelIndex--
					if m.selectionPanelIndex < m.selectionScrollOffset {
						m.selectionScrollOffset = m.selectionPanelIndex
					}
				}
				return m, nil
			case "down", "j":
				if m.selectionPanelIndex < maxIdx {
					m.selectionPanelIndex++
					if m.selectionPanelIndex >= m.selectionScrollOffset+maxVisible {
						m.selectionScrollOffset = m.selectionPanelIndex - maxVisible + 1
					}
				}
				return m, nil
			case "tab":

				if m.selectionPanelIndex < len(pkgNames) {
					nameToRemove := pkgNames[m.selectionPanelIndex]
					delete(m.markedPackages, nameToRemove)

					if m.selectionPanelIndex >= len(m.markedPackages) && m.selectionPanelIndex > 0 {
						m.selectionPanelIndex--
					}

					// Adjust scroll offset if out of bounds after deletion
					if m.selectionPanelIndex < m.selectionScrollOffset {
						m.selectionScrollOffset = m.selectionPanelIndex
					} else if len(m.markedPackages) <= maxVisible {
						m.selectionScrollOffset = 0
					}

					if len(m.markedPackages) == 0 {
						m.selectionPanelFocused = false
						m.statusMessage = "All selections cleared"
					} else {
						m.statusMessage = fmt.Sprintf("%d packages marked - [tab] to deselect", len(m.markedPackages))
					}
				}
				return m, nil
			case "enter":

				m.selectionPanelFocused = false
				if len(m.markedPackages) > 0 {
					if m.mode == modeInstall {
						var pkgsToInstall []string
						for name := range m.markedPackages {
							if !m.installedSet[name] {
								pkgsToInstall = append(pkgsToInstall, name)
							}
						}
						if len(pkgsToInstall) > 0 {
							sort.Strings(pkgsToInstall)
							m.showConfirmation = true
							m.confirmType = confirmInstall
							m.confirmPackages = pkgsToInstall
							m.confirmScrollOffset = 0
							m.markedPackages = make(map[string]bool)
							m.statusMessage = "Confirm installation"
						} else {
							m.statusMessage = "All marked packages are already installed"
						}
					} else if m.mode == modeUninstall {
						var pkgsToUninstall []string
						for name := range m.markedPackages {
							pkgsToUninstall = append(pkgsToUninstall, name)
						}
						sort.Strings(pkgsToUninstall)
						m.showConfirmation = true
						m.confirmType = confirmUninstall
						m.confirmPackages = pkgsToUninstall
						m.confirmScrollOffset = 0
						m.markedPackages = make(map[string]bool)
						m.statusMessage = "Confirm removal"
					} else if m.mode == modeUpdateSelective {
						var pkgsToUpdate []string
						for name := range m.markedPackages {
							pkgsToUpdate = append(pkgsToUpdate, name)
						}
						sort.Strings(pkgsToUpdate)
						m.showConfirmation = true
						m.confirmType = confirmSelectiveUpdate
						m.confirmPackages = pkgsToUpdate
						m.confirmScrollOffset = 0
						m.markedPackages = make(map[string]bool)
						m.statusMessage = "Confirm partial update"
					}
				}
				return m, nil
			}
			return m, nil
		}

		if m.textInput.Focused() {
			switch msg.String() {
			case "esc":
				if m.mode == modeUpdateSelective {
					m.mode = modeUpdate
					m.textInput.Blur()
					m.markedPackages = make(map[string]bool)
				} else {
					m.textInput.Blur()
				}
				return m, nil
			case "down":

				if m.selectedIndex > 0 {
					m.selectedIndex--
					if m.mode == modeInstall && len(m.filtered) > 0 {
						m.loadingInfo = true
						m.pendingInfoPackage = m.filtered[m.selectedIndex].Name
						return m, debouncePackageInfo(m.pendingInfoPackage)
					} else if m.mode == modeUninstall && len(m.filteredInstalled) > 0 {
						m.loadingInfo = true
						m.pendingInfoPackage = m.filteredInstalled[m.selectedIndex].Name
						return m, debouncePackageInfo(m.pendingInfoPackage)
					} else if m.mode == modeUpdateSelective && len(m.filtered) > 0 {
						m.loadingInfo = true
						m.pendingInfoPackage = m.filtered[m.selectedIndex].Name
						return m, debouncePackageInfo(m.pendingInfoPackage)
					}
				}
				return m, nil
			case "up":

				maxIndex := 0
				if m.mode == modeInstall || m.mode == modeUpdateSelective {
					maxIndex = len(m.filtered) - 1
				} else if m.mode == modeUninstall {
					maxIndex = len(m.filteredInstalled) - 1
				}
				if m.selectedIndex < maxIndex {
					m.selectedIndex++
					if m.mode == modeInstall && len(m.filtered) > 0 {
						m.loadingInfo = true
						m.pendingInfoPackage = m.filtered[m.selectedIndex].Name
						return m, debouncePackageInfo(m.pendingInfoPackage)
					} else if m.mode == modeUninstall && len(m.filteredInstalled) > 0 {
						m.loadingInfo = true
						m.pendingInfoPackage = m.filteredInstalled[m.selectedIndex].Name
						return m, debouncePackageInfo(m.pendingInfoPackage)
					} else if m.mode == modeUpdateSelective && len(m.filtered) > 0 {
						m.loadingInfo = true
						m.pendingInfoPackage = m.filtered[m.selectedIndex].Name
						return m, debouncePackageInfo(m.pendingInfoPackage)
					}
				}
				return m, nil
			case "enter":
				if m.mode == modeInstall && len(m.filtered) > 0 {

					if len(m.markedPackages) > 0 {
						var pkgsToInstall []string
						for name := range m.markedPackages {
							if !m.installedSet[name] {
								pkgsToInstall = append(pkgsToInstall, name)
							}
						}
						if len(pkgsToInstall) > 0 {
							sort.Strings(pkgsToInstall)
							m.showConfirmation = true
							m.confirmType = confirmInstall
							m.confirmPackages = pkgsToInstall
							m.confirmScrollOffset = 0
							m.markedPackages = make(map[string]bool)
							m.statusMessage = "Confirm installation"
						} else {
							m.statusMessage = "All marked packages are already installed"
						}
					} else {

						pkg := m.filtered[m.selectedIndex]
						if !pkg.Installed {
							m.showConfirmation = true
							m.confirmType = confirmInstall
							m.confirmPackages = []string{pkg.Name}
							m.confirmScrollOffset = 0
							m.statusMessage = "Confirm installation"
						} else {
							m.statusMessage = fmt.Sprintf("%s is already installed", pkg.Name)
						}
					}
				} else if m.mode == modeUninstall && len(m.filteredInstalled) > 0 {

					if len(m.markedPackages) > 0 {
						var pkgsToUninstall []string
						for name := range m.markedPackages {
							pkgsToUninstall = append(pkgsToUninstall, name)
						}
						sort.Strings(pkgsToUninstall)
						m.showConfirmation = true
						m.confirmType = confirmUninstall
						m.confirmPackages = pkgsToUninstall
						m.confirmScrollOffset = 0
						m.markedPackages = make(map[string]bool)
						m.statusMessage = "Confirm removal"
					} else {

						pkg := m.filteredInstalled[m.selectedIndex]
						m.showConfirmation = true
						m.confirmType = confirmUninstall
						m.confirmPackages = []string{pkg.Name}
						m.confirmScrollOffset = 0
						m.statusMessage = "Confirm removal"
					}
				} else if m.mode == modeUpdateSelective && len(m.filtered) > 0 {
					if len(m.markedPackages) > 0 {
						var pkgsToUpdate []string
						for name := range m.markedPackages {
							pkgsToUpdate = append(pkgsToUpdate, name)
						}
						sort.Strings(pkgsToUpdate)
						m.showConfirmation = true
						m.confirmType = confirmSelectiveUpdate
						m.confirmPackages = pkgsToUpdate
						m.confirmScrollOffset = 0
						m.markedPackages = make(map[string]bool)
						m.statusMessage = "Confirm partial update"
					} else {
						pkg := m.filtered[m.selectedIndex]
						m.showConfirmation = true
						m.confirmType = confirmSelectiveUpdate
						m.confirmPackages = []string{pkg.Name}
						m.confirmScrollOffset = 0
						m.statusMessage = "Confirm partial update"
					}
				}
				return m, nil
			case "tab":

				if m.mode == modeInstall && len(m.filtered) > 0 {
					pkg := m.filtered[m.selectedIndex]
					if m.markedPackages[pkg.Name] {
						delete(m.markedPackages, pkg.Name)
					} else {
						m.markedPackages[pkg.Name] = true
					}
					markedCount := len(m.markedPackages)
					if markedCount > 0 {
						m.statusMessage = fmt.Sprintf("%d packages marked", markedCount)
					} else {
						m.statusMessage = fmt.Sprintf("Found %d packages", len(m.filtered))
					}
				} else if m.mode == modeUninstall && len(m.filteredInstalled) > 0 {
					pkg := m.filteredInstalled[m.selectedIndex]
					if m.markedPackages[pkg.Name] {
						delete(m.markedPackages, pkg.Name)
					} else {
						m.markedPackages[pkg.Name] = true
					}
					markedCount := len(m.markedPackages)
					if markedCount > 0 {
						m.statusMessage = fmt.Sprintf("%d packages marked", markedCount)
					} else {
						m.statusMessage = fmt.Sprintf("%d installed packages", len(m.installed))
					}
				} else if m.mode == modeUpdateSelective && len(m.filtered) > 0 {
					pkg := m.filtered[m.selectedIndex]
					if m.markedPackages[pkg.Name] {
						delete(m.markedPackages, pkg.Name)
					} else {
						m.markedPackages[pkg.Name] = true
					}
					markedCount := len(m.markedPackages)
					if markedCount > 0 {
						m.statusMessage = fmt.Sprintf("%d packages marked", markedCount)
					} else {
						m.statusMessage = fmt.Sprintf("%d updates available", len(m.updatableAll))
					}
				}
				return m, nil
			}
			// All other keys go to text input
			var cmd tea.Cmd
			m.textInput, cmd = m.textInput.Update(msg)
			cmds = append(cmds, cmd)

			if m.mode == modeInstall {
				query := m.textInput.Value()
				if query != m.lastQuery {
					m.lastQuery = query

					repoFilters, searchQuery := parseRepoFilter(query)
					effectiveQueryLen := len(searchQuery)

					hasRepoFilter := len(repoFilters) > 0

					if effectiveQueryLen >= minSearchQueryLen || hasRepoFilter {

						m.filterAllPackages(query)
						m.selectedIndex = 0

						includesAUR := len(repoFilters) == 0 || repoFilters["aur"]
						shouldSearchAUR := includesAUR &&
							effectiveQueryLen >= minSearchQueryLen &&
							searchQuery != m.lastAURQuery

						if shouldSearchAUR {
							m.lastAURQuery = searchQuery
							m.searchingAUR = true
							cmds = append(cmds, searchAUR(searchQuery))
						}

						if len(m.filtered) > 0 {
							status := fmt.Sprintf("Found %d packages", len(m.filtered))
							if hasRepoFilter {
								status = fmt.Sprintf("Found %d %s packages", len(m.filtered), formatRepoFilters(repoFilters))
							}
							if m.searchingAUR {
								status += " (searching AUR...)"
							}
							m.statusMessage = status
							m.loadingInfo = true
							m.infoForPackage = m.filtered[0].Name
							cmds = append(cmds, getPackageInfo(m.filtered[0]))
						} else {
							if m.searchingAUR {
								m.statusMessage = "Searching AUR..."
							} else if hasRepoFilter && searchQuery == "" {
								m.statusMessage = fmt.Sprintf("No packages in %s", formatRepoFilters(repoFilters))
							} else {
								m.statusMessage = fmt.Sprintf("No matches for '%s'", query)
							}
							m.packageInfo = ""
							m.infoForPackage = ""
						}
					} else {
						m.filtered = []Package{}
						m.aurPackages = []Package{}
						m.lastAURQuery = ""
						m.packageInfo = ""
						m.infoForPackage = ""
						m.matchIndices = nil
						if len(m.repoPackages) > 0 {
							m.statusMessage = fmt.Sprintf("Type at least %d chars or use  to filter (c: e: m: a:) (%d repo packages)", minSearchQueryLen, len(m.repoPackages))
						} else {
							m.statusMessage = "Loading package database..."
						}
					}
				}
			} else if m.mode == modeUninstall {
				query := m.textInput.Value()
				if len(m.installed) > 0 {
					if query == "" {
						m.filteredInstalled = m.installed
						m.installedMatchIndices = nil
						m.statusMessage = fmt.Sprintf("%d installed packages", len(m.installed))
					} else {

						sourceFilters, searchQuery := parseUninstallFilter(query)
						hasSourceFilter := len(sourceFilters) > 0

						basePackages := m.installed

						if hasSourceFilter {
							var filtered []Package
							for _, pkg := range basePackages {

								if sourceFilters["total"] {
									filtered = append(filtered, pkg)
								} else {

									if sourceFilters["explicit"] && pkg.Explicit {
										filtered = append(filtered, pkg)
									}

									if sourceFilters["foreign"] && pkg.Source == "aur" {
										filtered = append(filtered, pkg)
									}

									if sourceFilters["orphan"] && pkg.Orphan {
										filtered = append(filtered, pkg)
									}
								}
							}
							basePackages = filtered
						}

						if searchQuery != "" {
							m.filteredInstalled = fuzzyFilter(basePackages, searchQuery)
							m.installedMatchIndices = computeAllMatchIndices(m.filteredInstalled, searchQuery)
						} else {
							m.filteredInstalled = basePackages
							m.installedMatchIndices = nil
						}

						if hasSourceFilter {
							m.statusMessage = fmt.Sprintf("Found %d %s packages", len(m.filteredInstalled), formatUninstallFilters(sourceFilters))
						} else {
							m.statusMessage = fmt.Sprintf("Showing %d of %d packages", len(m.filteredInstalled), len(m.installed))
						}
					}
					if m.selectedIndex >= len(m.filteredInstalled) {
						m.selectedIndex = 0
					}
					if len(m.filteredInstalled) > 0 && m.filteredInstalled[m.selectedIndex].Name != m.infoForPackage {
						m.loadingInfo = true
						m.infoForPackage = m.filteredInstalled[m.selectedIndex].Name
						cmds = append(cmds, getPackageInfo(m.filteredInstalled[m.selectedIndex]))
					}
				}
			} else if m.mode == modeUpdateSelective {
				query := m.textInput.Value()
				if len(m.updatableAll) > 0 {
					if query == "" {
						m.filtered = m.updatableAll
						m.matchIndices = nil
						m.statusMessage = fmt.Sprintf("%d updates available", len(m.updatableAll))
					} else {
						repoFilters, searchQuery := parseRepoFilter(query)
						hasRepoFilter := len(repoFilters) > 0

						basePackages := m.updatableAll
						if hasRepoFilter {
							var filtered []Package
							for _, pkg := range basePackages {
								if repoFilters["aur"] && pkg.Source == "aur" {
									filtered = append(filtered, pkg)
								} else if repoFilters["core"] && pkg.Source == "core" {
									filtered = append(filtered, pkg)
								} else if repoFilters["extra"] && pkg.Source == "extra" {
									filtered = append(filtered, pkg)
								} else if repoFilters["multilib"] && pkg.Source == "multilib" {
									filtered = append(filtered, pkg)
								}
							}
							basePackages = filtered
						}

						if searchQuery != "" {
							m.filtered = fuzzyFilter(basePackages, searchQuery)
							m.matchIndices = computeAllMatchIndices(m.filtered, searchQuery)
						} else {
							m.filtered = basePackages
							m.matchIndices = nil
						}

						if hasRepoFilter {
							m.statusMessage = fmt.Sprintf("Found %d %s updates", len(m.filtered), formatRepoFilters(repoFilters))
						} else {
							m.statusMessage = fmt.Sprintf("Showing %d of %d updates", len(m.filtered), len(m.updatableAll))
						}
					}

					if m.selectedIndex >= len(m.filtered) {
						m.selectedIndex = 0
					}

					if len(m.filtered) > 0 && m.filtered[m.selectedIndex].Name != m.infoForPackage {
						m.loadingInfo = true
						m.infoForPackage = m.filtered[m.selectedIndex].Name

						if cachedInfo, ok := m.infoCache[m.infoForPackage]; ok {
							m.packageInfo = cachedInfo
							m.loadingInfo = false
						} else {
							cmds = append(cmds, getPackageInfo(m.filtered[m.selectedIndex]))
						}
					} else if len(m.filtered) == 0 {

						m.packageInfo = ""
						m.infoForPackage = ""
					}
				}
			}
			return m, tea.Batch(cmds...)
		}

		switch msg.String() {
		case "q":
			return m, tea.Quit

		case "esc":
			if m.textInput.Focused() {
				m.textInput.Blur()
				return m, nil
			}

			if m.mode == modeUpdateSelective {
				m.mode = modeUpdate
				m.markedPackages = make(map[string]bool)
				m.statusMessage = fmt.Sprintf("%d updates available", len(m.updatableAll))
				return m, nil
			}

			if len(m.markedPackages) > 0 {
				m.markedPackages = make(map[string]bool)
				m.statusMessage = "Selections cleared"
				return m, nil
			}

		case "c":

			if m.mode == modeInstalled && !m.loading {
				m.showConfirmation = true
				m.confirmType = confirmCleanCache
				m.confirmScrollOffset = 0
				m.statusMessage = "Confirm cache cleaning"
				return m, nil
			}

		case "R":

			if m.mode == modeInstalled && !m.loading && m.dashboard.Orphans > 0 {

				cmd := exec.Command("paru", "-Qdtq")
				var orphanList bytes.Buffer
				cmd.Stdout = &orphanList
				if err := cmd.Run(); err != nil {
					m.statusMessage = fmt.Sprintf("Failed to get orphan list: %v", err)
					return m, nil
				}
				if orphanList.Len() == 0 {
					m.statusMessage = "No orphans to remove"
					return m, nil
				}
				orphans := strings.Fields(orphanList.String())
				m.confirmPackages = orphans
				m.showConfirmation = true
				m.confirmType = confirmRemoveOrphans
				m.confirmScrollOffset = 0
				m.statusMessage = "Confirm orphan removal"
				return m, nil
			}

		case "t":

			if m.mode == modeInstalled && !m.loading {
				m.mode = modeUninstall
				m.loading = true
				m.statusMessage = "Loading all packages..."
				m.selectedIndex = 0
				m.textInput.SetValue("t:")
				m.textInput.Placeholder = "Filter (t: total  e: explicit  f: foreign  o: orphan)..."
				m.markedPackages = make(map[string]bool)
				return m, getInstalledPackages()
			}

		case "e":

			if m.mode == modeInstalled && !m.loading {
				m.mode = modeUninstall
				m.loading = true
				m.statusMessage = "Loading explicit packages..."
				m.selectedIndex = 0
				m.textInput.SetValue("e:")
				m.textInput.Placeholder = "Filter (t: total  e: explicit  f: foreign  o: orphan)..."
				m.markedPackages = make(map[string]bool)
				return m, getInstalledPackages()
			}

		case "f":

			if m.mode == modeInstalled && !m.loading {
				m.mode = modeUninstall
				m.loading = true
				m.statusMessage = "Loading foreign packages..."
				m.selectedIndex = 0
				m.textInput.SetValue("f:")
				m.textInput.Placeholder = "Filter (t: total  e: explicit  f: foreign  o: orphan)..."
				m.markedPackages = make(map[string]bool)
				return m, getInstalledPackages()
			}

		case "o":

			if m.mode == modeInstalled && !m.loading {
				m.mode = modeUninstall
				m.loading = true
				m.statusMessage = "Loading orphan packages..."
				m.selectedIndex = 0
				m.textInput.SetValue("o:")
				m.textInput.Placeholder = "Filter (t: total  e: explicit  f: foreign  o: orphan)..."
				m.markedPackages = make(map[string]bool)
				return m, getInstalledPackages()
			}

		case "n":
			if !m.textInput.Focused() {
				m.mode = modeInstalled
				m.loading = true
				m.statusMessage = "Loading system statistics..."
				m.markedPackages = make(map[string]bool)
				return m, getDashboardData()
			}

		case "r":
			if m.mode != modeUninstall {
				m.mode = modeUninstall
				m.loading = true
				m.statusMessage = "Loading installed packages..."
				m.selectedIndex = 0
				m.textInput.SetValue("")
				m.textInput.Placeholder = "Filter (t: total  e: explicit  f: foreign  o: orphan)..."
				m.markedPackages = make(map[string]bool)
				return m, getInstalledPackages()
			}

		case "u":
			if !m.textInput.Focused() {
				if m.mode != modeUpdate {

					m.mode = modeUpdate
					m.markedPackages = make(map[string]bool)
				}

				m.loading = false
				m.statusMessage = "Syncing package databases..."
				m.updateOutput = ""
				m.pendingUpdates = nil
				m.updatableAll = nil
				m.filtered = nil
				m.matchIndices = nil
				return m, syncRepositoriesInTerminal()
			}

		case "s":
			if m.mode == modeUpdate {
				m.mode = modeUpdateSelective
				m.selectedIndex = 0
				m.textInput.SetValue("")
				m.textInput.Placeholder = "Search updates (c: e: m: a:)..."
				m.textInput.Focus()
				if len(m.pendingUpdates) > 0 {
					m.filtered = m.pendingUpdates
					m.matchIndices = nil
					m.statusMessage = fmt.Sprintf("%d updates available - type prefixes to filter", len(m.filtered))
					m.loadingInfo = true
					m.infoForPackage = m.filtered[0].Name
					return m, getPackageInfo(m.filtered[0])
				}
				return m, nil
			}
			return m, nil
		case "a":

			if m.mode == modeUpdate && !m.loading && len(m.pendingUpdates) > 0 {
				m.statusMessage = "Running system update..."
				return m, executeUpdateInTerminal()
			}

		case "y", "Y":

			if m.mode == modeUpdate && !m.loading && len(m.pendingUpdates) > 0 {
				m.statusMessage = "Running system update..."
				return m, executeUpdateInTerminal()
			}

		case "i":
			if m.mode != modeInstall {
				m.mode = modeInstall
				m.selectedIndex = 0
				m.filtered = []Package{}
				m.packageInfo = ""
				m.statusMessage = "Press [/] to search packages"
				m.textInput.SetValue("")
				m.textInput.Placeholder = "Search packages..."
				m.markedPackages = make(map[string]bool)
				return m, nil
			}

		case "down", "j":
			if m.mode == modeUpdate {
				if m.updateScrollOffset < len(m.pendingUpdates)-1 {
					m.updateScrollOffset++
				}
				return m, nil
			}
			if m.selectedIndex > 0 {
				m.infoScrollOffset = 0
				m.selectedIndex--
				if m.mode == modeInstall && len(m.filtered) > 0 {
					m.loadingInfo = true
					m.pendingInfoPackage = m.filtered[m.selectedIndex].Name
					return m, debouncePackageInfo(m.pendingInfoPackage)
				} else if m.mode == modeUninstall && len(m.filteredInstalled) > 0 {
					m.loadingInfo = true
					m.pendingInfoPackage = m.filteredInstalled[m.selectedIndex].Name
					return m, debouncePackageInfo(m.pendingInfoPackage)
				} else if m.mode == modeUpdateSelective && len(m.filtered) > 0 {
					m.loadingInfo = true
					m.pendingInfoPackage = m.filtered[m.selectedIndex].Name
					return m, debouncePackageInfo(m.pendingInfoPackage)
				}
			}

		case "up", "k":
			if m.mode == modeUpdate {
				if m.updateScrollOffset > 0 {
					m.updateScrollOffset--
				}
				return m, nil
			}
			maxIndex := 0
			if m.mode == modeInstall || m.mode == modeUpdateSelective {
				maxIndex = len(m.filtered) - 1
			} else if m.mode == modeUninstall {
				maxIndex = len(m.filteredInstalled) - 1
			}
			if m.selectedIndex < maxIndex {
				m.infoScrollOffset = 0
				m.selectedIndex++
				if m.mode == modeInstall && len(m.filtered) > 0 {
					m.loadingInfo = true
					m.pendingInfoPackage = m.filtered[m.selectedIndex].Name
					return m, debouncePackageInfo(m.pendingInfoPackage)
				} else if m.mode == modeUninstall && len(m.filteredInstalled) > 0 {
					m.loadingInfo = true
					m.pendingInfoPackage = m.filteredInstalled[m.selectedIndex].Name
					return m, debouncePackageInfo(m.pendingInfoPackage)
				} else if m.mode == modeUpdateSelective && len(m.filtered) > 0 {
					m.loadingInfo = true
					m.pendingInfoPackage = m.filtered[m.selectedIndex].Name
					return m, debouncePackageInfo(m.pendingInfoPackage)
				}
			}

		case "enter":
			if m.mode == modeInstall && len(m.filtered) > 0 {

				if len(m.markedPackages) > 0 {
					var pkgsToInstall []string
					for name := range m.markedPackages {

						if !m.installedSet[name] {
							pkgsToInstall = append(pkgsToInstall, name)
						}
					}
					if len(pkgsToInstall) > 0 {
						sort.Strings(pkgsToInstall)
						m.showConfirmation = true
						m.confirmType = confirmInstall
						m.confirmPackages = pkgsToInstall
						m.confirmScrollOffset = 0
						m.markedPackages = make(map[string]bool)
						m.statusMessage = "Confirm installation"
					} else {
						m.statusMessage = "All marked packages are already installed"
					}
				} else {

					pkg := m.filtered[m.selectedIndex]
					if !pkg.Installed {
						m.showConfirmation = true
						m.confirmType = confirmInstall
						m.confirmPackages = []string{pkg.Name}
						m.confirmScrollOffset = 0
						m.statusMessage = "Confirm installation"
					} else {
						m.statusMessage = fmt.Sprintf("%s is already installed", pkg.Name)
					}
				}
			} else if m.mode == modeUninstall && len(m.filteredInstalled) > 0 {

				if len(m.markedPackages) > 0 {
					var pkgsToUninstall []string
					for name := range m.markedPackages {
						pkgsToUninstall = append(pkgsToUninstall, name)
					}
					sort.Strings(pkgsToUninstall)
					m.showConfirmation = true
					m.confirmType = confirmUninstall
					m.confirmPackages = pkgsToUninstall
					m.confirmScrollOffset = 0
					m.markedPackages = make(map[string]bool)
					m.statusMessage = "Confirm removal"
				} else {

					pkg := m.filteredInstalled[m.selectedIndex]
					m.showConfirmation = true
					m.confirmType = confirmUninstall
					m.confirmPackages = []string{pkg.Name}
					m.confirmScrollOffset = 0
					m.statusMessage = "Confirm removal"
				}
			} else if m.mode == modeUpdateSelective && len(m.filtered) > 0 {

				if len(m.markedPackages) > 0 {
					var pkgsToUpdate []string
					for name := range m.markedPackages {
						pkgsToUpdate = append(pkgsToUpdate, name)
					}
					sort.Strings(pkgsToUpdate)
					m.showConfirmation = true
					m.confirmType = confirmSelectiveUpdate
					m.confirmPackages = pkgsToUpdate
					m.confirmScrollOffset = 0
					m.markedPackages = make(map[string]bool)
					m.statusMessage = "Confirm partial update"
				} else {

					pkg := m.filtered[m.selectedIndex]
					m.showConfirmation = true
					m.confirmType = confirmSelectiveUpdate
					m.confirmPackages = []string{pkg.Name}
					m.confirmScrollOffset = 0
					m.statusMessage = "Confirm partial update"
				}
			} else if m.mode == modeUpdate && len(m.pendingUpdates) > 0 {
				m.showConfirmation = true
				m.confirmType = confirmUpdate
				m.confirmScrollOffset = 0

				// Re-populate the confirmation list with all pending updates
				var packageNames []string
				for _, pkg := range m.pendingUpdates {
					packageNames = append(packageNames, pkg.Name)
				}
				m.confirmPackages = packageNames
				m.statusMessage = fmt.Sprintf("Confirm update for %d packages", len(m.pendingUpdates))
			}

		case "tab":

			if m.mode == modeInstall && len(m.filtered) > 0 {
				pkg := m.filtered[m.selectedIndex]
				if m.markedPackages[pkg.Name] {
					delete(m.markedPackages, pkg.Name)
				} else {
					m.markedPackages[pkg.Name] = true
				}
				markedCount := len(m.markedPackages)
				if markedCount > 0 {
					m.statusMessage = fmt.Sprintf("%d packages marked", markedCount)
				} else {
					m.statusMessage = fmt.Sprintf("Found %d packages", len(m.filtered))
				}
			} else if m.mode == modeUninstall && len(m.filteredInstalled) > 0 {
				pkg := m.filteredInstalled[m.selectedIndex]
				if m.markedPackages[pkg.Name] {
					delete(m.markedPackages, pkg.Name)
				} else {
					m.markedPackages[pkg.Name] = true
				}
				markedCount := len(m.markedPackages)
				if markedCount > 0 {
					m.statusMessage = fmt.Sprintf("%d packages marked", markedCount)
				} else {
					m.statusMessage = fmt.Sprintf("%d installed packages", len(m.installed))
				}
			} else if m.mode == modeUpdateSelective && len(m.filtered) > 0 {
				pkg := m.filtered[m.selectedIndex]
				if m.markedPackages[pkg.Name] {
					delete(m.markedPackages, pkg.Name)
				} else {
					m.markedPackages[pkg.Name] = true
				}
				markedCount := len(m.markedPackages)
				if markedCount > 0 {
					m.statusMessage = fmt.Sprintf("%d packages marked", markedCount)
				} else {
					m.statusMessage = fmt.Sprintf("%d updates available", len(m.updatableAll))
				}
			}

		case "/":
			if m.mode == modeUpdate {
				m.mode = modeUpdateSelective
				m.textInput.Placeholder = "Search updates (c: e: m: a:)..."
				m.textInput.Focus()
				if len(m.updatableAll) > 0 {
					m.filtered = m.updatableAll
					m.statusMessage = fmt.Sprintf("Type to search among %d updates", len(m.updatableAll))
					m.loadingInfo = true
					m.infoForPackage = m.filtered[0].Name
					return m, getPackageInfo(m.filtered[0])
				}
			} else if (m.mode == modeInstall || m.mode == modeUninstall || m.mode == modeUpdateSelective) && !m.textInput.Focused() {
				m.textInput.Focus()
				if m.mode == modeInstall && len(m.repoPackages) > 0 && m.textInput.Value() == "" {
					m.statusMessage = fmt.Sprintf("Type at least %d chars or use prefix (c: e: m: a:) to filter (%d repo packages)", minSearchQueryLen, len(m.repoPackages))
				} else if m.mode == modeUninstall && len(m.installed) > 0 && m.textInput.Value() == "" {
					m.statusMessage = fmt.Sprintf("Filter: t: total  e: explicit  f: foreign  o: orphan (%d installed)", len(m.installed))
				} else if m.mode == modeUpdateSelective && len(m.updatableAll) > 0 && m.textInput.Value() == "" {
					m.statusMessage = fmt.Sprintf("Filter updates with prefix (c: e: m: a:) among %d updates", len(m.updatableAll))
				}
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.textInput.Width = msg.Width - 6

	case repoPackagesMsg:
		m.loading = false
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Failed to load packages: %v", msg.err)
		} else {
			m.repoPackages = msg.packages

			m.installedSet = make(map[string]bool)
			for _, pkg := range m.repoPackages {
				if pkg.Installed {
					m.installedSet[pkg.Name] = true
				}
			}

			query := m.textInput.Value()
			if m.mode == modeInstall && query != "" {
				repoFilters, searchQuery := parseRepoFilter(query)
				hasRepoFilter := len(repoFilters) > 0
				effectiveQueryLen := len(searchQuery)

				if effectiveQueryLen >= minSearchQueryLen || hasRepoFilter {
					m.filterAllPackages(query)

					m.selectedIndex = 0

					if len(m.filtered) > 0 {
						status := fmt.Sprintf("Found %d packages", len(m.filtered))
						if hasRepoFilter {
							status = fmt.Sprintf("Found %d %s packages", len(m.filtered), formatRepoFilters(repoFilters))
						}
						if m.lastCompletedOp != "" {
							status = m.lastCompletedOp + " | " + status
						}
						m.statusMessage = status

						m.loadingInfo = true
						m.infoForPackage = m.filtered[0].Name
						return m, getPackageInfo(m.filtered[0])
					} else {
						m.statusMessage = fmt.Sprintf("No matches for '%s'", query)
					}
				} else {
					m.filtered = []Package{}
					m.matchIndices = nil
					if m.lastCompletedOp != "" {
						m.statusMessage = m.lastCompletedOp
					} else {
						m.statusMessage = fmt.Sprintf("Loaded %d repo packages - press [/] to search", len(m.repoPackages))
					}
				}
			} else {
				if m.lastCompletedOp != "" {
					m.statusMessage = m.lastCompletedOp
				} else {
					m.statusMessage = fmt.Sprintf("Loaded %d repo packages - press [/] to search", len(m.repoPackages))
				}
			}
		}

	case syncRepositoriesMsg:
		if msg.err != nil {
			m.showErrorOverlay = true
			m.errorTitle = "Sync Failed"
			m.errorMessage = "Failed to synchronize package databases"
			m.errorDetails = msg.err.Error()
			m.loading = false
			return m, nil
		}

		// Synchronization succeeded, now we can safely check for updates in the background
		m.loading = true
		m.statusMessage = "Checking for updates..."
		return m, checkUpdates()

	case aurSearchMsg:
		m.searchingAUR = false

		currentQuery := m.textInput.Value()
		isExactMatch := msg.query == m.lastAURQuery
		isUsefulPrefix := strings.HasPrefix(strings.ToLower(currentQuery), strings.ToLower(msg.query))

		if !isExactMatch && !isUsefulPrefix {

			return m, nil
		}

		if msg.err == nil {

			if !isExactMatch && isUsefulPrefix {

				if len(m.aurPackages) == 0 && len(msg.packages) > 0 {
					m.aurPackages = msg.packages
				}
			} else {

				if len(msg.packages) > 0 {
					m.aurPackages = msg.packages
				}

			}

			query := m.textInput.Value()
			if len(query) >= minSearchQueryLen {

				wasOnFirst := m.selectedIndex == 0
				prevSelected := ""
				if !wasOnFirst && m.selectedIndex < len(m.filtered) {
					prevSelected = m.filtered[m.selectedIndex].Name
				}

				m.filterAllPackages(query)

				if wasOnFirst {
					m.selectedIndex = 0
				} else if prevSelected != "" {
					for i, pkg := range m.filtered {
						if pkg.Name == prevSelected {
							m.selectedIndex = i
							break
						}
					}
				}
				if m.selectedIndex >= len(m.filtered) {
					m.selectedIndex = 0
				}

				if len(m.filtered) > 0 {
					m.statusMessage = fmt.Sprintf("Found %d packages (%d from AUR)", len(m.filtered), len(msg.packages))

					if m.filtered[m.selectedIndex].Name != m.infoForPackage {
						m.loadingInfo = true
						m.infoForPackage = m.filtered[m.selectedIndex].Name
						return m, getPackageInfo(m.filtered[m.selectedIndex])
					}
				} else {
					m.statusMessage = fmt.Sprintf("No matches for '%s'", query)
				}
			}
		} else if len(m.filtered) == 0 {
			m.statusMessage = fmt.Sprintf("No matches for '%s'", m.textInput.Value())
		}

	case packageInfoMsg:

		if msg.packageName == m.infoForPackage {
			m.loadingInfo = false
			if msg.err != nil {
				m.packageInfo = "Failed to load package info"
			} else {
				m.packageInfo = msg.info
				m.infoCache[msg.packageName] = msg.info
			}
		} else if msg.err == nil {
			m.infoCache[msg.packageName] = msg.info
		}

	case debounceTickMsg:

		if msg.packageName == m.pendingInfoPackage {
			m.infoForPackage = msg.packageName

			// Instant lookup if cached
			if cachedInfo, ok := m.infoCache[msg.packageName]; ok {
				m.loadingInfo = false
				m.packageInfo = cachedInfo
				return m, nil
			}

			// Find the package and fetch its info
			var pkg *Package
			if m.mode == modeInstall {
				for i := range m.filtered {
					if m.filtered[i].Name == msg.packageName {
						pkg = &m.filtered[i]
						break
					}
				}
			} else if m.mode == modeUninstall {
				for i := range m.filteredInstalled {
					if m.filteredInstalled[i].Name == msg.packageName {
						pkg = &m.filteredInstalled[i]
						break
					}
				}
			} else if m.mode == modeUpdate || m.mode == modeUpdateSelective {
				for i := range m.filtered {
					if m.filtered[i].Name == msg.packageName {
						pkg = &m.filtered[i]
						break
					}
				}
			}
			if pkg != nil {
				m.infoScrollOffset = 0
				return m, getPackageInfo(*pkg)
			}
		}

	case installedPackagesMsg:
		m.loading = false
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Error loading packages: %v", msg.err)
		} else {
			m.installed = msg.packages

			m.installedSet = make(map[string]bool)
			for _, pkg := range m.installed {
				m.installedSet[pkg.Name] = true
			}

			for i := range m.repoPackages {
				m.repoPackages[i].Installed = m.installedSet[m.repoPackages[i].Name]
			}

			for i := range m.filtered {
				m.filtered[i].Installed = m.installedSet[m.filtered[i].Name]
			}

			query := m.textInput.Value()
			if query != "" {

				sourceFilters, searchQuery := parseUninstallFilter(query)
				hasSourceFilter := len(sourceFilters) > 0

				basePackages := m.installed
				if hasSourceFilter {
					var filtered []Package
					for _, pkg := range basePackages {
						if sourceFilters["total"] {
							filtered = append(filtered, pkg)
						} else {
							if sourceFilters["explicit"] && pkg.Explicit {
								filtered = append(filtered, pkg)
							}
							if sourceFilters["foreign"] && pkg.Source == "aur" {
								filtered = append(filtered, pkg)
							}
							if sourceFilters["orphan"] && pkg.Orphan {
								filtered = append(filtered, pkg)
							}
						}
					}
					basePackages = filtered
				}

				if searchQuery != "" {
					m.filteredInstalled = fuzzyFilter(basePackages, searchQuery)
					m.installedMatchIndices = computeAllMatchIndices(m.filteredInstalled, searchQuery)
				} else {
					m.filteredInstalled = basePackages
					m.installedMatchIndices = nil
				}

				m.selectedIndex = 0

				if hasSourceFilter {
					status := fmt.Sprintf("Found %d %s packages", len(m.filteredInstalled), formatUninstallFilters(sourceFilters))
					if m.lastCompletedOp != "" {
						status = m.lastCompletedOp + " | " + status
					}
					m.statusMessage = status
				} else {
					status := fmt.Sprintf("%d packages - Press [/] to filter", len(m.filteredInstalled))
					if m.lastCompletedOp != "" {
						status = m.lastCompletedOp + " | " + status
					}
					m.statusMessage = status
				}
			} else {
				m.filteredInstalled = m.installed
				status := fmt.Sprintf("%d packages - Press [/] to filter", len(m.installed))
				if m.lastCompletedOp != "" {
					status = m.lastCompletedOp + " | " + status
				}
				m.statusMessage = status
			}

			if len(m.filteredInstalled) > 0 {
				m.loadingInfo = true
				m.infoForPackage = m.filteredInstalled[0].Name
				return m, getPackageInfo(m.filteredInstalled[0])
			}
		}

	case dashboardMsg:
		m.loading = false

		m.dashboard = msg.data
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Dashboard loaded with warnings: %v", msg.err)
		} else {

			if m.lastCompletedOp != "" {
				m.statusMessage = m.lastCompletedOp
			} else {
				m.statusMessage = "Dashboard loaded"
			}
		}

	case actionCompleteMsg:
		m.loading = false
		if msg.err != nil {
			m.statusMessage = msg.message
		} else {
			m.statusMessage = msg.message

			if m.mode == modeInstall {

				return m, loadRepoPackages()
			} else if m.mode == modeUninstall {
				return m, getInstalledPackages()
			}
		}

	case cleanCacheMsg:
		m.loading = false
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Cache clean failed: %v", msg.err)
		} else {
			m.statusMessage = "Cache cleaned successfully!"

			return m, getDashboardData()
		}

	case removeOrphansMsg:
		m.loading = false
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Orphan removal failed: %v", msg.err)
		} else {
			m.statusMessage = "Orphans removed successfully!"

			return m, getDashboardData()
		}

	case updateOutputMsg:
		m.loading = false
		m.updateOutput = msg.output
		if msg.err != nil {
			m.statusMessage = "Update failed"
		} else {
			m.statusMessage = "Update complete!"
		}

	case updateCheckMsg:
		m.loading = false
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Error checking updates: %v", msg.err)
			m.checkAfterUpdate = false
		} else if len(msg.packages) == 0 {
			if m.checkAfterUpdate {
				m.checkAfterUpdate = false
				m.loading = true
				m.statusMessage = "Syncing repositories to double-check..."
				return m, syncRepositoriesInTerminal()
			}
			m.statusMessage = "System is up to date!"
			m.updateOutput = "No updates available."
			m.pendingUpdates = nil
			m.updatableAll = nil
			m.filtered = nil
		} else {
			m.checkAfterUpdate = false
			m.updatableAll = msg.packages
			m.pendingUpdates = msg.packages

			m.filtered = m.updatableAll
			if m.textInput.Value() != "" {
				m.matchIndices = computeAllMatchIndices(m.filtered, m.textInput.Value())
			} else {
				m.matchIndices = nil
			}
			m.selectedIndex = 0
			m.confirmScrollOffset = 0
			m.statusMessage = fmt.Sprintf("%d updates available", len(msg.packages))
			m.updateOutput = ""

			if len(m.filtered) > 0 {
				m.loadingInfo = true
				m.pendingInfoPackage = m.filtered[0].Name
				return m, debouncePackageInfo(m.pendingInfoPackage)
			}
		}

	case execCompleteMsg:
		m.loading = false
		m.confirmPackages = nil
		
		// If it was an update operation, don't clear pendingUpdates yet, 
		// we'll wait for the checkUpdates() call to finish and update it.
		// For other operations (install/uninstall), we clear it because we'll refresh the whole list.
		if msg.operation != confirmUpdate && msg.operation != confirmSelectiveUpdate {
			m.pendingUpdates = nil
		}

		if msg.err != nil {
			opName := ""
			switch msg.operation {
			case confirmInstall:
				opName = "Installation"
			case confirmUninstall:
				opName = "Removal"
			case confirmUpdate:
				opName = "System Update"
			case confirmSelectiveUpdate:
				opName = "Selective Update"
			case confirmCleanCache:
				opName = "Cache Cleaning"
			case confirmRemoveOrphans:
				opName = "Orphan Removal"
			}

			m.showErrorOverlay = true
			m.errorTitle = fmt.Sprintf("%s Failed", opName)
			m.errorMessage = "The operation exited with a non-zero exit code."

			if exitErr, ok := msg.err.(*exec.ExitError); ok {
				m.errorDetails = fmt.Sprintf("Exit code: %d\n\nThe error output was displayed in the terminal.\nPlease check the terminal output for details.", exitErr.ExitCode())
			} else {
				m.errorDetails = fmt.Sprintf("Error: %v\n\nThe error output was displayed in the terminal.\nPlease check the terminal output for details.", msg.err)
			}

			m.statusMessage = fmt.Sprintf("%s failed", opName)
			m.lastCompletedOp = ""

			switch msg.operation {
			case confirmInstall:
				return m, loadRepoPackages()
			case confirmUninstall:
				return m, getInstalledPackages()
			case confirmUpdate:
				return m, checkUpdates()
			case confirmSelectiveUpdate:
				return m, checkUpdates()
			case confirmCleanCache, confirmRemoveOrphans:
				return m, getDashboardData()
			}
			return m, nil
		}

		switch msg.operation {
		case confirmInstall:
			if len(msg.packages) == 1 {
				m.lastCompletedOp = fmt.Sprintf("Installed: %s", msg.packages[0])
			} else {
				m.lastCompletedOp = fmt.Sprintf("Installed %d packages", len(msg.packages))
			}
			m.statusMessage = m.lastCompletedOp
			return m, loadRepoPackages()
		case confirmUninstall:
			if len(msg.packages) == 1 {
				m.lastCompletedOp = fmt.Sprintf("Removed: %s", msg.packages[0])
			} else {
				m.lastCompletedOp = fmt.Sprintf("Removed %d packages", len(msg.packages))
			}
			m.statusMessage = m.lastCompletedOp
			return m, getInstalledPackages()
		case confirmUpdate:
			m.lastCompletedOp = "System update completed"
			m.statusMessage = m.lastCompletedOp
			m.checkAfterUpdate = true
			m.pendingUpdates = nil
			m.updatableAll = nil
			m.filtered = nil
			return m, checkUpdates()
		case confirmSelectiveUpdate:
			if len(msg.packages) == 1 {
				m.lastCompletedOp = fmt.Sprintf("Updated: %s", msg.packages[0])
			} else {
				m.lastCompletedOp = fmt.Sprintf("Updated %d packages", len(msg.packages))
			}
			m.statusMessage = m.lastCompletedOp
			m.checkAfterUpdate = true

			// Remove updated packages from our local state immediately for better UX
			updatedMap := make(map[string]bool)
			for _, name := range msg.packages {
				updatedMap[name] = true
			}

			newPending := []Package{}
			for _, p := range m.pendingUpdates {
				if !updatedMap[p.Name] {
					newPending = append(newPending, p)
				}
			}
			m.pendingUpdates = newPending

			newUpdatableAll := []Package{}
			for _, p := range m.updatableAll {
				if !updatedMap[p.Name] {
					newUpdatableAll = append(newUpdatableAll, p)
				}
			}
			m.updatableAll = newUpdatableAll
			m.filtered = m.updatableAll // Refresh filtered list too

			return m, checkUpdates()
		case confirmCleanCache:
			m.lastCompletedOp = "Cache cleaned successfully"
			m.statusMessage = m.lastCompletedOp
			return m, getDashboardData()
		case confirmRemoveOrphans:
			if len(msg.packages) == 1 {
				m.lastCompletedOp = fmt.Sprintf("Removed orphan: %s", msg.packages[0])
			} else {
				m.lastCompletedOp = fmt.Sprintf("Removed %d orphan packages", len(msg.packages))
			}
			m.statusMessage = m.lastCompletedOp
			return m, getDashboardData()
		}
	}

	return m, tea.Batch(cmds...)
}
