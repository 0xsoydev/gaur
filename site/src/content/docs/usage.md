---
title: "usage"
description: "Master the keyboard shortcuts and workflows"
---

# usage

gaur is built for speed. Everything happens through the keyboard, no mouse required. Learn these shortcuts and you'll fly through package management.

## mode switching

Jump between views instantly:

| Key      | Action                      |
| -------- | --------------------------- |
| `i`      | Install mode                |
| `r`      | Remove mode                 |
| `u`      | Update mode                 |
| `d`      | Dashboard mode              |
| `,`      | Open settings               |
| `q`      | Quit                        |
| `Ctrl+C` | Force quit                  |

## getting around

Navigate lists and search:

| Key       | Action                     |
| --------- | -------------------------- |
| `/`       | Focus search box           |
| `↑` / `k` | Move up                    |
| `↓` / `j` | Move down                  |
| `PgUp`    | Jump 10 items up           |
| `PgDown`  | Jump 10 items down         |
| `Esc`     | Clear focus / deselect all |

## working with packages

Install, remove, or mark packages for batch operations:

| Key       | Action                           |
| --------- | -------------------------------- |
| `Tab`/`m` | Mark/unmark for batch processing |
| `Enter`   | Execute on selected/marked       |
| `*`       | Toggle selection panel           |

## updating

When updates are available, you have two options:

| Key     | Action                        |
| ------- | ----------------------------- |
| `Enter` | Full system upgrade           |
| `s`     | Switch to selective mode      |

**Heads up:** Selective updates can break dependencies. Proceed with caution.

## dashboard shortcuts

The dashboard gives you quick access to different package categories:

| Key | Action                         |
| --- | ------------------------------ |
| `t` | Remove mode → all packages     |
| `e` | Remove mode → explicit only    |
| `f` | Remove mode → AUR/foreign      |
| `o` | Remove mode → orphans          |
| `c` | Cache cleaning menu            |
| `R` | Nuke all orphan packages       |

## confirmation prompts

When gaur asks "are you sure?":

| Key           | Action            |
| ------------- | ----------------- |
| `y` / `Enter` | Confirm           |
| `n` / `Esc`   | Cancel            |
| `↑` / `↓`     | Scroll the list   |

## search filters

Narrow down results with prefixes.

### install mode

Filter by repository:

| Prefix | Repo     |
| ------ | -------- |
| `c:`   | Core     |
| `e:`   | Extra    |
| `m:`   | Multilib |
| `a:`   | AUR      |

**Example:** `ae:firefox` searches both official repos and the AUR for "firefox".

### remove mode

Filter installed packages by category:

| Prefix      | Shows                       |
| ----------- | --------------------------- |
| `t:`        | All installed packages      |
| `e:` / `l:` | Explicitly installed        |
| `f:` / `a:` | AUR/foreign packages        |
| `o:`        | Orphans                     |

**Example:** `of:google` finds orphaned AUR packages matching "google".
