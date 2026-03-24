---
title: "Configuration"
description: "How to customize gaur to your liking"
---

# Configuration

gaur is highly customizable. You can configure it either via the in-app settings menu or by editing its configuration file.

## In-App Settings

gaur features a convenient **Settings Overlay** that you can access at any time by pressing `,`.

From the overlay, you can instantly configure:
- **AUR Helper:** Switch between `paru` and `yay`.
- **Theme:** Instantly preview and apply different color themes.
- **Default View:** Set the view gaur opens with (`dashboard`, `install`, `update`, `remove`).
- **Border Type:** Change the UI borders (`rounded`, `normal`, `thick`, `double`).

## Config Options

On its first run, gaur will automatically generate a default configuration file located at:

`~/.config/gaur/config.toml`

The configuration file is written in TOML and uses a simple, intuitive structure.

### AUR Helper & Tools

You can set your preferred AUR helper and cache management tool.

```toml
[commands]
# Set your preferred AUR helper: "paru" (default) or "yay"
aur_helper = "paru"

# The tool used for cleaning the cache (defaults to paccache)
cache_tool = "paccache"
```

### Command Flags

Customize the flags passed to your package manager for installation and removal.

```toml
[commands]
# Custom flags for install/remove
install_flags = ""
remove_flags = "-Rns"
```

### Advanced Settings

gaur also supports advanced settings like custom cache directories and debouncing limits.

```toml
[advanced]
# Milliseconds to debounce package details fetching
debounce_ms = 150

# Set a custom cache directory (optional)
cache_dir = ""
```

## Custom Keybindings

You can remap keybindings to fit your workflow in the configuration file:

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

## Themes

gaur supports several built-in themes to match your terminal rice. You can change them through the settings menu, config file, or command line flag.

### Command Line Selection

Use the `--theme` flag to select a theme on startup:

```bash
gaur --theme catppuccin-mocha
```

### Listing Themes

To see all available themes:

```bash
gaur --list-themes
```

Check out our [Theme Gallery](/gaur/themes) to see what each theme looks like in action.
