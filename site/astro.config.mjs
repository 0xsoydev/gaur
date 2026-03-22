import { defineConfig } from 'astro/config';

import tailwind from '@astrojs/tailwind';

// https://astro.build/config
export default defineConfig({
  // You might need to set your base to the repo name if not a custom domain
  // base: '/gaur', 
  output: 'static',

  integrations: [tailwind()],
});