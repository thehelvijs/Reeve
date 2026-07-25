package main

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/thehelvijs/Reeve/contracts"
)

func toolByName(t *testing.T, ts *testServer, c *http.Client, name string) toolResponse {
	t.Helper()
	_, data := ts.do(t, c, http.MethodGet, "/api/v1/tools", nil, nil)
	var tools []toolResponse
	json.Unmarshal(data, &tools)
	for _, tl := range tools {
		if tl.Name == name {
			return tl
		}
	}
	t.Fatalf("tool %q not found in %s", name, data)
	return toolResponse{}
}

func TestToolStatusDerivation(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	hostID, token := enrollHost(t, ts, admin, "host-a")
	ts.do(t, nil, http.MethodPost, "/api/v1/ingest", samplePush(),
		map[string]string{"Authorization": "Bearer " + token})

	createTool(t, ts, admin, toolInput{Name: "Web", HostID: hostID, SourceType: "systemd", SourceRef: "nginx.service"})
	createTool(t, ts, admin, toolInput{Name: "Container", HostID: hostID, SourceType: "docker", SourceRef: "c1"})
	createTool(t, ts, admin, toolInput{Name: "Manual"})

	if s := toolByName(t, ts, admin, "Web").Status; s != contracts.StatusUp {
		t.Errorf("systemd active tool status = %q, want up", s)
	}
	if s := toolByName(t, ts, admin, "Container").Status; s != contracts.StatusUp {
		t.Errorf("running healthy container status = %q, want up", s)
	}
	if s := toolByName(t, ts, admin, "Manual").Status; s != contracts.StatusUnknown {
		t.Errorf("manual tool status = %q, want unknown", s)
	}
}

func TestToolStatusDown(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	hostID, token := enrollHost(t, ts, admin, "host-a")
	p := samplePush()
	p.Services = []contracts.ServiceState{{Unit: "broken.service", ActiveState: "failed", SubState: "failed"}}
	ts.do(t, nil, http.MethodPost, "/api/v1/ingest", p, map[string]string{"Authorization": "Bearer " + token})

	createTool(t, ts, admin, toolInput{Name: "Broken", HostID: hostID, SourceType: "systemd", SourceRef: "broken.service"})
	if s := toolByName(t, ts, admin, "Broken").Status; s != contracts.StatusDown {
		t.Errorf("failed service status = %q, want down", s)
	}
}

func TestToolStatusAgentOffline(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	hostID, _ := enrollHost(t, ts, admin, "host-a") // never pushes

	createTool(t, ts, admin, toolInput{Name: "NoAgent", HostID: hostID, SourceType: "systemd", SourceRef: "x.service"})
	if s := toolByName(t, ts, admin, "NoAgent").Status; s != contracts.StatusAgentOffline {
		t.Errorf("no-push host tool status = %q, want agent_offline", s)
	}
}

func TestHostMetricsEndpoint(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	hostID, token := enrollHost(t, ts, admin, "host-a")
	p := samplePush()
	p.Metrics = contracts.HostMetrics{CPUPct: 42, MemUsed: 100, MemTotal: 200}
	ts.do(t, nil, http.MethodPost, "/api/v1/ingest", p, map[string]string{"Authorization": "Bearer " + token})

	_, data := ts.do(t, admin, http.MethodGet, "/api/v1/hosts/"+hostID+"/metrics?range=24h", nil, nil)
	var out struct {
		Resolution string `json:"resolution"`
		Host       []struct {
			CPUPct float64 `json:"cpu_pct"`
		} `json:"host"`
	}
	json.Unmarshal(data, &out)
	if out.Resolution != "raw" {
		t.Errorf("resolution = %q, want raw", out.Resolution)
	}
	if len(out.Host) != 1 || out.Host[0].CPUPct != 42 {
		t.Errorf("host metrics: %+v", out.Host)
	}
}
