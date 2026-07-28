package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// With no relay configured the response has to carry the link, or the admin has
// just made an account nobody can reach.
func TestInviteWithoutMailReturnsTheLink(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)

	resp, data := ts.do(t, admin, http.MethodPost, "/api/admin/users",
		map[string]string{"email": "New@Example.com ", "role": "admin", "display_name": "New Person"}, nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("invite = %d: %s", resp.StatusCode, data)
	}
	var out inviteView
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if out.Emailed {
		t.Error("invite reports it was emailed with no relay configured")
	}
	if !strings.Contains(out.InviteLink, "/reset?token=") {
		t.Fatalf("invite link = %q", out.InviteLink)
	}
	if out.User.Email != "new@example.com" {
		t.Errorf("email = %q, want it trimmed and lowercased", out.User.Email)
	}
	if out.User.Role != "admin" || out.User.DisplayName != "New Person" {
		t.Errorf("invited user = %+v", out.User)
	}

	// The link is what sets the first password, and then it is spent.
	token := out.InviteLink[strings.Index(out.InviteLink, "token=")+len("token="):]
	resp, data = ts.do(t, ts.client(t), http.MethodPost, "/api/auth/reset",
		map[string]string{"token": token, "password": "invited-pass"}, nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("reset with the invite token = %d: %s", resp.StatusCode, data)
	}
	resp, data = ts.do(t, ts.client(t), http.MethodPost, "/api/auth/login",
		map[string]string{"email": "new@example.com", "password": "invited-pass"}, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login as the invited account = %d: %s", resp.StatusCode, data)
	}
}

// The account exists before the invite goes out, so a second invite to the same
// address is a conflict rather than a second account.
func TestInviteRejectsADuplicateAndABadRole(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)

	for _, tc := range []struct {
		name string
		body map[string]string
		want int
	}{
		{"first", map[string]string{"email": "dup@example.com"}, http.StatusCreated},
		{"duplicate", map[string]string{"email": "dup@example.com"}, http.StatusConflict},
		{"no at sign", map[string]string{"email": "nope"}, http.StatusBadRequest},
		{"empty", map[string]string{"email": ""}, http.StatusBadRequest},
		{"bad role", map[string]string{"email": "role@example.com", "role": "owner"}, http.StatusBadRequest},
	} {
		resp, data := ts.do(t, admin, http.MethodPost, "/api/admin/users", tc.body, nil)
		if resp.StatusCode != tc.want {
			t.Errorf("%s: invite = %d, want %d: %s", tc.name, resp.StatusCode, tc.want, data)
		}
	}
}

// An invited account with no password set cannot be signed into by guessing, and
// the default role is the safe one.
func TestInviteDefaultsToBasicAndCannotSignInYet(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)

	_, data := ts.do(t, admin, http.MethodPost, "/api/admin/users",
		map[string]string{"email": "quiet@example.com"}, nil)
	var out inviteView
	json.Unmarshal(data, &out)
	if out.User.Role != "basic" {
		t.Errorf("role = %q, want basic", out.User.Role)
	}
	resp, _ := ts.do(t, ts.client(t), http.MethodPost, "/api/auth/login",
		map[string]string{"email": "quiet@example.com", "password": ""}, nil)
	if resp.StatusCode == http.StatusOK {
		t.Error("an invited account signed in before its password was set")
	}
}

func TestInviteIsAdminOnly(t *testing.T) {
	ts := newTestServer(t)
	adminClient(t, ts)
	basic := ts.client(t)
	signup(t, ts, basic, "basic@example.com", "password123")
	resp, _ := ts.do(t, basic, http.MethodPost, "/api/admin/users",
		map[string]string{"email": "sneaky@example.com"}, nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("basic invite = %d, want 403", resp.StatusCode)
	}
}
