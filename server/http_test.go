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
	a.notifiers = newNotifiers()
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

func TestClientIPForwardedFor(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/tools", nil)
	r.Header.Set("X-Forwarded-For", "203.0.113.9, 10.0.0.1")
	if got := clientIP(r); got != "203.0.113.9" {
		t.Errorf("clientIP = %q, want 203.0.113.9", got)
	}
	r2 := httptest.NewRequest(http.MethodGet, "/api/v1/tools", nil)
	r2.RemoteAddr = "192.168.1.5:54321"
	if got := clientIP(r2); got != "192.168.1.5" {
		t.Errorf("clientIP = %q, want 192.168.1.5", got)
	}
}
