package main

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

// The heartbeat is only useful if a saved URL is actually reachable from this
// process, so the test-ping path is what proves the setting is wired to a GET.
func TestHeartbeatSaveAndTestPing(t *testing.T) {
	var pings atomic.Int64
	recv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		pings.Add(1)
	}))
	defer recv.Close()

	ts := newTestServer(t)
	c := ts.client(t)
	signup(t, ts, c, "boss@example.com", "password123")

	resp, _ := ts.do(t, c, http.MethodPost, "/api/admin/settings/test-heartbeat", nil, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("test ping with no url = %d, want 400", resp.StatusCode)
	}

	resp, _ = putSettings(t, ts, c, map[string]any{"heartbeat_url": "not a url"})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("saving a junk url = %d, want 400", resp.StatusCode)
	}

	resp, v := putSettings(t, ts, c, map[string]any{"heartbeat_url": recv.URL})
	if resp.StatusCode != http.StatusOK || v.HeartbeatURL != recv.URL {
		t.Fatalf("save = %d, url = %q", resp.StatusCode, v.HeartbeatURL)
	}

	resp, _ = ts.do(t, c, http.MethodPost, "/api/admin/settings/test-heartbeat", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("test ping = %d, want 200", resp.StatusCode)
	}
	if pings.Load() != 1 {
		t.Errorf("receiver saw %d pings, want 1", pings.Load())
	}

	dead := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer dead.Close()
	if err := pingHeartbeat(dead.URL); err == nil {
		t.Error("a 500 from the receiver reported success")
	}
}
