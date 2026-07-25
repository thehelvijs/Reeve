package main

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/thehelvijs/Reeve/contracts"
)

func TestListHostsIncludesLatestMetric(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.client(t)
	signup(t, ts, admin, "boss@example.com", "password123")

	withMetric, err := ts.app.db.CreateHost("h-metric", "linux", "", "hash-m", 60)
	if err != nil {
		t.Fatal(err)
	}
	noMetric, err := ts.app.db.CreateHost("h-empty", "linux", "", "hash-e", 60)
	if err != nil {
		t.Fatal(err)
	}
	sample := contracts.HostMetrics{CPUPct: 42, MemUsed: 4, MemTotal: 8, DiskUsed: 20, DiskTotal: 100}
	if err := ts.app.db.InsertHostMetric(withMetric.ID, sample, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}

	resp, data := ts.do(t, admin, http.MethodGet, "/api/v1/hosts", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list hosts = %d: %s", resp.StatusCode, data)
	}
	var hosts []hostView
	if err := json.Unmarshal(data, &hosts); err != nil {
		t.Fatal(err)
	}

	byID := map[string]hostView{}
	for _, h := range hosts {
		byID[h.ID] = h
	}

	m := byID[withMetric.ID].Metrics
	if m == nil {
		t.Fatal("host with a sample has no metrics block")
	}
	if m.CPUPct != 42 || m.MemUsed != 4 || m.MemTotal != 8 || m.DiskUsed != 20 || m.DiskTotal != 100 {
		t.Errorf("metrics = %+v, want the inserted sample", m)
	}
	if m.At == "" {
		t.Error("metrics block missing timestamp")
	}

	if byID[noMetric.ID].Metrics != nil {
		t.Error("host with no samples should omit the metrics block")
	}
}
