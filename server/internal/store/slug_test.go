package store

import (
	"testing"
)

// dbWithUser opens a temp database and returns it with an id tools can be
// created under, since tools reference a creator.
func dbWithUser(t *testing.T) (*DB, string) {
	t.Helper()
	db := openTemp(t)
	u, err := db.CreateUser("slug@example.com", "hash", RoleBasic)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	return db, u.ID
}

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Grafana":                "grafana",
		"Paperless-ngx":          "paperless-ngx",
		"  Home   Assistant  ":   "home-assistant",
		"Nextcloud (prod)":       "nextcloud-prod",
		"UPS #2 — battery":       "ups-2-battery",
		"already-a-slug":         "already-a-slug",
		"MiXeD CaSe 123":         "mixed-case-123",
		"___":                    "",
		"":                       "",
		"café":                   "café",
		"a/b\\c":                 "a-b-c",
		"trailing-punctuation!!": "trailing-punctuation",
	}
	for name, want := range cases {
		if got := Slugify(name); got != want {
			t.Errorf("Slugify(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestSlugifyTruncatesWithoutATrailingDash(t *testing.T) {
	long := ""
	for i := 0; i < 30; i++ {
		long += "ab "
	}
	got := Slugify(long)
	if len(got) > maxSlugLen {
		t.Errorf("Slugify produced %d characters, want at most %d", len(got), maxSlugLen)
	}
	if got[len(got)-1] == '-' {
		t.Errorf("Slugify(%q) = %q, which ends in a dash", long, got)
	}
}

// Two tools named the same must both get a working /go/ URL.
func TestUniqueToolSlugSuffixes(t *testing.T) {
	db, creator := dbWithUser(t)

	var slugs []string
	for i := 0; i < 3; i++ {
		tool, err := db.CreateTool(Tool{Name: "Grafana", CreatorID: creator})
		if err != nil {
			t.Fatalf("create tool %d: %v", i, err)
		}
		slugs = append(slugs, tool.Slug)
	}
	want := []string{"grafana", "grafana-2", "grafana-3"}
	for i, w := range want {
		if slugs[i] != w {
			t.Errorf("slug %d = %q, want %q", i, slugs[i], w)
		}
	}
}

// A tool keeping its own slug is not a collision with itself.
func TestUniqueToolSlugIgnoresTheExcludedTool(t *testing.T) {
	db, creator := dbWithUser(t)
	tool, err := db.CreateTool(Tool{Name: "Grafana", CreatorID: creator})
	if err != nil {
		t.Fatalf("create tool: %v", err)
	}
	got, err := db.UniqueToolSlug("grafana", tool.ID)
	if err != nil {
		t.Fatalf("UniqueToolSlug: %v", err)
	}
	if got != "grafana" {
		t.Errorf("UniqueToolSlug excluding the holder = %q, want grafana", got)
	}
}

// A name with nothing sluggable still has to produce a usable URL.
func TestCreateToolFallsBackWhenTheNameHasNoSlug(t *testing.T) {
	db, creator := dbWithUser(t)
	tool, err := db.CreateTool(Tool{Name: "!!!", CreatorID: creator})
	if err != nil {
		t.Fatalf("create tool: %v", err)
	}
	if tool.Slug != "tool" {
		t.Errorf("slug = %q, want the tool fallback", tool.Slug)
	}
}

func TestGetToolBySlug(t *testing.T) {
	db, creator := dbWithUser(t)
	made, err := db.CreateTool(Tool{Name: "Paperless ngx", CreatorID: creator})
	if err != nil {
		t.Fatalf("create tool: %v", err)
	}
	got, err := db.GetToolBySlug("paperless-ngx")
	if err != nil {
		t.Fatalf("GetToolBySlug: %v", err)
	}
	if got.ID != made.ID {
		t.Errorf("GetToolBySlug returned %q, want %q", got.ID, made.ID)
	}
	if _, err := db.GetToolBySlug("nothing-here"); err != ErrNotFound {
		t.Errorf("unknown slug error = %v, want ErrNotFound", err)
	}
}
