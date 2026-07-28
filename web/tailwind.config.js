import theme from '../theme/tailwind.config.js';

// Reuse the DESIGN.md token mapping; only the content globs differ here.
export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  // Without this, `dark:` falls back to Tailwind's prefers-color-scheme default
  // and reads the operating system instead of the theme the app is showing.
  darkMode: theme.darkMode,
  theme: theme.theme,
  plugins: [],
};
