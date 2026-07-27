package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/thehelvijs/Reeve/server/internal/store"
)

func createTool(t *testing.T, ts *testServer, c *http.Client, in toolInput) toolResponse {
	t.Helper()
	resp, data := ts.do(t, c, http.MethodPost, "/api/tools", in, nil)
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

	_, data := ts.do(t, c, http.MethodGet, "/api/tools", nil, nil)
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
	_, data := ts.do(t, other, http.MethodGet, "/api/tools", nil, nil)
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
	_, data := ts.do(t, other, http.MethodGet, "/api/tools", nil, nil)
	var tools []toolResponse
	json.Unmarshal(data, &tools)
	if len(tools) != 0 {
		t.Fatalf("restricted tool leaked into list: %+v", tools)
	}
	// Detail returns 404 (existence hidden), not 403.
	resp, _ := ts.do(t, other, http.MethodGet, "/api/tools/"+tr.ID, nil, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("restricted detail = %d, want 404", resp.StatusCode)
	}

	// After a direct grant, the user can see it.
	ts.do(t, owner, http.MethodPut, "/api/tools/"+tr.ID+"/visibility/user/"+dev.ID, nil, nil)
	resp, _ = ts.do(t, other, http.MethodGet, "/api/tools/"+tr.ID, nil, nil)
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
	_, gd := ts.do(t, admin, http.MethodPost, "/api/admin/groups", map[string]string{"name": "ops"}, nil)
	var g groupView
	json.Unmarshal(gd, &g)
	ts.do(t, admin, http.MethodPut, "/api/admin/groups/"+g.ID+"/members/"+dev.ID, nil, nil)
	ts.do(t, admin, http.MethodPut, "/api/tools/"+tr.ID+"/visibility/group/"+g.ID, nil, nil)

	_, data := ts.do(t, other, http.MethodGet, "/api/tools", nil, nil)
	var tools []toolResponse
	json.Unmarshal(data, &tools)
	if len(tools) != 1 {
		t.Fatalf("group-granted tool not visible: %+v", tools)
	}
}

// Listing and removing a visibility grant is how an owner audits and undoes who
// can see a restricted tool, so removal has to actually hide it again.
func TestToolVisibilityListAndRemove(t *testing.T) {
	ts := newTestServer(t)
	owner := ts.client(t)
	signup(t, ts, owner, "boss@example.com", "password123")
	tr := createTool(t, ts, owner, toolInput{Name: "Restricted", Visibility: "restricted"})

	dev := ts.client(t)
	_, devUser := signup(t, ts, dev, "dev@example.com", "password123")

	resp, data := ts.do(t, owner, http.MethodGet, "/api/tools/"+tr.ID+"/visibility", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list visibility = %d: %s", resp.StatusCode, data)
	}
	if got := strings.TrimSpace(string(data)); got != "[]" {
		t.Errorf("visibility on a fresh tool = %s, want []", got)
	}

	ts.do(t, owner, http.MethodPut, "/api/tools/"+tr.ID+"/visibility/user/"+devUser.ID, nil, nil)
	_, data = ts.do(t, owner, http.MethodGet, "/api/tools/"+tr.ID+"/visibility", nil, nil)
	var grants []visibilityGrantView
	json.Unmarshal(data, &grants)
	if len(grants) != 1 || grants[0].PrincipalType != "user" || grants[0].PrincipalID != devUser.ID {
		t.Fatalf("visibility grants = %+v", grants)
	}
	if resp, _ := ts.do(t, dev, http.MethodGet, "/api/tools/"+tr.ID, nil, nil); resp.StatusCode != http.StatusOK {
		t.Fatalf("granted user cannot see the tool: %d", resp.StatusCode)
	}

	if resp, _ = ts.do(t, owner, http.MethodDelete, "/api/tools/"+tr.ID+"/visibility/user/"+devUser.ID, nil, nil); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("remove visibility = %d", resp.StatusCode)
	}
	_, data = ts.do(t, owner, http.MethodGet, "/api/tools/"+tr.ID+"/visibility", nil, nil)
	json.Unmarshal(data, &grants)
	if len(grants) != 0 {
		t.Errorf("grants after removal = %+v", grants)
	}
	if resp, _ := ts.do(t, dev, http.MethodGet, "/api/tools/"+tr.ID, nil, nil); resp.StatusCode != http.StatusNotFound {
		t.Errorf("tool still visible after removal: %d, want 404", resp.StatusCode)
	}

	// A removal naming a principal kind that does not exist is a typo, not a
	// no-op, and only someone who may edit the tool may touch its visibility.
	if resp, _ = ts.do(t, owner, http.MethodDelete, "/api/tools/"+tr.ID+"/visibility/robot/x", nil, nil); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("bad principal type = %d, want 400", resp.StatusCode)
	}
	if resp, _ = ts.do(t, dev, http.MethodGet, "/api/tools/"+tr.ID+"/visibility", nil, nil); resp.StatusCode != http.StatusNotFound {
		t.Errorf("outsider listing visibility = %d, want 404", resp.StatusCode)
	}
}

func TestNonOwnerCannotEditOrDelete(t *testing.T) {
	ts := newTestServer(t)
	owner := ts.client(t)
	signup(t, ts, owner, "boss@example.com", "password123")
	tr := createTool(t, ts, owner, toolInput{Name: "Owned"})

	other := ts.client(t)
	signup(t, ts, other, "dev@example.com", "password123") // basic, non-owner

	resp, _ := ts.do(t, other, http.MethodPatch, "/api/tools/"+tr.ID, toolInput{Name: "Hacked"}, nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("non-owner edit = %d, want 403", resp.StatusCode)
	}
	resp, _ = ts.do(t, other, http.MethodDelete, "/api/tools/"+tr.ID, nil, nil)
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

	resp, data := ts.do(t, admin, http.MethodPatch, "/api/tools/"+tr.ID, toolInput{Name: "Renamed"}, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("admin edit = %d: %s", resp.StatusCode, data)
	}
}

func TestToolSearchFilter(t *testing.T) {
	ts := newTestServer(t)
	c := ts.client(t)
	signup(t, ts, c, "boss@example.com", "password123")
	createTool(t, ts, c, toolInput{Name: "Grafana"})
	pg := createTool(t, ts, c, toolInput{Name: "Postgres"})

	_, data := ts.do(t, c, http.MethodGet, "/api/tools?search=graf", nil, nil)
	var tools []toolResponse
	json.Unmarshal(data, &tools)
	if len(tools) != 1 || tools[0].Name != "Grafana" {
		t.Errorf("search mismatch: %+v", tools)
	}

	col, err := ts.app.db.CreateCollection(store.Collection{Name: "Databases", CreatorID: tools[0].CreatorID})
	if err != nil {
		t.Fatalf("create collection: %v", err)
	}
	if err := ts.app.db.AddCollectionTool(col.ID, pg.ID); err != nil {
		t.Fatalf("add collection tool: %v", err)
	}

	_, data = ts.do(t, c, http.MethodGet, "/api/tools?collection="+col.ID, nil, nil)
	json.Unmarshal(data, &tools)
	if len(tools) != 1 || tools[0].Name != "Postgres" {
		t.Errorf("collection filter mismatch: %+v", tools)
	}
}

// TestToolEndpointFieldsRejectScriptSchemes pins that a tool's navigation target
// stays a navigation target. Both fields end up in a Location header and in the
// endpoint JSON, so neither may name a scheme that executes.
func TestToolEndpointFieldsRejectScriptSchemes(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)

	rejected := []toolInput{
		{Name: "js scheme", Scheme: "javascript"},
		{Name: "js url", URL: "javascript:alert(document.cookie)"},
		{Name: "data url", URL: "data:text/html,<script>alert(1)</script>"},
		{Name: "file url", URL: "file:///etc/shadow"},
		{Name: "relative url", URL: "/not/absolute"},
		{Name: "hostless url", URL: "http://"},
	}
	for _, in := range rejected {
		resp, data := ts.do(t, admin, http.MethodPost, "/api/tools", in, nil)
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400 (%s)", in.Name, resp.StatusCode, data)
		}
	}

	accepted := []toolInput{
		{Name: "https tool", Scheme: "https", Address: "grafana.lan"},
		{Name: "http tool", Scheme: "http", Address: "10.0.0.5", Port: 3000},
		{Name: "url tool", URL: "https://grafana.lan:3000/d/abc"},
		{Name: "host follower"},
	}
	for _, in := range accepted {
		resp, data := ts.do(t, admin, http.MethodPost, "/api/tools", in, nil)
		if resp.StatusCode != http.StatusCreated {
			t.Errorf("%s: status = %d, want 201 (%s)", in.Name, resp.StatusCode, data)
		}
	}
}

// TestToolUpdateRejectsScriptSchemes covers the other door into the same field:
// a tool created clean and edited dirty afterwards.
func TestToolUpdateRejectsScriptSchemes(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	tool := createTool(t, ts, admin, toolInput{Name: "clean", Scheme: "https", Address: "grafana.lan"})

	resp, data := ts.do(t, admin, http.MethodPatch, "/api/tools/"+tool.ID,
		toolInput{Name: "clean", URL: "javascript:alert(1)"}, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("update status = %d, want 400 (%s)", resp.StatusCode, data)
	}

	_, data = ts.do(t, admin, http.MethodGet, "/api/tools/"+tool.ID, nil, nil)
	var after toolResponse
	json.Unmarshal(data, &after)
	if after.URL != "" {
		t.Errorf("rejected update still changed the stored URL to %q", after.URL)
	}
}
