package main

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/thehelvijs/Reeve/server/internal/auth"
	"github.com/thehelvijs/Reeve/server/internal/store"
)

func TestResetPasswordCLISetsPasswordAndDropsSessions(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "cli.db")
	db, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	hash, _ := auth.HashPassword("original-password")
	u, err := db.CreateUser("boss@example.com", hash, store.RoleAdmin)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	sess, err := db.CreateSession(u.ID, time.Hour)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	db.Close()

	if err := resetPasswordCLI(dbPath, "  BOSS@Example.com ", "fresh-password"); err != nil {
		t.Fatalf("resetPasswordCLI: %v", err)
	}

	db2, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("reopen db: %v", err)
	}
	defer db2.Close()
	after, err := db2.GetUserByID(u.ID)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if auth.VerifyPassword("original-password", after.PasswordHash) {
		t.Error("old password still verifies after a reset")
	}
	if _, err := db2.GetSession(sess.ID); err == nil {
		t.Error("session survived a password reset")
	}
}

func TestResetPasswordCLIUnknownEmail(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "cli.db")
	db, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	db.Close()
	if err := resetPasswordCLI(dbPath, "nobody@example.com", "whatever"); err == nil {
		t.Fatal("resetPasswordCLI on an unknown email returned nil, want an error")
	}
	if err := resetPasswordCLI(dbPath, "nobody@example.com", ""); err == nil {
		t.Fatal("resetPasswordCLI without a password returned nil, want an error")
	}
}

// adminResetPassword posts a reset for the target user and returns the response.
func adminResetPassword(t *testing.T, ts *testServer, c *http.Client, userID, password string) (*http.Response, map[string]any) {
	t.Helper()
	body := map[string]string{"password": password}
	resp, data := ts.do(t, c, http.MethodPost, "/api/admin/users/"+userID+"/password", body, nil)
	out := map[string]any{}
	json.Unmarshal(data, &out)
	return resp, out
}

func TestAdminResetPasswordSetsItAndSignsTargetOut(t *testing.T) {
	ts := newTestServer(t)
	adminClient := ts.client(t)
	signup(t, ts, adminClient, "boss@example.com", "password123")
	userClient := ts.client(t)
	_, target := signup(t, ts, userClient, "dev@example.com", "password123")

	// The target has a live session; it must not survive the reset.
	resp, _ := ts.do(t, userClient, http.MethodGet, "/api/me", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("target session precondition status = %d, want 200", resp.StatusCode)
	}

	resp, _ = adminResetPassword(t, ts, adminClient, target.ID, "chosen-password")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("reset status = %d, want 200", resp.StatusCode)
	}

	resp, _ = ts.do(t, userClient, http.MethodGet, "/api/me", nil, nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("target session after reset status = %d, want 401", resp.StatusCode)
	}
	if resp := login(t, ts, "dev@example.com", "chosen-password", "10.0.0.5"); resp.StatusCode != http.StatusOK {
		t.Errorf("login with the new password status = %d, want 200", resp.StatusCode)
	}
	if resp := login(t, ts, "dev@example.com", "password123", "10.0.0.6"); resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("login with the old password status = %d, want 401", resp.StatusCode)
	}
}

func TestAdminResetPasswordRejects(t *testing.T) {
	ts := newTestServer(t)
	adminClient := ts.client(t)
	_, admin := signup(t, ts, adminClient, "boss@example.com", "password123")
	userClient := ts.client(t)
	_, target := signup(t, ts, userClient, "dev@example.com", "password123")

	if resp, _ := adminResetPassword(t, ts, adminClient, admin.ID, "whatever"); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("self reset status = %d, want 400", resp.StatusCode)
	}
	if resp, _ := adminResetPassword(t, ts, adminClient, target.ID, ""); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("empty password status = %d, want 400", resp.StatusCode)
	}
	if resp, _ := adminResetPassword(t, ts, adminClient, "no-such-user", "whatever"); resp.StatusCode != http.StatusNotFound {
		t.Errorf("unknown user status = %d, want 404", resp.StatusCode)
	}
	if resp, _ := adminResetPassword(t, ts, userClient, admin.ID, "whatever"); resp.StatusCode != http.StatusForbidden {
		t.Errorf("basic user status = %d, want 403", resp.StatusCode)
	}
}

// Every path that sets a password enforces the same floor; a reset link must
// not be the way around the rule signup applies.
func TestEveryPasswordPathEnforcesTheMinimum(t *testing.T) {
	ts := newTestServer(t)
	adminClient := ts.client(t)
	signup(t, ts, adminClient, "boss@example.com", "password123")
	userClient := ts.client(t)
	_, target := signup(t, ts, userClient, "dev@example.com", "password123")

	if resp, _ := adminResetPassword(t, ts, adminClient, target.ID, "short"); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("admin reset with a 5-char password = %d, want 400", resp.StatusCode)
	}
	if resp := doReset(t, ts, "any-token", "short", "10.2.0.1"); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("mailed reset with a 5-char password = %d, want 400", resp.StatusCode)
	}
	if err := setPassword(ts.app.db, target.ID, "short"); err == nil {
		t.Error("setPassword accepted a 5-char password")
	}
}

// A password change must sign other devices out while keeping the caller in.
func TestChangePasswordDropsOtherSessions(t *testing.T) {
	ts := newTestServer(t)
	first := ts.client(t)
	signup(t, ts, first, "a@b.com", "password123")
	second := ts.client(t)
	if resp := loginWith(t, ts, second, "a@b.com", "password123"); resp.StatusCode != http.StatusOK {
		t.Fatalf("second login status = %d, want 200", resp.StatusCode)
	}

	body := map[string]string{"current_password": "password123", "new_password": "brand-new-password"}
	resp, _ := ts.do(t, first, http.MethodPost, "/api/me/password", body, nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("change password status = %d, want 204", resp.StatusCode)
	}

	resp, _ = ts.do(t, first, http.MethodGet, "/api/me", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("caller session after change status = %d, want 200", resp.StatusCode)
	}
	resp, _ = ts.do(t, second, http.MethodGet, "/api/me", nil, nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("other session after change status = %d, want 401", resp.StatusCode)
	}
}
