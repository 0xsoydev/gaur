---
title: "installation"
description: "Get gaur running on your Arch system"
---

# installation

gaur runs on Arch Linux and its derivatives: Manjaro, EndeavourOS, CachyOS, you name it.  
Here's how to get it.

## what you'll need

Make sure these are on your system before installing:

- **Arch Linux** (or any Arch-based distro)
- **An AUR helper:** [paru](https://github.com/Morganamilo/paru) or [yay](https://github.com/Jguer/yay)
- **fzf:** [fzf](https://github.com/junegunn/fzf) powers the fuzzy search
- **Go 1.21+** (only needed if you're building from source)

## install methods

### AUR (recommended)

The fastest route. Pick your helper:

```bash
yay -S gaur-bin
# or
paru -S gaur-bin
```

### go install

Already have Go set up? Grab the latest release directly:

```bash
go install github.com/prbhtkumr/gaur@latest
```

Make sure `$GOPATH/bin` is in your `$PATH`.

### from source

Want to run the bleeding edge?

1. Clone the repo:
   ```bash
   git clone https://github.com/prbhtkumr/gaur.git
   ```
2. Enter the directory:
   ```bash
   cd gaur
   ```
3. Build it:
   ```bash
   go build -o gaur .
   ```
4. Move the binary somewhere in your path:
   ```bash
   sudo mv gaur /usr/local/bin/
   ```

## verify

Check that everything's working:

```bash
gaur --version
```

If you see a version number, you're all set. Head over to [configuration](/docs/configuration) to customize your setup.
