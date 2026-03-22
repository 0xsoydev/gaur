/** @type {import('tailwindcss').Config} */
export default {
	content: ['./src/**/*.{astro,html,js,jsx,md,mdx,svelte,ts,tsx,vue}'],
	theme: {
		extend: {
			colors: {
				primary: '#cba6f7', // Mauve
				background: '#1e1e2e', // Base
				surface: '#181825', // Mantle
				crust: '#11111b',
				text: '#cdd6f4', // Text
				muted: '#a6adc8', // Subtext0
				accent: '#a6e3a1', // Green
				warning: '#fab387', // Peach
				highlight: '#74c7ec', // Sapphire
				pink: '#f5c2e7',
				blue: '#89b4fa',
				yellow: '#f9e2af',
			},
			fontFamily: {
				mono: ['"JetBrains Mono"', 'monospace'],
			},
			borderRadius: {
				gaur: '8px',
			},
			boxShadow: {
				glow: '0 0 15px rgba(116, 199, 236, 0.2)',
			}
		},
	},
	plugins: [],
};
