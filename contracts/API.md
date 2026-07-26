# Reeve REST API

Base path: `/api`. All responses are JSON. LAN-only; not exposed publicly.

## Authentication

- **Session cookie** (`reeve_session`) — set by `POST /api/auth/login` or
  `/signup`; used by the web UI. It resolves to that user's principal, never
  elevated.

The agent uses a separate host **enrollment token** (`Authorization: Bearer
rva_…`) for `POST /api/ingest` only.

## Errors

Every error uses one envelope with a stable machine code and a human message:

```json
{ "code": "not_found", "message": "tool not found" }
```

Common codes: `unauthenticated` (401), `forbidden` (403), `not_found` (404),
`bad_request` / `invalid_*` (400), `email_taken` / `duplicate_request` /
`name_taken` (409), `internal` (500).

## Catalog

### `GET /api/tools`

Returns the tools the caller may see (visibility applied per principal).
Query params: `search`, `collection` (a collection id, not a name), `host`,
`source_type`. Credentials are **never** included.

```json
[
  {
    "id": "…", "name": "Grafana", "slug": "grafana", "description": "",
    "collections": [{"id": "…", "name": "Metrics", "icon_url": "…"}],
    "tags": ["prod"], "scheme": "https", "address": "10.0.0.5", "port": 3000,
    "url": "", "physical_location": "", "host_id": "…",
    "source_type": "systemd", "source_ref": "grafana.service",
    "status": "up",                      // up | down | agent_offline | unknown
    "visibility": "public", "creator_id": "…", "can_edit": false,
    "log_alert_enabled": false
  }
]
```

### `GET /api/tools/{id}`

Same shape, single object. Returns **404** (not 403) when the caller may not see
the tool, so a restricted tool's existence stays hidden. Visibility: `public` = anonymous-visible; `restricted` = specific users/groups only.

Credentials are retrieved only via the separate, access-checked, audited
`POST /api/credentials/{id}/reveal` — never through the catalog API.

### `POST /api/tools`, `PATCH /api/tools/{id}`

Both accept `collection_ids`, an array of collection ids that **replaces** the
tool's whole membership set. An id the caller cannot see is a `400
invalid_collection`, and on create the tool is not kept. The `collections`
array in every tool response is filtered to the collections that caller may
see, so two callers can get different arrays for the same tool.

### `POST /api/tools` — the `slug`

`slug` is the name in `/go/<slug>` and is unique across the catalog. Omit it
and the server derives one from the name (`Paperless-ngx` → `paperless-ngx`),
adding `-2`, `-3` on collision. Supply one and it is taken literally: a
collision is `409 slug_taken` rather than a silent rename, because it is a URL
the caller is about to share. A slug with no letter or digit in it is `400
invalid_slug`. A `PATCH` that omits `slug` leaves it alone, so renaming a tool
never moves a link someone has bookmarked.

### `GET /api/public/tools` (no auth)

The unauthenticated portal surface. Returns only tools with `visibility: "public"`
(now defined as **anonymous-visible on the LAN**). A reduced DTO, not the same
shape as `GET /api/tools`: `id`, `name`, `description`, `collections`,
`slug`, `tags`, `scheme`, `address`, `port`, `url`, `physical_location`,
`host_id`, `source_type`, `status`. It omits `creator_id`, `source_ref`, `visibility`,
`can_edit`, and `log_alert_enabled`. Query params: `search`, `collection`,
`host`, `source_type`. Credentials are **never** included. Restricted tools
are never returned. `collections` carries public collections only, since the
caller is anonymous.

## Resolving where a service is

A tool with no `address` and no `url` follows its host: the agent reports the
host's own address on the route to the server on every push, and the server
resolves it at request time. That keeps a link working when the host's DHCP
lease changes. A tool that has an `address` or a `url` of its own always uses
it — the fallback only fills a blank.

### `GET /go/{slug}` (no auth for public tools)

`302` to wherever the tool is now, with the resolved URL in `Location`. This is
the link worth bookmarking or sharing.

- `404` when the slug is unknown **or** the caller may not see the tool, so the
  route cannot be used to enumerate the catalog.
- `409 no_endpoint` when the tool has no URL, no address, and no host that has
  reported one.
- A host that is offline still redirects to the last address it reported: a
  stale answer beats no answer.

### `GET /api/endpoints/{slug}` (no auth for public tools)

The same resolution as JSON, for scripts and for the UI.

```json
{
  "tool_id": "…", "slug": "grafana",
  "url": "http://192.168.1.42:3000",
  "scheme": "http", "address": "192.168.1.42", "port": 3000,
  "host_id": "…", "host_ip": "192.168.1.42",
  "source": "host",          // url | address | host
  "host_online": true
}
```

`source` says which rule answered, so a caller can tell a pinned tool from one
following its host. Same `404` and `409` semantics as `/go/{slug}`.

## Collections

A collection is a named, owned, avatared set of services. It replaces the old
free-text `category` field.

A collection's visibility gates the **collection**: its heading on the portal,
its page, and its appearance in a service's `collections` array. It never gates
the services inside it — a service's own `visibility` decides whether it is
listed at all, and a service is shown under every collection the viewer can
see, or under `Ungrouped` when it is in none they can see. So a public service
cannot be hidden by putting it in a restricted collection.

| Route | Auth | Notes |
| --- | --- | --- |
| `GET /api/public/collections` | none | public collections only, `can_edit` always `false` |
| `GET /api/collections` | user | the collections the caller may see |
| `POST /api/collections` | user | any signed-in user; `201`, `400 invalid_name`, `400 invalid_visibility`, `409 name_taken` |
| `GET /api/collections/{id}` | user | adds `tool_ids`; `404` when not visible |
| `PATCH /api/collections/{id}` | editor | `200`, `403 forbidden`, `404`, `409 name_taken` |
| `DELETE /api/collections/{id}` | creator or admin | `204`, `403 forbidden` |
| `PUT/DELETE /api/collections/{id}/tools/{toolId}` | editor | `204`; `404 not_found` for an unknown tool |
| `GET /api/collections/{id}/editors` | editor | list of `{principal_type, principal_id}` |
| `PUT/DELETE /api/collections/{id}/editors/{type}/{principalId}` | editor | `204`; `400 invalid_principal` unless `type` is `user` or `group` |
| `GET /api/collections/{id}/visibility` | editor | same shape as editors |
| `PUT/DELETE /api/collections/{id}/visibility/{type}/{principalId}` | editor | as above |
| `GET /api/collections/{id}/icon` | none | as tool icons: public collections to anyone, restricted only to principals who may see them |
| `POST/DELETE /api/collections/{id}/icon` | editor | multipart `icon` field, returns `{"icon_url": "…"}` |

"editor" means admin, the creator, or a user or group holding an editor grant.
A caller who may not see a collection gets `404`, not `403`, so restricted
names do not leak.

```json
{
  "id": "…", "name": "Manufacturing", "description": "Shop-floor tooling.",
  "visibility": "public", "creator_id": "…",
  "icon_url": "/api/collections/…/icon",
  "tool_count": 12, "can_edit": true, "created_at": "…"
}
```

### `GET /api/principals`

Authenticated. Returns the names a non-admin needs to fill a visibility or
editor picker:

```json
{ "users": [{"id": "…", "display_name": "…", "email": "…"}],
  "groups": [{"id": "…", "name": "ops"}] }
```

It exposes display name, email and group name to **any** signed-in user, and
nothing else. Any user can create a collection and grant access to it, so the
admin-only `/admin/users` and `/admin/groups` are not a usable source.

### `GET /api/public/hosts` (no auth)

Returns only hosts referenced by at least one public tool, trimmed to `id`,
`name`, and `status`. It omits `os`, `physical_location`, `agent_version`, and
`last_seen_at`. Hosts whose tools are all restricted, or that have no tools,
are omitted.

## Agent updates (admin)

The authed host payload from `GET /api/hosts` (and any other endpoint that
returns a `hostView`) carries two fields the public/anonymous host payload
never does:

- `auto_update` — the host's policy override: `default` (inherit the fleet
  setting), `on`, or `off`.
- `update_state` — one of:
  - `up_to_date` — the host reported the sha256 of one of the agent builds
    this server publishes, which is the same comparison the agent's own
    self-update makes. Version strings decide nothing: two builds of one
    version are different binaries.
  - `outdated` — running a binary this server does not publish, auto-update
    enabled, no slot yet.
  - `updating` — holds a rollout slot and is inside the stall window.
  - `stalled` — held a slot past the stall window without checking in on the
    new version.
  - `disabled` — auto-update is off, either by the host's own veto
    (`REEVE_AUTO_UPDATE=false`) or by policy (fleet default off with no
    per-host override, or an explicit `off` override).
  - `unknown` — the host has never reported a checksum, or the server ships no
    agent builds, so there is nothing to compare.

### `GET /api/admin/agent-updates`

Fleet rollout rollup.

```json
{
  "server_version": "1.4.0",
  "counts": {"up_to_date": 12, "outdated": 2, "updating": 1, "stalled": 1, "disabled": 3, "unknown": 0},
  "paused": true,
  "stalled": [{"id": "…", "name": "db-1"}]
}
```

`paused` is `true` whenever `stalled` is non-empty — a stalled host halts the
rollout for every other host until it's cleared. `counts.stalled` is always
greater than zero while `paused` is `true`: a host that can no longer update
(local veto or a policy of `off`) releases its slot instead of holding one, so
the banner can never name a host the page renders as fine. Each entry carries
the host `id` so the banner can link to the page where an admin takes it out of
the rollout.

### `POST /api/admin/agent-updates/resume`

Releases every rollout slot stamped before the stall cutoff, un-pausing the
rollout. `204 No Content`. Resume hands the same host its slot back on the next
push, so a host that genuinely cannot update re-stalls; setting that host's
policy to `off` is the way to take it out of the rollout for good.

### `PUT /api/admin/hosts/{id}/auto-update`

Body `{"policy": "default" | "on" | "off"}`. `200` with the updated `hostView`;
`400 invalid_policy` for anything else; `404 not_found` for an unknown host.

### `POST /api/admin/hosts/{id}/update-now`

Grants the host a rollout slot immediately, bypassing both the concurrency cap
and a paused rollout — this is an explicit operator override, not a paced grant.
`204 No Content`. `409 update_vetoed` if the host itself refuses updates
(`REEVE_AUTO_UPDATE=false`); `409 update_disabled` if auto-update is off for
the host by policy; `409 already_up_to_date` if the host already
reported the published build's checksum — stamping a slot for a host with
nothing to do would occupy a concurrency slot indefinitely if that host is
offline, silently pausing the rest of the fleet;
`409 version_unknown` if the server ships no agent builds or the host has never
reported a checksum, since a slot granted with nothing to compare can never
clear; `404 not_found` for an unknown host.

Setting the host's policy to `off` also releases any slot it holds, so "Never
update" reliably takes a host out of a rollout it is wedging. So does a push
reporting `auto_update_vetoed: true`.

### Settings: `agent_update`

`GET/PUT /api/admin/settings` carries an `agent_update` section alongside
`retention`, `smtp`, and `google`:

```json
{ "agent_update": {"enabled": true, "concurrency": 3, "stall_secs": 900} }
```

`enabled` is the fleet-wide default auto-update policy (a host's own
`auto_update` override wins over it). `concurrency` (1-100) caps how many
hosts may hold a rollout slot at once. `stall_secs` (1-86400) is how long a
host may hold a slot before it's considered stalled and pauses the rollout.
`PUT` validates both bounds and returns `400 invalid_agent_update` on failure;
like the other settings sections, omitting `agent_update` leaves it unchanged.

## Other endpoints (used by the web UI)

- Auth: `POST /auth/signup`, `/auth/login`, `/auth/logout`; `GET /me`.
- Tools: `POST /tools`, `PATCH/DELETE /tools/{id}`, visibility grants under
  `/tools/{id}/visibility/{ptype}/{pid}`.
- Collections: see the Collections section above; `GET /principals`.
- Credentials belong to a **host**, not a tool: `GET /hosts/{id}/credentials`
  (any signed-in user, metadata only), `POST /admin/hosts/{id}/credentials`,
  `PATCH/DELETE /admin/credentials/{id}`, `POST /credentials/{id}/reveal`
  (admin or a standing grant; every reveal is audited against the host).
- Access: `POST /hosts/{id}/access-requests`, `GET /access-requests?box=`,
  `POST /access-requests/{id}/approve|deny` (admin only — a host has no other
  owner), `/admin/hosts/{id}/access/{ptype}/{pid}`.
- Hosts: `GET /hosts`, `/hosts/{id}/inventory`, `/hosts/{id}/events`,
  `/hosts/{id}/uptime`, `/hosts/{id}/metrics` (the metrics response also
  carries `processes`, the latest top-by-CPU-and-memory snapshot).
- `GET /hosts/{id}/process-usage?window=1h|12h|24h|7d|30d` answers the same
  question over time instead of at one instant: per command, `cpu_avg`,
  `cpu_max`, `mem_avg`, `mem_max` and the `samples` behind them, top 25 by each
  of the two dimensions. Pushes are folded into five-minute buckets on ingest,
  so that is the floor on how finely a window can be sliced, and a week of it
  costs thousands of rows rather than millions. Kept for the 5m retention
  window; "Clear metrics" drops it with the rest.
- Admin (`role=admin`): `/admin/users`, `/admin/groups`, `/admin/hosts`,
  `/admin/webhooks`, `/admin/alerts`, `/admin/deliveries`,
  `/admin/audit/reveals|grants`, `/admin/server-info`, `/admin/agent-updates`
  (see Agent updates above).

### SSH install and the credential it keeps

`POST /admin/hosts/{id}/ssh-install` pushes the agent over SSH. By default it
then stores the login it was given as a credential on that host — `ssh_password`
or `ssh_key` — and grants the installing admin standing access to it. Send
`"skip_credential_save": true` to opt out; the checkbox in the deploy modal is
worded as the opt-out so an absent field means save. Nothing is stored when the
install fails, or when there was no secret to keep (agent-forwarded key, or
NOPASSWD sudo). The response carries `credential_saved`.

`DELETE /admin/hosts/{id}` removes a host along with its credentials, metrics,
events and command history. A tool pinned to it keeps a `host_id` that no
longer resolves, and the agent on the machine keeps running until it is
uninstalled.

## Host controls (admin only)

`POST /admin/hosts/{id}/commands` with `{action, target}` queues one action for
a host's agent; `GET /admin/hosts/{id}/commands` returns the last 20 with their
status, output, and the email of the admin who asked.

Actions are a fixed allowlist, mapped server-side to an argv and never passed to
a shell: `reboot` and `poweroff` (no target), `service_start|stop|restart`
(target is a systemd unit) and `container_start|stop|restart` (target is a
Docker container name).

- `400 unknown_action` — the action is not on the list.
- `400 unknown_target` — the host has not reported that unit or container, so
  the server will not name it.
- `409 control_unavailable` — the host's agent has not reported
  `control_enabled`: it predates the feature, or the machine runs it with
  `REEVE_ALLOW_CONTROL=false`. Control is on by default.

Delivery rides the push ack, because the agent is push-only and the server never
dials it: a command is collected on the host's next push (~15s), run, and its
outcome reported on the push after. Anything neither collected nor answered
within 10 minutes is expired. `reboot` and `poweroff` are acknowledged just
before they are invoked, since they kill the agent that would report them.

## Ingest (agent → server)

`POST /api/ingest` with `Authorization: Bearer rva_…`. Body is the
`contracts.Push` type (see `contracts.go`). Rejects unauthenticated, malformed,
or oversized pushes with the standard error envelope.

The agent reports `agent_checksum` (the sha256 of the binary it is running,
which is what the server judges `update_state` against),
`auto_update_vetoed` (`true` when the host set
`REEVE_AUTO_UPDATE=false` and will refuse any update) and `ip_address`, the
host's own address on the route to this server. An empty `ip_address` leaves
the stored one alone: a tick that could not work the address out is not
evidence the host moved. The endpoint replies
`200` with `{"check_now": bool}` (a `contracts.PushAck`), where `check_now` is
the server's instruction to run a self-update immediately.
