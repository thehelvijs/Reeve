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

## Accounts and groups

### `POST /api/admin/users`

Admin. Invites an account: `{email, role, display_name}`, where `role` is
`admin` or `basic` and defaults to `basic`. The server hashes random bytes
nobody holds as the password, so the invite link is the only way in.

```json
{ "user": {"id": "…", "email": "…", "role": "basic", …},
  "emailed": true, "invite_link": "https://…/reset?token=…" }
```

The link is a `password_resets` row that lives seven days, so an invite and a
reset consume the same token endpoint (`POST /auth/reset`). With a mail relay
configured the server sends the link and `emailed` is true. With no relay, or a
relay that refuses the message, `invite_link` carries it back for the admin to
hand over. The two fields are exclusive: the server never returns a link it
mailed.

### `GET /api/groups`

Authenticated. Returns the groups the caller may manage: every group for an
admin, the groups they moderate for anybody else. A member who moderates nothing
reads an empty list.

```json
[{ "id": "…", "name": "ops",
   "members": [{"user_id": "…", "email": "…", "display_name": "…",
                "avatar_url": "…", "role": "moderator"}] }]
```

The membership carries the person's name, so a moderator reads their own group
without also being handed `/admin/users`.

### Group membership

`POST /api/groups/{id}/members` takes `{email, role}` and adds an account that
already exists. An unknown address is a 404, which is also what a moderator gets
for an address outside their group: they never learn whether it has an account.
`PATCH /api/groups/{id}/members/{userId}` takes `{role}`.
`DELETE /api/groups/{id}/members/{userId}` removes one.

All three answer 403 unless the caller is an admin or a moderator **of that
group**. A moderator cannot demote or remove themselves (400 `self_lockout`),
and cannot create accounts or groups. Membership spans as many groups as an
account is put in, with a role in each.

### `GET /api/public/hosts` (no auth)

Returns only hosts referenced by at least one public tool, trimmed to `id`,
`name`, `status`, and what the portal map draws: `latitude`, `longitude` and
`pin_color`. It omits `os`, `physical_location`, `agent_version`, and
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
    per-host override, or an explicit `off` override). A host whose policy is off
    reads `updating` while it holds a slot an operator forced; a *paced* slot on
    such a host is dangling and still reads `disabled`.
  - `unknown` — the host has never reported a checksum, or the server ships no
    agent builds, so there is nothing to compare. Such a host is never chased by
    the rollout on its own; `update-now` still grants it a slot, and while it
    holds one the state reads `updating` (then `stalled`) like any other.

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
`204 No Content`. A policy of `off` is **not** a refusal here: it means "not on
the paced rollout", and this is the operator asking for one host now. Such a slot
is recorded as forced and sits outside the rollout entirely — it neither counts
against the concurrency cap nor pauses anyone if it goes stale, so pressing this
on a host that never comes back cannot halt the fleet. `409 update_vetoed` if the
host itself refuses updates (`REEVE_AUTO_UPDATE=false`), the one case nothing can
override because the agent ignores the ack; `409 already_up_to_date` if the host already
reported the published build's checksum — stamping a slot for a host with
nothing to do would occupy a concurrency slot indefinitely if that host is
offline, silently pausing the rest of the fleet;
`409 version_unknown` only if the server ships no agent builds — a host that has
never reported a checksum is granted a slot, since the agent compares its own
binary against the published one and does not need the server to know what it is
running, and that state cannot be fixed from the UI any other way; `404 not_found` for an unknown host.

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

### Settings: `server_update`

The same endpoint carries which stream of builds the server follows:

```json
{ "server_update": {"channel": "release"} }
```

`channel` is one of `release`, `develop` or `main`; anything else is
`400 invalid_update_channel`. It is the whole of what a client may say about
server updates — the image tag each channel resolves to belongs to the server,
so a caller can select among three published builds and nothing else.

Writing it publishes the resolved tag to `update-channel` beside the database,
which the updater sidecar reads on its next poll; the switch is a container
recreate, so the response to this `PUT` is the last thing the calling build
sends. A stored channel the running build does not recognise reads back as
`release`, so a downgrade cannot leave an instance chasing a tag nothing
publishes.

## Pipelines

Reeve reads run status from two forges: **GitLab** (gitlab.com or an instance
you host) and **GitHub** (github.com or an Enterprise Server). Either or both
may be connected; only the URL tells a hosted instance from a self-managed one.

### Settings: `gitlab` and `github`

`GET/PUT /api/admin/settings` carries one block per forge, the same three
fields each:

```json
{ "gitlab": {"enabled": true, "url": "https://gitlab.example.com", "token": "glpat-…"},
  "github": {"enabled": true, "url": "", "token": "github_pat_…"} }
```

`token` is write-only: it is sealed with the master key and reads back as
`token_set` only. An empty `url` means the hosted instance
(`https://gitlab.com`, `https://api.github.com`) rather than an error, and
reads back as that. An enabled block must carry an absolute `url` and a token
(stored or supplied), otherwise `400 invalid_gitlab` / `400 invalid_github`; a
disabled one is saved as written so it can be filled in over more than one
visit.

A GitLab token needs `read_api`. A GitHub token needs Actions and Metadata
read, which is `repo` on a classic token. Nothing is ever written to either.

### Pipeline groups

A pipeline group is an operator's own named set of repo paths, so a dashboard
does not depend on how the repos are arranged on the forge. The **provider is
the group's**, not the instance's: firmware on GitLab and tooling on GitHub sit
on one page.

`GET /api/pipeline-groups` (any signed-in account) lists them as stored, with
no call to a forge behind it:

```json
[{"id": "3f2a…", "name": "Firmware", "provider": "gitlab",
  "projects": ["firmware/Powerboard5"]}]
```

The writes are admin-only:

- `POST /api/admin/pipeline-groups` `{"name": "Firmware", "provider": "gitlab"}`
  → `201` with the record. `provider` is `gitlab` or `github`, anything else is
  `400 invalid_provider`. A name is unique; a repeat is `409 name_taken`.
- `PATCH /api/admin/pipeline-groups/{id}` `{"name": "…"}` → `204`. A group's
  provider is fixed at creation, since its stored paths belong to that forge.
- `DELETE /api/admin/pipeline-groups/{id}` → `204`; membership cascades.
- `POST|DELETE /api/admin/pipeline-groups/{id}/projects` `{"path": "group/repo"}`
  → `204`. The path rides in the body because a repo path carries slashes of
  its own. Adding is idempotent. A path with no slash is `400 invalid_path`, an
  unknown group is `404`, and a group already holding 200 projects is
  `400 group_full`.

### `GET /api/admin/repo-search?provider=&q=`

Admin only. Proxies the forge's own repo lookup so the group editor can offer
repos by name instead of asking for a full path. Returns up to 20 as
`{"name", "path", "url"}`, most recently active first, and an empty `q` is a
valid "what is there" query.

On GitLab this is project search. On GitHub it walks the token's own
repositories (newest push first, up to 300) and filters locally, because
GitHub's search spans all of GitHub rather than what a token can see.

An unknown provider is `400 invalid_provider`, a provider with no connection is
`412 forge_not_configured`, and a forge that refuses is `502 forge_failed`.

### `GET /api/pipelines`

Any signed-in account. Returns every group with the latest run of each member:

```json
{
  "configured": true,
  "groups": [{
    "id": "3f2a…",
    "name": "Firmware",
    "provider": "gitlab",
    "projects": [{
      "name": "Powerboard5", "path": "firmware/Powerboard5",
      "url": "https://gitlab.example.com/firmware/Powerboard5",
      "status": "failed", "ref": "develop",
      "updated_at": "2026-08-17T09:00:00Z",
      "pipeline_url": "https://gitlab.example.com/firmware/Powerboard5/-/pipelines/9"
    }]
  }]
}
```

`status` is lowercase and uses GitLab's vocabulary for **both** forges, so one
word has one meaning and a client needs one colour map: `success`, `failed`,
`running`, `pending`, `canceled`, `skipped`, `manual`, or empty for a repo that
has never run anything. A GitHub run maps onto it by conclusion
(`failure`/`timed_out`/`startup_failure` → `failed`, `cancelled` → `canceled`,
`neutral`/`stale` → `skipped`, `action_required` → `manual`) and, while it has
no conclusion yet, by status (`in_progress` → `running`, everything else
→ `pending`).

Projects come back failed-first, then by name. `configured` is false when no
forge is connected at all, which is a `200` rather than an error so a client can
prompt instead of failing. Each group reports its own trouble in `error` — its
provider not connected, or the forge unreachable — while the other groups still
render, and a single repo the forge will not answer for carries `error` on that
project. An empty group costs no call at all.

GitLab is read in one GraphQL request per group, each member project under its
own alias. GitHub has no batch query for workflow runs, so it is one request per
repo, at most 8 in flight.

## Notification channels (admin)

`GET/POST /admin/webhooks`, `PATCH/DELETE /admin/webhooks/{id}`,
`POST /admin/webhooks/test`. A channel is a URL, a scope, a `format`, a
`min_severity` gate and an encrypted `config`. Every read redacts `token` and
`routing_key`; sending either as an empty string on a `PATCH` keeps the stored
value.

`owner_type` is what the channel watches, and `owner_id` names it:

- `global` receives every event. `owner_id` is empty.
- `host` receives the host's own alerts and the alerts of the services on it.
- `tool` receives one service's alerts.
- `group` receives the alerts of the services a group can see, and of the hosts
  it can get into by credential.

Scope is the whole rule and the scopes add up: an event reaches every channel
whose scope covers it. A narrow channel never takes traffic off a broad one.
`POST` rejects a `tool`, `host` or `group` channel with no `owner_id` as
`400 invalid_owner`. Scope is fixed at creation; `PATCH` does not change it.

`events` narrows what a channel receives to named types: `down`, `log_error`,
`agent_offline`, `<metric>_high` for each threshold metric, and
`access_request`. An empty list is every type, and stays every type when a new
one is added, so selecting all of them stores as empty. A fire and its matching
resolve share a type, so a channel gets both or neither. An unknown name is
`400 invalid_events`. `GET /admin/webhook-formats` returns the list the form
offers, beside the receiver vocabulary.

`format` names the receiver, because each one rejects a body that is not its
own: Discord needs `content`, Slack needs `text`, Webex needs `markdown`.
Reeve's own payload reaches none of them, so it is rewritten on the way out.

- `auto` (the default) reads the receiver off the URL. Discord, Slack, Google
  Chat, Teams (both the retired connector and a Power Automate flow), Webex,
  ntfy, Telegram, PagerDuty and Opsgenie are recognised.
- A name overrides the URL: `discord`, `slack`, `mattermost`, `rocketchat`,
  `googlechat`, `teams`, `teamsflow`, `webex`, `ntfy`, `gotify`, `telegram`,
  `pagerduty`. Mattermost and Rocket.Chat run on the operator's own
  domain under a path a private sink could also use, so they are never guessed.
- `generic` sends Reeve's JSON untouched, which is what a sink of your own wants.
- `custom` posts `config.template`: JSON with `{{name}}` placeholders, replaced
  by that field of the payload and escaped for the string they sit in. A name
  the payload does not carry is left standing, so a typo shows up in the
  delivery. `GET /admin/webhook-formats` returns the format list and the
  placeholder names, which is what the form offers.

Every shaped message carries the severity and the subject in the emphasis its
receiver understands: markdown for Discord, Mattermost, Rocket.Chat, Teams and
Webex, its own syntax for Slack and Google Chat, and no emphasis for ntfy,
Gotify and Telegram. Below the message is a link to the page that shows what
fired, PagerDuty gets it in `links` instead. An alert is sent by a background
tick with no request to read the host from, so the link comes from the public URL
in Settings (or `REEVE_PUBLIC_URL`) when one is set, and otherwise from the last
address an admin reached the UI on. A loopback address never counts, because a link to localhost
helps nobody reading it in Discord. Until one of the two is known the message
carries no link.

Two receivers need a value no URL carries: Telegram takes `config.chat_id` and
PagerDuty takes `config.routing_key`. Without it the delivery fails naming the
missing field rather than passing on the receiver's own rejection.

`POST /admin/webhooks/test` delivers a probe. A saved channel is named by `id`;
a channel the form is still holding sends its `url`, `format` and `config`. A
receiver that answers badly is not an API failure: the response is `200` with
`{"ok": false, "error": "…"}` carrying the receiver's own words.

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
- `GET /hosts` includes the machine Reeve itself runs on, as the reserved id
  `__server__` with `is_server: true`. It is an ordinary host: install an agent
  on that machine (see `enroll-token` below) and it reports inventory, runs
  commands and joins the rollout like any other. Until one is installed the
  server samples that machine itself, so the row still carries metrics, disks
  and a heartbeat but no inventory; an agent reporting stands that sampling down,
  and going quiet past the row's offline window hands it back. It cannot be
  deleted.
- Hosts: `GET /hosts`, `/hosts/{id}/inventory`, `/hosts/{id}/events`,
  `/hosts/{id}/uptime`, `/hosts/{id}/metrics` (the metrics response also
  carries `processes`, the latest top-by-CPU-and-memory snapshot).
  The response also carries `disks`: `{ts, disks:[{mount, device, fs_type, used,
  total}]}` — every filesystem the agent reported, a replaceable snapshot rather
  than a series. `metrics.disk_used`/`disk_total` stay the root filesystem, which
  is what the chart and the disk alert threshold are built on. Kernel and virtual
  mounts are dropped by the agent, as are snap loop devices, and a bind mount is
  reported once per device. The machine running Reeve answers on this route like
  any other host, under the reserved id `__server__`.
  An inventory item is `{source_type, source_ref, name, state, detail, linked}`.
  `state` is what the machine says the thing is doing — a systemd active state or
  a container state, empty for a cron job. `detail` is the sub-state, the image,
  or the schedule. A container whose labels name it — `coolify.name` and
  friends, which a PaaS sets because the container name it generates is a uuid —
  reads under that name, with its own name joining the image in `detail`.
  `source_ref` is the container id either way: the name is cosmetic, and nothing
  links or acts on it.
  A process in the metrics payload carries `container` when it runs inside one,
  named as that container is listed, and omits it when it runs on the host. That
  is what tells php-fpm belonging to one deployment from php-fpm belonging to
  another on a machine hosting several.
  The inventory carries a fourth list, `processes`, alongside `services`,
  `containers` and `cron_jobs`. A process is linked by its command, not its pid,
  so a restart is the same thing still running, and one entry stands for however
  many copies of that command are up. `managed_by` on a container is the deployer
  that put it there and on a process is the container it runs inside, so a client
  can group either under what owns it.
- `GET /hosts/{id}/process-usage?window=1h|12h|24h|7d|30d` answers the same
  question over time instead of at one instant: per command, `cpu_avg`,
  `cpu_max`, `mem_avg`, `mem_max` and the `samples` behind them, top 25 by each
  of the two dimensions. Pushes are folded into five-minute buckets on ingest,
  so that is the floor on how finely a window can be sliced, and a week of it
  costs thousands of rows rather than millions. Kept for the 5m retention
  window; "Clear metrics" drops it with the rest.
- `PATCH /admin/hosts/{id}` sets `physical_location`, `latitude`, `longitude`
  and `pin_color` together; the location picker autosaves all four. `pin_color`
  is `#rrggbb` or empty for the brand accent, and anything else is a 400
  (`invalid_color`): the value is interpolated into the map marker's inline
  style. A `null` coordinate clears it.
- `GET /admin/geocode?q=` turns typed text into location suggestions:
  `[{label, lat, lon}]`, at most eight, empty for a query under three
  characters. The server asks Photon (the OSM typeahead geocoder) and the page
  asks the server, so the page keeps its `connect-src 'self'` policy and the
  geocoder sees one caller instead of every browser. The host location picker
  also matches the city list bundled in the page, which needs no request and
  works with no internet.
- Admin (`role=admin`): `/admin/users`, `/admin/groups`, `/admin/hosts`,
  `/admin/webhooks`, `/admin/alerts`, `/admin/deliveries`,
  `/admin/audit/reveals|grants`, `/admin/server-info`, `/admin/agent-updates`
  (see Agent updates above).

`PUT /admin/hosts/{id}/identity` with `{name, description}` renames a host and
sets what it is for, returning the updated `hostView`. Both are display only —
tools, telemetry and grants hang off the id — so this includes the server's own
row, whose name starts as a default. A blank name is refused; a blank
description clears it, and `description` is omitted from a host that has none.

### `POST /admin/hosts/{id}/enroll-token`

Mints a fresh enrollment token for a host that already exists and returns the
`install_command` for it. `POST /admin/hosts` shows a token
once and never again, so this is the way to install an agent on a host created
earlier — the server's own `__server__` row included. It revokes the previous
token: an agent still using it stops reporting until the new command is run.
Refused with `unreachable_server_url` when the UI is being reached on an address
no other machine can push to, the same check `ssh-install` makes.

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
uninstalled. The server's own host (`is_server`) answers `409 server_host`:
startup would recreate the row, without the history the delete threw away.

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
