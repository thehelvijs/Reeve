import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

// UI version comes from package.json (npm sets npm_package_version on scripts).
const uiVersion = process.env.npm_package_version ?? '0.0.1-alpha';

// UI builds into dist/ and is embedded into the server binary (Milestone 9).
// In dev, /api is proxied to the Go server so cookies stay same-origin.
// REEVE_DEV_PORT lets e2e point at a throwaway server without colliding with one already on 8080.
const devPort = process.env.REEVE_DEV_PORT ?? '8080';

export default defineConfig({
  plugins: [react()],
  define: { __UI_VERSION__: JSON.stringify(uiVersion) },
  build: { outDir: 'dist' },
  server: {
    proxy: {
      '/api': `http://127.0.0.1:${devPort}`,
      '/healthz': `http://127.0.0.1:${devPort}`,
      // Short links are resolved and redirected by the server, so the dev
      // server must hand them over rather than answering with the SPA shell.
      '/go': `http://127.0.0.1:${devPort}`,
    },
  },
});
