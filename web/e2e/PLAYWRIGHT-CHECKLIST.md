# Playwright UI checklist

Run through this with Playwright (screenshots + real clicks) after ANY frontend
change, including areas not touched. Passing typecheck/lint/build/e2e is NOT a
substitute — look at the screenshots and actually click things.

## What is now automated

Two specs cover part of this list, so a regression in them fails the build
rather than waiting for someone to notice:

- `zzz-visual.spec.ts` — pixel baselines for the three portal views, the modal
  over the list and over the map, and every authed page including ones a change
  did not touch. Data is masked and layout is asserted; regenerate baselines
  with `npm --prefix web run e2e -- --update-snapshots` and **look at the
  diff before accepting it**.
- `zzz-a11y.spec.ts` — axe against WCAG 2 A/AA on the portal, sign-in, the main
  authed pages and an open modal.

Neither replaces the sweep below. A snapshot cannot tell you the layout is
*good*, only that it has not changed, and axe checks only what a machine can.
`/admin/server` has no baseline on purpose: it renders the live state of the
machine the suite runs on.

`playwright.config.ts` always runs the e2e backend on `REEVE_DEV_PORT=7348` (it
sets the variable itself, unconditionally), so it never collides with `7338`,
which `npm run dev` uses by default and which a locally deployed Reeve
instance may already hold.

## Global chrome
- [ ] Version badge shows next to `Reeve` brand (app + portal).
- [ ] Search is centered; `Cmd/Ctrl+K` focuses it; hint shows the platform key.
- [ ] App (authed) and portal (anon) header/sidebar render without overlap; nav items all visible (Admin included).

## Authed app (sidebar layout)
- [ ] Sidebar shows all nav items + Admin section; active item highlighted.
- [ ] Each nav item routes correctly (Dashboard/Services/Hosts/Requests/Admin/*).
- [ ] User block (avatar + name) at sidebar bottom → profile; Sign out works.
- [ ] Content area scrolls; sidebar fixed.

## Portal (anon) — list / map / graph
- [ ] Switcher centered; each view fills full page width and height (no page scroll unless list overflows).
- [ ] LIST: services are cards in a grid, grouped under each host.
- [ ] MAP: dark Leaflet tiles, zoomed to host location (not whole globe), scroll-zoom, pins labelled.
- [ ] GRAPH: host nodes linked to service nodes with angled edges; hosts visually distinct; looks like a graph.

## Clicks → modals (test EVERY trigger)
- [ ] LIST: click host header → host modal; click service card → service modal.
- [ ] MAP: click host pin → host modal **on top of the map** (z-index).
- [ ] GRAPH: click host node → host modal; click service node → service modal.
- [ ] HOST modal: click a service row → switches to that service's modal.
- [ ] SERVICE modal: authed shows "Open service" → detail page; anon does not.

## Profile / account
- [ ] Avatar + thumbnail upload/remove; display name save; password change; delete account.

## Admin
- [ ] Users list (avatars/names), role toggle, deactivate, delete.
- [ ] Host detail → Settings: icon + thumbnail upload, location map picker (city autocomplete + pin).

## Forms
- [ ] Add-for-monitoring page: endpoint toggle (host&port / URL), save → tool detail.

## Notification channels
- [ ] Webhooks page: Receiver defaults to "Detect from URL"; a saved row names
      what the URL resolved to.
- [ ] Picking Telegram or PagerDuty reveals the id/key field; picking Custom
      template reveals the body editor and the variable list.
- [ ] Edit on a row opens the same fields, saves, and keeps the stored token when
      the token field is left blank.
- [ ] Test reports the receiver's own words on a rejection, not "failed".

## Agent updates
- [ ] Hosts page: no banner above the table unless the rollout is paused.
- [ ] Hosts row: outdated shows an amber pill, stalled a red one, up-to-date no pill.
- [ ] Host detail → Overview: Agent card shows version, state pill, policy select, Update now.
- [ ] Host detail → Overview: a vetoed host explains that the machine refuses updates.
- [ ] Settings → Agent updates: toggle, concurrency, stall timeout save and persist.
- [ ] Paused rollout: banner names the stalled hosts; Resume rollout clears it.
- [ ] Paused rollout: each named host is a link to its detail page, and the
      banner says setting Auto-update to Off takes it out of the rollout.
- [ ] Setting a stalled host's policy to Off clears the pause on the Hosts page.
