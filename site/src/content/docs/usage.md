---
title: "Usage"
description: "How to use Gaur to manage packages"
---

# Usage

Gaur is designed for speed and productivity, with a focus on keyboard-driven interactions. Master its keybindings to navigate and manage packages efficiently.

## Global Keybindings

These keybindings are available in all modes:

| Key      | Action                                        |
| -------- | --------------------------------------------- |
| `i`      | Switch to **Install** mode                    |
| `d`      | Switch to **Dash** (dashboard) mode            |
| `r`      | Switch to **Remove** mode                     |
| `u`      | Switch to **Update** mode / Check for updates |
| `q`      | Quit                                          |
| `Ctrl+C` | Force quit                                    |

## Navigation

Use these keys to move through lists and search for packages:

| Key       | Action                           |
| --------- | -------------------------------- |
| `/`       | Focus search input               |
| `↑` / `k` | Move selection up                |
| `↓` / `j` | Move selection down              |
| `PgUp`    | Jump 10 items up                 |
| `PgDown`  | Jump 10 items down               |
| `Esc`     | Defocus input / Clear selections |

## Package Operations

Manage your packages with these actions:

| Key       | Action                                     |
| --------- | ------------------------------------------ |
| `Tab`/`m` | Mark/unmark package for batch operation    |
| `Enter`   | Install/remove selected or marked packages |
| `*`       | Toggle selection panel focus               |

## Dashboard (Dash Mode)

Monitor and manage your system in **Dash Mode**:

| Key | Action                                       |
| --- | -------------------------------------------- |
| `t` | Jump to Remove mode → All packages           |
| `e` | Jump to Remove mode → Explicit packages      |
| `f` | Jump to Remove mode → Foreign (AUR) packages |
| `o` | Jump to Remove mode → Orphan packages        |
| `c` | Clean package cache                          |
| `R` | Remove all orphan packages                   |

## Confirmation Dialogs

When Gaur prompts for confirmation, use these keys:

| Key           | Action              |
| ------------- | ------------------- |
| `y` / `Enter` | Confirm operation   |
| `n` / `Esc`   | Cancel operation    |
| `↑` / `↓`     | Scroll package list |

## Search Filters

Gaur supports powerful search filters to narrow down your results.

### Install Mode

Prefix your search with repository filters:

| Prefix | Repository |
| ------ | ---------- |
| `c:`   | Core       |
| `e:`   | Extra      |
| `m:`   | Multilib   |
| `a:`   | AUR        |

**Example:** `ae:firefox` searches for "firefox" in both official repos and the AUR.

### Remove Mode

Filter installed packages by type:

| Prefix | Filter                 |
| ------ | ---------------------- |
| `t:`   | Total (all packages)   |
| `e:` / `l:`   | Explicitly installed (local) |
| `f:` / `a:`   | Foreign (AUR) packages |
| `o:`   | Orphan packages        |

**Example:** `of:google` searches for orphaned AUR packages matching \"google\".
