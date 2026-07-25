declare const __UI_VERSION__: string;

// UI version, injected at build from package.json (see vite.config.ts). Falls
// back to the default when the define is absent (e.g. bare dev server).
export const UI_VERSION = typeof __UI_VERSION__ !== 'undefined' ? __UI_VERSION__ : '0.0.1-alpha';

// Public source location. AGPL section 13 requires that anyone using this over a
// network be offered the corresponding source, so both shells link to it.
export const SOURCE_URL = 'https://github.com/thehelvijs/Reeve';
