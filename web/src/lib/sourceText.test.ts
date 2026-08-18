/// <reference types="vite/client" />
import { describe, expect, it } from 'vitest';

// A backslash-u escape is only an escape inside a JS string literal. In JSX text
// it is four characters that ship to the page, so the reader gets
// "Group → Settings" where an arrow was meant. A patch script wrote a whole
// help panel that way once, and neither the typecheck nor the linter can see it:
// it is valid text either way.
//
// The sources are read through Vite's own glob rather than node:fs, so this
// needs no node types in a project that has none.
const sources = import.meta.glob('../**/*.{ts,tsx}', { query: '?raw', import: 'default', eager: true });

const ESCAPE = /\\u[0-9a-fA-F]{4}/;

describe('source text', () => {
  it('carries no literal unicode escapes', () => {
    const offenders: string[] = [];
    for (const [path, body] of Object.entries(sources)) {
      // This file names the pattern on purpose.
      if (path.endsWith('sourceText.test.ts')) {
        continue;
      }
      (body as string).split('\n').forEach((line, i) => {
        if (ESCAPE.test(line)) {
          offenders.push(`${path}:${i + 1}`);
        }
      });
    }
    expect(offenders).toEqual([]);
  });

  // The glob is the whole point: an empty match would pass forever.
  it('reads the source tree', () => {
    expect(Object.keys(sources).length).toBeGreaterThan(20);
  });
});
