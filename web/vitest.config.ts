import { defineConfig } from 'vitest/config';

// Gate lane for the frontend: pure functions only, no browser and no jsdom, so
// it stays in the sub-second budget the pre-commit hook needs. Anything that
// needs a rendered page belongs in e2e/, which Playwright owns — vitest must
// not pick those specs up, hence the explicit include.
// Kept separate from vite.config.ts so `vite build` never has to resolve vitest.
export default defineConfig({
  test: {
    include: ['src/**/*.test.ts'],
    environment: 'node',
  },
});
