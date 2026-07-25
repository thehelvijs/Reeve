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
