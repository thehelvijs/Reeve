package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"path/filepath"
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
