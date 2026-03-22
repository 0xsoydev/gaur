---
title: "Configuration"
description: "How to customize Gaur to your liking"
---

# Configuration

Gaur is highly customizable via its configuration file. On its first run, it will automatically generate a default configuration file located at:

`~/.config/gaur/config.toml`

## Config Options

The configuration file is written in TOML and uses a simple, intuitive structure.

### AUR Helper

You can set your preferred AUR helper (e.g., `paru` or `yay`).

```toml
[commands]
# Set your preferred AUR helper: \"paru\" (default) or \"yay\"
aur_helper = \"paru\"
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

Gaur also supports advanced settings like custom cache directories.

```toml
[advanced]
# Set a custom cache directory (optional)
cache_dir = \"\"
```

## Themes

Gaur supports several built-in themes to match your terminal rice.

### Selecting a Theme

Use the `--theme` flag to select a theme on startup:

```bash
gaur --theme catppuccin-mocha
```

### Listing Themes

To see all available themes:

```bash
gaur --list-themes
```

Check out our [Showcase](/showcase) to see what each theme looks like in action.
