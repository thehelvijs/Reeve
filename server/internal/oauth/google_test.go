package oauth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func testConfig(base string) Config {
	return Config{
		ClientID:     "client-123",
		ClientSecret: "secret-456",
		RedirectURL:  "http://reeve.lan:8080/api/v1/auth/google/callback",
		AuthURL:      base + "/auth",
		TokenURL:     base + "/token",
		UserInfoURL:  base + "/userinfo",
	}
}

func TestValidate(t *testing.T) {
	c := testConfig("http://x")
	if err := c.Validate(); err != nil {
		t.Fatalf("Validate = %v, want nil", err)
	}
	for name, mutate := range map[string]func(*Config){
		"no client id":     func(c *Config) { c.ClientID = "" },
		"no client secret": func(c *Config) { c.ClientSecret = "" },
		"no redirect":      func(c *Config) { c.RedirectURL = "" },
	} {
		bad := testConfig("http://x")
		mutate(&bad)
		if err := bad.Validate(); err == nil {
			t.Errorf("%s: Validate = nil, want an error", name)
		}
	}
}

func TestAuthCodeURL(t *testing.T) {
	c := testConfig("http://provider")
	raw := c.AuthCodeURL("state-abc")
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	q := u.Query()
	want := map[string]string{
		"client_id":     "client-123",
		"redirect_uri":  c.RedirectURL,
		"response_type": "code",
		"state":         "state-abc",
	}
	for k, v := range want {
		if q.Get(k) != v {
			t.Errorf("%s = %q, want %q", k, q.Get(k), v)
		}
	}
	if !strings.Contains(q.Get("scope"), "email") {
		t.Errorf("scope = %q, want it to include email", q.Get("scope"))
	}
	if c.ClientSecret != "" && strings.Contains(raw, c.ClientSecret) {
		t.Error("auth URL leaks the client secret")
	}
}

func TestGoogleUsesProductionEndpoints(t *testing.T) {
	c := Google("id", "secret", "http://reeve.lan/cb")
	if c.AuthURL != GoogleAuthURL || c.TokenURL != GoogleTokenURL || c.UserInfoURL != GoogleUserInfoURL {
		t.Errorf("Google() endpoints = %q %q %q", c.AuthURL, c.TokenURL, c.UserInfoURL)
	}
}

// stubProvider serves the token and userinfo legs.
func stubProvider(t *testing.T, tokenBody, userInfoBody string, tokenStatus, userStatus int) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		if r.FormValue("grant_type") != "authorization_code" {
			t.Errorf("grant_type = %q", r.FormValue("grant_type"))
		}
		if r.FormValue("client_secret") != "secret-456" {
			t.Errorf("client_secret = %q", r.FormValue("client_secret"))
		}
		w.WriteHeader(tokenStatus)
		w.Write([]byte(tokenBody))
	})
	mux.HandleFunc("/userinfo", func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer access-789" {
			t.Errorf("Authorization = %q", got)
		}
		w.WriteHeader(userStatus)
		w.Write([]byte(userInfoBody))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestExchangeAndUserInfo(t *testing.T) {
	srv := stubProvider(t,
		`{"access_token":"access-789","token_type":"Bearer"}`,
		`{"sub":"1","email":"Dev@Example.com","email_verified":true,"name":"Dev"}`,
		200, 200)
	c := testConfig(srv.URL)

	token, err := c.Exchange(context.Background(), "code-1")
	if err != nil {
		t.Fatalf("Exchange: %v", err)
	}
	if token != "access-789" {
		t.Fatalf("token = %q", token)
	}
	p, err := c.UserInfo(context.Background(), token)
	if err != nil {
		t.Fatalf("UserInfo: %v", err)
	}
	if p.Email != "dev@example.com" {
		t.Errorf("email = %q, want it lowercased", p.Email)
	}
	if p.Name != "Dev" || p.Subject != "1" {
		t.Errorf("profile = %+v", p)
	}
}

func TestExchangeFailures(t *testing.T) {
	if _, err := testConfig("http://x").Exchange(context.Background(), ""); err == nil {
		t.Error("empty code returned nil error")
	}
	srv := stubProvider(t, `{"error":"invalid_grant"}`, `{}`, 400, 200)
	if _, err := testConfig(srv.URL).Exchange(context.Background(), "code"); err == nil {
		t.Error("provider 400 returned nil error")
	}
	srv2 := stubProvider(t, `{"token_type":"Bearer"}`, `{}`, 200, 200)
	if _, err := testConfig(srv2.URL).Exchange(context.Background(), "code"); err == nil {
		t.Error("missing access_token returned nil error")
	}
}

// An unverified provider email would let anyone claim a colleague's address.
func TestUserInfoRejectsUnverifiedOrEmptyEmail(t *testing.T) {
	cases := map[string]string{
		"unverified": `{"sub":"1","email":"dev@example.com","email_verified":false}`,
		"no email":   `{"sub":"1","email_verified":true}`,
		"not email":  `{"sub":"1","email":"dev","email_verified":true}`,
	}
	for name, body := range cases {
		srv := stubProvider(t, `{"access_token":"access-789"}`, body, 200, 200)
		if _, err := testConfig(srv.URL).UserInfo(context.Background(), "access-789"); err == nil {
			t.Errorf("%s: UserInfo = nil error, want a rejection", name)
		}
	}
}

func TestUserInfoProviderError(t *testing.T) {
	srv := stubProvider(t, `{"access_token":"access-789"}`, `nope`, 200, 401)
	if _, err := testConfig(srv.URL).UserInfo(context.Background(), "access-789"); err == nil {
		t.Error("provider 401 returned nil error")
	}
}

func TestDomainAllowed(t *testing.T) {
	cases := []struct {
		email   string
		allowed []string
		want    bool
	}{
		{"dev@example.com", nil, true},
		{"dev@example.com", []string{"example.com"}, true},
		{"dev@EXAMPLE.com", []string{"example.com"}, true},
		{"dev@example.com", []string{"@example.com"}, true},
		{"dev@other.com", []string{"example.com"}, false},
		{"dev@other.com", []string{"example.com", "other.com"}, true},
		{"notanemail", []string{"example.com"}, false},
		{"dev@sub.example.com", []string{"example.com"}, false},
	}
	for _, tc := range cases {
		if got := DomainAllowed(tc.email, tc.allowed); got != tc.want {
			t.Errorf("DomainAllowed(%q, %v) = %v, want %v", tc.email, tc.allowed, got, tc.want)
		}
	}
}

func TestParseDomains(t *testing.T) {
	got := ParseDomains(" Example.com, @other.com;third.com  ")
	want := []string{"example.com", "other.com", "third.com"}
	if len(got) != len(want) {
		t.Fatalf("ParseDomains = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ParseDomains = %v, want %v", got, want)
		}
	}
	if len(ParseDomains("   ")) != 0 {
		t.Error("blank input produced domains")
	}
}
