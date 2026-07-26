# DESIGN.md — UI style guide (Linear-inspired)

All UI in this project follows the design language below. It is inspired by
Linear's publicly observable aesthetic (not affiliated with or endorsed by
Linear). The whole system is built on restraint: dark canvas, one accent,
hairline structure, tight type. When in doubt, remove rather than add.

> Ready-made tokens: `theme/theme.css` (CSS variables — the source of truth) and
> `theme/tailwind.config.js`. When building a web UI, wire these into the app
> (import `theme.css`; use the Tailwind config or a v4 `@theme` block).

## Principles

- **Dark-first.** Dark is the default; light mode is a mapped equivalent, not
  the primary. Never use pure black (`#000000`) — use a near-black with a faint
  cool cast.
- **One accent, used sparingly.** A single chromatic accent for the brand mark,
  focus rings, and *one* primary action per view. No second accent color, no
  decorative use, no gradients, no spotlight cards.
- **Structure over shadow.** Hierarchy comes from a small ladder of surface
  shades and hairline borders — not drop shadows.
- **Quiet, dense, calm.** Minimal ornament; let content and product UI be the
  only texture. Reduce cognitive load; offer few choices per view.
- **Precision.** A tiny radius and spacing vocabulary, applied consistently.

## Color tokens

Surfaces (dark, default):
- `--canvas`       `#08090a`  page background (near-black, faint blue cast)
- `--surface-1`    `#0f1011`  cards, panels
- `--surface-2`    `#141516`  hovered / raised
- `--surface-3`    `#18191a`  dropdowns, popovers
Borders (hairline, 0.5–1px):
- `--border`       `#23252a`  default separation
- `--border-strong``#34343a`  emphasized edges / focus outlines on surfaces
Text:
- `--text`         `#f7f8f8`  primary
- `--text-muted`   `#8a8f98`  secondary / captions
Accent:
- `--accent`       `#e4f222`  acid-lime — brand, focus ring, primary CTA
- `--accent-hover` `#eef658`
- `--on-accent`    `#08090a`  text/icon color on top of the accent (dark, for contrast)

Use the accent for a single primary action per view. Everything else is
monochrome (surface + text + border).

## Typography

- **Family:** Inter (Inter Variable), with OpenType features **`cv01`** (single-
  story a) and **`ss03`** on, plus `zero` (slashed zero). These alternates are
  core to the look. Monospace for code: Berkeley Mono, falling back to
  `ui-monospace, monospace`.
- **Weights:** restrained. Body 400; emphasis 510–590; display 500–700. Avoid
  heavy bold as decoration.
- **Body:** 16px / line-height 1.5.
- **Display:** tight negative tracking — `letter-spacing: -0.022em` at ~48px and
  above. Larger type gets tighter, never looser.

## Spacing & radius

- **Spacing:** 8px base scale (4, 8, 12, 16, 24, 32, 48). Keep paddings compact
  in dense UI (8–12px); use larger outer padding (24px) around feature tiles.
- **Radius vocabulary (the entire set):**
  - button / input: `6px`
  - card / panel: `12px` (large tiles up to `16px`)
  - pill / avatar: `9999px`

## Elevation & motion

- Elevation = surface step up + hairline border (optionally a faint 1px top-edge
  highlight). Avoid large/soft drop shadows on dark.
- Motion is fast and purposeful: 120–200ms, ease-out, no bounce. Animate opacity
  and small transforms; don't animate layout or color decoratively.

## Components (defaults)

- **Primary button:** accent background, `--on-accent` (dark) label, radius 6px,
  compact padding (~8px 12px). One per view.
- **Secondary button:** transparent with `--border`, text `--text-muted` →
  `--text` on hover. No accent.
- **Card:** `--surface-1`, `--border` hairline, radius 12px.
- **Input:** `--surface-1`, `--border`; focus shows an accent ring (no glow).
- **Focus state:** always visible, always the accent ring — never removed.

## Keyboard

- **Enter confirms.** Every form, section, and modal with editable fields
  submits its one primary action on Enter — Save, Create, Continue, OK,
  whatever the main button says. No exceptions, including a card with a lone
  input and a Save button.
- Mechanically: wrap the fields in `<Form onSubmit={…}>` from
  `components/ui.tsx` and give the primary button `type="submit"`. `Button`
  defaults to `type="button"`, so secondary actions in the same form stay
  inert. A modal without a form takes `onSubmit` on `Modal`; an info-only
  modal passes `onClose` there, so Enter acknowledges it.
- A nested control that owns a smaller action (add-a-collection inside the
  service form, a city search inside the location picker) handles Enter itself
  and stops it, rather than submitting the outer form.
- **Escape closes** any modal.

## Don't

- No pure black, no second accent, no gradients, no glassmorphism.
- No drop shadows carrying hierarchy.
- No accent on secondary/decorative elements.
- No loose letter-spacing on headings; no heavy bold as ornament.
- No more than the three defined radii.
