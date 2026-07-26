package main

import (
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// stubGoogle serves the token and userinfo legs and a stand-in consent screen.
func stubGoogle(t *testing.T, profileJSON string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/auth", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("consent screen"))
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"access_token":"access-789","token_type":"Bearer"}`))
	})
	mux.HandleFunc("/userinfo", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(profileJSON))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// enableGoogle points the app at the stub provider and turns sign-in on.
func enableGoogle(t *testing.T, ts *testServer, provider *httptest.Server, domains string) {
	t.Helper()
	ts.app.cfg.PublicURL = ts.srv.URL
	ts.app.googleEndpoints = &oauthEndpoints{
		Auth:     provider.URL + "/auth",
		Token:    provider.URL + "/token",
		UserInfo: provider.URL + "/userinfo",
	}
	admin := ts.client(t)
	signup(t, ts, admin, "boss@example.com", "password123")
	resp, v := putSettings(t, ts, admin, map[string]any{"google": map[string]any{
		"enabled": true, "client_id": "client-123", "client_secret": "secret-456",
		"allowed_domains": domains,
	}})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("enable google status = %d, want 200", resp.StatusCode)
	}
	if !v.Google.Enabled || !v.Google.SecretSet {
		t.Fatalf("google settings after save = %+v", v.Google)
	}
}

// noFollow returns a client that surfaces redirects instead of chasing them.
func noFollow(t *testing.T) *http.Client {
	t.Helper()
	jar, _ := cookiejar.New(nil)
	return &http.Client{
		Jar:           jar,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
}

// startSignIn walks the /start leg and returns the state it minted.
func startSignIn(t *testing.T, ts *testServer, c *http.Client) (*http.Response, string) {
	t.Helper()
	resp, err := c.Get(ts.srv.URL + "/api/v1/auth/google/start")
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	resp.Body.Close()
	loc := resp.Header.Get("Location")
	state := ""
	if u, err := url.Parse(loc); err == nil {
		state = u.Query().Get("state")
	}
	return resp, state
}

func callback(t *testing.T, ts *testServer, c *http.Client, code, state string) *http.Response {
	t.Helper()
	q := url.Values{"code": {code}, "state": {state}}
	resp, err := c.Get(ts.srv.URL + "/api/v1/auth/google/callback?" + q.Encode())
	if err != nil {
		t.Fatalf("callback: %v", err)
	}
	resp.Body.Close()
	return resp
}

const verifiedProfile = `{"sub":"1","email":"dev@example.com","email_verified":true,"name":"Dev Example"}`

func TestGoogleStartRedirectsWithState(t *testing.T) {
	ts := newTestServer(t)
	provider := stubGoogle(t, verifiedProfile)
	enableGoogle(t, ts, provider, "")

	c := noFollow(t)
	resp, state := startSignIn(t, ts, c)
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("start status = %d, want 302", resp.StatusCode)
	}
	if !strings.HasPrefix(resp.Header.Get("Location"), provider.URL+"/auth") {
		t.Errorf("Location = %q", resp.Header.Get("Location"))
	}
	if state == "" {
		t.Error("no state in the provider URL")
	}
	var stateCookie string
	for _, ck := range resp.Cookies() {
		if ck.Name == oauthStateCookie {
			stateCookie = ck.Value
		}
	}
	if stateCookie != state {
		t.Errorf("state cookie = %q, want it to match the URL state %q", stateCookie, state)
	}
}

func TestGoogleSignInProvisionsAccount(t *testing.T) {
	ts := newTestServer(t)
	enableGoogle(t, ts, stubGoogle(t, verifiedProfile), "")

	c := noFollow(t)
	_, state := startSignIn(t, ts, c)
	resp := callback(t, ts, c, "code-1", state)
	if resp.StatusCode != http.StatusFound || resp.Header.Get("Location") != "/dashboard" {
		t.Fatalf("callback status = %d, location = %q", resp.StatusCode, resp.Header.Get("Location"))
	}

	u, err := ts.app.db.GetUserByEmail("dev@example.com")
	if err != nil {
		t.Fatalf("provisioned user missing: %v", err)
	}
	if u.Role != "basic" {
		t.Errorf("role = %q, want basic", u.Role)
	}
	if u.DisplayName != "Dev Example" {
		t.Errorf("display name = %q, want the provider's name", u.DisplayName)
	}
	// The session cookie works, and the account has no usable password.
	me, _ := ts.do(t, c, http.MethodGet, "/api/v1/me", nil, nil)
	if me.StatusCode != http.StatusOK {
		t.Errorf("me after google sign-in status = %d, want 200", me.StatusCode)
	}
	if resp := login(t, ts, "dev@example.com", "", "10.9.0.1"); resp.StatusCode == http.StatusOK {
		t.Error("provisioned account accepted an empty password")
	}
}

func TestGoogleSignInReusesExistingAccount(t *testing.T) {
	ts := newTestServer(t)
	enableGoogle(t, ts, stubGoogle(t, verifiedProfile), "")
	signup(t, ts, ts.client(t), "dev@example.com", "password123")
	before, _ := ts.app.db.CountUsers()

	c := noFollow(t)
	_, state := startSignIn(t, ts, c)
	if resp := callback(t, ts, c, "code-1", state); resp.Header.Get("Location") != "/dashboard" {
		t.Fatalf("location = %q", resp.Header.Get("Location"))
	}
	after, _ := ts.app.db.CountUsers()
	if after != before {
		t.Errorf("user count %d -> %d, want no new account", before, after)
	}
}

// The first federated sign-in pins the provider's subject to the account.
func TestGoogleSignInPinsTheSubject(t *testing.T) {
	ts := newTestServer(t)
	enableGoogle(t, ts, stubGoogle(t, verifiedProfile), "")
	signup(t, ts, ts.client(t), "dev@example.com", "password123")

	c := noFollow(t)
	_, state := startSignIn(t, ts, c)
	callback(t, ts, c, "code-1", state)

	u, err := ts.app.db.GetUserByEmail("dev@example.com")
	if err != nil {
		t.Fatalf("load user: %v", err)
	}
	if u.OAuthSubject != "1" {
		t.Errorf("oauth subject = %q, want %q", u.OAuthSubject, "1")
	}
}

// An email address can be reassigned by whoever runs the domain, so a second
// Google identity presenting the same address must not inherit the account.
func TestGoogleSignInRefusesADifferentSubjectForTheSameEmail(t *testing.T) {
	ts := newTestServer(t)
	enableGoogle(t, ts, stubGoogle(t, verifiedProfile), "")
	signup(t, ts, ts.client(t), "dev@example.com", "password123")

	first := noFollow(t)
	_, state := startSignIn(t, ts, first)
	if resp := callback(t, ts, first, "code-1", state); resp.Header.Get("Location") != "/dashboard" {
		t.Fatalf("first sign-in location = %q, want /dashboard", resp.Header.Get("Location"))
	}

	// Same address, different Google account.
	ts.app.googleEndpoints.UserInfo = stubGoogle(t,
		`{"sub":"impostor","email":"dev@example.com","email_verified":true}`).URL + "/userinfo"
	second := noFollow(t)
	_, state2 := startSignIn(t, ts, second)
	resp := callback(t, ts, second, "code-2", state2)
	if got := resp.Header.Get("Location"); got != "/login?oauth_error=subject_mismatch" {
		t.Errorf("location = %q, want the subject_mismatch refusal", got)
	}
	for _, ck := range resp.Cookies() {
		if ck.Name == sessionCookie && ck.Value != "" {
			t.Error("a session was issued to the impostor subject")
		}
	}
}

// State is the CSRF defense on the callback, so a mismatch must not sign in.
func TestGoogleCallbackRejectsStateMismatch(t *testing.T) {
	ts := newTestServer(t)
	enableGoogle(t, ts, stubGoogle(t, verifiedProfile), "")

	c := noFollow(t)
	startSignIn(t, ts, c)
	resp := callback(t, ts, c, "code-1", "not-the-state")
	assertLoginError(t, resp, "state_mismatch")

	fresh := noFollow(t)
	resp = callback(t, ts, fresh, "code-1", "any-state")
	assertLoginError(t, resp, "state_mismatch")
	if _, err := ts.app.db.GetUserByEmail("dev@example.com"); err == nil {
		t.Error("a rejected callback still provisioned an account")
	}
}

func TestGoogleCallbackEnforcesDomainAllowList(t *testing.T) {
	ts := newTestServer(t)
	enableGoogle(t, ts, stubGoogle(t, verifiedProfile), "allowed.com")

	c := noFollow(t)
	_, state := startSignIn(t, ts, c)
	assertLoginError(t, callback(t, ts, c, "code-1", state), "domain_not_allowed")
	if _, err := ts.app.db.GetUserByEmail("dev@example.com"); err == nil {
		t.Error("account provisioned for a disallowed domain")
	}
}

func TestGoogleCallbackRejectsUnverifiedEmail(t *testing.T) {
	ts := newTestServer(t)
	enableGoogle(t, ts, stubGoogle(t, `{"sub":"1","email":"dev@example.com","email_verified":false}`), "")

	c := noFollow(t)
	_, state := startSignIn(t, ts, c)
	assertLoginError(t, callback(t, ts, c, "code-1", state), "profile_failed")
}

// With signup closed and no domain allow-list, Google may only sign in people
// who already have an account.
func TestGoogleCallbackHonorsClosedSignup(t *testing.T) {
	ts := newTestServer(t)
	enableGoogle(t, ts, stubGoogle(t, verifiedProfile), "")
	admin := ts.client(t)
	if resp := loginWith(t, ts, admin, "boss@example.com", "password123"); resp.StatusCode != http.StatusOK {
		t.Fatalf("admin login status = %d", resp.StatusCode)
	}
	putSettings(t, ts, admin, map[string]any{"signup_enabled": false})

	c := noFollow(t)
	_, state := startSignIn(t, ts, c)
	assertLoginError(t, callback(t, ts, c, "code-1", state), "no_account")
}

// A domain allow-list is itself the admin's intent to admit that domain, so
// provisioning stays on even with signup closed.
func TestGoogleCallbackProvisionsForAllowedDomainWithClosedSignup(t *testing.T) {
	ts := newTestServer(t)
	enableGoogle(t, ts, stubGoogle(t, verifiedProfile), "example.com")
	admin := ts.client(t)
	loginWith(t, ts, admin, "boss@example.com", "password123")
	putSettings(t, ts, admin, map[string]any{"signup_enabled": false})

	c := noFollow(t)
	_, state := startSignIn(t, ts, c)
	if resp := callback(t, ts, c, "code-1", state); resp.Header.Get("Location") != "/dashboard" {
		t.Fatalf("location = %q, want /dashboard", resp.Header.Get("Location"))
	}
}

func TestGoogleCallbackRejectsDeactivatedAccount(t *testing.T) {
	ts := newTestServer(t)
	enableGoogle(t, ts, stubGoogle(t, verifiedProfile), "")
	_, target := signup(t, ts, ts.client(t), "dev@example.com", "password123")
	if err := ts.app.db.SetUserActive(target.ID, false); err != nil {
		t.Fatalf("deactivate: %v", err)
	}

	c := noFollow(t)
	_, state := startSignIn(t, ts, c)
	assertLoginError(t, callback(t, ts, c, "code-1", state), "account_disabled")
}

// Google matches redirect_uri exactly, so without a configured public URL the
// flow must refuse rather than send a host-derived URI it never registered.
func TestGoogleDisabledWithoutPublicURL(t *testing.T) {
	ts := newTestServer(t)
	enableGoogle(t, ts, stubGoogle(t, verifiedProfile), "")
	ts.app.cfg.PublicURL = ""

	c := noFollow(t)
	resp, _ := startSignIn(t, ts, c)
	assertLoginError(t, resp, "google_not_configured")

	_, data := ts.do(t, nil, http.MethodGet, "/api/v1/auth/status", nil, nil)
	if strings.Contains(string(data), `"google_enabled":true`) {
		t.Error("auth status advertises google sign-in with no public URL")
	}
}

func TestGoogleSettingsSecrecyAndValidation(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.client(t)
	signup(t, ts, admin, "boss@example.com", "password123")
	ts.app.cfg.PublicURL = ts.srv.URL

	resp, _ := putSettings(t, ts, admin, map[string]any{"google": map[string]any{
		"enabled": true, "client_id": "", "client_secret": "",
	}})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("enabling with no client id status = %d, want 400", resp.StatusCode)
	}

	putSettings(t, ts, admin, map[string]any{"google": map[string]any{
		"enabled": true, "client_id": "client-123", "client_secret": "secret-456",
		"allowed_domains": "Example.com, @other.com",
	}})
	_, data := ts.do(t, admin, http.MethodGet, "/api/v1/admin/settings", nil, nil)
	if strings.Contains(string(data), "secret-456") {
		t.Error("settings response leaked the client secret")
	}
	if raw, _ := ts.app.db.GetSetting(settingGoogleSecret); strings.Contains(raw, "secret-456") {
		t.Error("client secret stored in plaintext")
	}
	if got := ts.app.settingOr(settingGoogleDomains, ""); got != "example.com,other.com" {
		t.Errorf("normalized domains = %q", got)
	}
	// A save without a secret keeps the stored one.
	putSettings(t, ts, admin, map[string]any{"google": map[string]any{
		"enabled": true, "client_id": "client-123", "allowed_domains": "example.com",
	}})
	if got, _ := ts.app.sealedSetting(settingGoogleSecret); got != "secret-456" {
		t.Errorf("secret after a secretless save = %q, want it unchanged", got)
	}
}

func assertLoginError(t *testing.T, resp *http.Response, wantCode string) {
	t.Helper()
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("status = %d, want 302", resp.StatusCode)
	}
	loc := resp.Header.Get("Location")
	if !strings.Contains(loc, "oauth_error="+wantCode) {
		t.Fatalf("Location = %q, want oauth_error=%s", loc, wantCode)
	}
}
