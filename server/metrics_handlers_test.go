package main

import (
	"encoding/json"
	"net/http"
	"testing"

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
	resp, data := ts.do(t, nil, http.MethodPost, "/api/v1/ingest", push,
		map[string]string{"Authorization": "Bearer " + dockerToken})
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("ingest = %d: %s", resp.StatusCode, data)
	}

	resp, data = ts.do(t, admin, http.MethodGet, "/api/v1/hosts/"+dockerHost+"/metrics?range=1h", nil, nil)
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
	resp, data = ts.do(t, nil, http.MethodPost, "/api/v1/ingest", plain,
		map[string]string{"Authorization": "Bearer " + plainToken})
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("plain ingest = %d: %s", resp.StatusCode, data)
	}

	resp, data = ts.do(t, admin, http.MethodGet, "/api/v1/hosts/"+plainHost+"/metrics?range=1h", nil, nil)
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
