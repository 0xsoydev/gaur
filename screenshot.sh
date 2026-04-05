#!/bin/bash

# Path to main config and themes directory
MAIN_CONFIG="$HOME/.config/foot/foot.ini"
THEME_DIR="$HOME/.config/foot/themes"
TEMP_CONFIG="/tmp/gaur_theme_preview.ini"

# Map gaur themes to Foot files (Light themes removed)
declare -A THEMES=(
	["Catppuccin Frappe"]="catppuccin-frappe"
	["Catppuccin Macchiato"]="catppuccin-macchiato"
	["Catppuccin Mocha"]="catppuccin-mocha"
	["Dracula"]="dracula"
	["Gruvbox Dark"]="gruvbox-dark"
	["Monokai Pro"]="monokai-pro"
	["One Dark"]="onedark"
	["Rose Pine"]="rose-pine"
	["Solarized Dark"]="solarized-dark"
	["Tokyonight Night"]="tokyonight-night"
	["Tokyonight Storm"]="tokyonight-storm"
)

for gaur_theme in "${!THEMES[@]}"; do
	foot_theme="${THEMES[$gaur_theme]}"

	echo "Preparing: $gaur_theme (using $foot_theme)..."

	# Load Main Config
	echo "include=$MAIN_CONFIG" >"$TEMP_CONFIG"

	# Load Theme Config
	echo "include=$THEME_DIR/$foot_theme" >>"$TEMP_CONFIG"

	# Force Window Size in Config
	echo "" >>"$TEMP_CONFIG"
	echo "[main]" >>"$TEMP_CONFIG"
	echo "initial-window-size-pixels=910x630" >>"$TEMP_CONFIG"

	# Disable Client-Side Decorations (Removes borders and titlebar)
	echo "" >>"$TEMP_CONFIG"
	echo "[csd]" >>"$TEMP_CONFIG"
	echo "preferred=none" >>"$TEMP_CONFIG"
	echo "border-width=0" >>"$TEMP_CONFIG"

	while true; do
		# We run sh -c inside foot.
		# If user types n/no, we exit 1. If enter, exit 0.
		foot -c "$TEMP_CONFIG" \
			--title "Screenshot: $gaur_theme" \
			sh -c "
                ./gaur --theme '$gaur_theme'; 
                echo -ne '\n📸 Done? (Enter=Next, n=Retry): '; 
                read -r ans; 
                case \"\$ans\" in 
                    n|N|no|No) exit 1 ;; 
                    *) exit 0 ;; 
                esac"

		# Check the exit code of the foot command above
		if [ $? -eq 0 ]; then
			break # User hit Enter (Success), break retry loop, go to next theme
		else
			echo "♻️ Retrying $gaur_theme..."
			# The loop repeats, relaunching the window
		fi
	done
done

echo "All themes previewed!"
