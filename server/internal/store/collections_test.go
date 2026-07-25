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
