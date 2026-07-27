package main

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/thehelvijs/Reeve/contracts"
	"github.com/thehelvijs/Reeve/server/internal/store"
)

func TestHostMetricsEndpointContainers(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)

	dockerHost, dockerToken := enrollHost(t, ts, admin, "docker-host")
	push := samplePush()
	push.Metrics = contracts.HostMetrics{CPUPct: 10, MemUsed: 2000, MemTotal: 8000}
	push.Containers = []contracts.ContainerState{{ID: "c1", Name: "web", Image: "nginx", State: "running", Health: "healthy"}}
	push.ContainerStats = []contracts.ContainerSample{{ContainerID: "c1", CPUPct: 2, MemUsed: 1000, MemLimit: 5000}}
	resp, data := ts.do(t, nil, http.MethodPost, "/api/ingest", push,
		map[string]string{"Authorization": "Bearer " + dockerToken})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("ingest = %d: %s", resp.StatusCode, data)
	}

	resp, data = ts.do(t, admin, http.MethodGet, "/api/hosts/"+dockerHost+"/metrics?range=1h", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("metrics = %d: %s", resp.StatusCode, data)
	}
	var body struct {
		Containers []store.ContainerPoint `json:"containers"`
	}
	if err := json.Unmarshal(data, &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Containers) != 1 {
		t.Fatalf("containers = %d, want 1: %s", len(body.Containers), data)
	}
	c := body.Containers[0]
	if c.Name != "web" {
		t.Errorf("Name = %q, want %q", c.Name, "web")
	}
	if c.ContainerID != "c1" {
		t.Errorf("ContainerID = %q, want %q", c.ContainerID, "c1")
	}
	if c.CPUPct != 2 {
		t.Errorf("CPUPct = %v, want 2", c.CPUPct)
	}
	if c.MemUsed != 1000 {
		t.Errorf("MemUsed = %d, want 1000", c.MemUsed)
	}

	// A docker-less host reports no container series, so the UI panel stays hidden.
	plainHost, plainToken := enrollHost(t, ts, admin, "plain-host")
	plain := samplePush()
	plain.Metrics = contracts.HostMetrics{CPUPct: 5, MemUsed: 1000, MemTotal: 8000}
	plain.Containers = nil
	plain.ContainerStats = nil
	resp, data = ts.do(t, nil, http.MethodPost, "/api/ingest", plain,
		map[string]string{"Authorization": "Bearer " + plainToken})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("plain ingest = %d: %s", resp.StatusCode, data)
	}

	resp, data = ts.do(t, admin, http.MethodGet, "/api/hosts/"+plainHost+"/metrics?range=1h", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("plain metrics = %d: %s", resp.StatusCode, data)
	}
	if err := json.Unmarshal(data, &body); err != nil {
		t.Fatalf("decode plain: %v", err)
	}
	if len(body.Containers) != 0 {
		t.Fatalf("docker-less containers = %d, want 0: %s", len(body.Containers), data)
	}
}

func TestHostUptimeEndpoint(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.client(t)
	signup(t, ts, admin, "boss@example.com", "password123")
	basic := ts.client(t)
	signup(t, ts, basic, "dev@example.com", "password123")

	host, err := ts.app.db.CreateHost("h1", "linux", "", "hh1", 60)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	ev, err := ts.app.db.CreateAlertEvent(
		store.AlertEvent{SubjectKey: "k", HostID: host.ID, Type: "agent_offline", Severity: "error", Message: "m"},
		now.Add(-time.Hour),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := ts.app.db.ResolveAlertEvent(ev.ID, now.Add(-time.Hour+100*time.Second)); err != nil {
		t.Fatal(err)
	}

	for _, c := range []*http.Client{admin, basic} {
		resp, data := ts.do(t, c, http.MethodGet, "/api/hosts/"+host.ID+"/uptime?range=24h", nil, nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("host uptime = %d: %s", resp.StatusCode, data)
		}
		var body struct {
			Range     string  `json:"range"`
			UptimePct float64 `json:"uptime_pct"`
		}
		if err := json.Unmarshal(data, &body); err != nil {
			t.Fatalf("unmarshal: %v: %s", err, data)
		}
		if body.Range != "24h" {
			t.Errorf("range = %q, want 24h", body.Range)
		}
		if body.UptimePct <= 0 || body.UptimePct >= 100 {
			t.Errorf("uptime_pct = %v, want between 0 and 100", body.UptimePct)
		}
	}
}

func TestHostUptimeNotFound(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.client(t)
	signup(t, ts, admin, "boss@example.com", "password123")
	resp, data := ts.do(t, admin, http.MethodGet, "/api/hosts/nope/uptime", nil, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404: %s", resp.StatusCode, data)
	}
}

func TestToolUptimeEndpoint(t *testing.T) {
	ts := newTestServer(t)
	owner := ts.client(t)
	signup(t, ts, owner, "boss@example.com", "password123")
	tool := createTool(t, ts, owner, toolInput{Name: "svc"})

	now := time.Now().UTC()
	ev, err := ts.app.db.CreateAlertEvent(
		store.AlertEvent{SubjectKey: "k", ToolID: tool.ID, Type: "down", Severity: "error", Message: "m"},
		now.Add(-2*time.Hour),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := ts.app.db.ResolveAlertEvent(ev.ID, now.Add(-2*time.Hour+60*time.Second)); err != nil {
		t.Fatal(err)
	}

	resp, data := ts.do(t, owner, http.MethodGet, "/api/tools/"+tool.ID+"/uptime?range=7d", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("tool uptime = %d: %s", resp.StatusCode, data)
	}
	var body struct {
		Range     string  `json:"range"`
		UptimePct float64 `json:"uptime_pct"`
	}
	if err := json.Unmarshal(data, &body); err != nil {
		t.Fatalf("unmarshal: %v: %s", err, data)
	}
	if body.Range != "7d" {
		t.Errorf("range = %q, want 7d", body.Range)
	}
	if body.UptimePct <= 0 || body.UptimePct >= 100 {
		t.Errorf("uptime_pct = %v, want between 0 and 100", body.UptimePct)
	}
}

func TestToolUptimeHiddenForOutsider(t *testing.T) {
	ts := newTestServer(t)
	owner := ts.client(t)
	signup(t, ts, owner, "boss@example.com", "password123")
	tool := createTool(t, ts, owner, toolInput{Name: "Secret", Visibility: "restricted"})

	outsider := ts.client(t)
	signup(t, ts, outsider, "dev@example.com", "password123")
	resp, data := ts.do(t, outsider, http.MethodGet, "/api/tools/"+tool.ID+"/uptime", nil, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("outsider tool uptime = %d, want 404: %s", resp.StatusCode, data)
	}
}

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

	resp, data := ts.do(t, admin, http.MethodGet, "/api/admin/server-metrics?range=24h", nil, nil)
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
	_, hdata := ts.do(t, admin, http.MethodGet, "/api/hosts", nil, nil)
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
	resp2, _ := ts.do(t, basic, http.MethodGet, "/api/admin/server-metrics", nil, nil)
	if resp2.StatusCode != http.StatusForbidden {
		t.Fatalf("basic server-metrics = %d, want 403", resp2.StatusCode)
	}
}

func TestHostProcessUsageEndpoint(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	hostID, token := enrollHost(t, ts, admin, "busy-host")

	push := samplePush()
	push.Processes = []contracts.ProcessSample{
		{PID: 1, User: "plex", Command: "Plex Transcoder", CPUPct: 40, MemRSS: 1 << 20},
		{PID: 2, User: "root", Command: "dockerd", CPUPct: 5, MemRSS: 1 << 30},
	}
	for i := 0; i < 2; i++ {
		resp, data := ts.do(t, nil, http.MethodPost, "/api/ingest", push,
			map[string]string{"Authorization": "Bearer " + token})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("ingest = %d: %s", resp.StatusCode, data)
		}
	}

	resp, data := ts.do(t, admin, http.MethodGet, "/api/hosts/"+hostID+"/process-usage?window=1h", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("process-usage = %d: %s", resp.StatusCode, data)
	}
	var body struct {
		Window    string               `json:"window"`
		Processes []store.ProcessUsage `json:"processes"`
	}
	if err := json.Unmarshal(data, &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Window != time.Hour.String() {
		t.Errorf("window = %q, want %q", body.Window, time.Hour.String())
	}
	if len(body.Processes) != 2 {
		t.Fatalf("processes = %+v, want both commands: %s", body.Processes, data)
	}
	top := body.Processes[0]
	if top.Command != "Plex Transcoder" || top.CPUAvg != 40 || top.Samples != 2 {
		t.Errorf("top = %+v, want the transcoder averaged over two pushes", top)
	}
}

// An unknown host must 404 rather than answer with an empty window, which would
// read as "this machine does nothing".
func TestHostProcessUsageUnknownHost(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	resp, _ := ts.do(t, admin, http.MethodGet, "/api/hosts/nope/process-usage", nil, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
}

// The metrics response carries every filesystem the agent reported, not just
// the root one the charts are built on.
func TestHostMetricsEndpointCarriesDisks(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	hostID, token := enrollHost(t, ts, admin, "disk-host")

	push := samplePush()
	push.Metrics.Disks = []contracts.DiskUsage{
		{Mount: "/", Device: "/dev/sda3", FSType: "ext4", Used: 16, Total: 30},
		{Mount: "/mnt/media", Device: "/dev/sdb1", FSType: "ext4", Used: 900, Total: 1000},
	}
	resp, data := ts.do(t, nil, http.MethodPost, "/api/ingest", push,
		map[string]string{"Authorization": "Bearer " + token})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("ingest = %d: %s", resp.StatusCode, data)
	}

	resp, data = ts.do(t, admin, http.MethodGet, "/api/hosts/"+hostID+"/metrics?range=1h", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("metrics = %d: %s", resp.StatusCode, data)
	}
	var body struct {
		Disks store.HostDisks `json:"disks"`
	}
	if err := json.Unmarshal(data, &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Disks.Disks) != 2 {
		t.Fatalf("disks = %+v, want both filesystems: %s", body.Disks.Disks, data)
	}
	if body.Disks.Disks[1].Mount != "/mnt/media" || body.Disks.Disks[1].Total != 1000 {
		t.Errorf("second filesystem = %+v, want /mnt/media", body.Disks.Disks[1])
	}
}

// A host that has never reported sends an empty list, not null, so the UI has
// nothing to guard against.
func TestHostMetricsEndpointDisksEmptyForSilentHost(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	h, _ := ts.app.db.CreateHost("silent-disks", "linux", "", "hash-sd", 60)

	_, data := ts.do(t, admin, http.MethodGet, "/api/hosts/"+h.ID+"/metrics", nil, nil)
	var body struct {
		Disks store.HostDisks `json:"disks"`
	}
	if err := json.Unmarshal(data, &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Disks.Disks == nil {
		t.Error("disks is null; the UI would have to guard against it")
	}
}
