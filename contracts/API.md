# Reeve REST API (v1)

Base path: `/api/v1`. All responses are JSON. LAN-only; not exposed publicly.

## Authentication

- **Session cookie** (`reeve_session`) — set by `POST /api/v1/auth/login` or
  `/signup`; used by the web UI. It resolves to that user's principal, never
  elevated.

The agent uses a separate host **enrollment token** (`Authorization: Bearer
rva_…`) for `POST /api/v1/ingest` only.

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
Query params: `search`, `collection` (a collection id, not a name), `host`,
`source_type`. Credentials are **never** included.

```json
[
  {
    "id": "…", "name": "Grafana", "description": "",
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

### `GET /api/v1/tools/{id}`

Same shape, single object. Returns **404** (not 403) when the caller may not see
the tool, so a restricted tool's existence stays hidden. Visibility: `public` = anonymous-visible; `restricted` = specific users/groups only.

Credentials are retrieved only via the separate, access-checked, audited
`POST /api/v1/credentials/{id}/reveal` — never through the catalog API.

### `POST /api/v1/tools`, `PATCH /api/v1/tools/{id}`

Both accept `collection_ids`, an array of collection ids that **replaces** the
tool's whole membership set. An id the caller cannot see is a `400
invalid_collection`, and on create the tool is not kept. The `collections`
array in every tool response is filtered to the collections that caller may
see, so two callers can get different arrays for the same tool.

### `GET /api/v1/public/tools` (no auth)

The unauthenticated portal surface. Returns only tools with `visibility: "public"`
(now defined as **anonymous-visible on the LAN**). A reduced DTO, not the same
shape as `GET /api/v1/tools`: `id`, `name`, `description`, `collections`,
`tags`, `scheme`, `address`, `port`, `url`, `physical_location`, `host_id`,
`source_type`, `status`. It omits `creator_id`, `source_ref`, `visibility`,
`can_edit`, and `log_alert_enabled`. Query params: `search`, `collection`,
`host`, `source_type`. Credentials are **never** included. Restricted tools
are never returned. `collections` carries public collections only, since the
caller is anonymous.

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
| `GET /api/v1/public/collections` | none | public collections only, `can_edit` always `false` |
| `GET /api/v1/collections` | user | the collections the caller may see |
| `POST /api/v1/collections` | user | any signed-in user; `201`, `400 invalid_name`, `400 invalid_visibility`, `409 name_taken` |
| `GET /api/v1/collections/{id}` | user | adds `tool_ids`; `404` when not visible |
| `PATCH /api/v1/collections/{id}` | editor | `200`, `403 forbidden`, `404`, `409 name_taken` |
| `DELETE /api/v1/collections/{id}` | creator or admin | `204`, `403 forbidden` |
| `PUT/DELETE /api/v1/collections/{id}/tools/{toolId}` | editor | `204`; `404 not_found` for an unknown tool |
| `GET /api/v1/collections/{id}/editors` | editor | list of `{principal_type, principal_id}` |
| `PUT/DELETE /api/v1/collections/{id}/editors/{type}/{principalId}` | editor | `204`; `400 invalid_principal` unless `type` is `user` or `group` |
| `GET /api/v1/collections/{id}/visibility` | editor | same shape as editors |
| `PUT/DELETE /api/v1/collections/{id}/visibility/{type}/{principalId}` | editor | as above |
| `GET /api/v1/collections/{id}/icon` | none | as tool icons: public collections to anyone, restricted only to principals who may see them |
| `POST/DELETE /api/v1/collections/{id}/icon` | editor | multipart `icon` field, returns `{"icon_url": "…"}` |

"editor" means admin, the creator, or a user or group holding an editor grant.
A caller who may not see a collection gets `404`, not `403`, so restricted
names do not leak.

```json
{
  "id": "…", "name": "Manufacturing", "description": "Shop-floor tooling.",
  "visibility": "public", "creator_id": "…",
  "icon_url": "/api/v1/collections/…/icon",
  "tool_count": 12, "can_edit": true, "created_at": "…"
}
```

### `GET /api/v1/principals`

Authenticated. Returns the names a non-admin needs to fill a visibility or
editor picker:

```json
{ "users": [{"id": "…", "display_name": "…", "email": "…"}],
  "groups": [{"id": "…", "name": "ops"}] }
```

It exposes display name, email and group name to **any** signed-in user, and
nothing else. Any user can create a collection and grant access to it, so the
admin-only `/admin/users` and `/admin/groups` are not a usable source.

### `GET /api/v1/public/hosts` (no auth)

Returns only hosts referenced by at least one public tool, trimmed to `id`,
`name`, and `status`. It omits `os`, `physical_location`, `agent_version`, and
`last_seen_at`. Hosts whose tools are all restricted, or that have no tools,
are omitted.

## Other endpoints (used by the web UI)

- Auth: `POST /auth/signup`, `/auth/login`, `/auth/logout`; `GET /me`.
- Tools: `POST /tools`, `PATCH/DELETE /tools/{id}`, visibility grants under
  `/tools/{id}/visibility/{ptype}/{pid}`.
- Collections: see the Collections section above; `GET /principals`.
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

`POST /api/v1/ingest` with `Authorization: Bearer rva_…`. Body is the
`contracts.Push` type (see `contracts.go`). Rejects unauthenticated, malformed,
or wrong-protocol pushes with the standard error envelope.

**Protocol v2:** The agent reports `auto_update_vetoed` (`true` when the host set
`REEVE_AUTO_UPDATE=false` and will refuse any update). The endpoint replies
`200` with `{"check_now": bool}` (a `contracts.PushAck`), where `check_now` is
the server's instruction to run a self-update immediately; earlier versions
reply `204 No Content`.
