/** Linear-inspired Tailwind theme — generated from DESIGN.md.
 *  Pairs with theme.css; the CSS variables there are the source of truth,
 *  so light/dark switching works automatically via [data-theme].
 *
 *  This is Tailwind v3 (tailwind.config.js). On Tailwind v4, skip this file
 *  and put an `@theme` block in your CSS referencing the same variables.
 */
module.exports = {
  content: ['./index.html', './src/**/*.{html,js,ts,jsx,tsx,vue,svelte}'],
  theme: {
    extend: {
      colors: {
        canvas: 'var(--canvas)',
        surface: {
          1: 'var(--surface-1)',
          2: 'var(--surface-2)',
          3: 'var(--surface-3)',
        },
        hairline: 'var(--border)',
        'hairline-strong': 'var(--border-strong)',
        content: 'var(--text)',
        muted: 'var(--text-muted)',
        accent: {
          DEFAULT: 'var(--accent)',
          hover: 'var(--accent-hover)',
          fg: 'var(--on-accent)',
        },
      },
      borderRadius: {
        button: 'var(--radius-button)',
        card: 'var(--radius-card)',
        tile: 'var(--radius-tile)',
        pill: 'var(--radius-pill)',
      },
      fontFamily: {
        sans: ['Inter Variable', 'Inter', 'system-ui', 'sans-serif'],
        mono: ['Berkeley Mono', 'ui-monospace', 'monospace'],
      },
      letterSpacing: {
        display: '-0.022em',
      },
      transitionTimingFunction: {
        linear2: 'cubic-bezier(0.16, 1, 0.3, 1)',
      },
    },
  },
  plugins: [],
};
