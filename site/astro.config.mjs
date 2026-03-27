import { defineConfig } from 'astro/config';

import tailwind from '@astrojs/tailwind';

// https://astro.build/config
export default defineConfig({
  site: 'https://gaur.prbhtkumr.xyz',
  base: '/', 
  output: 'static',

  integrations: [tailwind()],
});