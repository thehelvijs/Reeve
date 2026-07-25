# Reeve REST API (v1)

Base path: `/api/v1`. All responses are JSON. LAN-only; not exposed publicly.

## Authentication

- **Session cookie** (`lv_session`) — set by `POST /api/v1/auth/login` or
  `/signup`; used by the web UI. It resolves to that user's principal, never
  elevated.

The agent uses a separate host **enrollment token** (`Authorization: Bearer
lva_…`) for `POST /api/v1/ingest` only.

## Errors

Every error uses one envelope with a stable machine code and a human message:

```json
{ "code": "not_found", "message": "tool not found" }
```

Common codes: `unauthenticated` (401), `forbidden` (403), `not_found` (404),
`bad_request` / `invalid_*` (400), `email_taken` / `duplicate_request` /
`name_taken` (409), `internal` (500).

## Catalog

### `GET /api/v1/tools`

Returns the tools the caller may see (visibility applied per principal).
Query params: `search`, `category`, `host`, `source_type`. Credentials are
**never** included.

```json
[
  {
    "id": "…", "name": "Grafana", "description": "", "category": "metrics",
    "tags": ["prod"], "scheme": "https", "address": "10.0.0.5", "port": 3000,
    "url": "", "physical_location": "", "host_id": "…",
    "source_type": "systemd", "source_ref": "grafana.service",
    "status": "up",                      // up | down | agent_offline | unknown
    "visibility": "public", "creator_id": "…", "can_edit": false,
    "log_alert_enabled": false
  }
]
```

### `GET /api/v1/tools/{id}`

Same shape, single object. Returns **404** (not 403) when the caller may not see
the tool, so a restricted tool's existence stays hidden. Visibility: `public` = anonymous-visible; `restricted` = specific users/groups only.

Credentials are retrieved only via the separate, access-checked, audited
`POST /api/v1/credentials/{id}/reveal` — never through the catalog API.

### `GET /api/v1/public/tools` (no auth)

The unauthenticated portal surface. Returns only tools with `visibility: "public"`
(now defined as **anonymous-visible on the LAN**). A reduced DTO, not the same
shape as `GET /api/v1/tools`: `id`, `name`, `description`, `category`, `tags`,
`scheme`, `address`, `port`, `url`, `physical_location`, `host_id`,
`source_type`, `status`. It omits `creator_id`, `source_ref`, `visibility`,
`can_edit`, and `log_alert_enabled`. Query params: `search`, `category`,
`host`, `source_type`. Credentials are **never** included. Restricted tools
are never returned.

### `GET /api/v1/public/hosts` (no auth)

Returns only hosts referenced by at least one public tool, trimmed to `id`,
`name`, and `status`. It omits `os`, `physical_location`, `agent_version`, and
`last_seen_at`. Hosts whose tools are all restricted, or that have no tools,
are omitted.

## Other endpoints (used by the web UI)

- Auth: `POST /auth/signup`, `/auth/login`, `/auth/logout`; `GET /me`.
- Tools: `POST /tools`, `PATCH/DELETE /tools/{id}`, visibility grants under
  `/tools/{id}/visibility/{ptype}/{pid}`.
- Credentials: `GET/POST /tools/{id}/credentials`, `PATCH/DELETE
  /credentials/{id}`, `POST /credentials/{id}/reveal`.
- Access: `POST /tools/{id}/access-requests`, `GET /access-requests?box=`,
  `POST /access-requests/{id}/approve|deny`, `/tools/{id}/access/{ptype}/{pid}`.
- Hosts: `GET /hosts`, `/hosts/{id}/inventory`, `/hosts/{id}/events`,
  `/hosts/{id}/uptime`.
- Admin (`role=admin`): `/admin/users`, `/admin/groups`, `/admin/hosts`,
  `/admin/webhooks`, `/admin/alerts`, `/admin/deliveries`,
  `/admin/audit/reveals|grants`, `/admin/server-info`.

## Ingest (agent → server)

`POST /api/v1/ingest` with `Authorization: Bearer lva_…`. Body is the
`contracts.Push` type (see `contracts.go`). Rejects unauthenticated, malformed,
or wrong-protocol pushes with the standard error envelope.
