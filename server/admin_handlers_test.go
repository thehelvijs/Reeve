package main

import (
	"encoding/json"
	"net/http"
	"testing"
)

// Promoting and deactivating another account is the only way to hand over or
// shut off access, so both halves and the self-lockout guard are checked.
func TestUpdateUserRoleAndActive(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.client(t)
	_, adminUser := signup(t, ts, admin, "boss@example.com", "password123")
	dev := ts.client(t)
	_, devUser := signup(t, ts, dev, "dev@example.com", "password123")

	resp, data := ts.do(t, admin, http.MethodPatch, "/api/v1/admin/users/"+devUser.ID,
		map[string]any{"role": "admin"}, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("promote = %d: %s", resp.StatusCode, data)
	}
	var view adminUserView
	json.Unmarshal(data, &view)
	if view.Role != "admin" || !view.Active {
		t.Fatalf("promoted view = %+v", view)
	}

	// A deactivated account's session stops working, which is the point.
	if resp, data = ts.do(t, admin, http.MethodPatch, "/api/v1/admin/users/"+devUser.ID,
		map[string]any{"active": false}, nil); resp.StatusCode != http.StatusOK {
		t.Fatalf("deactivate = %d: %s", resp.StatusCode, data)
	}
	json.Unmarshal(data, &view)
	if view.Active {
		t.Error("view still reports the account active")
	}
	if resp, _ = ts.do(t, dev, http.MethodGet, "/api/v1/tools", nil, nil); resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("deactivated user still authenticated: %d", resp.StatusCode)
	}

	for name, body := range map[string]map[string]any{
		"demote self":     {"role": "basic"},
		"deactivate self": {"active": false},
	} {
		resp, data := ts.do(t, admin, http.MethodPatch, "/api/v1/admin/users/"+adminUser.ID, body, nil)
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("%s = %d, want 400: %s", name, resp.StatusCode, data)
		}
	}

	cases := map[string]struct {
		id   string
		body map[string]any
		want int
	}{
		"no changes":   {devUser.ID, map[string]any{}, http.StatusBadRequest},
		"invalid role": {devUser.ID, map[string]any{"role": "wizard"}, http.StatusBadRequest},
		"unknown user": {"nobody", map[string]any{"role": "basic"}, http.StatusNotFound},
	}
	for name, tc := range cases {
		resp, data := ts.do(t, admin, http.MethodPatch, "/api/v1/admin/users/"+tc.id, tc.body, nil)
		if resp.StatusCode != tc.want {
			t.Errorf("%s = %d, want %d: %s", name, resp.StatusCode, tc.want, data)
		}
	}
}

// The audit endpoints are the read side of the hash-chained trail: an admin has
// to be able to see who revealed what, and narrow it by user and by tool.
func TestAuditListsAndFilters(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.client(t)
	_, adminUser := signup(t, ts, admin, "boss@example.com", "password123")
	dev := ts.client(t)
	_, devUser := signup(t, ts, dev, "dev@example.com", "password123")

	toolA := createTool(t, ts, admin, toolInput{Name: "Box A"})
	toolB := createTool(t, ts, admin, toolInput{Name: "Box B"})
	credA := createCred(t, ts, admin, toolA.ID, credentialInput{Type: "kv", Secret: map[string]string{"k": "a"}})
	credB := createCred(t, ts, admin, toolB.ID, credentialInput{Type: "kv", Secret: map[string]string{"k": "b"}})

	// One reveal per tool by the admin, one by the dev after a grant.
	revealSecret(t, ts, admin, credA.ID)
	revealSecret(t, ts, admin, credB.ID)
	ts.do(t, admin, http.MethodPut, "/api/v1/tools/"+toolA.ID+"/access/user/"+devUser.ID, nil, nil)
	if code, _ := revealSecret(t, ts, dev, credA.ID); code != http.StatusOK {
		t.Fatalf("granted reveal = %d", code)
	}

	reveals := func(query string) []map[string]any {
		t.Helper()
		resp, data := ts.do(t, admin, http.MethodGet, "/api/v1/admin/audit/reveals"+query, nil, nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("reveals%s = %d: %s", query, resp.StatusCode, data)
		}
		var out []map[string]any
		json.Unmarshal(data, &out)
		return out
	}

	if got := len(reveals("")); got != 3 {
		t.Fatalf("unfiltered reveals = %d, want 3", got)
	}
	if got := len(reveals("?tool=" + toolA.ID)); got != 2 {
		t.Errorf("reveals for tool A = %d, want 2", got)
	}
	if got := len(reveals("?user=" + devUser.ID)); got != 1 {
		t.Errorf("reveals by dev = %d, want 1", got)
	}
	if got := len(reveals("?user=" + adminUser.ID + "&tool=" + toolB.ID)); got != 1 {
		t.Errorf("reveals by admin on tool B = %d, want 1", got)
	}
	if got := len(reveals("?user=nobody")); got != 0 {
		t.Errorf("reveals for an unknown user = %d, want 0", got)
	}
	// A window that ends before anything happened excludes everything.
	if got := len(reveals("?to=2000-01-01T00:00:00Z")); got != 0 {
		t.Errorf("reveals before the epoch window = %d, want 0", got)
	}

	resp, data := ts.do(t, admin, http.MethodGet, "/api/v1/admin/audit/grants?tool="+toolA.ID, nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("grants = %d: %s", resp.StatusCode, data)
	}
	var grants []map[string]any
	json.Unmarshal(data, &grants)
	if len(grants) != 1 || grants[0]["action"] != "grant" {
		t.Errorf("grant audit for tool A = %s", data)
	}
}

func TestServerInfoAdminOnly(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.client(t)
	signup(t, ts, admin, "boss@example.com", "password123") // first user = admin

	resp, data := ts.do(t, admin, http.MethodGet, "/api/v1/admin/server-info", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("admin server-info = %d: %s", resp.StatusCode, data)
	}
	var info map[string]any
	if err := json.Unmarshal(data, &info); err != nil {
		t.Fatalf("bad json: %v", err)
	}
	for _, key := range []string{"version", "uptime_secs", "go", "db", "counts", "host"} {
		if _, ok := info[key]; !ok {
			t.Errorf("server-info missing %q: %s", key, data)
		}
	}

	basic := ts.client(t)
	signup(t, ts, basic, "dev@example.com", "password123") // second user = basic
	resp2, _ := ts.do(t, basic, http.MethodGet, "/api/v1/admin/server-info", nil, nil)
	if resp2.StatusCode != http.StatusForbidden {
		t.Errorf("basic user server-info = %d, want 403", resp2.StatusCode)
	}
}
