---
title: "configuration"
description: "Customize gaur to fit your workflow"
---

# configuration

gaur works great out of the box, but you can tweak everything. Change settings on the fly or edit the config file directly.

## in-app settings

Press `,` anywhere in gaur to open the settings overlay. From there you can adjust:

- **AUR Helper:** Toggle between `paru` and `yay`
- **Theme:** Preview and apply color schemes instantly
- **Default View:** Choose what gaur opens to (`dashboard`, `install`, `update`, `remove`)
- **Border Style:** Pick your aesthetic (`rounded`, `normal`, `thick`, `double`)

Changes apply immediately. No restart needed.

## config file

On first run, gaur creates a config at:

`~/.config/config.toml`

The format is TOML. Simple and readable.

### aur helper & tools

Set your preferred AUR helper and cache cleaner:

```toml
[commands]
# "paru" (default) or "yay"
aur_helper = "paru"

# cache cleaning tool (paccache by default)
cache_tool = "paccache"
```

### package manager flags

Pass custom flags to install and remove operations:

```toml
[commands]
install_flags = ""
remove_flags = "-Rns"
```

### advanced options

Fine-tune performance and paths:

```toml
[advanced]
# debounce delay for package lookups (ms)
debounce_ms = 150

# custom cache directory (leave empty for default)
cache_dir = ""
```

## custom keybindings

Remap any key to fit your muscle memory:

```toml
[keys]
quit = ["q", "ctrl+c"]
install_mode = ["i", "alt+2"]
remove_mode = ["r", "alt+4"]
update_mode = ["u", "alt+3"]
dashboard_mode = ["d", "alt+1"]
search = "/"
mark = "tab"
selective = "s"
settings = ","
confirm = "enter"
cancel = "esc"
```

## themes

gaur bundles several popular themes. Switch via settings, config file, or command line.

### command line

Launch with a specific theme:

```bash
gaur --theme catppuccin-mocha
```

### list all themes

See what's available:

```bash
gaur --list-themes
```

Browse the [Theme Gallery](/themes) to preview each one.
