package main

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/thehelvijs/Reeve/contracts"
)

func enrollHost(t *testing.T, ts *testServer, admin *http.Client, name string) (hostID, token string) {
	t.Helper()
	resp, data := ts.do(t, admin, http.MethodPost, "/api/v1/admin/hosts", map[string]any{"name": name}, nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("enroll host = %d: %s", resp.StatusCode, data)
	}
	var out struct {
		Host        hostView `json:"host"`
		EnrollToken string   `json:"enroll_token"`
		RunCommand  string   `json:"run_command"`
	}
	json.Unmarshal(data, &out)
	if out.EnrollToken == "" || out.RunCommand == "" {
		t.Fatalf("missing token or run command: %s", data)
	}
	return out.Host.ID, out.EnrollToken
}

func samplePush() contracts.Push {
	return contracts.Push{
		ProtocolVersion: contracts.PushProtocolVersion,
		AgentVersion:    "0.1.0",
		SentAt:          time.Unix(1_700_000_000, 0).UTC(),
		Services:        []contracts.ServiceState{{Unit: "nginx.service", ActiveState: "active", SubState: "running"}},
		Containers:      []contracts.ContainerState{{ID: "c1", Name: "web", Image: "nginx", State: "running", Health: "healthy"}},
		CronJobs:        []contracts.CronState{{Name: "backup", Schedule: "0 3 * * *"}},
		LogEvents:       []contracts.LogEvent{{Source: "web", Level: "error", Message: "boom", At: time.Unix(1_700_000_000, 0).UTC()}},
	}
}

func TestIngestStoresTelemetry(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	hostID, token := enrollHost(t, ts, admin, "host-a")

	resp, data := ts.do(t, nil, http.MethodPost, "/api/v1/ingest", samplePush(),
		map[string]string{"Authorization": "Bearer " + token})
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("ingest = %d: %s", resp.StatusCode, data)
	}

	// Host shows online.
	_, data = ts.do(t, admin, http.MethodGet, "/api/v1/hosts", nil, nil)
	var hosts []hostView
	json.Unmarshal(data, &hosts)
	if len(hosts) != 1 || hosts[0].Status != "online" || hosts[0].AgentVersion != "0.1.0" {
		t.Fatalf("host status after push: %+v", hosts)
	}

	// Inventory reflects the pushed items.
	_, data = ts.do(t, admin, http.MethodGet, "/api/v1/hosts/"+hostID+"/inventory", nil, nil)
	var inv inventoryResponse
	json.Unmarshal(data, &inv)
	if len(inv.Services) != 1 || inv.Services[0].SourceRef != "nginx.service" {
		t.Errorf("services: %+v", inv.Services)
	}
	if len(inv.Containers) != 1 || inv.Containers[0].Name != "web" {
		t.Errorf("containers: %+v", inv.Containers)
	}
	if len(inv.CronJobs) != 1 || inv.CronJobs[0].Name != "backup" {
		t.Errorf("cron: %+v", inv.CronJobs)
	}

	// Log event stored.
	var n int
	ts.app.db.SQL().QueryRow(`SELECT COUNT(*) FROM log_events WHERE host_id = ?`, hostID).Scan(&n)
	if n != 1 {
		t.Errorf("log events = %d, want 1", n)
	}
}

func TestIngestReplacesSnapshot(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	hostID, token := enrollHost(t, ts, admin, "host-a")
	hdr := map[string]string{"Authorization": "Bearer " + token}

	ts.do(t, nil, http.MethodPost, "/api/v1/ingest", samplePush(), hdr)
	// Second push with a different service replaces the first snapshot.
	p := samplePush()
	p.Services = []contracts.ServiceState{{Unit: "redis.service", ActiveState: "active", SubState: "running"}}
	ts.do(t, nil, http.MethodPost, "/api/v1/ingest", p, hdr)

	_, data := ts.do(t, admin, http.MethodGet, "/api/v1/hosts/"+hostID+"/inventory", nil, nil)
	var inv inventoryResponse
	json.Unmarshal(data, &inv)
	if len(inv.Services) != 1 || inv.Services[0].SourceRef != "redis.service" {
		t.Errorf("snapshot not replaced: %+v", inv.Services)
	}
}

func TestIngestBadTokenRejected(t *testing.T) {
	ts := newTestServer(t)
	adminClient(t, ts)
	before := ts.app.ingestRejected.Load()

	resp, _ := ts.do(t, nil, http.MethodPost, "/api/v1/ingest", samplePush(),
		map[string]string{"Authorization": "Bearer rva_bogus"})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("bad token ingest = %d, want 401", resp.StatusCode)
	}
	if ts.app.ingestRejected.Load() != before+1 {
		t.Error("rejected counter not incremented")
	}
}

func TestIngestBadProtocolRejected(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	_, token := enrollHost(t, ts, admin, "host-a")
	p := samplePush()
	p.ProtocolVersion = 999
	resp, _ := ts.do(t, nil, http.MethodPost, "/api/v1/ingest", p,
		map[string]string{"Authorization": "Bearer " + token})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("bad protocol ingest = %d, want 400", resp.StatusCode)
	}
}

func TestInventoryMarksLinked(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	hostID, token := enrollHost(t, ts, admin, "host-a")
	ts.do(t, nil, http.MethodPost, "/api/v1/ingest", samplePush(),
		map[string]string{"Authorization": "Bearer " + token})

	// Link a tool to the nginx unit.
	createTool(t, ts, admin, toolInput{Name: "Nginx", HostID: hostID, SourceType: "systemd", SourceRef: "nginx.service"})

	_, data := ts.do(t, admin, http.MethodGet, "/api/v1/hosts/"+hostID+"/inventory", nil, nil)
	var inv inventoryResponse
	json.Unmarshal(data, &inv)
	if len(inv.Services) != 1 || !inv.Services[0].Linked {
		t.Errorf("nginx unit not marked linked: %+v", inv.Services)
	}
	if inv.Containers[0].Linked {
		t.Error("unlinked container marked linked")
	}
}

func TestCreateHostAdminOnly(t *testing.T) {
	ts := newTestServer(t)
	adminClient(t, ts)
	basic := ts.client(t)
	signup(t, ts, basic, "dev@example.com", "password123")
	resp, _ := ts.do(t, basic, http.MethodPost, "/api/v1/admin/hosts", map[string]any{"name": "x"}, nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("basic create host = %d, want 403", resp.StatusCode)
	}
}
