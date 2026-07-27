# DESIGN.md - UI style guide

All UI in this project follows the design language below. The model is GitLab's
Pajamas: a light canvas, dense tables, one accent kept for the brand, and
structure that names itself. An operator reads Reeve next to a terminal all day.
The numbers must stay legible. Nothing here is atmosphere.

> Ready-made tokens: `theme/theme.css` (CSS variables, the source of truth) and
> `theme/tailwind.config.js`. When you build a web UI, wire these into the app.
> Import `theme.css`, then use the Tailwind config or a v4 `@theme` block.

## Principles

- **Light, and only light.** White content, a faint grey chrome, near-black text.
  There is no dark theme and no theme toggle. One theme means one thing to keep
  correct.
- **Two colors do two jobs.** Blue (`--link`) carries every interactive state:
  links, focus rings, bars, chart lines. Acid-lime (`--accent`) is the brand. It
  appears on the wordmark and on the primary button of a form, and never as text.
- **Every column has a name.** A list of records is a table with a header row. A
  bar, a pill, or a button in a row must be readable without a guess about what
  it measures.
- **Say what the state means.** A count carries what it counts out of and a line
  that explains the state. `unknown` is a name from the server. The UI says "not
  monitored".
- **Structure over shadow.** Hierarchy comes from hairline borders, a rule under
  each page title, and a small surface ladder. A shadow belongs only on something
  that floats over the page.
- **Quiet and dense.** 14px body text, compact rows, little ornament. Reduce the
  work the reader does.

## Color tokens

Surfaces:
- `--canvas`       `#ffffff`  content background, cards, inputs
- `--surface-1`    `#fbfafd`  sidebar, table header, row hover
- `--surface-2`    `#f0f0f2`  pressed and selected fills, code blocks
- `--surface-3`    `#ffffff`  popovers, with a border and `--shadow-pop`

Borders:
- `--border`       `#dcdcde`  default separation
- `--border-strong``#bfbfc3`  input and button edges

Text:
- `--text`         `#1f1e24`  primary
- `--text-muted`   `#626168`  secondary and captions, 5.7:1 on white

Brand:
- `--accent`       `#e4f222`  acid-lime, always a fill
- `--accent-hover` `#d5e300`
- `--on-accent`    `#1f1e24`  text on top of the accent

Interaction:
- `--link`         `#1068bf`  links, focus rings, neutral bars and chart lines
- `--link-hover`   `#0b5cad`
- `--focus`        `#1068bf`

Status. Each state has four steps: a text-safe tone, a solid fill for a dot or a
bar, and a soft/line pair for a badge.
- up:   `#24663b` / `#108548` / `#ecf4ee` / `#c3e6cd`
- down: `#ae1800` / `#dd2b0e` / `#fcf1ef` / `#fdd4cd`
- warn: `#8f4700` / `#ab6100` / `#fdf1dd` / `#f5d9a8`
- idle: `#626168` / `#a4a3a8` / `#f0f0f2` / `#dcdcde`

Never put the accent on text, a border, a chart line, or a status. It measures
1.2:1 against white, so it disappears. Use `--link` or a status tone instead.

## Typography

- **Family:** Inter (Inter Variable), with `zero` for a slashed zero. Monospace
  for code and endpoints: Berkeley Mono, falling back to `ui-monospace`.
- **Weights:** body 400, emphasis 500-600, headings 600. No heavy bold as
  ornament.
- **Body:** 14px / line-height 1.5. Captions and table cells drop to 12px.
- **Page title:** 20px, weight 600, normal tracking. Large type does not get
  tighter here.
- **Eyebrow:** 11px, uppercase, weight 600, `letter-spacing: 0.05em`, muted.
  Table headers and section labels use it.
- **Numbers in a column:** `tabular-nums`, so digits line up down a table.

## Spacing & radius

- **Spacing:** 8px base scale (4, 8, 12, 16, 24, 32, 48). Table cells stay tight
  at 8-12px. Cards take 16px.
- **Radius vocabulary (the entire set):**
  - button / input / badge: `4px`
  - card / panel / table: `8px`
  - large tile: `12px`
  - pill / avatar: `9999px`

## Elevation & motion

- Elevation is a border plus a surface step. `--shadow-pop` is for a modal, a
  dropdown, or a map tooltip, and nothing else.
- Motion is fast: 120-200ms, ease-out, no bounce. Animate opacity and small
  transforms. Do not animate layout or color for decoration.

## Components (defaults)

- **Primary button:** accent fill, `--on-accent` label, weight 600, radius 4px.
  One per form or settings card, so a page of independent cards carries one each.
  A view that is a single form carries exactly one.
- **Secondary button:** white fill, `--border-strong` edge, dark label, grey
  hover.
- **Danger button:** white fill, `--down-line` edge, `--down` label, red hover.
- **Card:** `--canvas`, `--border` hairline, radius 8px.
- **Table:** header row on `--surface-1` in eyebrow type, hairline row dividers,
  `--surface-1` row hover. The first cell links the record by name in `--link`.
  Actions go in a right-aligned last column.
- **Tabs:** a filter over the list below. The active tab takes a 2px `--link`
  underline and weight 600. Each tab carries its count.
- **Pill:** a status badge in the soft/line/text steps of its tone.
- **Input:** `--canvas` fill, `--border-strong` edge, blue focus ring.
- **Focus state:** always visible, always the blue ring. Never removed.
- **Inline link:** blue and underlined. color alone does not mark a link inside
  a sentence.
- **Charts:** series colors come from the fixed palette in `HostMetrics.tsx`,
  assigned in order and never by rank. The order alternates dark and light steps,
  which is what keeps neighbouring series apart for a colorblind reader. The
  palette passes a lightness band, a chroma floor, deutan and tritan separation,
  and 3:1 contrast on white. Re-validate it before you change it.

## Writing

- Name a state the way an operator reads it, not the way the server stores it.
  "unreachable" is not "down": the service may still run on a host that stopped
  reporting.
- One name per thing, everywhere. A pill, a tab, and a tile that describe the
  same state use the same word. `lib/statusTone.ts` owns that mapping.
- A button says what happens. "Add host", then a host appears.
- An empty state points at the next action. An error says what broke and what to
  do.

## Keyboard

- **Enter confirms.** Every form, section, and modal with editable fields submits
  its one primary action on Enter. This includes a card with a lone input and a
  Save button.
- Mechanically: wrap the fields in `<Form onSubmit={…}>` from `components/ui.tsx`
  and give the primary button `type="submit"`. `Button` defaults to
  `type="button"`, so a secondary action in the same form stays inert. A modal
  without a form takes `onSubmit` on `Modal`. An info-only modal passes `onClose`
  there, so Enter acknowledges it.
- A nested control that owns a smaller action handles Enter itself and stops it,
  rather than submitting the outer form.
- **Escape closes** any modal.

## Do not

- No dark theme, no theme toggle, no `prefers-color-scheme` branch.
- No accent as text, border, chart line, or status color.
- No second brand color, no gradient, no glassmorphism.
- No drop shadow that carries hierarchy.
- No unlabeled list rows.
- No raw server vocabulary in the UI.
- No more than the four defined radii.
