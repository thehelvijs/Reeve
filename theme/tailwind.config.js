/** Reeve Tailwind theme — generated from DESIGN.md.
 *  Pairs with theme.css; the CSS variables there are the source of truth.
 *
 *  This is Tailwind v3 (tailwind.config.js). On Tailwind v4, skip this file
 *  and put an `@theme` block in your CSS referencing the same variables.
 */
module.exports = {
  content: ['./index.html', './src/**/*.{html,js,ts,jsx,tsx,vue,svelte}'],
  // Tokens cover almost everything, so `dark:` is for the few properties a
  // variable cannot carry — a filter, a blend mode. The selector mirrors the
  // variable map: dark is anything not explicitly marked light, so it holds
  // before the boot script runs and with JavaScript off.
  darkMode: ['selector', ':root:not([data-theme="light"])'],
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
        link: {
          DEFAULT: 'var(--link)',
          hover: 'var(--link-hover)',
        },
        // A lone measure: lime on dark, blue on light.
        data: 'var(--data-primary)',
        // Status tones: bare name for text, -solid for a dot or bar fill,
        // -soft/-line for a badge.
        up: {
          DEFAULT: 'var(--up)',
          solid: 'var(--up-solid)',
          soft: 'var(--up-soft)',
          line: 'var(--up-line)',
        },
        down: {
          DEFAULT: 'var(--down)',
          solid: 'var(--down-solid)',
          soft: 'var(--down-soft)',
          line: 'var(--down-line)',
        },
        warn: {
          DEFAULT: 'var(--warn)',
          solid: 'var(--warn-solid)',
          soft: 'var(--warn-soft)',
          line: 'var(--warn-line)',
        },
        idle: {
          DEFAULT: 'var(--idle)',
          solid: 'var(--idle-solid)',
          soft: 'var(--idle-soft)',
          line: 'var(--idle-line)',
        },
      },
      outlineColor: {
        image: 'var(--image-outline)',
      },
      borderRadius: {
        button: 'var(--radius-button)',
        card: 'var(--radius-card)',
        tile: 'var(--radius-tile)',
        pill: 'var(--radius-pill)',
      },
      boxShadow: {
        pop: 'var(--shadow-pop)',
      },
      fontFamily: {
        sans: ['Inter Variable', 'Inter', 'system-ui', 'sans-serif'],
        mono: ['Berkeley Mono', 'ui-monospace', 'monospace'],
      },
      fontSize: {
        // The table/eyebrow label size, small caps by convention.
        eyebrow: ['11px', { lineHeight: '16px', letterSpacing: '0.05em' }],
      },
      // The DEFAULT keys are what every bare `transition-*` utility emits, so the
      // motion tokens apply app-wide without a duration or easing class per site.
      transitionTimingFunction: {
        DEFAULT: 'var(--ease)',
      },
      transitionDuration: {
        DEFAULT: 'var(--dur)',
      },
    },
  },
  plugins: [],
};
