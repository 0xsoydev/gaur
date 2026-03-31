---
title: "internals"
description: "A look under the hood"
---

# internals

Curious how gaur works? Here's the technical breakdown.

## architecture

gaur follows a clean separation between UI and system operations. The codebase is modular and readable.

### core components

1. **Model (`model.go`):** Central state management. Holds config, package lists, search results, selected items, and UI state.

2. **View (`view.go`, `styles.go`):** Renders the TUI using [Lip Gloss](https://github.com/charmbracelet/lipgloss). Handles layout calculations and theme application.

3. **Commands (`commands.go`):** System interaction layer. Executes `pacman`, AUR helpers (`paru`/`yay`), and `fzf` for searching.

4. **Dashboard (`feature_dashboard.go`):** Gathers system stats: disk usage, cache size, orphan counts, repo distribution.

## aur helper integration

gaur doesn't replace your package manager. It wraps it.

- **Search:** When you type in install mode, gaur queries the local `pacman` database and the AUR through your configured helper. Results are debounced to avoid hammering the system.

- **Fuzzy matching:** Search results pipe through `fzf` for fast, relevance-ranked filtering.

- **Handoff:** Operations that need user input (password prompts, conflict resolution) hand control to the AUR helper via `tea.ExecProcess`. This keeps the terminal interactive and safe.

## real-time state sync

After every install, remove, or update operation, gaur runs a full refresh. It re-scans the system to update:

- Dashboard statistics
- Repository distribution charts
- Package lists and counts

The UI always reflects the actual state of your Arch installation. No stale data, no manual refreshes.
