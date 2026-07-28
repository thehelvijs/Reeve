package main

import (
	"encoding/json"
	"net/http"
	"testing"
)

// newGroup creates a group as an admin and returns its view.
func newGroup(t *testing.T, ts *testServer, admin *http.Client, name string) groupView {
	t.Helper()
	resp, data := ts.do(t, admin, http.MethodPost, "/api/admin/groups", map[string]string{"name": name}, nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create group = %d: %s", resp.StatusCode, data)
	}
	var g groupView
	if err := json.Unmarshal(data, &g); err != nil {
		t.Fatal(err)
	}
	return g
}

func groupsFor(t *testing.T, ts *testServer, c *http.Client) []groupView {
	t.Helper()
	resp, data := ts.do(t, c, http.MethodGet, "/api/groups", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list groups = %d: %s", resp.StatusCode, data)
	}
	var out []groupView
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

// The whole point of the role: a basic account manages the membership of the one
// group it moderates, and nothing else.
func TestModeratorManagesOnlyTheirOwnGroup(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	mod := ts.client(t)
	_, modUser := signup(t, ts, mod, "mod@example.com", "password123")
	_, dev := signup(t, ts, ts.client(t), "dev@example.com", "password123")

	ops := newGroup(t, ts, admin, "ops")
	other := newGroup(t, ts, admin, "other")

	// A plain member sees no group to manage.
	ts.do(t, admin, http.MethodPut, "/api/admin/groups/"+ops.ID+"/members/"+modUser.ID, nil, nil)
	if got := groupsFor(t, ts, mod); len(got) != 0 {
		t.Fatalf("a plain member was handed %d groups to manage", len(got))
	}

	// Promoted, they see exactly that group, with its members.
	resp, data := ts.do(t, admin, http.MethodPatch, "/api/groups/"+ops.ID+"/members/"+modUser.ID,
		map[string]string{"role": "moderator"}, nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("promote = %d: %s", resp.StatusCode, data)
	}
	got := groupsFor(t, ts, mod)
	if len(got) != 1 || got[0].ID != ops.ID {
		t.Fatalf("moderator sees %+v, want only ops", got)
	}
	if len(got[0].Members) != 1 || got[0].Members[0].Email != "mod@example.com" || got[0].Members[0].Role != "moderator" {
		t.Fatalf("member view = %+v", got[0].Members)
	}

	// They add someone by email, and set that member's role.
	resp, data = ts.do(t, mod, http.MethodPost, "/api/groups/"+ops.ID+"/members",
		map[string]string{"email": "Dev@Example.com"}, nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("add by email = %d: %s", resp.StatusCode, data)
	}
	resp, _ = ts.do(t, mod, http.MethodPatch, "/api/groups/"+ops.ID+"/members/"+dev.ID,
		map[string]string{"role": "moderator"}, nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("set member role = %d", resp.StatusCode)
	}
	if got = groupsFor(t, ts, mod); len(got[0].Members) != 2 {
		t.Fatalf("members = %+v", got[0].Members)
	}

	// And they cannot touch the group they do not moderate.
	for _, tc := range []struct {
		name   string
		method string
		path   string
		body   any
	}{
		{"add", http.MethodPost, "/api/groups/" + other.ID + "/members", map[string]string{"email": "dev@example.com"}},
		{"promote", http.MethodPatch, "/api/groups/" + other.ID + "/members/" + dev.ID, map[string]string{"role": "moderator"}},
		{"remove", http.MethodDelete, "/api/groups/" + other.ID + "/members/" + dev.ID, nil},
	} {
		resp, _ := ts.do(t, mod, tc.method, tc.path, tc.body, nil)
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("%s in another group = %d, want 403", tc.name, resp.StatusCode)
		}
	}
}

// A moderator holds no power over accounts: they cannot invite, list users, or
// name an address that has no account yet.
func TestModeratorCannotReachAccounts(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	mod := ts.client(t)
	_, modUser := signup(t, ts, mod, "mod@example.com", "password123")
	ops := newGroup(t, ts, admin, "ops")
	ts.do(t, admin, http.MethodPut, "/api/admin/groups/"+ops.ID+"/members/"+modUser.ID, nil, nil)
	ts.do(t, admin, http.MethodPatch, "/api/groups/"+ops.ID+"/members/"+modUser.ID,
		map[string]string{"role": "moderator"}, nil)

	for _, tc := range []struct {
		name   string
		method string
		path   string
		body   any
		want   int
	}{
		{"list users", http.MethodGet, "/api/admin/users", nil, http.StatusForbidden},
		{"invite", http.MethodPost, "/api/admin/users", map[string]string{"email": "new@example.com"}, http.StatusForbidden},
		{"create group", http.MethodPost, "/api/admin/groups", map[string]string{"name": "mine"}, http.StatusForbidden},
		{"delete group", http.MethodDelete, "/api/admin/groups/" + ops.ID, nil, http.StatusForbidden},
		{"unknown email", http.MethodPost, "/api/groups/" + ops.ID + "/members", map[string]string{"email": "ghost@example.com"}, http.StatusNotFound},
		{"bad role", http.MethodPatch, "/api/groups/" + ops.ID + "/members/" + modUser.ID, map[string]string{"role": "owner"}, http.StatusBadRequest},
	} {
		resp, data := ts.do(t, mod, tc.method, tc.path, tc.body, nil)
		if resp.StatusCode != tc.want {
			t.Errorf("%s = %d, want %d: %s", tc.name, resp.StatusCode, tc.want, data)
		}
	}
}

// A moderator who could demote or remove themselves would leave a group nobody
// but an admin can manage, from a page that no longer lists it.
func TestModeratorCannotStandThemselvesDown(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	mod := ts.client(t)
	_, modUser := signup(t, ts, mod, "mod@example.com", "password123")
	ops := newGroup(t, ts, admin, "ops")
	ts.do(t, admin, http.MethodPut, "/api/admin/groups/"+ops.ID+"/members/"+modUser.ID, nil, nil)
	ts.do(t, admin, http.MethodPatch, "/api/groups/"+ops.ID+"/members/"+modUser.ID,
		map[string]string{"role": "moderator"}, nil)

	resp, _ := ts.do(t, mod, http.MethodPatch, "/api/groups/"+ops.ID+"/members/"+modUser.ID,
		map[string]string{"role": "member"}, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("self demote = %d, want 400", resp.StatusCode)
	}
	resp, _ = ts.do(t, mod, http.MethodDelete, "/api/groups/"+ops.ID+"/members/"+modUser.ID, nil, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("self remove = %d, want 400", resp.StatusCode)
	}
	// An admin can, because an admin can always put it back.
	resp, _ = ts.do(t, admin, http.MethodPatch, "/api/groups/"+ops.ID+"/members/"+modUser.ID,
		map[string]string{"role": "member"}, nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("admin demote = %d, want 204", resp.StatusCode)
	}
}

// Adding a member twice must not quietly demote a moderator, and one account
// belongs to as many groups as it is put in.
func TestMembershipSpansGroupsAndKeepsRoles(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	_, dev := signup(t, ts, ts.client(t), "dev@example.com", "password123")
	ops := newGroup(t, ts, admin, "ops")
	platform := newGroup(t, ts, admin, "platform")

	ts.do(t, admin, http.MethodPost, "/api/groups/"+ops.ID+"/members",
		map[string]string{"email": "dev@example.com", "role": "moderator"}, nil)
	ts.do(t, admin, http.MethodPost, "/api/groups/"+platform.ID+"/members",
		map[string]string{"email": "dev@example.com"}, nil)
	// The plain add carries no role, so it must not overwrite the moderator row.
	ts.do(t, admin, http.MethodPut, "/api/admin/groups/"+ops.ID+"/members/"+dev.ID, nil, nil)

	roles := map[string]string{}
	for _, g := range groupsFor(t, ts, admin) {
		for _, m := range g.Members {
			if m.UserID == dev.ID {
				roles[g.Name] = m.Role
			}
		}
	}
	if roles["ops"] != "moderator" || roles["platform"] != "member" {
		t.Fatalf("roles across groups = %+v", roles)
	}
}
