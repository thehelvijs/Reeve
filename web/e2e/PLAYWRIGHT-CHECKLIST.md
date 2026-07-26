# Playwright UI checklist

Run through this with Playwright (screenshots + real clicks) after ANY frontend
change, including areas not touched. Passing typecheck/lint/build/e2e is NOT a
substitute — look at the screenshots and actually click things.

`playwright.config.ts` always runs the e2e backend on `REEVE_DEV_PORT=8099` (it
sets the variable itself, unconditionally), so it never collides with `8080`,
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
- [ ] Host detail: icon + thumbnail upload, location map picker (city autocomplete + pin).

## Forms
- [ ] Add-for-monitoring page: endpoint toggle (host&port / URL), save → tool detail.

## Agent updates
- [ ] Hosts page: rollup line shows the server version and per-state counts.
- [ ] Hosts row: outdated shows an amber pill, stalled a red one, up-to-date no pill.
- [ ] Host detail: Agent card shows version, state pill, policy select, Update now.
- [ ] Host detail: a vetoed host explains that the machine refuses updates.
- [ ] Settings → Agent updates: toggle, concurrency, stall timeout save and persist.
- [ ] Paused rollout: banner names the stalled hosts; Resume rollout clears it.
