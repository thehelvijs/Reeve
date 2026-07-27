import { useSyncExternalStore } from 'react';

export type Theme = 'dark' | 'light';

// Kept in step with web/public/theme-boot.js, which applies the same key and the
// same default before the bundle loads. theme.test.ts asserts they agree.
export const THEME_KEY = 'reeve-theme';
export const DEFAULT_THEME: Theme = 'dark';

// Anything unrecognized resolves to the default rather than to the operating
// system's preference: dark is the product's choice, not a guess about the desk.
export function readTheme(raw: string | null): Theme {
  if (raw === 'light' || raw === 'dark') {
    return raw;
  }
  return DEFAULT_THEME;
}

export function otherTheme(theme: Theme): Theme {
  if (theme === 'dark') {
    return 'light';
  }
  return 'dark';
}

// The canvas the browser should paint behind the page, for the theme-color meta.
export function canvasFor(theme: Theme): string {
  if (theme === 'light') {
    return '#ffffff';
  }
  return '#1f1e24';
}

const listeners = new Set<() => void>();

// The DOM attribute is the single source of truth at runtime: the boot script
// sets it, this reads it, and nothing keeps a second copy that could drift.
function snapshot(): Theme {
  if (document.documentElement.dataset.theme === 'light') {
    return 'light';
  }
  return 'dark';
}

function subscribe(fn: () => void): () => void {
  listeners.add(fn);
  return () => {
    listeners.delete(fn);
  };
}

export function setTheme(theme: Theme): void {
  document.documentElement.dataset.theme = theme;
  const meta = document.querySelector('meta[name="theme-color"]');
  if (meta) {
    meta.setAttribute('content', canvasFor(theme));
  }
  try {
    window.localStorage.setItem(THEME_KEY, theme);
  } catch {
    // A blocked store only costs persistence; the page still switches.
  }
  for (const fn of listeners) {
    fn();
  }
}

// Components that hand colors to a canvas library (uPlot, Leaflet, React Flow)
// read this so they redraw when the theme changes; everything else uses tokens.
export function useTheme(): Theme {
  return useSyncExternalStore(subscribe, snapshot, () => DEFAULT_THEME);
}

// Resolves a token to the value the active theme gives it. Only for a library
// that needs a color string rather than a CSS variable.
export function cssVar(name: string): string {
  return getComputedStyle(document.documentElement).getPropertyValue(name).trim();
}
