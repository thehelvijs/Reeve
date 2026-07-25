package main

import (
	"net/http"
	"strings"
	"testing"

	"github.com/thehelvijs/Reeve/server/internal/store"
)

// effectiveThreshold resolves one threshold through the same set the alerting
// pass and the handlers use.
func effectiveThreshold(t *testing.T, ts *testServer, hostID, metric string) (store.Threshold, bool) {
	t.Helper()
	set, err := ts.app.db.LoadThresholds()
	if err != nil {
		t.Fatalf("LoadThresholds: %v", err)
	}
	return set.Effective(hostID, metric)
}

func TestThresholdsAdminGetSet(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.client(t)
	signup(t, ts, admin, "boss@example.com", "password123")

	resp, data := ts.do(t, admin, http.MethodGet, "/api/v1/admin/thresholds", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get thresholds = %d: %s", resp.StatusCode, data)
	}
	if !strings.Contains(string(data), "cpu") || !strings.Contains(string(data), "window_secs") {
		t.Errorf("thresholds payload missing keys: %s", data)
	}

	body := map[string]any{
		"window_secs": 120,
		"thresholds": map[string]any{
			"cpu":  map[string]any{"enabled": true, "threshold": 70},
			"mem":  map[string]any{"enabled": true, "threshold": 80},
			"disk": map[string]any{"enabled": false, "threshold": 95},
			"temp": map[string]any{"enabled": true, "threshold": 75},
		},
	}
	resp2, _ := ts.do(t, admin, http.MethodPut, "/api/v1/admin/thresholds", body, nil)
	if resp2.StatusCode != http.StatusNoContent && resp2.StatusCode != http.StatusOK {
		t.Fatalf("put thresholds = %d", resp2.StatusCode)
	}
	if th, ok := effectiveThreshold(t, ts, "", "cpu"); !ok || th.Value != 70 {
		t.Errorf("cpu global after PUT = %+v", th)
	}

	basic := ts.client(t)
	signup(t, ts, basic, "dev@example.com", "password123")
	resp3, _ := ts.do(t, basic, http.MethodGet, "/api/v1/admin/thresholds", nil, nil)
	if resp3.StatusCode != http.StatusForbidden {
		t.Errorf("basic get thresholds = %d, want 403", resp3.StatusCode)
	}
}

func TestHostThresholdsAdminGetSet(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.client(t)
	signup(t, ts, admin, "boss@example.com", "password123")
	host, err := ts.app.db.CreateHost("h1", "linux", "", "hh1", 60)
	if err != nil {
		t.Fatal(err)
	}

	resp, data := ts.do(t, admin, http.MethodGet, "/api/v1/admin/hosts/"+host.ID+"/thresholds", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get host thresholds = %d: %s", resp.StatusCode, data)
	}
	for _, key := range []string{"cpu", "mem", "disk", "temp"} {
		if !strings.Contains(string(data), key) {
			t.Errorf("host thresholds payload missing %q: %s", key, data)
		}
	}

	body := map[string]any{
		"thresholds": map[string]any{
			"cpu": map[string]any{"enabled": true, "threshold": 50},
		},
	}
	resp2, _ := ts.do(t, admin, http.MethodPut, "/api/v1/admin/hosts/"+host.ID+"/thresholds", body, nil)
	if resp2.StatusCode != http.StatusNoContent && resp2.StatusCode != http.StatusOK {
		t.Fatalf("put host thresholds = %d", resp2.StatusCode)
	}
	if th, ok := effectiveThreshold(t, ts, host.ID, "cpu"); !ok || th.Value != 50 {
		t.Errorf("cpu override after PUT = %+v", th)
	}

	basic := ts.client(t)
	signup(t, ts, basic, "dev@example.com", "password123")
	resp3, _ := ts.do(t, basic, http.MethodPut, "/api/v1/admin/thresholds", body, nil)
	if resp3.StatusCode != http.StatusForbidden {
		t.Errorf("basic put thresholds = %d, want 403", resp3.StatusCode)
	}
}

func TestHostThresholdsReset(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.client(t)
	signup(t, ts, admin, "boss@example.com", "password123")
	host, err := ts.app.db.CreateHost("h1", "linux", "", "hh1", 60)
	if err != nil {
		t.Fatal(err)
	}

	body := map[string]any{
		"thresholds": map[string]any{
			"cpu": map[string]any{"enabled": true, "threshold": 50},
		},
	}
	resp, _ := ts.do(t, admin, http.MethodPut, "/api/v1/admin/hosts/"+host.ID+"/thresholds", body, nil)
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		t.Fatalf("put host thresholds = %d", resp.StatusCode)
	}
	if th, ok := effectiveThreshold(t, ts, host.ID, "cpu"); !ok || th.Value != 50 {
		t.Errorf("cpu override after PUT = %+v", th)
	}

	basic := ts.client(t)
	signup(t, ts, basic, "dev@example.com", "password123")
	respBasic, _ := ts.do(t, basic, http.MethodDelete, "/api/v1/admin/hosts/"+host.ID+"/thresholds", nil, nil)
	if respBasic.StatusCode != http.StatusForbidden {
		t.Errorf("basic delete host thresholds = %d, want 403", respBasic.StatusCode)
	}

	respDel, _ := ts.do(t, admin, http.MethodDelete, "/api/v1/admin/hosts/"+host.ID+"/thresholds", nil, nil)
	if respDel.StatusCode != http.StatusNoContent {
		t.Fatalf("delete host thresholds = %d, want 204", respDel.StatusCode)
	}
	if th, ok := effectiveThreshold(t, ts, host.ID, "cpu"); !ok || th.Value != 90 {
		t.Errorf("cpu after reset = %+v, want fallback to global 90", th)
	}
}

func TestHostThresholdsUnknownHost(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.client(t)
	signup(t, ts, admin, "boss@example.com", "password123")

	resp, _ := ts.do(t, admin, http.MethodGet, "/api/v1/admin/hosts/unknown/thresholds", nil, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("get unknown host thresholds = %d, want 404", resp.StatusCode)
	}

	body := map[string]any{
		"thresholds": map[string]any{
			"cpu": map[string]any{"enabled": true, "threshold": 50},
		},
	}
	resp2, _ := ts.do(t, admin, http.MethodPut, "/api/v1/admin/hosts/unknown/thresholds", body, nil)
	if resp2.StatusCode != http.StatusNotFound {
		t.Errorf("put unknown host thresholds = %d, want 404", resp2.StatusCode)
	}

	resp3, _ := ts.do(t, admin, http.MethodDelete, "/api/v1/admin/hosts/unknown/thresholds", nil, nil)
	if resp3.StatusCode != http.StatusNotFound {
		t.Errorf("delete unknown host thresholds = %d, want 404", resp3.StatusCode)
	}
}
