package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/thehelvijs/Reeve/server/internal/crypto"
	"github.com/thehelvijs/Reeve/server/internal/store"
)

// testServer spins up the full app against a temp DB and an httptest server.
type testServer struct {
	app *app
	srv *httptest.Server
}

func newTestServer(t *testing.T) *testServer {
	return newTestServerKey(t, bytes.Repeat([]byte{3}, 32))
}

func newTestServerKey(t *testing.T, key []byte) *testServer {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	cipher, err := crypto.New(key)
	if err != nil {
		t.Fatalf("cipher: %v", err)
	}
	db, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	a := &app{db: db, cipher: cipher, cfg: config{DBPath: dbPath, SessionTTL: time.Hour, Version: "test"}, startedAt: time.Now().UTC(), agentFS: agentDistFS(), scriptFS: installScripts}
	srv := httptest.NewServer(a.routes())
	t.Cleanup(srv.Close)
	return &testServer{app: a, srv: srv}
}

// client returns an http.Client with a cookie jar so sessions persist.
func (ts *testServer) client(t *testing.T) *http.Client {
	t.Helper()
	jar, _ := cookiejar.New(nil)
	return &http.Client{Jar: jar}
}

func (ts *testServer) do(t *testing.T, c *http.Client, method, path string, body any, hdrs map[string]string) (*http.Response, []byte) {
	t.Helper()
	var rdr io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, ts.srv.URL+path, rdr)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	// Same-origin by default, as a browser on this server would be; a test
	// exercising the origin check overrides it through hdrs.
	req.Header.Set("Origin", ts.srv.URL)
	for k, v := range hdrs {
		req.Header.Set(k, v)
	}
	if c == nil {
		c = &http.Client{}
	}
	resp, err := c.Do(req)
	if err != nil {
		t.Fatalf("do %s %s: %v", method, path, err)
	}
	data, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	return resp, data
}

func TestHealthz(t *testing.T) {
	ts := newTestServer(t)
	resp, data := ts.do(t, nil, http.MethodGet, "/healthz", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var body map[string]string
	json.Unmarshal(data, &body)
	if body["status"] != "ok" {
		t.Errorf("unexpected body: %s", data)
	}
}

// The cookie name is part of the deployment's surface (proxies, browser state),
// so pin it: renaming it signs everyone out and must be deliberate.
func TestLoginSetsTheReeveSessionCookie(t *testing.T) {
	ts := newTestServer(t)
	c := ts.client(t)
	signup(t, ts, c, "boss@example.com", "password123")

	resp, data := ts.do(t, c, http.MethodPost, "/api/v1/auth/login",
		map[string]string{"email": "boss@example.com", "password": "password123"}, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login = %d: %s", resp.StatusCode, data)
	}
	var found *http.Cookie
	for _, ck := range resp.Cookies() {
		if ck.Name == "reeve_session" {
			found = ck
		}
	}
	if found == nil {
		t.Fatalf("no reeve_session cookie in %v", resp.Cookies())
	}
	if !found.HttpOnly {
		t.Error("session cookie is not HttpOnly")
	}
	if sessionCookie != "reeve_session" {
		t.Errorf("sessionCookie = %q, want reeve_session", sessionCookie)
	}
	if oauthStateCookie != "reeve_oauth_state" {
		t.Errorf("oauthStateCookie = %q, want reeve_oauth_state", oauthStateCookie)
	}
}

func captureLog(t *testing.T) (*bytes.Buffer, func()) {
	t.Helper()
	buf := &bytes.Buffer{}
	prevOut := log.Writer()
	prevFlags := log.Flags()
	log.SetOutput(buf)
	log.SetFlags(0)
	return buf, func() {
		log.SetOutput(prevOut)
		log.SetFlags(prevFlags)
	}
}

func TestLogRequestsLogsApiNotStatic(t *testing.T) {
	buf, restore := captureLog(t)
	defer restore()
	a := &app{cfg: config{LogRequests: true}}
	h := a.logRequests(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/tools", nil))
	out := buf.String()
	if !strings.Contains(out, "GET /api/v1/tools") || !strings.Contains(out, "418") {
		t.Errorf("expected api log line with method/path/status, got %q", out)
	}

	buf.Reset()
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/assets/app.js", nil))
	if buf.String() != "" {
		t.Errorf("static path should not be logged, got %q", buf.String())
	}
}

func TestLogRequestsToggleOff(t *testing.T) {
	buf, restore := captureLog(t)
	defer restore()
	a := &app{cfg: config{LogRequests: false}}
	h := a.logRequests(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/tools", nil))
	if buf.String() != "" {
		t.Errorf("logging disabled but produced %q", buf.String())
	}
}

func TestClientIPForwardedForOnlyWhenProxyTrusted(t *testing.T) {
	spoofed := httptest.NewRequest(http.MethodGet, "/api/v1/tools", nil)
	spoofed.Header.Set("X-Forwarded-For", "203.0.113.9, 10.0.0.1")
	spoofed.RemoteAddr = "192.168.1.5:54321"

	untrusting := &app{cfg: config{}}
	if got := untrusting.clientIP(spoofed); got != "192.168.1.5" {
		t.Errorf("clientIP = %q, want the peer address 192.168.1.5 when no proxy is trusted", got)
	}

	trusting := &app{cfg: config{TrustProxyHeaders: true}}
	if got := trusting.clientIP(spoofed); got != "203.0.113.9" {
		t.Errorf("clientIP = %q, want 203.0.113.9", got)
	}

	bare := httptest.NewRequest(http.MethodGet, "/api/v1/tools", nil)
	bare.RemoteAddr = "192.168.1.5:54321"
	if got := trusting.clientIP(bare); got != "192.168.1.5" {
		t.Errorf("clientIP = %q, want 192.168.1.5", got)
	}
}

func TestRequireSameOriginRejectsCrossSiteCookieWrite(t *testing.T) {
	a := &app{cfg: config{PublicURL: "http://reeve.lan:8080"}}
	reached := false
	h := a.requireSameOrigin(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) { reached = true }))

	withCookie := func(method, origin string) *http.Request {
		r := httptest.NewRequest(method, "http://reeve.lan:8080/api/v1/credentials/c1/reveal", nil)
		r.AddCookie(&http.Cookie{Name: sessionCookie, Value: "s1"})
		if origin != "" {
			r.Header.Set("Origin", origin)
		}
		return r
	}

	cases := []struct {
		name       string
		req        *http.Request
		wantPassed bool
	}{
		{"cross-origin post", withCookie(http.MethodPost, "http://evil.lan"), false},
		{"same-host different port", withCookie(http.MethodPost, "http://reeve.lan:9999"), false},
		{"no origin header", withCookie(http.MethodPost, ""), false},
		{"matching public url", withCookie(http.MethodPost, "http://reeve.lan:8080"), true},
		{"safe method", withCookie(http.MethodGet, "http://evil.lan"), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reached = false
			w := httptest.NewRecorder()
			h.ServeHTTP(w, tc.req)
			if reached != tc.wantPassed {
				t.Errorf("handler reached = %v, want %v (status %d)", reached, tc.wantPassed, w.Code)
			}
		})
	}
}

// The agent authenticates with a bearer token and sends no Origin, so ingest
// must stay reachable; only a session cookie brings the browser's ambient
// credential into play.
func TestRequireSameOriginLeavesTokenCallersAlone(t *testing.T) {
	a := &app{cfg: config{}}
	reached := false
	h := a.requireSameOrigin(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) { reached = true }))
	r := httptest.NewRequest(http.MethodPost, "/api/v1/ingest", nil)
	r.Header.Set("Authorization", "Bearer rva_deadbeef")
	h.ServeHTTP(httptest.NewRecorder(), r)
	if !reached {
		t.Error("token-authenticated ingest was rejected by the origin check")
	}
}

func TestSecurityHeaders(t *testing.T) {
	h := securityHeaders(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	for header, want := range map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"Referrer-Policy":        "no-referrer",
	} {
		if got := w.Header().Get(header); got != want {
			t.Errorf("%s = %q, want %q", header, got, want)
		}
	}
	csp := w.Header().Get("Content-Security-Policy")
	for _, want := range []string{"script-src 'self'", "frame-ancestors 'none'", "object-src 'none'"} {
		if !strings.Contains(csp, want) {
			t.Errorf("CSP %q is missing %q", csp, want)
		}
	}
}
