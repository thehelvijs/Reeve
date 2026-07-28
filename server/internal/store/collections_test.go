package store

import "testing"

func mustUser(t *testing.T, db *DB, email string) string {
	t.Helper()
	u, err := db.CreateUser(email, "hash", RoleBasic)
	if err != nil {
		t.Fatalf("create user %s: %v", email, err)
	}
	return u.ID
}

func TestCollectionCRUD(t *testing.T) {
	db := openTemp(t)
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

func TestCollectionVisibility(t *testing.T) {
	db := openTemp(t)
	owner := mustUser(t, db, "owner@example.com")
	editor := mustUser(t, db, "editor@example.com")
	viewer := mustUser(t, db, "viewer@example.com")
	stranger := mustUser(t, db, "stranger@example.com")
	grp, err := db.CreateGroup("ops")
	if err != nil {
		t.Fatalf("create group: %v", err)
	}
	member := mustUser(t, db, "member@example.com")
	if err := db.AddGroupMember(grp.ID, member, GroupRoleMember); err != nil {
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
	grpEdit, _ := db.CreateCollection(Collection{Name: "GroupEdited", CreatorID: owner, Visibility: VisibilityRestricted})
	if err := db.AddCollectionEditor(grpEdit.ID, "group", grp.ID); err != nil {
		t.Fatalf("add group editor: %v", err)
	}

	cases := []struct {
		name string
		user string
		id   string
		want bool
	}{
		{"public to stranger", stranger, pub.ID, true},
		{"restricted to creator", owner, res.ID, true},
		{"restricted to user editor", editor, res.ID, true},
		{"restricted to granted user", viewer, res.ID, true},
		{"restricted to stranger", stranger, res.ID, false},
		{"group grant to member", member, grpOnly.ID, true},
		{"group grant to non-member", stranger, grpOnly.ID, false},
		{"group editor to member", member, grpEdit.ID, true},
		{"group editor to non-member", stranger, grpEdit.ID, false},
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
	db := openTemp(t)
	owner := mustUser(t, db, "owner@example.com")
	editor := mustUser(t, db, "editor@example.com")
	stranger := mustUser(t, db, "stranger@example.com")
	grp, _ := db.CreateGroup("leads")
	lead := mustUser(t, db, "lead@example.com")
	if err := db.AddGroupMember(grp.ID, lead, GroupRoleMember); err != nil {
		t.Fatalf("add member: %v", err)
	}

	c, _ := db.CreateCollection(Collection{Name: "Manufacturing", CreatorID: owner})
	if err := db.AddCollectionEditor(c.ID, "user", editor); err != nil {
		t.Fatalf("add user editor: %v", err)
	}
	if err := db.AddCollectionEditor(c.ID, "group", grp.ID); err != nil {
		t.Fatalf("add group editor: %v", err)
	}

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

	grants, _ := db.ListCollectionEditors(c.ID)
	if len(grants) != 2 {
		t.Errorf("ListCollectionEditors = %+v, want 2 grants", grants)
	}

	if err := db.RemoveCollectionEditor(c.ID, "user", editor); err != nil {
		t.Fatalf("remove editor: %v", err)
	}
	if got, _ := db.CanEditCollection(editor, false, c.ID); got {
		t.Error("removed editor can still edit")
	}

	if err := db.RemoveCollectionVisibility(c.ID, "user", stranger); err != nil {
		t.Fatalf("remove absent visibility grant: %v", err)
	}
}

func TestCollectionMembership(t *testing.T) {
	db := openTemp(t)
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

	if err := db.AddCollectionTool(c.ID, t1.ID); err != nil {
		t.Fatalf("add tool: %v", err)
	}
	if err := db.AddCollectionTool(c.ID, t1.ID); err != nil {
		t.Fatalf("re-add tool is not idempotent: %v", err)
	}
	if err := db.RemoveCollectionTool(c.ID, t1.ID); err != nil {
		t.Fatalf("remove tool: %v", err)
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

	if err := db.SetToolCollections(t1.ID, []string{c.ID}); err != nil {
		t.Fatalf("set tool collections: %v", err)
	}
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
	db := openTemp(t)
	u := mustUser(t, db, "a@example.com")
	grp, _ := db.CreateGroup("ops")
	c, _ := db.CreateCollection(Collection{Name: "Ops", CreatorID: u, Visibility: VisibilityRestricted})
	if err := db.AddCollectionEditor(c.ID, "group", grp.ID); err != nil {
		t.Fatalf("add editor: %v", err)
	}
	if err := db.AddCollectionVisibility(c.ID, "group", grp.ID); err != nil {
		t.Fatalf("add visibility: %v", err)
	}

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
