package main

import (
	"encoding/json"
	"net/http"
	"testing"
)

func adminClient(t *testing.T, ts *testServer) *http.Client {
	t.Helper()
	c := ts.client(t)
	signup(t, ts, c, "boss@example.com", "password123")
	return c
}

func TestGroupCRUDAndMembership(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	_, dev := signup(t, ts, ts.client(t), "dev@example.com", "password123")

	// Create.
	resp, data := ts.do(t, admin, http.MethodPost, "/api/v1/admin/groups", map[string]string{"name": "ops"}, nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create group = %d: %s", resp.StatusCode, data)
	}
	var g groupView
	json.Unmarshal(data, &g)

	// Add member.
	resp, _ = ts.do(t, admin, http.MethodPut, "/api/v1/admin/groups/"+g.ID+"/members/"+dev.ID, nil, nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("add member = %d", resp.StatusCode)
	}

	// List reflects the member.
	_, data = ts.do(t, admin, http.MethodGet, "/api/v1/admin/groups", nil, nil)
	var groups []groupView
	json.Unmarshal(data, &groups)
	if len(groups) != 1 || len(groups[0].Members) != 1 || groups[0].Members[0] != dev.ID {
		t.Fatalf("membership not reflected: %+v", groups)
	}

	// Remove member.
	ts.do(t, admin, http.MethodDelete, "/api/v1/admin/groups/"+g.ID+"/members/"+dev.ID, nil, nil)
	_, data = ts.do(t, admin, http.MethodGet, "/api/v1/admin/groups", nil, nil)
	json.Unmarshal(data, &groups)
	if len(groups[0].Members) != 0 {
		t.Errorf("member not removed: %+v", groups[0])
	}

	// Delete cascades membership (no FK error, group gone).
	resp, _ = ts.do(t, admin, http.MethodDelete, "/api/v1/admin/groups/"+g.ID, nil, nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete group = %d", resp.StatusCode)
	}
	_, data = ts.do(t, admin, http.MethodGet, "/api/v1/admin/groups", nil, nil)
	json.Unmarshal(data, &groups)
	if len(groups) != 0 {
		t.Errorf("group not deleted: %+v", groups)
	}
}

// A rename must keep the group's identity, since visibility and credential
// grants point at the id — renaming must not orphan them.
func TestRenameGroup(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	_, dev := signup(t, ts, ts.client(t), "dev@example.com", "password123")

	_, data := ts.do(t, admin, http.MethodPost, "/api/v1/admin/groups", map[string]string{"name": "ops"}, nil)
	var g groupView
	json.Unmarshal(data, &g)
	ts.do(t, admin, http.MethodPut, "/api/v1/admin/groups/"+g.ID+"/members/"+dev.ID, nil, nil)
	ts.do(t, admin, http.MethodPost, "/api/v1/admin/groups", map[string]string{"name": "taken"}, nil)

	resp, data := ts.do(t, admin, http.MethodPatch, "/api/v1/admin/groups/"+g.ID, map[string]string{"name": "  platform  "}, nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("rename = %d: %s", resp.StatusCode, data)
	}

	_, data = ts.do(t, admin, http.MethodGet, "/api/v1/admin/groups", nil, nil)
	var groups []groupView
	json.Unmarshal(data, &groups)
	var renamed *groupView
	for i := range groups {
		if groups[i].ID == g.ID {
			renamed = &groups[i]
		}
	}
	if renamed == nil {
		t.Fatalf("the group lost its id on rename: %s", data)
	}
	if renamed.Name != "platform" {
		t.Errorf("name = %q, want the trimmed %q", renamed.Name, "platform")
	}
	if len(renamed.Members) != 1 || renamed.Members[0] != dev.ID {
		t.Errorf("rename dropped the membership: %+v", renamed.Members)
	}

	cases := map[string]struct {
		id   string
		body map[string]string
		want int
	}{
		"blank name":    {g.ID, map[string]string{"name": "   "}, http.StatusBadRequest},
		"name taken":    {g.ID, map[string]string{"name": "taken"}, http.StatusConflict},
		"unknown group": {"nope", map[string]string{"name": "whatever"}, http.StatusNotFound},
	}
	for name, tc := range cases {
		resp, data := ts.do(t, admin, http.MethodPatch, "/api/v1/admin/groups/"+tc.id, tc.body, nil)
		if resp.StatusCode != tc.want {
			t.Errorf("%s = %d, want %d: %s", name, resp.StatusCode, tc.want, data)
		}
	}
}

func TestDuplicateGroupName(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	ts.do(t, admin, http.MethodPost, "/api/v1/admin/groups", map[string]string{"name": "ops"}, nil)
	resp, _ := ts.do(t, admin, http.MethodPost, "/api/v1/admin/groups", map[string]string{"name": "ops"}, nil)
	if resp.StatusCode != http.StatusConflict {
		t.Errorf("dup group name = %d, want 409", resp.StatusCode)
	}
}

func TestGroupsAdminOnly(t *testing.T) {
	ts := newTestServer(t)
	adminClient(t, ts)
	basic := ts.client(t)
	signup(t, ts, basic, "dev@example.com", "password123")
	resp, _ := ts.do(t, basic, http.MethodGet, "/api/v1/admin/groups", nil, nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("basic on groups = %d, want 403", resp.StatusCode)
	}
}

func TestAddMissingUserToGroup(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	_, data := ts.do(t, admin, http.MethodPost, "/api/v1/admin/groups", map[string]string{"name": "ops"}, nil)
	var g groupView
	json.Unmarshal(data, &g)
	resp, _ := ts.do(t, admin, http.MethodPut, "/api/v1/admin/groups/"+g.ID+"/members/nouser", nil, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("add missing user = %d, want 404", resp.StatusCode)
	}
}
