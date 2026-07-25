package main

import (
	"encoding/json"
	"net/http"
	"testing"
)

// newUser signs a fresh client up and returns it with the created user row.
func newUser(t *testing.T, ts *testServer, email string) (*http.Client, userView) {
	t.Helper()
	c := ts.client(t)
	resp, v := signup(t, ts, c, email, "password123")
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("signup %s = %d", email, resp.StatusCode)
	}
	return c, v
}

func jsonString(t *testing.T, body []byte, key string) string {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatalf("unmarshal %s: %v", body, err)
	}
	s, _ := m[key].(string)
	if s == "" {
		t.Fatalf("field %q missing from %s", key, body)
	}
	return s
}

func containsName(t *testing.T, body []byte, name string) bool {
	t.Helper()
	var any1 any
	if err := json.Unmarshal(body, &any1); err != nil {
		t.Fatalf("unmarshal %s: %v", body, err)
	}
	return hasName(any1, name)
}

func hasName(v any, name string) bool {
	switch t := v.(type) {
	case map[string]any:
		if s, ok := t["name"].(string); ok && s == name {
			return true
		}
		for _, child := range t {
			if hasName(child, name) {
				return true
			}
		}
	case []any:
		for _, child := range t {
			if hasName(child, name) {
				return true
			}
		}
	}
	return false
}

func TestCollectionCreateAndEditPermissions(t *testing.T) {
	ts := newTestServer(t)
	admin, _ := newUser(t, ts, "boss@example.com")
	owner, _ := newUser(t, ts, "owner@example.com")
	stranger, strangerUser := newUser(t, ts, "stranger@example.com")

	// Any signed-in user can create.
	resp, body := ts.do(t, owner, http.MethodPost, "/api/v1/collections",
		map[string]any{"name": "Manufacturing", "description": "Shop floor."}, nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create by basic user = %d, want 201: %s", resp.StatusCode, body)
	}
	id := jsonString(t, body, "id")

	// A stranger cannot edit.
	resp, _ = ts.do(t, stranger, http.MethodPatch, "/api/v1/collections/"+id,
		map[string]any{"name": "Hijacked"}, nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("stranger PATCH = %d, want 403", resp.StatusCode)
	}

	// The creator can.
	resp, _ = ts.do(t, owner, http.MethodPatch, "/api/v1/collections/"+id,
		map[string]any{"name": "Manufacturing", "description": "Updated."}, nil)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("creator PATCH = %d, want 200", resp.StatusCode)
	}

	// An added editor can.
	resp, _ = ts.do(t, owner, http.MethodPut,
		"/api/v1/collections/"+id+"/editors/user/"+strangerUser.ID, nil, nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("add editor = %d, want 204", resp.StatusCode)
	}
	resp, _ = ts.do(t, stranger, http.MethodPatch, "/api/v1/collections/"+id,
		map[string]any{"name": "Manufacturing"}, nil)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("editor PATCH = %d, want 200", resp.StatusCode)
	}

	// An admin can.
	resp, _ = ts.do(t, admin, http.MethodPatch, "/api/v1/collections/"+id,
		map[string]any{"name": "Manufacturing"}, nil)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("admin PATCH = %d, want 200", resp.StatusCode)
	}

	// An editor who is not the creator cannot delete.
	resp, _ = ts.do(t, stranger, http.MethodDelete, "/api/v1/collections/"+id, nil, nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("editor DELETE = %d, want 403", resp.StatusCode)
	}
	// The creator can.
	resp, _ = ts.do(t, owner, http.MethodDelete, "/api/v1/collections/"+id, nil, nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("creator DELETE = %d, want 204", resp.StatusCode)
	}
}

func TestCollectionValidation(t *testing.T) {
	ts := newTestServer(t)
	owner, _ := newUser(t, ts, "boss@example.com")

	resp, _ := ts.do(t, owner, http.MethodPost, "/api/v1/collections",
		map[string]any{"name": "   "}, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("blank name = %d, want 400", resp.StatusCode)
	}
	resp, _ = ts.do(t, owner, http.MethodPost, "/api/v1/collections",
		map[string]any{"name": "Ops", "visibility": "secret"}, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("bad visibility = %d, want 400", resp.StatusCode)
	}
	resp, body := ts.do(t, owner, http.MethodPost, "/api/v1/collections",
		map[string]any{"name": "Ops"}, nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create = %d: %s", resp.StatusCode, body)
	}
	resp, _ = ts.do(t, owner, http.MethodPost, "/api/v1/collections",
		map[string]any{"name": "Ops"}, nil)
	if resp.StatusCode != http.StatusConflict {
		t.Errorf("duplicate name = %d, want 409", resp.StatusCode)
	}

	id := jsonString(t, body, "id")
	resp, _ = ts.do(t, owner, http.MethodPut, "/api/v1/collections/"+id+"/editors/robot/x", nil, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("bad principal type = %d, want 400", resp.StatusCode)
	}
	resp, _ = ts.do(t, owner, http.MethodPut, "/api/v1/collections/"+id+"/tools/nosuchtool", nil, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("unknown tool = %d, want 404", resp.StatusCode)
	}
}

func TestCollectionToolsAndGrantRoutes(t *testing.T) {
	ts := newTestServer(t)
	owner, _ := newUser(t, ts, "boss@example.com")
	_, viewer := newUser(t, ts, "viewer@example.com")

	_, body := ts.do(t, owner, http.MethodPost, "/api/v1/collections",
		map[string]any{"name": "Metrics"}, nil)
	id := jsonString(t, body, "id")
	tool := createTool(t, ts, owner, toolInput{Name: "Grafana"})

	resp, _ := ts.do(t, owner, http.MethodPut, "/api/v1/collections/"+id+"/tools/"+tool.ID, nil, nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("add tool = %d, want 204", resp.StatusCode)
	}
	_, body = ts.do(t, owner, http.MethodGet, "/api/v1/collections/"+id, nil, nil)
	var detail struct {
		ToolCount int      `json:"tool_count"`
		CanEdit   bool     `json:"can_edit"`
		ToolIDs   []string `json:"tool_ids"`
	}
	json.Unmarshal(body, &detail)
	if detail.ToolCount != 1 || len(detail.ToolIDs) != 1 || detail.ToolIDs[0] != tool.ID {
		t.Errorf("detail after add = %+v", detail)
	}
	if !detail.CanEdit {
		t.Error("creator can_edit = false")
	}

	resp, _ = ts.do(t, owner, http.MethodDelete, "/api/v1/collections/"+id+"/tools/"+tool.ID, nil, nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("remove tool = %d, want 204", resp.StatusCode)
	}

	ts.do(t, owner, http.MethodPut, "/api/v1/collections/"+id+"/visibility/user/"+viewer.ID, nil, nil)
	_, body = ts.do(t, owner, http.MethodGet, "/api/v1/collections/"+id+"/visibility", nil, nil)
	var grants []visibilityGrantView
	json.Unmarshal(body, &grants)
	if len(grants) != 1 || grants[0].PrincipalID != viewer.ID {
		t.Errorf("visibility grants = %+v", grants)
	}
	ts.do(t, owner, http.MethodDelete, "/api/v1/collections/"+id+"/visibility/user/"+viewer.ID, nil, nil)
	_, body = ts.do(t, owner, http.MethodGet, "/api/v1/collections/"+id+"/visibility", nil, nil)
	json.Unmarshal(body, &grants)
	if len(grants) != 0 {
		t.Errorf("visibility grants after revoke = %+v", grants)
	}

	ts.do(t, owner, http.MethodPut, "/api/v1/collections/"+id+"/editors/user/"+viewer.ID, nil, nil)
	_, body = ts.do(t, owner, http.MethodGet, "/api/v1/collections/"+id+"/editors", nil, nil)
	json.Unmarshal(body, &grants)
	if len(grants) != 1 || grants[0].PrincipalID != viewer.ID {
		t.Errorf("editor grants = %+v", grants)
	}
}

func TestCollectionVisibilityOverHTTP(t *testing.T) {
	ts := newTestServer(t)
	owner, _ := newUser(t, ts, "boss@example.com")
	stranger, _ := newUser(t, ts, "stranger@example.com")

	_, body := ts.do(t, owner, http.MethodPost, "/api/v1/collections",
		map[string]any{"name": "Secret", "visibility": "restricted"}, nil)
	id := jsonString(t, body, "id")

	resp, _ := ts.do(t, stranger, http.MethodGet, "/api/v1/collections/"+id, nil, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("invisible GET = %d, want 404 so the name does not leak", resp.StatusCode)
	}

	_, body = ts.do(t, nil, http.MethodGet, "/api/v1/public/collections", nil, nil)
	if containsName(t, body, "Secret") {
		t.Error("public collection list leaked a restricted collection")
	}

	_, body = ts.do(t, stranger, http.MethodGet, "/api/v1/collections", nil, nil)
	if containsName(t, body, "Secret") {
		t.Error("authenticated list leaked a collection the caller cannot see")
	}
	_, body = ts.do(t, owner, http.MethodGet, "/api/v1/collections", nil, nil)
	if !containsName(t, body, "Secret") {
		t.Error("creator's own collection missing from their list")
	}
}

func TestListPrincipals(t *testing.T) {
	ts := newTestServer(t)
	admin, _ := newUser(t, ts, "boss@example.com")
	basic, _ := newUser(t, ts, "dev@example.com")
	ts.do(t, admin, http.MethodPost, "/api/v1/admin/groups", map[string]string{"name": "ops"}, nil)

	resp, body := ts.do(t, basic, http.MethodGet, "/api/v1/principals", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("principals for a basic user = %d: %s", resp.StatusCode, body)
	}
	var out principalsView
	json.Unmarshal(body, &out)
	if len(out.Users) != 2 || len(out.Groups) != 1 {
		t.Errorf("principals = %+v, want 2 users and 1 group", out)
	}
	if out.Groups[0].Name != "ops" {
		t.Errorf("group name = %q, want ops", out.Groups[0].Name)
	}
	for _, field := range []string{"password_hash", "role", "active"} {
		if containsKey(body, field) {
			t.Errorf("principals leaked %q: %s", field, body)
		}
	}

	resp, _ = ts.do(t, nil, http.MethodGet, "/api/v1/principals", nil, nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("anonymous principals = %d, want 401", resp.StatusCode)
	}
}

func containsKey(body []byte, key string) bool {
	var v any
	if err := json.Unmarshal(body, &v); err != nil {
		return false
	}
	return hasKey(v, key)
}

func hasKey(v any, key string) bool {
	switch t := v.(type) {
	case map[string]any:
		if _, ok := t[key]; ok {
			return true
		}
		for _, child := range t {
			if hasKey(child, key) {
				return true
			}
		}
	case []any:
		for _, child := range t {
			if hasKey(child, key) {
				return true
			}
		}
	}
	return false
}
