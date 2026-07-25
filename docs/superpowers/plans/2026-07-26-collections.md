# Collections Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the free-text `tools.category` field with owned, avatared, permissioned **collections**, and group the public portal by collection instead of by host.

**Architecture:** A new `collections` table plus three join tables (`collection_tools`, `collection_editors`, `collection_visibility`), mirroring the shape of the existing `tool_visibility` / `credential_access` tables so a user group works as a principal wherever a single user does. The Go store gains `collections.go`, the HTTP layer gains `collection_handlers.go` and a `/api/v1/collections` route family, and the React UI gains a Collections nav entry, two pages, an editor modal and a group-by toggle on the portal.

**Tech Stack:** Go 1.x + `net/http` `ServeMux` with method+path patterns, SQLite via the existing `store.DB`, React 18 + TypeScript + Vite + Tailwind, Playwright for e2e. No new dependencies.

## Global Constraints

- Reeve is **pre-release**. Schema changes edit `server/internal/store/migrations/0001_init.sql` **in place**. Do not create `0002_*.sql`. Do not write any data-migration SQL. The DB is thrown away and recreated.
- `agent/` must not import `server/`. Nothing in this plan touches `agent/`.
- Comments: one line maximum, ever. Only for a non-obvious *why*. Never restate the code, never narrate a diff, never justify a gating `if`.
- No inline ternaries (`?:`) in Go or TypeScript. Use explicit `if`/`else` that assigns the value. **Exception:** JSX conditional rendering (`{cond && <X/>}`) is fine and is used throughout the codebase.
- Always brace control-statement bodies, even single-line.
- Every task ends with a commit. Conventional Commits: `type: summary`, imperative, ~50 chars, no trailing period. **No AI-attribution trailers of any kind.**
- Stage explicitly with `git add <paths>`. Never `git add -A` or `git add .`.
- The gate must pass before each commit: `make gate` (go vet + go test) for Go tasks, `make web-check` (tsc --noEmit + eslint --max-warnings 0) for web tasks.
- UI follows `DESIGN.md`: near-black canvas, single acid-lime accent `#e4f222` used only for brand / focus rings / one primary action per view, hairline borders and a surface ladder instead of shadows, radii 6 / 12 / 9999px.
- Accent colour literal: `#e4f222`. Tailwind tokens in use: `bg-canvas`, `bg-surface-1`, `bg-surface-2`, `bg-surface-3`, `border-hairline`, `text-content`, `text-muted`, `text-accent`, `rounded-button`, `rounded-pill`.
- Collection default visibility is `public`. Everything is visible unless somebody restricts it.

---

## File Structure

**Created:**

| File | Responsibility |
| --- | --- |
| `server/internal/store/collections.go` | `Collection` struct, CRUD, the visibility clause, membership, editor and visibility grants |
| `server/internal/store/collections_test.go` | Store-level tests for all of the above |
| `server/collection_handlers.go` | `/api/v1/collections` HTTP handlers including the icon trio |
| `server/collection_handlers_test.go` | Handler-level permission and payload tests |
| `server/principal_handlers.go` | `GET /api/v1/principals` for non-admin pickers |
| `web/src/pages/Collections.tsx` | `/collections` grid |
| `web/src/pages/CollectionDetail.tsx` | `/collections/:id` detail |
| `web/src/components/CollectionEditor.tsx` | Create/edit modal |
| `web/src/components/CollectionInfoModal.tsx` | Portal read-only modal |
| `web/src/components/PrincipalPicker.tsx` | Shared user/group grant picker over `/api/v1/principals` |
| `web/e2e/collections.spec.ts` | Portal grouping and permission e2e |

**Modified:**

| File | Change |
| --- | --- |
| `server/internal/store/migrations/0001_init.sql` | Four new tables; drop `category` column and `idx_tools_category` |
| `server/internal/store/tools.go` | Drop `Tool.Category`; `ToolFilter.CollectionID`; collection membership helpers |
| `server/internal/store/groups.go` | `DeleteGroup` clears collection editor and visibility rows |
| `server/tool_handlers.go` | `collections` in both response shapes, `collection_ids` input, `?collection=` |
| `server/app.go` | Register the new routes |
| `contracts/contracts.go` | `ToolDTO.Category` → `ToolDTO.Collections` |
| `contracts/API.md` | Document the route family and the tool payload change |
| `web/src/api.ts` | `Collection`, `CollectionRef`, `Principals` types; `Tool.collections` |
| `web/src/lib/group.ts` | `groupToolsByCollection` beside `groupToolsByHost` |
| `web/src/pages/Portal.tsx` | Group-by toggle, collection headings, info modal |
| `web/src/components/Layout.tsx` | Collections nav entry; relabel admin Groups |
| `web/src/components/NavIcon.tsx` | `collections` icon |
| `web/src/pages/Catalog.tsx` | Collection filter and pills |
| `web/src/pages/ToolFormPage.tsx` | Collection multi-select with inline create |
| `web/src/pages/ToolDetail.tsx` | Collections row |
| `web/src/components/ToolInfoModal.tsx` | Collection pills |
| `web/src/App.tsx` | Two new routes |

**Parallelism:** Tasks 1–6 are the Go side and run in sequence. Tasks 7–12 are the web side. Task 7 (`api.ts` types) only needs the shapes written in this plan, so the web chain may start immediately and in parallel with the Go chain. Task 13 integrates and must run last.

---

## Task 1: Schema and collection CRUD

**Files:**
- Modify: `server/internal/store/migrations/0001_init.sql`
- Create: `server/internal/store/collections.go`
- Test: `server/internal/store/collections_test.go`

**Interfaces:**
- Consumes: `store.NewID()`, `store.ErrNotFound`, `db.exec1(query, args...)`, `db.sql` — all already in `server/internal/store`.
- Produces:
  - `type Collection struct { ID, Name, Description, Visibility, CreatorID, IconPath string; CreatedAt time.Time }`
  - `func (db *DB) CreateCollection(c Collection) (Collection, error)`
  - `func (db *DB) GetCollection(id string) (Collection, error)`
  - `func (db *DB) UpdateCollection(c Collection) error`
  - `func (db *DB) DeleteCollection(id string) error`
  - `func (db *DB) SetCollectionIconPath(id, path string) error`
  - `func (db *DB) ListCollectionsVisibleTo(userID string, isAdmin bool) ([]Collection, error)`
  - `func (db *DB) ListPublicCollections() ([]Collection, error)`
  - `func (db *DB) CanSeeCollection(userID string, isAdmin bool, id string) (bool, error)`

- [ ] **Step 1: Add the tables to the schema**

In `server/internal/store/migrations/0001_init.sql`, insert this block immediately after the `CREATE INDEX idx_group_members_user` line:

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

The `collection_tools` table references `tools(id)`, so this block must sit **after** `CREATE TABLE tools`. If `CREATE TABLE tools` comes later in the file, put the whole block directly after `CREATE INDEX idx_tools_category` instead, and note that that index is removed in Task 3.

- [ ] **Step 2: Write the failing test**

Create `server/internal/store/collections_test.go`. Look at an existing test in the same package first (`server/internal/store/rollup_test.go`) for how a test DB is opened, and reuse that exact helper — do not invent a new one.

```go
package store

import "testing"

func TestCollectionCRUD(t *testing.T) {
	db := testDB(t)
	u := mustUser(t, db, "a@example.com")

	c, err := db.CreateCollection(Collection{Name: "Manufacturing", Description: "Shop floor.", CreatorID: u})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if c.ID == "" {
		t.Fatal("create returned no id")
	}
	if c.Visibility != VisibilityPublic {
		t.Errorf("visibility = %q, want public by default", c.Visibility)
	}

	got, err := db.GetCollection(c.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Name != "Manufacturing" || got.Description != "Shop floor." {
		t.Errorf("round trip mismatch: %+v", got)
	}

	c.Name = "Embedded"
	c.Visibility = VisibilityRestricted
	if err := db.UpdateCollection(c); err != nil {
		t.Fatalf("update: %v", err)
	}
	got, _ = db.GetCollection(c.ID)
	if got.Name != "Embedded" || got.Visibility != VisibilityRestricted {
		t.Errorf("update not applied: %+v", got)
	}

	if _, err := db.CreateCollection(Collection{Name: "Embedded", CreatorID: u}); err == nil {
		t.Error("duplicate name accepted, want unique constraint error")
	}

	if err := db.DeleteCollection(c.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := db.GetCollection(c.ID); err != ErrNotFound {
		t.Errorf("get after delete = %v, want ErrNotFound", err)
	}
	if err := db.DeleteCollection(c.ID); err != ErrNotFound {
		t.Errorf("double delete = %v, want ErrNotFound", err)
	}
}
```

Add the two helpers at the bottom of the file only if the package does not already have equivalents — check first:

```go
func mustUser(t *testing.T, db *DB, email string) string {
	t.Helper()
	u, err := db.CreateUser(email, "hash", "basic")
	if err != nil {
		t.Fatalf("create user %s: %v", email, err)
	}
	return u.ID
}
```

Match `CreateUser`'s real signature from `server/internal/store/users.go`; adjust the helper if it differs.

- [ ] **Step 3: Run it and confirm it fails**

Run: `go test ./server/internal/store/ -run TestCollectionCRUD -v`
Expected: a compile failure — `undefined: Collection`.

- [ ] **Step 4: Write `collections.go`**

```go
package store

import (
	"database/sql"
	"time"
)

// Collection is a named, owned set of tools used to group the catalog.
type Collection struct {
	ID          string
	Name        string
	Description string
	Visibility  string
	CreatorID   string
	IconPath    string
	CreatedAt   time.Time
}

const collectionSelect = `SELECT id, name, description, visibility, creator_id, icon_path, created_at
	FROM collections`

// CreateCollection inserts a collection, defaulting to public visibility.
func (db *DB) CreateCollection(c Collection) (Collection, error) {
	c.ID = NewID()
	c.CreatedAt = time.Now().UTC()
	if c.Visibility == "" {
		c.Visibility = VisibilityPublic
	}
	_, err := db.sql.Exec(
		`INSERT INTO collections(id, name, description, visibility, creator_id, icon_path, created_at)
		 VALUES (?,?,?,?,?,?,?)`,
		c.ID, c.Name, c.Description, c.Visibility, c.CreatorID, c.IconPath,
		c.CreatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return Collection{}, err
	}
	return c, nil
}

// GetCollection returns a collection by id, or ErrNotFound.
func (db *DB) GetCollection(id string) (Collection, error) {
	row := db.sql.QueryRow(collectionSelect+` WHERE id = ?`, id)
	c, err := scanCollection(row)
	if err == sql.ErrNoRows {
		return Collection{}, ErrNotFound
	}
	return c, err
}

// UpdateCollection overwrites the editable fields of a collection.
func (db *DB) UpdateCollection(c Collection) error {
	return db.exec1(
		`UPDATE collections SET name=?, description=?, visibility=? WHERE id=?`,
		c.Name, c.Description, c.Visibility, c.ID)
}

// DeleteCollection removes a collection; membership and grants cascade.
func (db *DB) DeleteCollection(id string) error {
	return db.exec1(`DELETE FROM collections WHERE id = ?`, id)
}

// SetCollectionIconPath sets (or clears, when empty) a collection's icon path.
func (db *DB) SetCollectionIconPath(id, path string) error {
	return db.exec1(`UPDATE collections SET icon_path = ? WHERE id = ?`, path, id)
}

// ListCollectionsVisibleTo returns the collections a user may see, ordered by
// name. Admins see everything.
func (db *DB) ListCollectionsVisibleTo(userID string, isAdmin bool) ([]Collection, error) {
	q := collectionSelect
	var args []any
	if !isAdmin {
		q += ` WHERE ` + collectionVisibleClause
		args = visibleArgs(userID)
	}
	return db.queryCollections(q+` ORDER BY name`, args...)
}

// ListPublicCollections returns anonymous-visible collections ordered by name.
func (db *DB) ListPublicCollections() ([]Collection, error) {
	return db.queryCollections(collectionSelect + ` WHERE visibility = 'public' ORDER BY name`)
}

// CanSeeCollection reports whether the user may see the collection.
func (db *DB) CanSeeCollection(userID string, isAdmin bool, id string) (bool, error) {
	q := `SELECT 1 FROM collections WHERE id = ?`
	args := []any{id}
	if !isAdmin {
		q += ` AND (` + collectionVisibleClause + `)`
		args = append(args, visibleArgs(userID)...)
	}
	var one int
	if err := db.sql.QueryRow(q, args...).Scan(&one); err != nil {
		return false, nil
	}
	return true, nil
}

// collectionVisibleClause matches collections a non-admin may see. Bind with
// visibleArgs: public OR creator OR editor (user/group) OR granted (user/group).
const collectionVisibleClause = `(
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
)`

func visibleArgs(userID string) []any {
	return []any{userID, userID, userID, userID, userID}
}

func (db *DB) queryCollections(q string, args ...any) ([]Collection, error) {
	rows, err := db.sql.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Collection
	for rows.Next() {
		c, err := scanCollection(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func scanCollection(row scanner) (Collection, error) {
	var c Collection
	var created string
	if err := row.Scan(&c.ID, &c.Name, &c.Description, &c.Visibility,
		&c.CreatorID, &c.IconPath, &created); err != nil {
		return Collection{}, err
	}
	c.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	return c, nil
}
```

`scanner` is the existing interface in `server/internal/store/tools.go`; confirm it covers both `*sql.Row` and `*sql.Rows`. If it only covers `*sql.Rows`, use the same `Query`-then-`Next` shape `GetTool` uses rather than widening the interface.

- [ ] **Step 5: Run the test**

Run: `go test ./server/internal/store/ -run TestCollectionCRUD -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add server/internal/store/migrations/0001_init.sql server/internal/store/collections.go server/internal/store/collections_test.go
git commit -m "feat: add collections table and store CRUD"
```

---

## Task 2: Membership, editors and visibility grants

**Files:**
- Modify: `server/internal/store/collections.go`
- Modify: `server/internal/store/groups.go:47-55` (`DeleteGroup`)
- Test: `server/internal/store/collections_test.go`

**Interfaces:**
- Consumes: everything Task 1 produced; `store.VisibilityGrant` from `server/internal/store/tools.go`.
- Produces:
  - `func (db *DB) SetCollectionTools(collectionID string, toolIDs []string) error` — replaces the whole membership set in one transaction
  - `func (db *DB) AddCollectionTool(collectionID, toolID string) error`
  - `func (db *DB) RemoveCollectionTool(collectionID, toolID string) error`
  - `func (db *DB) ListCollectionToolIDs(collectionID string) ([]string, error)`
  - `func (db *DB) SetToolCollections(toolID string, collectionIDs []string) error`
  - `func (db *DB) CollectionsForTools(toolIDs []string) (map[string][]Collection, error)` — tool id → its collections, unfiltered
  - `func (db *DB) CountCollectionTools(collectionID string) (int, error)`
  - `func (db *DB) AddCollectionEditor(collectionID, principalType, principalID string) error`
  - `func (db *DB) RemoveCollectionEditor(collectionID, principalType, principalID string) error`
  - `func (db *DB) ListCollectionEditors(collectionID string) ([]VisibilityGrant, error)`
  - `func (db *DB) AddCollectionVisibility(collectionID, principalType, principalID string) error`
  - `func (db *DB) RemoveCollectionVisibility(collectionID, principalType, principalID string) error`
  - `func (db *DB) ListCollectionVisibility(collectionID string) ([]VisibilityGrant, error)`
  - `func (db *DB) CanEditCollection(userID string, isAdmin bool, collectionID string) (bool, error)`

- [ ] **Step 1: Write the failing tests**

Append to `server/internal/store/collections_test.go`:

```go
func TestCollectionVisibility(t *testing.T) {
	db := testDB(t)
	owner := mustUser(t, db, "owner@example.com")
	editor := mustUser(t, db, "editor@example.com")
	viewer := mustUser(t, db, "viewer@example.com")
	stranger := mustUser(t, db, "stranger@example.com")
	grp, err := db.CreateGroup("ops")
	if err != nil {
		t.Fatalf("create group: %v", err)
	}
	member := mustUser(t, db, "member@example.com")
	if err := db.AddGroupMember(grp.ID, member); err != nil {
		t.Fatalf("add member: %v", err)
	}

	pub, _ := db.CreateCollection(Collection{Name: "Public", CreatorID: owner})
	res, _ := db.CreateCollection(Collection{Name: "Restricted", CreatorID: owner, Visibility: VisibilityRestricted})
	if err := db.AddCollectionEditor(res.ID, "user", editor); err != nil {
		t.Fatalf("add editor: %v", err)
	}
	if err := db.AddCollectionVisibility(res.ID, "user", viewer); err != nil {
		t.Fatalf("add visibility: %v", err)
	}
	grpOnly, _ := db.CreateCollection(Collection{Name: "GroupOnly", CreatorID: owner, Visibility: VisibilityRestricted})
	if err := db.AddCollectionVisibility(grpOnly.ID, "group", grp.ID); err != nil {
		t.Fatalf("add group visibility: %v", err)
	}

	cases := []struct {
		name   string
		user   string
		id     string
		want   bool
	}{
		{"public to stranger", stranger, pub.ID, true},
		{"restricted to creator", owner, res.ID, true},
		{"restricted to user editor", editor, res.ID, true},
		{"restricted to granted user", viewer, res.ID, true},
		{"restricted to stranger", stranger, res.ID, false},
		{"group grant to member", member, grpOnly.ID, true},
		{"group grant to non-member", stranger, grpOnly.ID, false},
	}
	for _, tc := range cases {
		got, _ := db.CanSeeCollection(tc.user, false, tc.id)
		if got != tc.want {
			t.Errorf("%s: CanSeeCollection = %v, want %v", tc.name, got, tc.want)
		}
	}

	got, _ := db.CanSeeCollection(stranger, true, res.ID)
	if !got {
		t.Error("admin cannot see restricted collection")
	}

	list, _ := db.ListCollectionsVisibleTo(stranger, false)
	if len(list) != 1 || list[0].ID != pub.ID {
		t.Errorf("stranger sees %d collections, want only the public one", len(list))
	}
	pubList, _ := db.ListPublicCollections()
	if len(pubList) != 1 || pubList[0].ID != pub.ID {
		t.Errorf("ListPublicCollections = %+v, want only the public one", pubList)
	}
}

func TestCollectionEditRights(t *testing.T) {
	db := testDB(t)
	owner := mustUser(t, db, "owner@example.com")
	editor := mustUser(t, db, "editor@example.com")
	stranger := mustUser(t, db, "stranger@example.com")
	grp, _ := db.CreateGroup("leads")
	lead := mustUser(t, db, "lead@example.com")
	db.AddGroupMember(grp.ID, lead)

	c, _ := db.CreateCollection(Collection{Name: "Manufacturing", CreatorID: owner})
	db.AddCollectionEditor(c.ID, "user", editor)
	db.AddCollectionEditor(c.ID, "group", grp.ID)

	cases := []struct {
		name    string
		user    string
		isAdmin bool
		want    bool
	}{
		{"creator", owner, false, true},
		{"user editor", editor, false, true},
		{"group editor", lead, false, true},
		{"admin", stranger, true, true},
		{"stranger", stranger, false, false},
	}
	for _, tc := range cases {
		got, _ := db.CanEditCollection(tc.user, tc.isAdmin, c.ID)
		if got != tc.want {
			t.Errorf("%s: CanEditCollection = %v, want %v", tc.name, got, tc.want)
		}
	}

	db.RemoveCollectionEditor(c.ID, "user", editor)
	if got, _ := db.CanEditCollection(editor, false, c.ID); got {
		t.Error("removed editor can still edit")
	}
}

func TestCollectionMembership(t *testing.T) {
	db := testDB(t)
	u := mustUser(t, db, "a@example.com")
	c, _ := db.CreateCollection(Collection{Name: "Metrics", CreatorID: u})
	t1, _ := db.CreateTool(Tool{Name: "grafana", CreatorID: u})
	t2, _ := db.CreateTool(Tool{Name: "prom", CreatorID: u})

	if err := db.SetCollectionTools(c.ID, []string{t1.ID, t2.ID}); err != nil {
		t.Fatalf("set tools: %v", err)
	}
	ids, _ := db.ListCollectionToolIDs(c.ID)
	if len(ids) != 2 {
		t.Fatalf("membership = %v, want 2 tools", ids)
	}
	n, _ := db.CountCollectionTools(c.ID)
	if n != 2 {
		t.Errorf("count = %d, want 2", n)
	}

	if err := db.SetCollectionTools(c.ID, []string{t2.ID}); err != nil {
		t.Fatalf("replace tools: %v", err)
	}
	ids, _ = db.ListCollectionToolIDs(c.ID)
	if len(ids) != 1 || ids[0] != t2.ID {
		t.Errorf("after replace = %v, want only t2", ids)
	}

	byTool, _ := db.CollectionsForTools([]string{t2.ID})
	if len(byTool[t2.ID]) != 1 || byTool[t2.ID][0].Name != "Metrics" {
		t.Errorf("CollectionsForTools = %+v, want Metrics for t2", byTool)
	}

	if err := db.DeleteTool(t2.ID); err != nil {
		t.Fatalf("delete tool: %v", err)
	}
	ids, _ = db.ListCollectionToolIDs(c.ID)
	if len(ids) != 0 {
		t.Errorf("membership after tool delete = %v, want empty", ids)
	}

	db.SetToolCollections(t1.ID, []string{c.ID})
	if err := db.DeleteCollection(c.ID); err != nil {
		t.Fatalf("delete collection: %v", err)
	}
	var n2 int
	db.sql.QueryRow(`SELECT COUNT(*) FROM collection_tools`).Scan(&n2)
	if n2 != 0 {
		t.Errorf("collection_tools rows after collection delete = %d, want 0", n2)
	}
}

func TestDeleteGroupClearsCollectionGrants(t *testing.T) {
	db := testDB(t)
	u := mustUser(t, db, "a@example.com")
	grp, _ := db.CreateGroup("ops")
	c, _ := db.CreateCollection(Collection{Name: "Ops", CreatorID: u, Visibility: VisibilityRestricted})
	db.AddCollectionEditor(c.ID, "group", grp.ID)
	db.AddCollectionVisibility(c.ID, "group", grp.ID)

	if err := db.DeleteGroup(grp.ID); err != nil {
		t.Fatalf("delete group: %v", err)
	}
	if e, _ := db.ListCollectionEditors(c.ID); len(e) != 0 {
		t.Errorf("editors after group delete = %+v, want empty", e)
	}
	if v, _ := db.ListCollectionVisibility(c.ID); len(v) != 0 {
		t.Errorf("visibility after group delete = %+v, want empty", v)
	}
}
```

Note: `db.CreateTool(Tool{...})` still takes a `Category` field at this point; leave it unset. Task 3 removes the field and these tests keep compiling.

Foreign keys must be enforced for the cascade assertions. Check that the store opens SQLite with `_foreign_keys=on` (or runs `PRAGMA foreign_keys = ON`) in `server/internal/store/*.go`. If it does not, **stop and report it** rather than adding the pragma — that is a behaviour change beyond this task's scope.

- [ ] **Step 2: Run them and confirm they fail**

Run: `go test ./server/internal/store/ -run 'TestCollection|TestDeleteGroup' -v`
Expected: compile failure — `db.AddCollectionEditor undefined`.

- [ ] **Step 3: Implement in `collections.go`**

Append:

```go
// SetCollectionTools replaces a collection's whole membership set.
func (db *DB) SetCollectionTools(collectionID string, toolIDs []string) error {
	return db.replaceMembership(`collection_id`, collectionID, `tool_id`, toolIDs)
}

// SetToolCollections replaces the set of collections a tool belongs to.
func (db *DB) SetToolCollections(toolID string, collectionIDs []string) error {
	return db.replaceMembership(`tool_id`, toolID, `collection_id`, collectionIDs)
}

func (db *DB) replaceMembership(keyCol, keyID, valCol string, valIDs []string) error {
	tx, err := db.sql.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM collection_tools WHERE `+keyCol+` = ?`, keyID); err != nil {
		return err
	}
	stmt := `INSERT INTO collection_tools(` + keyCol + `, ` + valCol + `) VALUES (?, ?) ON CONFLICT DO NOTHING`
	for _, v := range valIDs {
		if v == "" {
			continue
		}
		if _, err := tx.Exec(stmt, keyID, v); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// AddCollectionTool puts a tool in a collection (idempotent).
func (db *DB) AddCollectionTool(collectionID, toolID string) error {
	_, err := db.sql.Exec(
		`INSERT INTO collection_tools(collection_id, tool_id) VALUES (?, ?) ON CONFLICT DO NOTHING`,
		collectionID, toolID)
	return err
}

// RemoveCollectionTool takes a tool out of a collection.
func (db *DB) RemoveCollectionTool(collectionID, toolID string) error {
	_, err := db.sql.Exec(
		`DELETE FROM collection_tools WHERE collection_id = ? AND tool_id = ?`, collectionID, toolID)
	return err
}

// ListCollectionToolIDs returns the tool ids in a collection.
func (db *DB) ListCollectionToolIDs(collectionID string) ([]string, error) {
	rows, err := db.sql.Query(`SELECT tool_id FROM collection_tools WHERE collection_id = ?`, collectionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// CountCollectionTools returns how many tools are in a collection.
func (db *DB) CountCollectionTools(collectionID string) (int, error) {
	var n int
	err := db.sql.QueryRow(`SELECT COUNT(*) FROM collection_tools WHERE collection_id = ?`,
		collectionID).Scan(&n)
	return n, err
}

// CollectionsForTools maps each given tool id to the collections it belongs to,
// without applying any visibility filter.
func (db *DB) CollectionsForTools(toolIDs []string) (map[string][]Collection, error) {
	out := map[string][]Collection{}
	if len(toolIDs) == 0 {
		return out, nil
	}
	q := `SELECT ct.tool_id, c.id, c.name, c.description, c.visibility, c.creator_id, c.icon_path, c.created_at
		FROM collection_tools ct JOIN collections c ON c.id = ct.collection_id
		WHERE ct.tool_id IN (` + placeholders(len(toolIDs)) + `) ORDER BY c.name`
	args := make([]any, len(toolIDs))
	for i, id := range toolIDs {
		args[i] = id
	}
	rows, err := db.sql.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var toolID string
		var c Collection
		var created string
		if err := rows.Scan(&toolID, &c.ID, &c.Name, &c.Description, &c.Visibility,
			&c.CreatorID, &c.IconPath, &created); err != nil {
			return nil, err
		}
		c.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		out[toolID] = append(out[toolID], c)
	}
	return out, rows.Err()
}

func placeholders(n int) string {
	if n == 0 {
		return ""
	}
	return strings.Repeat("?,", n-1) + "?"
}

// AddCollectionEditor grants edit rights on a collection to a user or group.
func (db *DB) AddCollectionEditor(collectionID, principalType, principalID string) error {
	_, err := db.sql.Exec(
		`INSERT INTO collection_editors(collection_id, principal_type, principal_id)
		 VALUES (?, ?, ?) ON CONFLICT DO NOTHING`, collectionID, principalType, principalID)
	return err
}

// RemoveCollectionEditor revokes edit rights.
func (db *DB) RemoveCollectionEditor(collectionID, principalType, principalID string) error {
	_, err := db.sql.Exec(
		`DELETE FROM collection_editors WHERE collection_id = ? AND principal_type = ? AND principal_id = ?`,
		collectionID, principalType, principalID)
	return err
}

// ListCollectionEditors returns the principals with edit rights.
func (db *DB) ListCollectionEditors(collectionID string) ([]VisibilityGrant, error) {
	return db.listGrants(`collection_editors`, collectionID)
}

// AddCollectionVisibility grants sight of a restricted collection.
func (db *DB) AddCollectionVisibility(collectionID, principalType, principalID string) error {
	_, err := db.sql.Exec(
		`INSERT INTO collection_visibility(collection_id, principal_type, principal_id)
		 VALUES (?, ?, ?) ON CONFLICT DO NOTHING`, collectionID, principalType, principalID)
	return err
}

// RemoveCollectionVisibility revokes a visibility grant.
func (db *DB) RemoveCollectionVisibility(collectionID, principalType, principalID string) error {
	_, err := db.sql.Exec(
		`DELETE FROM collection_visibility WHERE collection_id = ? AND principal_type = ? AND principal_id = ?`,
		collectionID, principalType, principalID)
	return err
}

// ListCollectionVisibility returns the principals granted sight of a collection.
func (db *DB) ListCollectionVisibility(collectionID string) ([]VisibilityGrant, error) {
	return db.listGrants(`collection_visibility`, collectionID)
}

func (db *DB) listGrants(table, collectionID string) ([]VisibilityGrant, error) {
	rows, err := db.sql.Query(
		`SELECT principal_type, principal_id FROM `+table+` WHERE collection_id = ?`, collectionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []VisibilityGrant{}
	for rows.Next() {
		var g VisibilityGrant
		if err := rows.Scan(&g.PrincipalType, &g.PrincipalID); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

// CanEditCollection reports whether the user may modify the collection:
// admin, creator, or an editor principal.
func (db *DB) CanEditCollection(userID string, isAdmin bool, collectionID string) (bool, error) {
	if isAdmin {
		var one int
		err := db.sql.QueryRow(`SELECT 1 FROM collections WHERE id = ?`, collectionID).Scan(&one)
		return err == nil, nil
	}
	var one int
	err := db.sql.QueryRow(`SELECT 1 FROM collections WHERE id = ? AND (
		creator_id = ?
		OR id IN (SELECT collection_id FROM collection_editors
		          WHERE principal_type='user' AND principal_id = ?)
		OR id IN (SELECT collection_id FROM collection_editors
		          WHERE principal_type='group'
		            AND principal_id IN (SELECT group_id FROM group_members WHERE user_id = ?))
	)`, collectionID, userID, userID, userID).Scan(&one)
	if err != nil {
		return false, nil
	}
	return true, nil
}
```

Add `"strings"` to the import block.

- [ ] **Step 4: Extend `DeleteGroup`**

In `server/internal/store/groups.go`, add two statements alongside the two that are already there, and update the existing doc comment's single line so it still reads true:

```go
func (db *DB) DeleteGroup(id string) error {
	db.sql.Exec(`DELETE FROM tool_visibility WHERE principal_type='group' AND principal_id = ?`, id)
	db.sql.Exec(`DELETE FROM credential_access WHERE principal_type='group' AND principal_id = ?`, id)
	db.sql.Exec(`DELETE FROM collection_editors WHERE principal_type='group' AND principal_id = ?`, id)
	db.sql.Exec(`DELETE FROM collection_visibility WHERE principal_type='group' AND principal_id = ?`, id)
	return db.exec1(`DELETE FROM groups WHERE id = ?`, id)
}
```

- [ ] **Step 5: Run the tests**

Run: `go test ./server/internal/store/ -run 'TestCollection|TestDeleteGroup' -v`
Expected: PASS, all four tests.

- [ ] **Step 6: Commit**

```bash
git add server/internal/store/collections.go server/internal/store/collections_test.go server/internal/store/groups.go
git commit -m "feat: add collection membership and grant store"
```

---

## Task 3: Retire `tools.category`

**Files:**
- Modify: `server/internal/store/migrations/0001_init.sql`
- Modify: `server/internal/store/tools.go`
- Modify: `server/tool_handlers_test.go:158-161`
- Modify: `contracts/contracts.go:102`

**Interfaces:**
- Consumes: `SetToolCollections`, `CollectionsForTools` from Task 2.
- Produces:
  - `store.Tool` no longer has `Category`.
  - `store.ToolFilter` has `CollectionID string` in place of `Category string`.
  - `contracts.ToolDTO` has `Collections []CollectionRef` in place of `Category string`, where `type CollectionRef struct { ID, Name, IconURL string }` with json tags `id`, `name`, `icon_url`.

- [ ] **Step 1: Update the schema**

In `0001_init.sql`, delete the `category TEXT NOT NULL DEFAULT '',` line from `CREATE TABLE tools` and delete the `CREATE INDEX idx_tools_category ON tools(category);` line.

- [ ] **Step 2: Run the store tests and watch them fail**

Run: `go test ./server/internal/store/ 2>&1 | head -30`
Expected: failures from SQL referencing a `category` column that no longer exists.

- [ ] **Step 3: Strip `Category` from the store**

In `server/internal/store/tools.go`:
- Remove `Category string` from `Tool`.
- In `ToolFilter`, replace `Category string` with `CollectionID string`.
- Remove `category` from the `INSERT` column list, the `VALUES` placeholders and the argument list in `CreateTool`.
- Remove `category=?` and its argument from `UpdateTool`.
- Remove `&t.Category` from `scanTool`'s `Scan` call and `category,` from `toolSelect`.
- In `ListToolsVisibleTo` and `ListPublicTools`, replace the `f.Category` block with:

```go
	if f.CollectionID != "" {
		where = append(where, `id IN (SELECT tool_id FROM collection_tools WHERE collection_id = ?)`)
		args = append(args, f.CollectionID)
	}
```

- [ ] **Step 4: Update the contract type**

In `contracts/contracts.go`, replace `Category string \`json:"category"\`` on `ToolDTO` with:

```go
	Collections []CollectionRef `json:"collections"`
```

and add, next to `ToolDTO`:

```go
// CollectionRef is the compact collection shape embedded in a tool payload.
type CollectionRef struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	IconURL string `json:"icon_url"`
}
```

- [ ] **Step 5: Fix the compile errors this causes**

Run `go build ./... 2>&1 | head -30` and repair each site. Expect `server/tool_handlers.go` (handled properly in Task 5 — for now delete the `Category` lines from `toolInput`, `publicToolResponse`, `toolToResponse`, `handleCreateTool`, `handleUpdateTool`, and change the two `Category: r.URL.Query().Get("category")` filter lines to `CollectionID: r.URL.Query().Get("collection")`).

In `server/tool_handlers_test.go`, the `?category=database` assertion around line 158 becomes a `?collection=` assertion. Rewrite that block to create a collection, put the tool in it, then filter:

```go
	c := ts.mustCreateCollection(t, c /* admin client */, "Databases")
	ts.mustAddCollectionTool(t, c /* admin client */, c.ID, dbToolID)
	_, data = ts.do(t, c, http.MethodGet, "/api/v1/tools?collection="+c.ID, nil, nil)
```

If the test harness in `server/api_v1_test.go` has no such helpers yet, inline plain `ts.do` calls against `POST /api/v1/collections` and `PUT /api/v1/collections/{id}/tools/{toolId}` instead — those routes land in Task 4, so if you are running Task 3 before Task 4, temporarily assert on the unfiltered list and leave a failing-by-design `t.Skip("collection filter covered in Task 5")`, then remove the skip in Task 5. Do not leave a skip behind at the end of Task 5.

- [ ] **Step 6: Run the whole Go suite**

Run: `make gate`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add server/internal/store/migrations/0001_init.sql server/internal/store/tools.go server/tool_handlers.go server/tool_handlers_test.go contracts/contracts.go
git commit -m "refactor: replace tool category with collections"
```

---

## Task 4: Collection HTTP handlers, icons and routes

**Files:**
- Create: `server/collection_handlers.go`
- Create: `server/principal_handlers.go`
- Modify: `server/app.go:95-115` (catalog route block)
- Modify: `server/icon_handlers.go` (append a collection icon section)
- Test: `server/collection_handlers_test.go`

**Interfaces:**
- Consumes: Task 1 and 2 store methods; `writeJSON`, `writeError`, `decodeJSON`, `iconURL`, `readImageUpload`, `writeImageFile`, `a.iconDir()`, `rbac.FromContext`, `rbac.RequireAuth` — all existing in `server/`.
- Produces: the route family below, and `type collectionView struct` with fields `ID`, `Name`, `Description`, `Visibility`, `CreatorID`, `IconURL`, `ToolCount`, `CanEdit`, `CreatedAt`, plus `ToolIDs []string` on the single-collection response.

Read `server/group_handlers.go` and `server/tool_visibility_handlers.go` before writing this file. Match their error-response vocabulary exactly (`not_found`, `forbidden`, `bad_request`, `invalid_name`, `name_taken`, `internal`).

- [ ] **Step 1: Write the failing handler tests**

Create `server/collection_handlers_test.go`. Open `server/group_handlers_test.go` first and reuse its harness verbatim — the same `newTestServer`, the same client/session helpers, the same `ts.do` signature. Do not invent a new harness.

```go
package main

import (
	"net/http"
	"testing"
)

func TestCollectionCreateAndEditPermissions(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.adminClient(t)
	owner := ts.userClient(t, "owner@example.com")
	stranger := ts.userClient(t, "stranger@example.com")

	// Any signed-in user can create.
	status, body := ts.do(t, owner, http.MethodPost, "/api/v1/collections",
		map[string]any{"name": "Manufacturing", "description": "Shop floor."}, nil)
	if status != http.StatusCreated {
		t.Fatalf("create by basic user = %d, want 201: %s", status, body)
	}
	id := jsonString(t, body, "id")

	// A stranger cannot edit.
	status, _ = ts.do(t, stranger, http.MethodPatch, "/api/v1/collections/"+id,
		map[string]any{"name": "Hijacked"}, nil)
	if status != http.StatusForbidden {
		t.Errorf("stranger PATCH = %d, want 403", status)
	}

	// The creator can.
	status, _ = ts.do(t, owner, http.MethodPatch, "/api/v1/collections/"+id,
		map[string]any{"name": "Manufacturing", "description": "Updated."}, nil)
	if status != http.StatusOK {
		t.Errorf("creator PATCH = %d, want 200", status)
	}

	// An added editor can.
	strangerID := ts.userID(t, "stranger@example.com")
	status, _ = ts.do(t, owner, http.MethodPut,
		"/api/v1/collections/"+id+"/editors/user/"+strangerID, nil, nil)
	if status != http.StatusNoContent {
		t.Fatalf("add editor = %d, want 204", status)
	}
	status, _ = ts.do(t, stranger, http.MethodPatch, "/api/v1/collections/"+id,
		map[string]any{"name": "Manufacturing"}, nil)
	if status != http.StatusOK {
		t.Errorf("editor PATCH = %d, want 200", status)
	}

	// An admin can.
	status, _ = ts.do(t, admin, http.MethodPatch, "/api/v1/collections/"+id,
		map[string]any{"name": "Manufacturing"}, nil)
	if status != http.StatusOK {
		t.Errorf("admin PATCH = %d, want 200", status)
	}

	// An editor who is not the creator cannot delete.
	status, _ = ts.do(t, stranger, http.MethodDelete, "/api/v1/collections/"+id, nil, nil)
	if status != http.StatusForbidden {
		t.Errorf("editor DELETE = %d, want 403", status)
	}
	// The creator can.
	status, _ = ts.do(t, owner, http.MethodDelete, "/api/v1/collections/"+id, nil, nil)
	if status != http.StatusNoContent {
		t.Errorf("creator DELETE = %d, want 204", status)
	}
}

func TestCollectionVisibilityOverHTTP(t *testing.T) {
	ts := newTestServer(t)
	owner := ts.userClient(t, "owner@example.com")
	stranger := ts.userClient(t, "stranger@example.com")
	anon := ts.anonClient(t)

	_, body := ts.do(t, owner, http.MethodPost, "/api/v1/collections",
		map[string]any{"name": "Secret", "visibility": "restricted"}, nil)
	id := jsonString(t, body, "id")

	status, _ := ts.do(t, stranger, http.MethodGet, "/api/v1/collections/"+id, nil, nil)
	if status != http.StatusNotFound {
		t.Errorf("invisible GET = %d, want 404 so the name does not leak", status)
	}

	_, body = ts.do(t, anon, http.MethodGet, "/api/v1/public/collections", nil, nil)
	if containsName(t, body, "Secret") {
		t.Error("public collection list leaked a restricted collection")
	}

	_, body = ts.do(t, stranger, http.MethodGet, "/api/v1/collections", nil, nil)
	if containsName(t, body, "Secret") {
		t.Error("authenticated list leaked a collection the caller cannot see")
	}
}

func TestToolPayloadCollections(t *testing.T) {
	ts := newTestServer(t)
	owner := ts.userClient(t, "owner@example.com")
	stranger := ts.userClient(t, "stranger@example.com")

	_, body := ts.do(t, owner, http.MethodPost, "/api/v1/collections",
		map[string]any{"name": "Hidden", "visibility": "restricted"}, nil)
	hidden := jsonString(t, body, "id")
	_, body = ts.do(t, owner, http.MethodPost, "/api/v1/collections",
		map[string]any{"name": "Shown"}, nil)
	shown := jsonString(t, body, "id")

	status, body := ts.do(t, owner, http.MethodPost, "/api/v1/tools",
		map[string]any{"name": "grafana", "collection_ids": []string{hidden, shown}}, nil)
	if status != http.StatusCreated {
		t.Fatalf("create tool = %d: %s", status, body)
	}
	toolID := jsonString(t, body, "id")

	_, body = ts.do(t, owner, http.MethodGet, "/api/v1/tools/"+toolID, nil, nil)
	if !containsName(t, body, "Hidden") || !containsName(t, body, "Shown") {
		t.Error("creator's tool payload is missing a collection they can see")
	}

	_, body = ts.do(t, stranger, http.MethodGet, "/api/v1/tools/"+toolID, nil, nil)
	if containsName(t, body, "Hidden") {
		t.Error("tool payload leaked a collection the caller cannot see")
	}
	if !containsName(t, body, "Shown") {
		t.Error("tool payload dropped a public collection")
	}

	status, _ = ts.do(t, owner, http.MethodPost, "/api/v1/tools",
		map[string]any{"name": "bogus", "collection_ids": []string{"no-such-id"}}, nil)
	if status != http.StatusBadRequest {
		t.Errorf("unknown collection id = %d, want 400", status)
	}
}
```

`jsonString(t, body, key)` and `containsName(t, body, name)` are small helpers: unmarshal into `map[string]any` / `[]map[string]any` and read the field. If the harness already has equivalents, use those instead; otherwise add these two at the bottom of the file. `ts.anonClient` and `ts.userID` likewise — check `server/api_v1_test.go` for what already exists before adding anything.

- [ ] **Step 2: Run and confirm failure**

Run: `go test ./server/ -run TestCollection -v`
Expected: 404s everywhere, because the routes do not exist.

- [ ] **Step 3: Write `collection_handlers.go`**

Shape (fill in each handler following the existing `group_handlers.go` style):

```go
package main

import (
	"net/http"
	"strings"

	"github.com/thehelvijs/Reeve/server/internal/auth"
	"github.com/thehelvijs/Reeve/server/internal/rbac"
	"github.com/thehelvijs/Reeve/server/internal/store"
)

type collectionView struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Visibility  string `json:"visibility"`
	CreatorID   string `json:"creator_id"`
	IconURL     string `json:"icon_url"`
	ToolCount   int    `json:"tool_count"`
	CanEdit     bool   `json:"can_edit"`
	CreatedAt   string `json:"created_at"`
}

type collectionDetailView struct {
	collectionView
	ToolIDs []string `json:"tool_ids"`
}

type collectionInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Visibility  string `json:"visibility"`
}
```

Handlers to write:

| Handler | Behaviour |
| --- | --- |
| `handleListPublicCollections` | `db.ListPublicCollections()`, `CanEdit` always false, always a `[]` not `null` |
| `handleListCollections` | `db.ListCollectionsVisibleTo(p.UserID, p.IsAdmin())`, `CanEdit` per row via `db.CanEditCollection` |
| `handleCreateCollection` | any authed user; trim name; empty name → 400 `invalid_name`; validate visibility with the existing `validVisibility`; duplicate name → 409 `name_taken`; 201 with the view |
| `handleGetCollection` | `loadVisibleCollection`; returns `collectionDetailView` with `ToolIDs` |
| `handleUpdateCollection` | `loadEditableCollection`; same validation as create; 200 with the view |
| `handleDeleteCollection` | `loadVisibleCollection`, then admin-or-creator only, else 403; 204 |
| `handleAddCollectionTool` / `handleRemoveCollectionTool` | `loadEditableCollection`, verify the tool exists (404 if not), 204 |
| `handleListCollectionEditors` / `Add` / `Remove` | `loadEditableCollection`; validate `{type}` is `user` or `group` (400 `bad_request` otherwise); 204 on write |
| `handleListCollectionVisibility` / `Add` / `Remove` | same |

Two loaders, mirroring `loadVisibleTool` / `loadEditableTool` in `server/tool_handlers.go`:

```go
// loadVisibleCollection fetches the collection and 404s if the principal may
// not see it, so a restricted name never leaks.
func (a *app) loadVisibleCollection(w http.ResponseWriter, r *http.Request, p auth.Principal) (store.Collection, bool) {
	id := r.PathValue("id")
	ok, _ := a.db.CanSeeCollection(p.UserID, p.IsAdmin(), id)
	if !ok {
		writeError(w, http.StatusNotFound, "not_found", "collection not found")
		return store.Collection{}, false
	}
	c, err := a.db.GetCollection(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "collection not found")
		return store.Collection{}, false
	}
	return c, true
}

// loadEditableCollection additionally requires admin, creator or editor rights.
func (a *app) loadEditableCollection(w http.ResponseWriter, r *http.Request, p auth.Principal) (store.Collection, bool) {
	c, ok := a.loadVisibleCollection(w, r, p)
	if !ok {
		return store.Collection{}, false
	}
	can, _ := a.db.CanEditCollection(p.UserID, p.IsAdmin(), c.ID)
	if !can {
		writeError(w, http.StatusForbidden, "forbidden", "only an editor, the creator or an admin can modify this collection")
		return store.Collection{}, false
	}
	return c, true
}
```

- [ ] **Step 4: Add collection icons**

Append a `// --- collection icons ---` section to `server/icon_handlers.go`, copying the tool-icon trio and swapping the permission check:

- `handleUploadCollectionIcon` / `handleDeleteCollectionIcon` gate on `a.db.CanEditCollection`, store under base `"collection-"+id`, and call `a.db.SetCollectionIconPath`.
- `handleServeCollectionIcon` is unauthenticated: serve when `c.Visibility == store.VisibilityPublic`, or when a principal is present and `a.db.CanSeeCollection` returns true. Otherwise 404. Set `Cache-Control: no-store` like the others.
- The URL helper call is `iconURL("collections", id, path)`. Check `iconURL`'s implementation and confirm the first argument is the URL path segment; if it is not that shape, follow whatever it actually does.

- [ ] **Step 5: Write `principal_handlers.go`**

```go
package main

import "net/http"

type principalsView struct {
	Users  []principalUser  `json:"users"`
	Groups []principalGroup `json:"groups"`
}

type principalUser struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
}

type principalGroup struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// handleListPrincipals gives any signed-in user the names needed to fill a
// visibility or editor picker; it exposes nothing beyond name and email.
func (a *app) handleListPrincipals(w http.ResponseWriter, _ *http.Request) {
	users, err := a.db.ListUsers()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not list principals")
		return
	}
	groups, err := a.db.ListGroups()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not list principals")
		return
	}
	out := principalsView{Users: []principalUser{}, Groups: []principalGroup{}}
	for _, u := range users {
		out.Users = append(out.Users, principalUser{ID: u.ID, DisplayName: u.DisplayName, Email: u.Email})
	}
	for _, g := range groups {
		out.Groups = append(out.Groups, principalGroup{ID: g.ID, Name: g.Name})
	}
	writeJSON(w, http.StatusOK, out)
}
```

Confirm `db.ListUsers()`'s real name and signature in `server/internal/store/users.go` and adapt.

- [ ] **Step 6: Register the routes**

In `server/app.go`, add a `// Collections.` block after the catalog block. The unauthenticated ones go up with the other public routes.

```go
	mux.HandleFunc("GET /api/v1/public/collections", a.handleListPublicCollections)
	mux.HandleFunc("GET /api/v1/collections/{id}/icon", a.handleServeCollectionIcon)
```

```go
	// Collections.
	mux.Handle("GET /api/v1/collections", authed(http.HandlerFunc(a.handleListCollections)))
	mux.Handle("POST /api/v1/collections", authed(http.HandlerFunc(a.handleCreateCollection)))
	mux.Handle("GET /api/v1/collections/{id}", authed(http.HandlerFunc(a.handleGetCollection)))
	mux.Handle("PATCH /api/v1/collections/{id}", authed(http.HandlerFunc(a.handleUpdateCollection)))
	mux.Handle("DELETE /api/v1/collections/{id}", authed(http.HandlerFunc(a.handleDeleteCollection)))
	mux.Handle("POST /api/v1/collections/{id}/icon", authed(http.HandlerFunc(a.handleUploadCollectionIcon)))
	mux.Handle("DELETE /api/v1/collections/{id}/icon", authed(http.HandlerFunc(a.handleDeleteCollectionIcon)))
	mux.Handle("PUT /api/v1/collections/{id}/tools/{toolId}", authed(http.HandlerFunc(a.handleAddCollectionTool)))
	mux.Handle("DELETE /api/v1/collections/{id}/tools/{toolId}", authed(http.HandlerFunc(a.handleRemoveCollectionTool)))
	mux.Handle("GET /api/v1/collections/{id}/editors", authed(http.HandlerFunc(a.handleListCollectionEditors)))
	mux.Handle("PUT /api/v1/collections/{id}/editors/{ptype}/{pid}", authed(http.HandlerFunc(a.handleAddCollectionEditor)))
	mux.Handle("DELETE /api/v1/collections/{id}/editors/{ptype}/{pid}", authed(http.HandlerFunc(a.handleRemoveCollectionEditor)))
	mux.Handle("GET /api/v1/collections/{id}/visibility", authed(http.HandlerFunc(a.handleListCollectionVisibility)))
	mux.Handle("PUT /api/v1/collections/{id}/visibility/{ptype}/{pid}", authed(http.HandlerFunc(a.handleAddCollectionVisibility)))
	mux.Handle("DELETE /api/v1/collections/{id}/visibility/{ptype}/{pid}", authed(http.HandlerFunc(a.handleRemoveCollectionVisibility)))
	mux.Handle("GET /api/v1/principals", authed(http.HandlerFunc(a.handleListPrincipals)))
```

`GET /api/v1/collections/{id}/icon` is unauthenticated but `GET /api/v1/collections/{id}` is not; Go's `ServeMux` treats these as distinct patterns, so both register cleanly.

- [ ] **Step 7: Run the tests**

Run: `go test ./server/ -run 'TestCollection|TestToolPayload' -v`
Expected: `TestCollectionCreateAndEditPermissions` and `TestCollectionVisibilityOverHTTP` PASS. `TestToolPayloadCollections` still fails — `collection_ids` lands in Task 5.

- [ ] **Step 8: Commit**

```bash
git add server/collection_handlers.go server/principal_handlers.go server/icon_handlers.go server/app.go server/collection_handlers_test.go
git commit -m "feat: add collection API routes"
```

---

## Task 5: Wire collections into the tool payload

**Files:**
- Modify: `server/tool_handlers.go`
- Test: `server/collection_handlers_test.go` (`TestToolPayloadCollections` from Task 4)
- Test: `server/tool_handlers_test.go` (remove the Task 3 skip, assert `?collection=`)

**Interfaces:**
- Consumes: `db.CollectionsForTools`, `db.SetToolCollections`, `db.CanSeeCollection`, `contracts.CollectionRef`.
- Produces: `collection_ids []string` accepted on tool create and update; `collections` present on `toolResponse` and `publicToolResponse`.

- [ ] **Step 1: Add the input field**

In `toolInput`, add:

```go
	CollectionIDs []string `json:"collection_ids"`
```

- [ ] **Step 2: Add a filtering helper**

```go
// visibleCollectionRefs maps tool ids to the collections each caller may see.
// A nil principal means anonymous, which sees public collections only.
func (a *app) visibleCollectionRefs(toolIDs []string, p *auth.Principal) map[string][]contracts.CollectionRef {
	out := map[string][]contracts.CollectionRef{}
	byTool, err := a.db.CollectionsForTools(toolIDs)
	if err != nil {
		return out
	}
	for toolID, cols := range byTool {
		refs := []contracts.CollectionRef{}
		for _, c := range cols {
			visible := c.Visibility == store.VisibilityPublic
			if !visible && p != nil {
				visible, _ = a.db.CanSeeCollection(p.UserID, p.IsAdmin(), c.ID)
			}
			if !visible {
				continue
			}
			refs = append(refs, contracts.CollectionRef{
				ID: c.ID, Name: c.Name, IconURL: iconURL("collections", c.ID, c.IconPath),
			})
		}
		out[toolID] = refs
	}
	return out
}
```

- [ ] **Step 3: Populate the responses**

`toolToResponse` cannot reach the DB, so fill `Collections` at the call sites:

- `handleListTools`: build `ids` from the tools, call `visibleCollectionRefs(ids, &p)` once, then assign `tr.Collections = refs[t.ID]` inside the loop; when the map has no entry use `[]contracts.CollectionRef{}` so the JSON is `[]` and not `null`.
- `handleListPublicTools`: same with `nil` for the principal, assigning to the new `Collections []contracts.CollectionRef \`json:"collections"\`` field on `publicToolResponse`.
- `handleGetTool`, `handleCreateTool`, `handleUpdateTool`: single-tool calls with `&p`.

One batched `CollectionsForTools` call per list request. Do not call it per tool.

- [ ] **Step 4: Validate and persist `collection_ids`**

Add to `server/tool_handlers.go`:

```go
// applyCollectionIDs replaces a tool's collection membership, rejecting ids the
// caller cannot see.
func (a *app) applyCollectionIDs(w http.ResponseWriter, toolID string, ids []string, p auth.Principal) bool {
	for _, id := range ids {
		ok, _ := a.db.CanSeeCollection(p.UserID, p.IsAdmin(), id)
		if !ok {
			writeError(w, http.StatusBadRequest, "invalid_collection", "unknown collection id")
			return false
		}
	}
	if err := a.db.SetToolCollections(toolID, ids); err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not set collections")
		return false
	}
	return true
}
```

Call it in `handleCreateTool` right after `CreateTool` succeeds, and in `handleUpdateTool` right after `UpdateTool` succeeds, in both cases before writing the response. On create, if it fails, delete the just-created tool so a rejected request leaves nothing behind.

- [ ] **Step 5: Run the tests**

Run: `go test ./server/ -run 'TestCollection|TestToolPayload|TestTool' -v`
Expected: PASS, including `TestToolPayloadCollections`. Remove any `t.Skip` left from Task 3 and make the `?collection=` assertion real.

- [ ] **Step 6: Full gate**

Run: `make gate`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add server/tool_handlers.go server/tool_handlers_test.go server/collection_handlers_test.go
git commit -m "feat: expose collections on the tool payload"
```

---

## Task 6: Document the API

**Files:**
- Modify: `contracts/API.md`

- [ ] **Step 1: Update the tool sections**

Find every `category` mention (`contracts/API.md:31`, `:37`, `:60`, `:63` at time of writing) and replace:
- Query param `category` → `collection` (an id, not a name), on both `GET /api/v1/tools` and `GET /api/v1/public/tools`.
- The example body's `"category": "metrics"` → `"collections": [{"id": "…", "name": "Metrics", "icon_url": "…"}]`.
- The field list for the public tool shape: drop `category`, add `collections`.
- Note that `POST`/`PATCH` accept `collection_ids`, an array that replaces the whole membership set, and that an id the caller cannot see is a `400 invalid_collection`.

- [ ] **Step 2: Add the collections section**

Add a `## Collections` section in the file's existing style, listing every route from Task 4 with its method, auth requirement and status codes, the `collectionView` JSON example, and one paragraph stating the rule: a collection's visibility gates the collection only; a service's own visibility decides whether it is listed, and a service appears under every collection the viewer can see, or under Ungrouped.

Also document `GET /api/v1/principals` and say plainly that it exposes display name, email and group name to any signed-in user.

- [ ] **Step 3: Check nothing else references the old field**

Run: `grep -rn "category" --include='*.go' --include='*.md' --include='*.ts' --include='*.tsx' server/ contracts/ web/src/ web/e2e/ README.md`
Expected: no hits outside `docs/superpowers/`. Fix any that remain.

- [ ] **Step 4: Commit**

```bash
git add contracts/API.md
git commit -m "docs: document the collections API"
```

---

## Task 7: Web types and API surface

**Files:**
- Modify: `web/src/api.ts`

**Interfaces:**
- Produces, for every later web task:

```ts
export interface CollectionRef {
  id: string;
  name: string;
  icon_url: string;
}

export interface Collection {
  id: string;
  name: string;
  description: string;
  visibility: 'public' | 'restricted';
  creator_id: string;
  icon_url: string;
  tool_count: number;
  can_edit: boolean;
  created_at: string;
}

export interface CollectionDetail extends Collection {
  tool_ids: string[];
}

export interface Principals {
  users: { id: string; display_name: string; email: string }[];
  groups: { id: string; name: string }[];
}
```

- [ ] **Step 1: Add the types**

Add the four interfaces above next to the existing `Group` interface. On `Tool`, replace `category: string;` with `collections: CollectionRef[];`.

- [ ] **Step 2: Typecheck and watch it fail**

Run: `cd web && npx tsc --noEmit`
Expected: errors in `Catalog.tsx`, `Portal.tsx`, `ToolDetail.tsx`, `ToolInfoModal.tsx`, `ToolFormPage.tsx` — every `t.category` reader. Tasks 9 and 11 fix them.

- [ ] **Step 3: Commit**

```bash
git add web/src/api.ts
git commit -m "feat: add collection types to the web api client"
```

Commit with the typecheck red is acceptable **only here**, because the type change is what drives Tasks 9 and 11. If you are running the web tasks as one sequence, fold this commit into Task 9 instead so no commit leaves the tree broken.

---

## Task 8: Group tools by collection

**Files:**
- Modify: `web/src/lib/group.ts`

**Interfaces:**
- Produces:

```ts
export interface CollectionGroup {
  collection: CollectionRef | null;
  tools: Tool[];
}
export function groupToolsByCollection(tools: Tool[]): CollectionGroup[]
```

- [ ] **Step 1: Implement it**

Append to `web/src/lib/group.ts`:

```ts
// groupToolsByCollection buckets tools under each collection they carry, with
// tools in no visible collection collected under a null header sorted last.
export function groupToolsByCollection(tools: Tool[]): CollectionGroup[] {
  const byCollection = new Map<string, { collection: CollectionRef; tools: Tool[] }>();
  const ungrouped: Tool[] = [];
  for (const t of tools) {
    if (t.collections.length === 0) {
      ungrouped.push(t);
      continue;
    }
    for (const c of t.collections) {
      const entry = byCollection.get(c.id);
      if (entry) {
        entry.tools.push(t);
      } else {
        byCollection.set(c.id, { collection: c, tools: [t] });
      }
    }
  }
  const groups: CollectionGroup[] = Array.from(byCollection.values()).sort((a, b) =>
    a.collection.name.localeCompare(b.collection.name),
  );
  if (ungrouped.length > 0) {
    groups.push({ collection: null, tools: ungrouped });
  }
  return groups;
}
```

Import `CollectionRef` alongside the existing `Host, Tool` import.

A tool in two visible collections appears under both. That is deliberate.

- [ ] **Step 2: Typecheck**

Run: `cd web && npx tsc --noEmit 2>&1 | grep group.ts`
Expected: no output.

- [ ] **Step 3: Commit**

```bash
git add web/src/lib/group.ts
git commit -m "feat: group tools by collection"
```

---

## Task 9: Portal group-by toggle

**Files:**
- Modify: `web/src/pages/Portal.tsx`
- Create: `web/src/components/CollectionInfoModal.tsx`

**Interfaces:**
- Consumes: `groupToolsByCollection` (Task 8), `Collection`/`CollectionRef` (Task 7), `GET /api/v1/public/collections`.

- [ ] **Step 1: Build `CollectionInfoModal`**

Read `web/src/components/HostInfoModal.tsx` and mirror it exactly: same `Modal` wrapper, same prop shape, same spacing. Props:

```tsx
export default function CollectionInfoModal({
  collection,
  tools,
  onClose,
  onOpenTool,
}: {
  collection: CollectionRef & { description?: string };
  tools: Tool[];
  onClose: () => void;
  onOpenTool: (id: string) => void;
})
```

Body: `EntityIcon` at 40px, the name as the title, the description as muted body text when present, then the tools as a clickable list reusing whatever row markup `HostInfoModal` uses.

- [ ] **Step 2: Add the toggle to `Portal.tsx`**

- Add `type GroupBy = 'collection' | 'host';`
- Read the initial value from the URL: `useSearchParams()`, `?by=host` → `'host'`, anything else → `'collection'`. Writing the state back to the URL uses `setSearchParams`, and `by=collection` is omitted from the URL since it is the default.
- Render a second segmented control to the right of the existing `list / map / graph` one, only when `view === 'list'`, using the same markup and classes as the existing control with the labels `Collection` and `Host`.
- Replace `const groups = useMemo(() => groupToolsByHost(filtered, hosts), ...)` with a branch on `groupBy` — an explicit `if`/`else`, not a ternary — producing a discriminated union the JSX renders as either the existing host heading or a new collection heading.
- The collection heading matches the existing host heading: `EntityIcon` at 22px, the name, the count in `text-xs text-muted`. No status dot (a collection has no status). Clicking it opens `CollectionInfoModal`. The `Ungrouped` heading is plain text with no button and sorts last.
- Anonymous visitors get collection descriptions from `GET /api/v1/public/collections`; signed-in visitors from `GET /api/v1/collections`. Fetch that list in the same `useEffect` that already loads tools and hosts, on the same 15s poll, and key it by id to enrich the `CollectionRef` embedded in each tool.
- The `t.category && <Pill>` block on the tool card becomes `t.collections.map((c) => <Pill key={c.id}>{c.name}</Pill>)`.

- [ ] **Step 3: Verify by hand**

Run the dev stack: `cd web && npm run dev` in one shell, `REEVE_MASTER_KEY=AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA= REEVE_DB=/tmp/reeve-dev.db REEVE_ADDR=127.0.0.1:8080 go run ./server` in another. Open `http://127.0.0.1:5173/`, confirm the portal defaults to collection grouping, the Host toggle switches it and puts `?by=host` in the address bar, and a reload on that URL keeps host grouping. Kill both processes when done.

- [ ] **Step 4: Check**

Run: `make web-check`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add web/src/pages/Portal.tsx web/src/components/CollectionInfoModal.tsx
git commit -m "feat: group the portal by collection"
```

---

## Task 10: Collections pages and editor

**Files:**
- Create: `web/src/pages/Collections.tsx`
- Create: `web/src/pages/CollectionDetail.tsx`
- Create: `web/src/components/CollectionEditor.tsx`
- Create: `web/src/components/PrincipalPicker.tsx`
- Modify: `web/src/App.tsx`
- Modify: `web/src/components/Layout.tsx`
- Modify: `web/src/components/NavIcon.tsx`

**Interfaces:**
- Consumes: Task 7 types, Task 4 routes.
- Produces: routes `/collections` and `/collections/:id`.

- [ ] **Step 1: Add the nav icon**

In `web/src/components/NavIcon.tsx`, add `| 'collections'` to `IconName` and an entry to `paths`. A stacked-layers glyph in the same 16px, stroke-only style as its neighbours:

```tsx
  collections: (
    <>
      <path d="M8 2 14 5l-6 3-6-3 6-3Z" />
      <rect x="3.5" y="9.5" width="9" height="4" rx="1" />
    </>
  ),
```

- [ ] **Step 2: Add the nav entry and relabel admin Groups**

In `web/src/components/Layout.tsx`, insert into `primaryNav` between Services and Hosts:

```ts
  { to: '/collections', label: 'Collections', icon: 'collections', end: false },
```

and change the admin entry's label from `'Groups'` to `'User groups'`. Leave its route and icon alone.

- [ ] **Step 3: Build `PrincipalPicker`**

```tsx
export default function PrincipalPicker({
  grants,
  onAdd,
  onRemove,
}: {
  grants: VisibilityGrant[];
  onAdd: (type: 'user' | 'group', id: string) => void;
  onRemove: (type: 'user' | 'group', id: string) => void;
})
```

It fetches `/api/v1/principals` once on mount, renders the current grants as removable pills (copy the pill markup from `AdminGroups.tsx`'s member list) and a `<select>` of the principals not yet granted, labelled `Add user…` / `Add group…`. `VisibilityGrant` is `{ principal_type: 'user' | 'group'; principal_id: string }` — check `web/src/pages/ToolDetail.tsx` for the existing declaration and export it from `api.ts` rather than declaring a second one.

- [ ] **Step 4: Build `CollectionEditor`**

A `Modal` (size `lg`) used for both create and edit, taking `collection?: CollectionDetail`. Fields:
- Name (`Input`, required, trimmed).
- Description (a `<textarea>` styled like `Input`; three rows).
- Avatar: `IconUploader` with `path={`/api/v1/collections/${id}/icon`}`. On create the collection does not exist yet, so hide the uploader until the first save and show it on reopen.
- Visibility: a `Select` of `public` / `restricted`; when `restricted`, render a `PrincipalPicker` over `/api/v1/collections/{id}/visibility`.
- Editors: a `PrincipalPicker` over `/api/v1/collections/{id}/editors`, always shown.
- Services: a scrollable checkbox list of every tool from `GET /api/v1/tools`, diffed against `tool_ids` on save via `PUT`/`DELETE /api/v1/collections/{id}/tools/{toolId}`.

Save posts to `/api/v1/collections` or patches `/api/v1/collections/{id}`, then calls `onSaved(collection)`. Errors render in an `ErrorText`. One primary `Button` per the design system; Cancel is `variant="secondary"`.

- [ ] **Step 5: Build `Collections.tsx`**

`PageHeader` titled `Collections`, subtitle `Group services so the portal reads by team, not by machine.`, action `New collection` (available to every signed-in user). Body: a `grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3` of `Card`s, each with `EntityIcon` at 36px, name, truncated description, `{tool_count} services`, and a `<Pill tone="down">restricted</Pill>` when applicable. Clicking a card navigates to `/collections/{id}`. Empty state uses `EmptyState` with the same copy pattern as `AdminGroups.tsx`.

- [ ] **Step 6: Build `CollectionDetail.tsx`**

`BackLink` to `/collections`, then a header with `EntityIcon` at 48px, the name, the description, the count, an `Edit` button when `can_edit`, and a `Delete` button (`variant="danger"`, `window.confirm` first) when `can_edit` and the viewer is the creator or an admin. Below, the member services as the same card grid `Catalog.tsx` uses — read that file and reuse its card markup rather than writing a third variant.

- [ ] **Step 7: Register the routes**

In `web/src/App.tsx`, inside the `Protected`/`Layout` block next to `/catalog`:

```tsx
        <Route path="/collections" element={<Collections />} />
        <Route path="/collections/:id" element={<CollectionDetail />} />
```

- [ ] **Step 8: Check and eyeball**

Run: `make web-check` — expected PASS.
Then run the dev stack (Task 9 Step 3), sign in, create a collection, add a service to it, set it restricted, grant a user, and confirm the portal reflects each change. Kill both processes.

- [ ] **Step 9: Commit**

```bash
git add web/src/pages/Collections.tsx web/src/pages/CollectionDetail.tsx web/src/components/CollectionEditor.tsx web/src/components/PrincipalPicker.tsx web/src/components/NavIcon.tsx web/src/components/Layout.tsx web/src/App.tsx
git commit -m "feat: add collections pages and editor"
```

---

## Task 11: Catalog, tool form, tool detail

**Files:**
- Modify: `web/src/pages/ToolFormPage.tsx`
- Modify: `web/src/pages/Catalog.tsx`
- Modify: `web/src/pages/ToolDetail.tsx:112`
- Modify: `web/src/components/ToolInfoModal.tsx:40`

- [ ] **Step 1: `ToolFormPage` — collections multi-select**

- Drop `category` from the `f` state object, the load effect and the submit body.
- Add `const [collectionIDs, setCollectionIDs] = useState<string[]>([])`, seeded from `t.collections.map((c) => c.id)` when editing.
- Add `collection_ids: collectionIDs` to the submit body.
- In the `Organize` section, replace the `Category` `Field` with a `Collections` `Field` holding a checkbox list of `GET /api/v1/collections` (name + `EntityIcon` at 18px per row, scrollable at `max-h-48`), plus an inline `+ New collection` control: a text input and an Add button that `POST`s `{name}` to `/api/v1/collections`, appends the result to the list and selects it. Show the API error inline on failure (a duplicate name is a `409`).
- The `Tags` field keeps its half of the two-column grid.

- [ ] **Step 2: `Catalog` — collection filter**

- Replace the `category` state with `collectionID`.
- Replace the `useMemo` that derives categories from `tools.map((t) => t.category)` with a fetch of `GET /api/v1/collections` into state.
- The filter `<select>` lists collection names by id; the predicate becomes `t.collections.some((c) => c.id === collectionID)`.
- The card pill block renders one `Pill` per entry in `t.collections`, keyed by `c.id`, keeping the existing `restricted` pill beside them.
- Update the `EmptyState` copy from `"No services match your search or category."` to `"No services match your search or collection."`.

- [ ] **Step 3: `ToolDetail` — collections row**

Replace `<Detail label="Category" value={tool.category} />` with a `Collections` row rendering one `Pill` per collection, each linking to `/collections/{id}`, and an em-dash when the array is empty. Match the existing `Detail` row's label column width.

- [ ] **Step 4: `ToolInfoModal` — pills**

Replace `{tool.category && <Pill>{tool.category}</Pill>}` with `{tool.collections.map((c) => <Pill key={c.id}>{c.name}</Pill>)}`.

- [ ] **Step 5: Check**

Run: `make web-check`
Expected: PASS, and `grep -rn "category" web/src/` returns nothing.

- [ ] **Step 6: Commit**

```bash
git add web/src/pages/ToolFormPage.tsx web/src/pages/Catalog.tsx web/src/pages/ToolDetail.tsx web/src/components/ToolInfoModal.tsx
git commit -m "feat: pick collections when adding a service"
```

---

## Task 12: End-to-end tests

**Files:**
- Create: `web/e2e/collections.spec.ts`
- Modify: `web/e2e/portal.spec.ts` if it asserts on host grouping

**Interfaces:**
- Consumes: everything above.

Read `web/e2e/portal.spec.ts` and `web/e2e/z-admin-creates.spec.ts` first. The suite runs `fullyParallel: false, workers: 1` and the filename prefixes (`z-`, `zz-`) order the files — pick a filename that puts this spec after whatever creates the fixtures it needs.

- [ ] **Step 1: Write the spec**

```ts
import { expect, test } from '@playwright/test';

test.describe('collections', () => {
  test('portal groups by collection and toggles to host', async ({ page }) => {
    await page.goto('/');
    await expect(page.getByRole('button', { name: 'Collection' })).toBeVisible();
    await page.getByRole('button', { name: 'Host' }).click();
    await expect(page).toHaveURL(/by=host/);
    await page.reload();
    await expect(page.getByRole('button', { name: 'Host' })).toHaveClass(/bg-surface-2/);
  });

  test('a restricted collection is absent for an anonymous visitor', async ({ page }) => {
    await page.goto('/');
    await expect(page.getByRole('heading', { name: 'Secret' })).toHaveCount(0);
  });
});
```

Fill in the fixture setup the way the neighbouring specs do — sign in as the admin created by the earlier spec, create a `Secret` restricted collection with a public service in it, sign out, then assert. Assert on user-visible text, not on internal class names, wherever the existing specs manage to.

- [ ] **Step 2: Run**

Run: `cd web && npx playwright test e2e/collections.spec.ts`
Expected: PASS. The config starts its own server and Vite; do not have another copy on 8080 or 5173.

- [ ] **Step 3: Run the whole e2e suite**

Run: `cd web && npx playwright test`
Expected: PASS. Fix any neighbouring spec that asserted on host grouping or on a category field.

- [ ] **Step 4: Commit**

```bash
git add web/e2e/
git commit -m "test: cover collection grouping end to end"
```

---

## Task 13: Full gate and visual sweep

**Files:** none created; fixes land in whichever file the sweep implicates.

- [ ] **Step 1: Run every gate**

```sh
make gate
make web-check
python3 -m pytest scripts/
cd web && npx playwright test
```

All four must pass.

- [ ] **Step 2: Build and run the real binary**

```sh
make web-build && make server-assets && make build
```

Then start it on a fixed port with a throwaway DB, check nothing of the user's is already on that port first.

- [ ] **Step 3: Seed a realistic fixture**

Through the UI: an admin, one basic user, one user group, three collections (`Manufacturing`, `Embedded`, `Ops`) with avatars and descriptions, one of them restricted, and six services spread across them, including one service in two collections and one in none.

- [ ] **Step 4: Screenshot sweep**

Capture and **look at** every one of these. Passing typecheck is not evidence the UI is right.

Anonymous: portal list grouped by collection; portal list grouped by host; portal map; portal graph; a collection info modal open over the list; the portal with a search term active; the portal at 375px width.

Signed in as the basic user: portal; dashboard; services catalog; the collection filter open; a tool detail page; the add-service form with the collections multi-select and the inline create open; `/collections`; a collection detail page; the collection editor modal with the visibility picker showing; `/collections` when the user can see none.

Signed in as the admin: the same set, plus admin users, admin **User groups**, webhooks, alerts, audit, server, settings — the pages this work did not touch, to catch anything a global change broke.

Both themes if the app has a light mode; check `DESIGN.md` and `theme/theme.css` for whether it does.

- [ ] **Step 5: Judge each screenshot**

Against `DESIGN.md`: near-black canvas, hairline borders, the surface ladder, the accent used only for brand / focus / one primary action per view, the three radii. Fix anything off and re-shoot that view.

- [ ] **Step 6: Kill every process you started**

The built binary, any dev server, any Playwright leftovers. Confirm with `pgrep -af 'reeve|vite|go run'`.

- [ ] **Step 7: Commit any fixes**

```bash
git add <the files the sweep implicated>
git commit -m "fix: <what the sweep found>"
```

If the sweep found nothing, there is nothing to commit — say so rather than inventing a commit.

---

## Self-Review Notes

Spec coverage checked: schema (T1), permissions (T1, T2, T4), collection visibility semantics (T1, T2, T4 tests), the full route list (T4), tool payload change (T3, T5), `?collection=` (T3, T5), `/api/v1/principals` (T4), portal toggle (T9), sidebar (T10), `/collections` and `/collections/:id` (T10), editor (T10), `ToolFormPage` multi-select (T11), catalog / detail / info modal (T11), store and handler tests (T1, T2, T4, T5), e2e (T12), screenshot sweep (T13), `API.md` (T6). No eval suite, per the spec — no LLM in this path.

Naming checked across tasks: `Collection`, `CollectionRef`, `CollectionDetail`, `collectionView`, `collectionVisibleClause`, `CanSeeCollection`, `CanEditCollection`, `SetToolCollections`, `CollectionsForTools`, `groupToolsByCollection`, `visibleCollectionRefs`, `applyCollectionIDs` are each defined once and used consistently.

Known coupling: Task 3 breaks `server/tool_handlers.go` compilation until its Step 5 patches it, and Task 7 leaves the web typecheck red until Task 9. Both are called out in place with the fix.
