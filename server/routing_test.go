package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/thehelvijs/Reeve/contracts"
)

func TestAlertSeverityAssigned(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)

	// agent_offline -> error.
	_, token := enrollHostWin(t, ts, admin, "h", 60)
	ts.do(t, nil, http.MethodPost, "/api/ingest", samplePush(),
		map[string]string{"Authorization": "Bearer " + token})
	base := time.Now().UTC().Add(2 * time.Minute)
	ts.app.evaluateAlerts(base)
	ts.app.evaluateAlerts(base.Add(61 * time.Second))
	var sev string
	if err := ts.app.db.SQL().QueryRow(
		`SELECT severity FROM alert_events WHERE type='agent_offline'`).Scan(&sev); err != nil {
		t.Fatalf("query agent_offline severity: %v", err)
	}
	if sev != "error" {
		t.Errorf("agent_offline severity = %q, want error", sev)
	}
}

func TestThresholdAlertSeverityWarning(t *testing.T) {
	ts := newTestServer(t)
	host, err := ts.app.db.CreateHost("h1", "linux", "", "hh1", 600)
	if err != nil {
		t.Fatal(err)
	}
	ts.app.db.TouchHost(host.ID, "test")
	base := time.Now().UTC()
	if err := ts.app.db.InsertHostMetric(host.ID, contracts.HostMetrics{CPUPct: 95, MemTotal: 100, MemUsed: 1}, base); err != nil {
		t.Fatal(err)
	}
	ts.app.evaluateAlerts(base)
	ts.app.evaluateAlerts(base.Add(6 * time.Minute))
	var sev string
	if err := ts.app.db.SQL().QueryRow(
		`SELECT severity FROM alert_events WHERE type='cpu_high'`).Scan(&sev); err != nil {
		t.Fatalf("query cpu_high severity: %v", err)
	}
	if sev != "warning" {
		t.Errorf("cpu_high severity = %q, want warning", sev)
	}
}

func TestSeverityRoutingEnqueue(t *testing.T) {
	ts := newTestServer(t)
	ts.app.db.CreateWebhook("global", "", "http://info.invalid", "generic", "{}", "info")
	ts.app.db.CreateWebhook("global", "", "http://err.invalid", "generic", "{}", "error")

	// A warning host alert should reach only the info channel.
	ts.app.enqueue(alertSubject{key: "host:h:cpu_high", altype: "cpu_high", hostID: "h", message: "m", severity: "warning"}, "fired", time.Now().UTC())
	if got := countRows(t, ts, `SELECT COUNT(*) FROM webhook_deliveries`); got != 1 {
		t.Fatalf("warning enqueued %d deliveries, want 1 (info channel only)", got)
	}

	// An error host alert reaches both.
	ts.app.enqueue(alertSubject{key: "host:h:down", altype: "down", hostID: "h", message: "m", severity: "error"}, "fired", time.Now().UTC())
	if got := countRows(t, ts, `SELECT COUNT(*) FROM webhook_deliveries`); got != 3 {
		t.Fatalf("after error enqueue total deliveries = %d, want 3", got)
	}
}

func TestChannelConfigEncryptedAndRedacted(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)

	resp, data := ts.do(t, admin, http.MethodPost, "/api/admin/webhooks", map[string]any{
		"owner_type":   "global",
		"format":       "webhook",
		"url":          "http://sink.invalid",
		"min_severity": "warning",
		"config":       map[string]string{"token": "supersecret"},
	}, nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create webhook channel = %d: %s", resp.StatusCode, data)
	}

	var raw string
	if err := ts.app.db.SQL().QueryRow(`SELECT config FROM webhooks WHERE format='webhook'`).Scan(&raw); err != nil {
		t.Fatalf("read raw config: %v", err)
	}
	if raw == "" || raw == "{}" {
		t.Fatalf("config not stored: %q", raw)
	}
	if strings.Contains(raw, "supersecret") {
		t.Fatalf("plaintext token found in stored config: %q", raw)
	}

	_, data = ts.do(t, admin, http.MethodGet, "/api/admin/webhooks", nil, nil)
	var views []channelView
	if err := json.Unmarshal(data, &views); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	var found bool
	for _, v := range views {
		if v.Format == "webhook" {
			found = true
			if v.Config["token"] == "supersecret" {
				t.Fatalf("token leaked on read: %v", v.Config)
			}
			if v.MinSeverity != "warning" {
				t.Errorf("min_severity = %q, want warning", v.MinSeverity)
			}
		}
	}
	if !found {
		t.Fatalf("webhook channel not in list: %s", data)
	}
}

func TestSeverityRankOrder(t *testing.T) {
	if severityOrError("") != "error" {
		t.Errorf("empty severity should default to error")
	}
	if severityOrError("warning") != "warning" {
		t.Errorf("set severity should pass through")
	}
}

func TestCreateRejectsUnknownKind(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	resp, _ := ts.do(t, admin, http.MethodPost, "/api/admin/webhooks",
		map[string]any{"owner_type": "global", "format": "sms", "url": "http://x.invalid"}, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("unknown kind status = %d, want 400", resp.StatusCode)
	}
}

func TestCreateRejectsMissingURLForHTTPKind(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	resp, _ := ts.do(t, admin, http.MethodPost, "/api/admin/webhooks",
		map[string]any{"owner_type": "global", "format": "webhook", "url": ""}, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("missing url status = %d, want 400", resp.StatusCode)
	}
}

// An unrouted /api/ path must answer with the error envelope, not the SPA: a
// non-browser caller reads a 200 full of HTML as a successful request.
func TestUnknownAPIPathIsNotSwallowedBySPA(t *testing.T) {
	ts := newTestServer(t)
	resp, data := ts.do(t, nil, http.MethodGet, "/api/v1/hosts", nil, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404: %s", resp.StatusCode, data)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Errorf("content-type = %q, want json", ct)
	}
}
