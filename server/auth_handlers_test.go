package main

import (
	"encoding/json"
	"net/http"
	"testing"
)

func signup(t *testing.T, ts *testServer, c *http.Client, email, pass string) (*http.Response, userView) {
	t.Helper()
	resp, data := ts.do(t, c, http.MethodPost, "/api/v1/auth/signup",
		credentials{Email: email, Password: pass}, nil)
	var v userView
	json.Unmarshal(data, &v)
	return resp, v
}

func TestFirstUserIsAdmin(t *testing.T) {
	ts := newTestServer(t)
	c := ts.client(t)
	resp, v := signup(t, ts, c, "boss@example.com", "password123")
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("signup status = %d", resp.StatusCode)
	}
	if v.Role != "admin" {
		t.Errorf("first user role = %q, want admin", v.Role)
	}
}

func TestSecondUserIsBasic(t *testing.T) {
	ts := newTestServer(t)
	signup(t, ts, ts.client(t), "boss@example.com", "password123")
	_, v := signup(t, ts, ts.client(t), "dev@example.com", "password123")
	if v.Role != "basic" {
		t.Errorf("second user role = %q, want basic", v.Role)
	}
}

func TestSignupWeakPassword(t *testing.T) {
	ts := newTestServer(t)
	resp, _ := signup(t, ts, ts.client(t), "a@b.com", "short")
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

func TestSignupDuplicateEmail(t *testing.T) {
	ts := newTestServer(t)
	signup(t, ts, ts.client(t), "a@b.com", "password123")
	resp, _ := signup(t, ts, ts.client(t), "a@b.com", "password123")
	if resp.StatusCode != http.StatusConflict {
		t.Errorf("status = %d, want 409", resp.StatusCode)
	}
}

func TestSignupDisabledBlocksNonFirst(t *testing.T) {
	ts := newTestServer(t)
	signup(t, ts, ts.client(t), "boss@example.com", "password123") // first = admin, always allowed
	ts.app.db.SetSetting(settingSignupEnabled, "false")
	resp, _ := signup(t, ts, ts.client(t), "dev@example.com", "password123")
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("status = %d, want 403", resp.StatusCode)
	}
}

func TestLoginGoodAndBad(t *testing.T) {
	ts := newTestServer(t)
	signup(t, ts, ts.client(t), "a@b.com", "password123")

	resp, _ := ts.do(t, ts.client(t), http.MethodPost, "/api/v1/auth/login",
		credentials{Email: "a@b.com", Password: "password123"}, nil)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("good login status = %d, want 200", resp.StatusCode)
	}
	resp, _ = ts.do(t, ts.client(t), http.MethodPost, "/api/v1/auth/login",
		credentials{Email: "a@b.com", Password: "wrong"}, nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("bad login status = %d, want 401", resp.StatusCode)
	}
}

func TestMeRequiresAuth(t *testing.T) {
	ts := newTestServer(t)
	resp, _ := ts.do(t, ts.client(t), http.MethodGet, "/api/v1/me", nil, nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
}

func TestSessionCookieFlowAndLogout(t *testing.T) {
	ts := newTestServer(t)
	c := ts.client(t)
	signup(t, ts, c, "a@b.com", "password123")

	resp, data := ts.do(t, c, http.MethodGet, "/api/v1/me", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("me status = %d", resp.StatusCode)
	}
	var v userView
	json.Unmarshal(data, &v)
	if v.Email != "a@b.com" {
		t.Errorf("me email = %q", v.Email)
	}

	ts.do(t, c, http.MethodPost, "/api/v1/auth/logout", nil, nil)
	resp, _ = ts.do(t, c, http.MethodGet, "/api/v1/me", nil, nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("after logout status = %d, want 401", resp.StatusCode)
	}
}

func TestAuthStatusSetupFlag(t *testing.T) {
	ts := newTestServer(t)
	// No users yet -> setup required.
	_, data := ts.do(t, nil, http.MethodGet, "/api/v1/auth/status", nil, nil)
	var s map[string]bool
	json.Unmarshal(data, &s)
	if !s["setup_required"] {
		t.Errorf("fresh server should require setup: %v", s)
	}
	// After the first signup -> no longer required.
	signup(t, ts, ts.client(t), "boss@example.com", "password123")
	_, data = ts.do(t, nil, http.MethodGet, "/api/v1/auth/status", nil, nil)
	json.Unmarshal(data, &s)
	if s["setup_required"] {
		t.Errorf("setup should be complete after first user: %v", s)
	}
}

// fromIP builds the header map that pins a request's apparent source IP.
func fromIP(ip string) map[string]string {
	return map[string]string{"X-Forwarded-For": ip}
}

func login(t *testing.T, ts *testServer, email, pass, ip string) *http.Response {
	t.Helper()
	resp, _ := ts.do(t, ts.client(t), http.MethodPost, "/api/v1/auth/login",
		credentials{Email: email, Password: pass}, fromIP(ip))
	return resp
}

// loginWith logs in on a caller-supplied client so its cookie jar keeps the
// resulting session.
func loginWith(t *testing.T, ts *testServer, c *http.Client, email, pass string) *http.Response {
	t.Helper()
	resp, _ := ts.do(t, c, http.MethodPost, "/api/v1/auth/login",
		credentials{Email: email, Password: pass}, nil)
	return resp
}

func TestLoginLocksOutAfterRepeatedFailures(t *testing.T) {
	ts := newTestServer(t)
	signup(t, ts, ts.client(t), "a@b.com", "password123")

	for i := 0; i < 5; i++ {
		if resp := login(t, ts, "a@b.com", "wrong", "10.0.0.1"); resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("attempt %d status = %d, want 401", i+1, resp.StatusCode)
		}
	}
	resp := login(t, ts, "a@b.com", "wrong", "10.0.0.1")
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("attempt 6 status = %d, want 429", resp.StatusCode)
	}
	if resp.Header.Get("Retry-After") == "" {
		t.Error("429 response carries no Retry-After header")
	}
	// The lockout must hold even for the right password, or it is no lockout.
	if resp := login(t, ts, "a@b.com", "password123", "10.0.0.1"); resp.StatusCode != http.StatusTooManyRequests {
		t.Errorf("correct password during lockout status = %d, want 429", resp.StatusCode)
	}
}

func TestLoginLockoutFollowsTheEmailAcrossIPs(t *testing.T) {
	ts := newTestServer(t)
	signup(t, ts, ts.client(t), "a@b.com", "password123")
	for i := 0; i < 5; i++ {
		login(t, ts, "a@b.com", "wrong", "10.0.0.1")
	}
	if resp := login(t, ts, "a@b.com", "password123", "10.0.0.9"); resp.StatusCode != http.StatusTooManyRequests {
		t.Errorf("status from a new IP = %d, want 429 (email is locked)", resp.StatusCode)
	}
}

func TestLoginLockoutDoesNotAffectOtherAccounts(t *testing.T) {
	ts := newTestServer(t)
	signup(t, ts, ts.client(t), "a@b.com", "password123")
	signup(t, ts, ts.client(t), "c@d.com", "password123")
	for i := 0; i < 5; i++ {
		login(t, ts, "a@b.com", "wrong", "10.0.0.1")
	}
	if resp := login(t, ts, "c@d.com", "password123", "10.0.0.2"); resp.StatusCode != http.StatusOK {
		t.Errorf("unrelated account status = %d, want 200", resp.StatusCode)
	}
}

func TestLoginSuccessClearsFailureCount(t *testing.T) {
	ts := newTestServer(t)
	signup(t, ts, ts.client(t), "a@b.com", "password123")
	for i := 0; i < 4; i++ {
		login(t, ts, "a@b.com", "wrong", "10.0.0.1")
	}
	if resp := login(t, ts, "a@b.com", "password123", "10.0.0.1"); resp.StatusCode != http.StatusOK {
		t.Fatalf("login before lockout status = %d, want 200", resp.StatusCode)
	}
	for i := 0; i < 4; i++ {
		if resp := login(t, ts, "a@b.com", "wrong", "10.0.0.1"); resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("post-reset attempt %d status = %d, want 401", i+1, resp.StatusCode)
		}
	}
}
