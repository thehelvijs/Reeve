package main

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/thehelvijs/Reeve/server/internal/store"
)

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
		resp, data := ts.do(t, c, http.MethodGet, "/api/v1/hosts/"+host.ID+"/uptime?range=24h", nil, nil)
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
	resp, data := ts.do(t, admin, http.MethodGet, "/api/v1/hosts/nope/uptime", nil, nil)
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

	resp, data := ts.do(t, owner, http.MethodGet, "/api/v1/tools/"+tool.ID+"/uptime?range=7d", nil, nil)
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
	resp, data := ts.do(t, outsider, http.MethodGet, "/api/v1/tools/"+tool.ID+"/uptime", nil, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("outsider tool uptime = %d, want 404: %s", resp.StatusCode, data)
	}
}
