package main

import (
	"net/http"
	"testing"
	"time"

	"github.com/thehelvijs/Reeve/contracts"
)

func TestClearHostMetricsAdminOnly(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.client(t)
	signup(t, ts, admin, "boss@example.com", "password123")

	host, err := ts.app.db.CreateHost("h1", "linux", "", "hash1", 60)
	if err != nil {
		t.Fatal(err)
	}
	if err := ts.app.db.InsertHostMetric(host.ID, contracts.HostMetrics{CPUPct: 5, MemTotal: 100}, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}

	basic := ts.client(t)
	signup(t, ts, basic, "dev@example.com", "password123")
	resp, _ := ts.do(t, basic, http.MethodDelete, "/api/v1/admin/hosts/"+host.ID+"/metrics", nil, nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("basic clear = %d, want 403", resp.StatusCode)
	}

	resp2, data := ts.do(t, admin, http.MethodDelete, "/api/v1/admin/hosts/"+host.ID+"/metrics", nil, nil)
	if resp2.StatusCode != http.StatusNoContent {
		t.Fatalf("admin clear = %d: %s", resp2.StatusCode, data)
	}
	pts, err := ts.app.db.QueryHostMetrics(host.ID, "raw", time.Unix(0, 0))
	if err != nil {
		t.Fatal(err)
	}
	if len(pts) != 0 {
		t.Errorf("expected 0 metric points after clear, got %d", len(pts))
	}

	resp3, _ := ts.do(t, admin, http.MethodDelete, "/api/v1/admin/hosts/nope/metrics", nil, nil)
	if resp3.StatusCode != http.StatusNotFound {
		t.Errorf("clear unknown host = %d, want 404", resp3.StatusCode)
	}
}
