import theme from '../theme/tailwind.config.js';

// Reuse the DESIGN.md token mapping; only the content globs differ here.
export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: theme.theme,
  plugins: [],
};
