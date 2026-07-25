package main

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/thehelvijs/Reeve/contracts"
	"github.com/thehelvijs/Reeve/server/internal/store"
)

func TestServerMetricsEndpointAndHiddenHost(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.client(t)
	signup(t, ts, admin, "boss@example.com", "password123")

	if err := ts.app.db.EnsureServerHost("linux"); err != nil {
		t.Fatal(err)
	}
	sample := contracts.HostMetrics{CPUPct: 7, MemUsed: 1, MemTotal: 2}
	if err := ts.app.db.InsertHostMetric(store.ServerHostID, sample, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}

	resp, data := ts.do(t, admin, http.MethodGet, "/api/v1/admin/server-metrics?range=24h", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("server-metrics = %d: %s", resp.StatusCode, data)
	}
	var body struct {
		Host []store.MetricPoint `json:"host"`
	}
	if err := json.Unmarshal(data, &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Host) == 0 || body.Host[len(body.Host)-1].CPUPct != 7 {
		t.Fatalf("expected the sampled point, got %+v", body.Host)
	}

	// The reserved self host must never appear in the host list.
	_, hdata := ts.do(t, admin, http.MethodGet, "/api/v1/hosts", nil, nil)
	var hosts []hostView
	if err := json.Unmarshal(hdata, &hosts); err != nil {
		t.Fatal(err)
	}
	for _, h := range hosts {
		if h.ID == store.ServerHostID {
			t.Fatal("reserved server host leaked into the host list")
		}
	}

	// Non-admins cannot read server metrics.
	basic := ts.client(t)
	signup(t, ts, basic, "dev@example.com", "password123")
	resp2, _ := ts.do(t, basic, http.MethodGet, "/api/v1/admin/server-metrics", nil, nil)
	if resp2.StatusCode != http.StatusForbidden {
		t.Fatalf("basic server-metrics = %d, want 403", resp2.StatusCode)
	}
}
