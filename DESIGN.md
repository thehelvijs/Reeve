# DESIGN.md - UI style guide

All UI in this project follows the design language below. The model is GitLab's
Pajamas: dense tables, one accent kept for the brand, and structure that names
itself. An operator reads Reeve next to a terminal all day. The numbers must stay
legible. Nothing here is atmosphere.

> Ready-made tokens: `theme/theme.css` (CSS variables, the source of truth) and
> `theme/tailwind.config.js`. When you build a web UI, wire these into the app.
> Import `theme.css`, then use the Tailwind config or a v4 `@theme` block.

## Principles

- **Two themes, dark by default.** Both are first-class and both are gated: the
  axe sweep and the pixel sweep run every view twice. A token without a value in
  one map is invisible in exactly one theme, so nothing hardcodes a color that a
  theme should own.
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

Dark is `:root`, so it holds with no attribute and with JavaScript off. Light is
`[data-theme='light']`. `web/public/theme-boot.js` applies the stored choice
before first paint; `src/lib/theme.ts` owns the key, the default and the toggle.

| token | dark | light | use |
| --- | --- | --- | --- |
| `--canvas` | `#1f1e24` | `#ffffff` | content background, cards, inputs |
| `--surface-1` | `#28272d` | `#fbfafd` | sidebar, table header, row hover |
| `--surface-2` | `#333238` | `#f0f0f2` | pressed fills, code blocks |
| `--surface-3` | `#28272d` | `#ffffff` | popovers, with a border and `--shadow-pop` |
| `--border` | `#434248` | `#dcdcde` | default separation |
| `--border-strong` | `#535158` | `#bfbfc3` | input and button edges |
| `--text` | `#ececef` | `#1f1e24` | primary |
| `--text-muted` | `#bfbfc3` | `#626168` | secondary and captions |

Dark is a gray-950 canvas rather than a near-black one, and its borders are
visible rather than implied. The inky version that came before was hard to read.

Brand, the same in both themes because it is only ever a fill:
- `--accent` `#e4f222` acid-lime, `--on-accent` `#1f1e24` for text on top of it.

Interaction. `--link` carries links, focus rings, neutral bars and chart lines:
- dark `#63a6e9`, light `#1068bf`.

Status. Each state has four steps in each theme: a text-safe tone, a solid fill
for a dot or a bar, and a soft/line pair for a badge. Read the values from
`theme/theme.css`; the shape is `--up`, `--up-solid`, `--up-soft`, `--up-line`,
and the same for `down`, `warn` and `idle`.

Never put the accent on text, a border, a chart line, or a status. It clears no
text-contrast bar on either canvas, so it disappears. Use `--link` or a status
tone instead.

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

- **Page header:** one component for every page, list or detail. Title at 20px,
  the record's icon and current state beside it, an optional subtitle, one
  right-aligned action, closed by a hairline rule. A detail page adds a back link
  above it. No page grows its own heading.
- **Section:** everything below the page header is one. A 14px heading, the count
  of what is under it, a line saying what it is for, one control on the right,
  then the card, table or form. Sections in a view stack on one gap. A section
  brings no margin of its own.
- **Facts:** a definition list for what a record *is*: eyebrow term, 14px value.
  What it is *doing* goes in bars or a chart, not here.
- **Primary button:** accent fill, `--on-accent` label, weight 600, radius 4px.
  One per form or settings card, so a page of independent cards carries one each.
  A view that is a single form carries exactly one.
- **Secondary button:** `--canvas` fill, `--border-strong` edge, `--text` label,
  `--surface-2` hover.
- **Danger button:** `--canvas` fill, `--down-line` edge, `--down` label,
  `--down-soft` hover.
- **Card:** `--canvas`, `--border` hairline, radius 8px.
- **Table:** header row on `--surface-1` in eyebrow type, hairline row dividers,
  `--surface-1` row hover. The first cell links the record by name in `--link`.
  Actions go in a right-aligned last column.
- **Tabs:** a filter over the list below. The active tab takes a 2px `--link`
  underline and weight 600. Each tab carries its count.
- **Pill:** a status badge in the soft/line/text steps of its tone.
- **Input:** `--canvas` fill, `--border-strong` edge, blue focus ring.
- **Focus state:** always visible, always the blue ring. Never removed.
- **Inline link:** blue and underlined. Color alone does not mark a link inside
  a sentence.
- **Theme toggle:** one control, in the app sidebar and in the portal header, so
  an anonymous visitor can switch too. It names what it will do, not what is on.
- **Charts:** a card per measure, its current reading in the header, and the plot
  under it. A single-series plot carries no legend, because the header is the
  readout. A multi-series plot keeps uPlot's legend, which names the lines.
  Fewer than two samples is a sentence, not a plot: one point draws nothing and
  pads the time axis out to years.
  A plot with one line, and a resource bar, take `--data-primary`,
  which is the brand lime on dark and the link blue on light. Lime reads 13.4:1 on
  the dark canvas and 1.2:1 on white, so it carries data in exactly one theme.
  A plot with several lines takes slots instead, `--series-1` to `--series-8`,
  assigned in order and never by rank, so a series keeps its color when the drawn
  count changes. Each theme has its own eight steps, selected against its own canvas
  rather than flipped from the other. Both sets pass a lightness band, a chroma
  floor, deutan and tritan separation, and 3:1 contrast. `Chart.tsx` reads
  `--axis`, `--grid` and `--tick` at draw time, so nothing in a plot is
  hardcoded. Re-validate a set before you change it.

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

- No third theme, and no `prefers-color-scheme` branch: dark is the product's
  default, not a guess about the desk.
- No color literal in a component. If a library needs a string, read the token
  with `cssVar` and depend on `useTheme` so it redraws.
- No accent as text, border, chart line, or status color.
- No second brand color, no gradient, no glassmorphism.
- No drop shadow that carries hierarchy.
- No unlabeled list rows, and no state word without a column naming it.
- No raw server vocabulary in the UI.
- No more than the four defined radii.
