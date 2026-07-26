package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestAgentInstallCommandShape(t *testing.T) {
	a := &app{cfg: config{PublicURL: "http://10.0.0.2:8080"}}
	cmd := a.agentInstallCommand(httptest.NewRequest(http.MethodPost, "/", nil), "tok-123")
	for _, want := range []string{
		"curl -fsSL http://10.0.0.2:8080/install.sh",
		"REEVE_SERVER_URL=http://10.0.0.2:8080",
		"REEVE_AGENT_TOKEN=tok-123",
		"sudo",
		"bash",
	} {
		if !strings.Contains(cmd, want) {
			t.Errorf("install command missing %q: %s", want, cmd)
		}
	}
}

// With no REEVE_PUBLIC_URL the enroll commands use the address the admin's
// browser reached the UI on, which on a LAN is the server's LAN address.
func TestAgentCommandsFallBackToRequestHost(t *testing.T) {
	a := &app{cfg: config{Addr: "127.0.0.1:8080"}}
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	r.Host = "192.168.1.50:8080"
	for _, cmd := range []string{a.agentInstallCommand(r, "tok"), a.agentRunCommand(r, "tok")} {
		if !strings.Contains(cmd, "REEVE_SERVER_URL=http://192.168.1.50:8080") {
			t.Errorf("command did not use the request host: %s", cmd)
		}
		if strings.Contains(cmd, "127.0.0.1") {
			t.Errorf("command leaked the bind address: %s", cmd)
		}
	}
}

func TestUpdateHostLocation(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.client(t)
	signup(t, ts, admin, "admin@example.com", "password123")

	h, err := ts.app.db.CreateHost("riga-box", "linux", "", "tok", 60)
	if err != nil {
		t.Fatalf("create host: %v", err)
	}

	body := map[string]any{"physical_location": "Riga, Latvia", "latitude": 56.946, "longitude": 24.106}
	resp, data := ts.do(t, admin, http.MethodPatch, "/api/v1/admin/hosts/"+h.ID, body, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("patch host = %d: %s", resp.StatusCode, data)
	}
	var v struct {
		PhysicalLocation string   `json:"physical_location"`
		Latitude         *float64 `json:"latitude"`
		Longitude        *float64 `json:"longitude"`
	}
	json.Unmarshal(data, &v)
	if v.PhysicalLocation != "Riga, Latvia" || v.Latitude == nil || *v.Latitude != 56.946 || v.Longitude == nil {
		t.Fatalf("location not persisted: %s", data)
	}

	// Listing reflects the stored coordinates.
	_, list := ts.do(t, admin, http.MethodGet, "/api/v1/hosts", nil, nil)
	if !bytes.Contains(list, []byte("56.946")) {
		t.Errorf("host list missing latitude: %s", list)
	}
}

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
