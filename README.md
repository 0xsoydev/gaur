<div align="center">

<img src="gaur.png" alt="Gaur" width="800" />

# Gaur

**A beautiful, interactive TUI for Arch Linux package management**

Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) • Powered by [paru](https://github.com/Morganamilo/paru) or [yay](https://github.com/Jguer/yay)

> ⚠️ **Disclaimer:** This project is mostly vibecoded and continues to be developed through vibecoding.  
> Do report rough edges, and expect an occasional "it works on my machine" moment (trying my best to eliminate those).

</div>

---

## Table of Contents

- [Features](#-features)
- [Requirements](#-requirements)
- [Interface](#-interface)
- [Installation](#-installation)
- [Configuration](#-configuration)
- [Usage](#-usage)
- [How It Works](#-how-it-works)
- [License](#-license)

## ✨ Features

### 📦 Package Management

- **Fuzzy Search** — Lightning-fast fuzzy matching powered by `fzf` with match highlighting
- **Dynamic Helper Support** — Seamlessly switch between `paru` and `yay` via configuration
- **Repository Filtering** — Filter by source with prefixes: `c:` (core), `e:` (extra), `m:` (multilib), `a:` (aur)
- **Batch Operations** — Mark multiple packages with `Tab` and install/remove them all at once
- **Selective Updates** — Carefully manage your system by selecting specific packages to update rather than full updates
- **Interactive Hand-off** — Correctly handles terminal prompts for password entry and conflict resolution
- **Real-time Package Details** — View detailed package information with debounced loading

### 📊 System Dashboard

- **Package Statistics** — Total, explicit, foreign (AUR), and orphan package counts
- **Disk Space Analysis** — Capacity, usage breakdown (packages, cache, other, free) with visual proportional bars
- **Repository Distribution** — Breakdown of packages by repository (core, extra, multilib, AUR)
- **XDG Compliance** — Automatically respects `$XDG_CACHE_HOME` for all cache operations
- **Top Packages** — See your largest installed packages at a glance
- **Top Cache Hogs** — Identifies the largest packages taking up space across both pacman and helper caches
- **Cache Management** — Clean package caches via an interactive menu, choose keep policies, or selectively delete packages from cache
- **Orphan Removal** — Identify and remove orphaned packages

### 🎨 Interface

- **Mode-specific Theming** — Each mode (Install, Dash, Remove, Update) has its own color scheme
- **In-App Settings Menu** — Press `,` to instantly change themes, borders, and helpers without restarting
- **Selection Panel** — Dedicated panel for managing marked packages
- **Centered Dialogs** — All confirmation and error boxes are perfectly centered line-by-line
- **Automatic Refresh** — The entire UI refreshes automatically after any system change to ensure data integrity

## 📋 Requirements

- Arch Linux (or Arch-based distribution)
- [paru](https://github.com/Morganamilo/paru) or [yay](https://github.com/Jguer/yay) — AUR helper
- [fzf](https://github.com/junegunn/fzf) — Fuzzy finder (for search)
- Go 1.21+ (for building from source)

## 🖼️ Interface

```
╭─────────────────────────────────────────────────────────╭──────────────────╮
│ Repository   : extra                                    | Selected (2) [*] │
│ Name         : firefox                                  |  firefox         │
│ Version      : 133.0-1                                  |  firefoxpwa      |
│ Description  : Fast, Private & Safe Web Browser         ╰──────────────────│
│ Architecture : x86_64                                                      │
│ URL          : https://www.mozilla.org/firefox                             │
│                                                                            │
├────────────────────────────────────────────────────────────────────────────┤
│  extra/firefox-i18n-an 147.0.2-1                                           │
│  extra/firefox-i18n-af 147.0.2-1                                           │
│ *extra/firefoxpwa 2.18.0.1                                                 │
│>*extra/firefox 147.0.2-1 [installed]                                       │
│                                                                            │
│Found 610 packages (492 from AUR)                                           │
│> firefox                                                                   │
╰────────────────────────────────────────────────────────────────────────────╯
       [/] search  [tab] mark  [i]nstall  [d]ash  [r]emove  [u]pdate  [,] settings  [q]uit
```

## 🚀 Installation

### From Source

```bash
git clone https://github.com/prbhtkumr/gaur.git
cd gaur
go build -o gaur .
sudo mv gaur /usr/local/bin/
```

### Using go install

```bash
go install github.com/prbhtkumr/gaur@latest
```

## ⚙️ Configuration

Gaur creates a configuration file at `~/.config/gaur/config.toml` on first run. You can also configure most settings directly in the app using the **Settings Menu** (`,`).

```toml
[commands]
# Set your preferred AUR helper: "paru" (default) or "yay"
aur_helper = "paru"
# The tool used for cleaning the cache (defaults to paccache)
cache_tool = "paccache"
# Custom flags for install/remove
install_flags = ""
remove_flags = "-Rns"

[advanced]
# Set a custom cache directory (optional)
cache_dir = ""
```

## 📖 Usage

```bash
gaur
```

### Keybindings

#### Global

| Key      | Action                                        |
| -------- | --------------------------------------------- |
| `i`      | Switch to **Install** mode                    |
| `d`      | Switch to **Dash** (dashboard) mode            |
| `r`      | Switch to **Remove** mode                     |
| `u`      | Switch to **Update** mode / Check for updates |
| `,`      | Open **Settings** menu                        |
| `q`      | Quit                                          |
| `Ctrl+C` | Force quit                                    |

#### Navigation

| Key       | Action                           |
| --------- | -------------------------------- |
| `/`       | Focus search input               |
| `↑` / `k` | Move selection up                |
| `↓` / `j` | Move selection down              |
| `PgUp`    | Jump 10 items up                 |
| `PgDown`  | Jump 10 items down               |
| `Esc`     | Defocus input / Clear selections |

#### Package Operations

| Key       | Action                                     |
| --------- | ------------------------------------------ |
| `Tab`/`m` | Mark/unmark package for batch operation    |
| `Enter`   | Install/remove selected or marked packages |
| `*`       | Toggle selection panel focus               |

#### Update Mode

| Key     | Action                                         |
| ------- | ---------------------------------------------- |
| `Enter` | Proceed with a full system update              |
| `s`     | Switch to **Selective Update** mode            |

#### Dashboard (Dash Mode)

| Key | Action                                       |
| --- | -------------------------------------------- |
| `t` | Jump to Remove mode → All packages           |
| `e` | Jump to Remove mode → Explicit packages      |
| `f` | Jump to Remove mode → Foreign (AUR) packages |
| `o` | Jump to Remove mode → Orphan packages        |
| `c` | Open Cache Cleaning menu                     |
| `R` | Remove all orphan packages                   |

#### Confirmation Dialogs

| Key           | Action              |
| ------------- | ------------------- |
| `y` / `Enter` | Confirm operation   |
| `n` / `Esc`   | Cancel operation    |
| `↑` / `↓`     | Scroll package list |

### Search Filters

#### Install Mode

Prefix your search with repository filters:

| Prefix | Repository |
| ------ | ---------- |
| `c:`   | Core       |
| `e:`   | Extra      |
| `m:`   | Multilib   |
| `a:`   | AUR        |

Combine filters: `ae:firefox` searches AUR and Extra for "firefox"

#### Remove Mode

Filter installed packages by type:

| Prefix | Filter                 |
| ------ | ---------------------- |
| `t:`   | Total (all packages)   |
| `e:` / `l:`   | Explicitly installed (local) |
| `f:` / `a:`   | Foreign (AUR) packages |
| `o:`   | Orphan packages        |

Combined: `of:google` searches for orphaned AUR packages matching "google".

### Color Legend

| Color      | Source   |
| ---------- | -------- |
| 🟢 Green   | core     |
| 🔵 Blue    | extra    |
| 🟠 Orange  | multilib |
| 🟣 Magenta | AUR      |
| ⚪ Grey    | Other    |

### Themes

Gaur supports customizable color themes. Use the `--theme` flag to select a theme on startup, or press `,` to open the Settings menu and change the theme live.

```bash
gaur --theme catppuccin-mocha
```

To list available themes from the command line:

```bash
gaur --list-themes
```

## 🔧 How It Works

1. **Package Database** — Loads repository packages from local pacman cache on startup
2. **AUR Search** — Queries AUR via your configured helper (debounced)
3. **Fuzzy Matching** — Uses `fzf --filter` for fast, relevance-ranked fuzzy matching
4. **Interactive Operations** — Hands off to the AUR helper in the terminal for install/remove/update with full interactivity (password prompts, conflict resolution, etc.)
5. **Unified Refresh** — Re-scans the system after any change to update dashboard stats and lists instantly.

## 📄 License

GPLv3 License — See [LICENSE](LICENSE) for details.

---

<div align="center">

**[Report Bug](https://github.com/prbhtkumr/gaur/issues)** · **[Request Feature](https://github.com/prbhtkumr/gaur/issues)**

</div>
