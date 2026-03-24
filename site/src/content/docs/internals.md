---
title: "Internals"
description: "How gaur works under the hood"
---

# Project Internals

This section provides a technical deep dive into how gaur operates, its architecture, and the design decisions that power its interactive experience.

## Architecture Overview

gaur is built using a modular architecture that separates the UI logic from the system package management operations.

### Component Breakdown

1.  **The Model (`model.go`)**: The central state of the application. It holds configuration, package lists, search results, and UI state (like selected index and marked packages).
2.  **The View (`view.go`, `styles.go`)**: Responsible for rendering the TUI using the [Lip Gloss](https://github.com/charmbracelet/lipgloss) library. It calculates layout dimensions and applies themes.
3.  **The Commands (`commands.go`)**: Handles interaction with the host system. This includes executing `pacman`, the AUR helper (`paru`/`yay`), and `fzf` for fuzzy searching.
4.  **The Dashboard (`feature_dashboard.go`)**: A specialized module that gathers system statistics, calculates disk usage, and identifies cache hogs.

## Working with AUR Helpers

gaur doesn't reinvent package management. Instead, it acts as a high-fidelity wrapper around established tools:

- **Searching**: When you type in Install mode, gaur triggers a debounced search that queries the local `pacman` database and the AUR (via the configured helper).
- **Fuzzy Matching**: Results are piped through `fzf` for lightning-fast, relevance-ranked matching.
- **Hand-off**: For operations that require user interaction (like password entry or conflict resolution), gaur uses `tea.ExecProcess` to hand control over to the AUR helper in the terminal, ensuring full interactivity and safety.

## Real-time Refresh Logic

After any operation that modifies the system state (install, remove, update), gaur triggers a unified refresh. This re-scans the system to update dashboard stats, repo distributions, and package lists instantly, ensuring the UI always reflects the current state of your Arch Linux installation.
