---
title: "Installation"
description: "How to install gaur on Arch Linux"
---

# Installation

gaur is designed specifically for Arch Linux and its derivatives (e.g., Manjaro, EndeavourOS). You can install it using several methods.

## Prerequisites

Before installing gaur, ensure you have the following on your system:

- **Arch Linux** (or an Arch-based distribution)
- **AUR Helper:** [paru](https://github.com/Morganamilo/paru) or [yay](https://github.com/Jguer/yay)
- **Fuzzy Finder:** [fzf](https://github.com/junegunn/fzf) (required for searching)
- **Go 1.21+** (only if building from source)

## Methods

### From the AUR (Recommended)

gaur is available on the AUR. Use your favorite helper:

```bash
yay -S gaur-bin
# or
paru -S gaur-bin
```

### Using go install

If you have Go installed, you can install the latest release directly:

```bash
go install github.com/prbhtkumr/gaur@latest
```

Ensure your `$GOPATH/bin` is in your `$PATH`.

### From Source

To build gaur from the latest source code:

1. Clone the repository:
   ```bash
   git clone https://github.com/prbhtkumr/gaur.git
   ```
2. Navigate to the project directory:
   ```bash
   cd gaur
   ```
3. Build the binary:
   ```bash
   go build -o gaur .
   ```
4. Move the binary to your local path:
   ```bash
   sudo mv gaur /usr/local/bin/
   ```

## Verification

After installation, verify it's working by running:

```bash
gaur --version
```

If everything is correct, you are ready to [configure](/docs/configuration) your setup.
