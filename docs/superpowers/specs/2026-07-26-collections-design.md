# Collections

## Problem

The portal groups services by host, which answers "what runs where" and not
"what does this team use". Services carry a free-text `category` string that
nothing owns, nothing validates and nobody can attach an avatar or a
description to. A visitor who is not signed in lands on a flat list keyed by
machine names that mean nothing to them.

## Solution

A **collection** is a named, owned, avatared set of services: Manufacturing,
Embedded, Ops. It replaces `tools.category`. The portal groups by collection by
default and keeps host grouping behind a toggle.

Collections are deliberately separate from the existing user **groups**. A
person on one team routinely uses services from several collections, so
membership of a team and the set of collections a person sees are different
questions. Groups stay a set of users; collections stay a set of services. The
two meet only where a group is used as a principal in a collection's editor or
visibility grants.

## Data model

All of this goes into `server/internal/store/migrations/0001_init.sql` in
place. Reeve is pre-release; there is no data to migrate and no `0002_*.sql`.

```sql
CREATE TABLE collections (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    visibility  TEXT NOT NULL DEFAULT 'public'
                  CHECK (visibility IN ('public', 'restricted')),
    creator_id  TEXT NOT NULL REFERENCES users(id),
    icon_path   TEXT NOT NULL DEFAULT '',
    created_at  TEXT NOT NULL
);
CREATE INDEX idx_collections_creator ON collections(creator_id);

CREATE TABLE collection_tools (
    collection_id TEXT NOT NULL REFERENCES collections(id) ON DELETE CASCADE,
    tool_id       TEXT NOT NULL REFERENCES tools(id) ON DELETE CASCADE,
    PRIMARY KEY (collection_id, tool_id)
);
CREATE INDEX idx_collection_tools_tool ON collection_tools(tool_id);

CREATE TABLE collection_editors (
    collection_id  TEXT NOT NULL REFERENCES collections(id) ON DELETE CASCADE,
    principal_type TEXT NOT NULL CHECK (principal_type IN ('user', 'group')),
    principal_id   TEXT NOT NULL,
    PRIMARY KEY (collection_id, principal_type, principal_id)
);
CREATE INDEX idx_collection_editors_principal
    ON collection_editors(principal_type, principal_id);

CREATE TABLE collection_visibility (
    collection_id  TEXT NOT NULL REFERENCES collections(id) ON DELETE CASCADE,
    principal_type TEXT NOT NULL CHECK (principal_type IN ('user', 'group')),
    principal_id   TEXT NOT NULL,
    PRIMARY KEY (collection_id, principal_type, principal_id)
);
CREATE INDEX idx_collection_visibility_principal
    ON collection_visibility(principal_type, principal_id);
```

`tools.category` and `idx_tools_category` are removed from the same file.

`principal_type` mirrors `tool_visibility` and `credential_access`, so a user
group works as a principal everywhere a single user does, and `DeleteGroup`
gains two more explicit cleanup statements alongside the two it already runs.

Deleting a collection cascades its membership and grant rows. Deleting a tool
cascades its membership rows. Neither deletes the other's subject.

## Permissions

| Action | Allowed for |
| --- | --- |
| Create | any authenticated user |
| See | admin, creator, any editor principal, any visibility principal, or `visibility = 'public'` (which includes anonymous) |
| Edit name, description, avatar, visibility, services, editors, grants | admin, creator, any editor principal |
| Delete | admin or creator |

Editor implies see. A collection defaults to `public`, matching the default on
`tools.visibility`: everything is visible unless somebody restricts it.

The store expresses "see" as one clause, bound with the user id four times,
directly parallel to the existing `toolVisibleClause`:

```
visibility = 'public'
OR creator_id = ?
OR id IN (SELECT collection_id FROM collection_editors
          WHERE principal_type='user' AND principal_id = ?)
OR id IN (SELECT collection_id FROM collection_editors
          WHERE principal_type='group'
            AND principal_id IN (SELECT group_id FROM group_members WHERE user_id = ?))
OR id IN (SELECT collection_id FROM collection_visibility
          WHERE principal_type='user' AND principal_id = ?)
OR id IN (SELECT collection_id FROM collection_visibility
          WHERE principal_type='group'
            AND principal_id IN (SELECT group_id FROM group_members WHERE user_id = ?))
```

### How collection visibility interacts with service visibility

A collection's visibility gates the collection: its heading on the portal, its
page, its sidebar entry, and its appearance in the collections list on a
service.

It never gates the services inside it. Whether a service is listed at all is
decided by that service's own `visibility` and grants, exactly as today. A
service is then shown under every collection the viewer can see, and under
`Ungrouped` if it belongs to no collection the viewer can see.

The consequence, which is the intended one: a public service cannot be hidden
by putting it in a restricted collection, and adding a service to a restricted
collection cannot silently remove it from somebody's portal.

## API

`contracts/API.md` is updated in the same commit as the handlers.

```
GET    /api/v1/public/collections                              anonymous, public only
GET    /api/v1/collections                                     visible to caller
POST   /api/v1/collections                                     any authenticated user
GET    /api/v1/collections/{id}
PATCH  /api/v1/collections/{id}
DELETE /api/v1/collections/{id}
PUT    /api/v1/collections/{id}/tools/{toolId}
DELETE /api/v1/collections/{id}/tools/{toolId}
GET    /api/v1/collections/{id}/editors
PUT    /api/v1/collections/{id}/editors/{type}/{principalId}
DELETE /api/v1/collections/{id}/editors/{type}/{principalId}
GET    /api/v1/collections/{id}/visibility
PUT    /api/v1/collections/{id}/visibility/{type}/{principalId}
DELETE /api/v1/collections/{id}/visibility/{type}/{principalId}
GET    /api/v1/collections/{id}/icon                           unauthenticated, as tool icons
POST   /api/v1/collections/{id}/icon
DELETE /api/v1/collections/{id}/icon
```

Collection view:

```json
{
  "id": "…", "name": "Manufacturing", "description": "Shop-floor tooling.",
  "visibility": "public", "creator_id": "…", "icon_url": "/api/v1/collections/…/icon",
  "tool_count": 12, "can_edit": true, "created_at": "…"
}
```

`can_edit` is computed per caller and is always `false` on the public endpoint.
`GET /api/v1/collections/{id}` additionally returns `tool_ids`.

Every write route except create and the two list routes is gated by the same
`can_edit` check; delete is gated by admin-or-creator. A caller who cannot see
a collection gets `404`, not `403`, so restricted names do not leak.

### Changes to existing endpoints

`Tool` loses `category` and gains `collections`, an array of
`{id, name, icon_url}` filtered to the collections the caller can see:

```json
{ "id": "…", "name": "Grafana",
  "collections": [{"id": "…", "name": "Metrics", "icon_url": "…"}] }
```

`POST /api/v1/tools` and `PATCH /api/v1/tools/{id}` accept `collection_ids`,
an array replacing the full membership set. Unknown ids are a `400`. A caller
may put a service into any collection they can see; collection membership is a
property of the service, so it follows the service's own edit permission.

The `?category=` query parameter on `GET /api/v1/tools` and
`GET /api/v1/public/tools` becomes `?collection=<id>`.

### New supporting endpoint

`GET /api/v1/principals` (authenticated) returns
`{"users": [{"id", "display_name", "email"}], "groups": [{"id", "name"}]}`.

The existing visibility picker in `ToolDetail` reads `/api/v1/admin/users` and
`/api/v1/admin/groups`, which a basic user cannot call. Since any user can now
create a collection and grant access to it, a non-admin picker needs a source
of principal names. This endpoint exposes display name, email and group name to
any signed-in user, and nothing else. On a LAN tool where the team already
shares a service catalogue that is the intended level of disclosure.

## UI

`DESIGN.md` governs. Avatars reuse `IconUploader` and the existing icon
directory; collection cards reuse `Card`, `EntityIcon` and `Pill`.

**Portal** (`web/src/pages/Portal.tsx`). A second segmented control sits beside
`list / map / graph`, reading `by: Collection | Host`. Collection is the
default; the choice is reflected as `?by=host` so a view is linkable. Map and
graph are unaffected. A collection heading shows avatar, name and count and
opens a `CollectionInfoModal` — same shape as `HostInfoModal` — with the
avatar, description and the services in it. `Ungrouped` sorts last and has no
modal.

`web/src/lib/group.ts` gains `groupToolsByCollection`, sitting next to the
existing `groupToolsByHost` and returning the same `{header, tools}[]` shape.

**Sidebar** (`web/src/components/Layout.tsx`). A `Collections` entry between
`Services` and `Hosts`. The admin `Groups` entry is relabelled `User groups`;
that page is otherwise untouched.

**`/collections`** — grid of collection cards (avatar, name, description,
count, a `restricted` pill where it applies) and a `New collection` button
available to every signed-in user.

**`/collections/:id`** — header with avatar, name, description and count; the
services in it as a grid; an `Edit` action when `can_edit`.

**Collection editor** — name, description, avatar upload, visibility
public/restricted with a principal picker when restricted, an editors picker,
and a service multi-select. Both pickers are one component over
`/api/v1/principals`, shared with the reworked tool visibility panel.

**`ToolFormPage`** — the `category` text input becomes a multi-select of the
collections the user can see, with an inline "create new" that posts a
collection and selects it. This is the add-a-service-and-pick-its-group flow.

**`Catalog`** — the category dropdown becomes a collection dropdown; the card
pills list collection names. `ToolDetail` swaps its `Category` row for
`Collections`, and `ToolInfoModal` shows collection pills.

## Testing

Store (`server/internal/store/collections_test.go`): CRUD; the visibility
clause across all five paths (public, creator, user editor, group editor, user
grant, group grant) and the negative case; membership listing; cascade on
collection delete and on tool delete; `DeleteGroup` clearing collection editor
and visibility rows.

Handlers (`server/collection_handlers_test.go`): a basic user can create; a
non-editor `PATCH` is `403`; creator, editor and admin `PATCH` succeed; a
non-creator basic user's `DELETE` is `403` and the creator's succeeds; a caller
who cannot see a collection gets `404` on `GET`; the public endpoint omits
restricted collections; the tool payload strips collections the caller cannot
see; `collection_ids` with an unknown id is `400`; `?collection=` filters.

E2E (`web/e2e/`): the portal defaults to collection grouping, the host toggle
works and survives a reload via `?by=host`, and a restricted collection is
absent for an anonymous visitor.

Front-end work ends with a Playwright screenshot sweep across portal (list,
map, graph, both groupings, anonymous and signed in), catalog, tool detail,
tool form, the two collection pages, and the untouched admin pages.

No LLM is involved anywhere in this feature, so there is no eval suite.

## Out of scope

Nesting collections, per-collection webhooks or alerts, collection-scoped
credential grants, and reordering services within a collection.
