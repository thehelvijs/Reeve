package main

import (
	"encoding/json"
	"net/http"
	"testing"
)

func createTool(t *testing.T, ts *testServer, c *http.Client, in toolInput) toolResponse {
	t.Helper()
	resp, data := ts.do(t, c, http.MethodPost, "/api/v1/tools", in, nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create tool = %d: %s", resp.StatusCode, data)
	}
	var tr toolResponse
	json.Unmarshal(data, &tr)
	return tr
}

func TestToolCreateAndList(t *testing.T) {
	ts := newTestServer(t)
	c := ts.client(t)
	signup(t, ts, c, "boss@example.com", "password123")

	tr := createTool(t, ts, c, toolInput{Name: "Grafana", Address: "10.0.0.5", Port: 3000, Tags: []string{"metrics"}})
	if tr.Name != "Grafana" || !tr.CanEdit {
		t.Fatalf("unexpected tool: %+v", tr)
	}

	_, data := ts.do(t, c, http.MethodGet, "/api/v1/tools", nil, nil)
	var tools []toolResponse
	json.Unmarshal(data, &tools)
	if len(tools) != 1 || tools[0].Tags[0] != "metrics" {
		t.Fatalf("list mismatch: %+v", tools)
	}
}

func TestPublicToolVisibleToOthers(t *testing.T) {
	ts := newTestServer(t)
	owner := ts.client(t)
	signup(t, ts, owner, "boss@example.com", "password123")
	createTool(t, ts, owner, toolInput{Name: "Public Tool"})

	other := ts.client(t)
	signup(t, ts, other, "dev@example.com", "password123")
	_, data := ts.do(t, other, http.MethodGet, "/api/v1/tools", nil, nil)
	var tools []toolResponse
	json.Unmarshal(data, &tools)
	if len(tools) != 1 {
		t.Fatalf("public tool not visible to other user: %+v", tools)
	}
	if tools[0].CanEdit {
		t.Error("non-owner should not be able to edit")
	}
}

func TestRestrictedToolHiddenFromOutsiders(t *testing.T) {
	ts := newTestServer(t)
	owner := ts.client(t)
	signup(t, ts, owner, "boss@example.com", "password123") // admin (first)
	tr := createTool(t, ts, owner, toolInput{Name: "Secret", Visibility: "restricted"})

	other := ts.client(t)
	_, dev := signup(t, ts, other, "dev@example.com", "password123")

	// Absent from the list.
	_, data := ts.do(t, other, http.MethodGet, "/api/v1/tools", nil, nil)
	var tools []toolResponse
	json.Unmarshal(data, &tools)
	if len(tools) != 0 {
		t.Fatalf("restricted tool leaked into list: %+v", tools)
	}
	// Detail returns 404 (existence hidden), not 403.
	resp, _ := ts.do(t, other, http.MethodGet, "/api/v1/tools/"+tr.ID, nil, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("restricted detail = %d, want 404", resp.StatusCode)
	}

	// After a direct grant, the user can see it.
	ts.do(t, owner, http.MethodPut, "/api/v1/tools/"+tr.ID+"/visibility/user/"+dev.ID, nil, nil)
	resp, _ = ts.do(t, other, http.MethodGet, "/api/v1/tools/"+tr.ID, nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("after grant detail = %d, want 200", resp.StatusCode)
	}
}

func TestRestrictedToolVisibleViaGroup(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.client(t)
	signup(t, ts, admin, "boss@example.com", "password123")
	tr := createTool(t, ts, admin, toolInput{Name: "GroupOnly", Visibility: "restricted"})

	other := ts.client(t)
	_, dev := signup(t, ts, other, "dev@example.com", "password123")

	// Group with dev as member, granted visibility.
	_, gd := ts.do(t, admin, http.MethodPost, "/api/v1/admin/groups", map[string]string{"name": "ops"}, nil)
	var g groupView
	json.Unmarshal(gd, &g)
	ts.do(t, admin, http.MethodPut, "/api/v1/admin/groups/"+g.ID+"/members/"+dev.ID, nil, nil)
	ts.do(t, admin, http.MethodPut, "/api/v1/tools/"+tr.ID+"/visibility/group/"+g.ID, nil, nil)

	_, data := ts.do(t, other, http.MethodGet, "/api/v1/tools", nil, nil)
	var tools []toolResponse
	json.Unmarshal(data, &tools)
	if len(tools) != 1 {
		t.Fatalf("group-granted tool not visible: %+v", tools)
	}
}

func TestNonOwnerCannotEditOrDelete(t *testing.T) {
	ts := newTestServer(t)
	owner := ts.client(t)
	signup(t, ts, owner, "boss@example.com", "password123")
	tr := createTool(t, ts, owner, toolInput{Name: "Owned"})

	other := ts.client(t)
	signup(t, ts, other, "dev@example.com", "password123") // basic, non-owner

	resp, _ := ts.do(t, other, http.MethodPatch, "/api/v1/tools/"+tr.ID, toolInput{Name: "Hacked"}, nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("non-owner edit = %d, want 403", resp.StatusCode)
	}
	resp, _ = ts.do(t, other, http.MethodDelete, "/api/v1/tools/"+tr.ID, nil, nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("non-owner delete = %d, want 403", resp.StatusCode)
	}
}

func TestAdminCanEditAnyTool(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.client(t)
	signup(t, ts, admin, "boss@example.com", "password123") // first = admin
	dev := ts.client(t)
	signup(t, ts, dev, "dev@example.com", "password123")
	tr := createTool(t, ts, dev, toolInput{Name: "DevTool"})

	resp, data := ts.do(t, admin, http.MethodPatch, "/api/v1/tools/"+tr.ID, toolInput{Name: "Renamed"}, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("admin edit = %d: %s", resp.StatusCode, data)
	}
}

func TestToolSearchFilter(t *testing.T) {
	ts := newTestServer(t)
	c := ts.client(t)
	signup(t, ts, c, "boss@example.com", "password123")
	createTool(t, ts, c, toolInput{Name: "Grafana", Category: "metrics"})
	createTool(t, ts, c, toolInput{Name: "Postgres", Category: "database"})

	_, data := ts.do(t, c, http.MethodGet, "/api/v1/tools?search=graf", nil, nil)
	var tools []toolResponse
	json.Unmarshal(data, &tools)
	if len(tools) != 1 || tools[0].Name != "Grafana" {
		t.Errorf("search mismatch: %+v", tools)
	}

	_, data = ts.do(t, c, http.MethodGet, "/api/v1/tools?category=database", nil, nil)
	json.Unmarshal(data, &tools)
	if len(tools) != 1 || tools[0].Name != "Postgres" {
		t.Errorf("category filter mismatch: %+v", tools)
	}
}
