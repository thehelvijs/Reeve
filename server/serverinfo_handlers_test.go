package main

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestServerInfoAdminOnly(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.client(t)
	signup(t, ts, admin, "boss@example.com", "password123") // first user = admin

	resp, data := ts.do(t, admin, http.MethodGet, "/api/v1/admin/server-info", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("admin server-info = %d: %s", resp.StatusCode, data)
	}
	var info map[string]any
	if err := json.Unmarshal(data, &info); err != nil {
		t.Fatalf("bad json: %v", err)
	}
	for _, key := range []string{"version", "uptime_secs", "go", "db", "counts", "host"} {
		if _, ok := info[key]; !ok {
			t.Errorf("server-info missing %q: %s", key, data)
		}
	}

	basic := ts.client(t)
	signup(t, ts, basic, "dev@example.com", "password123") // second user = basic
	resp2, _ := ts.do(t, basic, http.MethodGet, "/api/v1/admin/server-info", nil, nil)
	if resp2.StatusCode != http.StatusForbidden {
		t.Errorf("basic user server-info = %d, want 403", resp2.StatusCode)
	}
}
