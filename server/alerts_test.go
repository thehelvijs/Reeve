package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/thehelvijs/Reeve/contracts"
)

// enrollHostWin enrolls a host with a custom offline window so a test can hold
// it "online" across an injected time span.
func enrollHostWin(t *testing.T, ts *testServer, admin *http.Client, name string, offlineAfter int) (string, string) {
	t.Helper()
	resp, data := ts.do(t, admin, http.MethodPost, "/api/admin/hosts",
		map[string]any{"name": name, "offline_after_secs": offlineAfter}, nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("enroll = %d: %s", resp.StatusCode, data)
	}
	var out struct {
		Host        hostView `json:"host"`
		EnrollToken string   `json:"enroll_token"`
	}
	json.Unmarshal(data, &out)
	return out.Host.ID, out.EnrollToken
}

func countRows(t *testing.T, ts *testServer, q string, args ...any) int {
	t.Helper()
	var n int
	if err := ts.app.db.SQL().QueryRow(q, args...).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	return n
}

func TestDownAlertDebounceFireResolve(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	hostID, token := enrollHostWin(t, ts, admin, "h", 3600)
	hdr := map[string]string{"Authorization": "Bearer " + token}

	down := samplePush()
	down.Services = []contracts.ServiceState{{Unit: "web.service", ActiveState: "failed", SubState: "failed"}}
	ts.do(t, nil, http.MethodPost, "/api/ingest", down, hdr)
	createTool(t, ts, admin, toolInput{Name: "Web", HostID: hostID, SourceType: "systemd", SourceRef: "web.service"})
	ts.app.db.CreateWebhook("global", "", "http://sink.invalid", "", "", "")

	t0 := time.Now().UTC()
	ts.app.evaluateAlerts(t0) // pending
	if n := countRows(t, ts, `SELECT COUNT(*) FROM alert_events`); n != 0 {
		t.Fatalf("fired before debounce: %d events", n)
	}
	ts.app.evaluateAlerts(t0.Add(30 * time.Second)) // still pending
	if n := countRows(t, ts, `SELECT COUNT(*) FROM alert_events`); n != 0 {
		t.Fatalf("fired at 30s (<60s debounce): %d events", n)
	}
	ts.app.evaluateAlerts(t0.Add(61 * time.Second)) // fire
	if n := countRows(t, ts, `SELECT COUNT(*) FROM alert_events WHERE type='down' AND resolved_at IS NULL`); n != 1 {
		t.Fatalf("expected 1 open down event, got %d", n)
	}
	if n := countRows(t, ts, `SELECT COUNT(*) FROM webhook_deliveries`); n != 1 {
		t.Fatalf("expected 1 delivery enqueued, got %d", n)
	}

	// Recover: service active again.
	up := samplePush()
	up.Services = []contracts.ServiceState{{Unit: "web.service", ActiveState: "active", SubState: "running"}}
	ts.do(t, nil, http.MethodPost, "/api/ingest", up, hdr)
	ts.app.evaluateAlerts(t0.Add(70 * time.Second)) // resolve
	if n := countRows(t, ts, `SELECT COUNT(*) FROM alert_events WHERE resolved_at IS NOT NULL`); n != 1 {
		t.Errorf("expected event resolved, got %d resolved", n)
	}
	if n := countRows(t, ts, `SELECT COUNT(*) FROM webhook_deliveries`); n != 2 {
		t.Errorf("expected recovery delivery, total deliveries = %d", n)
	}
}

func TestFlapSuppressed(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	hostID, token := enrollHostWin(t, ts, admin, "h", 3600)
	hdr := map[string]string{"Authorization": "Bearer " + token}
	down := samplePush()
	down.Services = []contracts.ServiceState{{Unit: "web.service", ActiveState: "failed"}}
	ts.do(t, nil, http.MethodPost, "/api/ingest", down, hdr)
	createTool(t, ts, admin, toolInput{Name: "Web", HostID: hostID, SourceType: "systemd", SourceRef: "web.service"})

	t0 := time.Now().UTC()
	ts.app.evaluateAlerts(t0) // pending
	up := samplePush()
	up.Services = []contracts.ServiceState{{Unit: "web.service", ActiveState: "active", SubState: "running"}}
	ts.do(t, nil, http.MethodPost, "/api/ingest", up, hdr)
	ts.app.evaluateAlerts(t0.Add(10 * time.Second)) // back to ok before debounce
	if n := countRows(t, ts, `SELECT COUNT(*) FROM alert_events`); n != 0 {
		t.Errorf("flap fired an alert: %d events", n)
	}
}

func TestAgentOfflineAlert(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	_, token := enrollHostWin(t, ts, admin, "h", 60)
	ts.do(t, nil, http.MethodPost, "/api/ingest", samplePush(),
		map[string]string{"Authorization": "Bearer " + token})
	ts.app.db.CreateWebhook("global", "", "http://sink.invalid", "", "", "")

	// last_seen is ~now; evaluate far in the future so the host reads offline.
	base := time.Now().UTC().Add(2 * time.Minute)
	ts.app.evaluateAlerts(base)                       // pending
	ts.app.evaluateAlerts(base.Add(61 * time.Second)) // fire
	if n := countRows(t, ts, `SELECT COUNT(*) FROM alert_events WHERE type='agent_offline'`); n != 1 {
		t.Errorf("expected 1 agent_offline alert, got %d", n)
	}
}

func TestLogErrorAlert(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	hostID, token := enrollHostWin(t, ts, admin, "h", 3600)
	p := samplePush()
	p.LogEvents = []contracts.LogEvent{{Source: "web", Level: "error", Message: "boom", At: time.Now().UTC()}}
	ts.do(t, nil, http.MethodPost, "/api/ingest", p, map[string]string{"Authorization": "Bearer " + token})

	// Tool with log alerts on, matching the log source "web".
	createTool(t, ts, admin, toolInput{Name: "Web", HostID: hostID, SourceType: "systemd", SourceRef: "web"})
	ts.app.db.SQL().Exec(`UPDATE tools SET log_alert_enabled = 1 WHERE name = 'Web'`)

	ts.app.evaluateAlerts(time.Now().UTC()) // log_error debounce is 0 -> fires immediately
	if n := countRows(t, ts, `SELECT COUNT(*) FROM alert_events WHERE type='log_error'`); n != 1 {
		t.Errorf("expected 1 log_error alert, got %d", n)
	}
}

func TestThresholdAlertFiresAndResolves(t *testing.T) {
	ts := newTestServer(t)
	host, err := ts.app.db.CreateHost("h1", "linux", "", "hh1", 7200)
	if err != nil {
		t.Fatal(err)
	}
	ts.app.db.TouchHost(host.ID, "test") // mark online (sets last_seen_at)

	base := time.Now().UTC()
	// Breaching sample: cpu 95 > global 90.
	if err := ts.app.db.InsertHostMetric(host.ID, contracts.HostMetrics{CPUPct: 95, MemTotal: 100, MemUsed: 1}, base); err != nil {
		t.Fatal(err)
	}

	// First pass: pending (not yet past the window).
	ts.app.evaluateAlerts(base)
	st := ts.app.db.GetAlertState("host:" + host.ID + ":cpu_high")
	if st.State != "pending" {
		t.Fatalf("state after first pass = %q, want pending", st.State)
	}
	// After the window: fires.
	ts.app.evaluateAlerts(base.Add(6 * time.Minute))
	st = ts.app.db.GetAlertState("host:" + host.ID + ":cpu_high")
	if st.State != "firing" {
		t.Fatalf("state after window = %q, want firing", st.State)
	}

	// Recovery sample under threshold → resolves.
	if err := ts.app.db.InsertHostMetric(host.ID, contracts.HostMetrics{CPUPct: 10, MemTotal: 100, MemUsed: 1}, base.Add(7*time.Minute)); err != nil {
		t.Fatal(err)
	}
	ts.app.evaluateAlerts(base.Add(7 * time.Minute))
	st = ts.app.db.GetAlertState("host:" + host.ID + ":cpu_high")
	if st.State != "ok" {
		t.Fatalf("state after recovery = %q, want ok", st.State)
	}
}

func TestThresholdAlertResolvesOnDisable(t *testing.T) {
	ts := newTestServer(t)
	host, err := ts.app.db.CreateHost("h1", "linux", "", "hh1", 7200)
	if err != nil {
		t.Fatal(err)
	}
	ts.app.db.TouchHost(host.ID, "test") // mark online (sets last_seen_at)

	base := time.Now().UTC()
	// Breaching sample: cpu 95 > global 90.
	if err := ts.app.db.InsertHostMetric(host.ID, contracts.HostMetrics{CPUPct: 95, MemTotal: 100, MemUsed: 1}, base); err != nil {
		t.Fatal(err)
	}

	ts.app.evaluateAlerts(base)                      // pending
	ts.app.evaluateAlerts(base.Add(6 * time.Minute)) // fires (past window)
	st := ts.app.db.GetAlertState("host:" + host.ID + ":cpu_high")
	if st.State != "firing" {
		t.Fatalf("state before disable = %q, want firing", st.State)
	}

	// Disable the cpu threshold while still firing.
	if err := ts.app.db.SetThreshold(host.ID, "cpu", false, 90); err != nil {
		t.Fatal(err)
	}
	ts.app.evaluateAlerts(base.Add(7 * time.Minute))
	st = ts.app.db.GetAlertState("host:" + host.ID + ":cpu_high")
	if st.State != "ok" {
		t.Fatalf("state after disabling threshold = %q, want ok (auto-resolved)", st.State)
	}
}

func TestLoadThresholdAlertFires(t *testing.T) {
	ts := newTestServer(t)
	host, err := ts.app.db.CreateHost("h1", "linux", "", "hh1", 7200)
	if err != nil {
		t.Fatal(err)
	}
	ts.app.db.TouchHost(host.ID, "test") // mark online (sets last_seen_at)

	if err := ts.app.db.SetThreshold(host.ID, "load", true, 1); err != nil {
		t.Fatal(err)
	}

	base := time.Now().UTC()
	if err := ts.app.db.InsertHostMetric(host.ID, contracts.HostMetrics{Load1: 5, MemTotal: 100, MemUsed: 1}, base); err != nil {
		t.Fatal(err)
	}

	ts.app.evaluateAlerts(base)                      // pending
	ts.app.evaluateAlerts(base.Add(6 * time.Minute)) // past window: fires
	st := ts.app.db.GetAlertState("host:" + host.ID + ":load_high")
	if st.State != "firing" {
		t.Fatalf("state after window = %q, want firing", st.State)
	}
}

func TestNetRateThresholdFires(t *testing.T) {
	ts := newTestServer(t)
	host, err := ts.app.db.CreateHost("h1", "linux", "", "hh1", 7200)
	if err != nil {
		t.Fatal(err)
	}
	ts.app.db.TouchHost(host.ID, "test")
	if err := ts.app.db.SetThreshold("", "net", true, 1000); err != nil {
		t.Fatal(err)
	}

	base := time.Now().UTC()
	// Two samples 10s apart, +1 MiB rx → ~104857 B/s, over the 1000 B/s limit.
	if err := ts.app.db.InsertHostMetric(host.ID, contracts.HostMetrics{NetRx: 1000, MemTotal: 100, MemUsed: 1}, base); err != nil {
		t.Fatal(err)
	}
	if err := ts.app.db.InsertHostMetric(host.ID, contracts.HostMetrics{NetRx: 1000 + 1048576, MemTotal: 100, MemUsed: 1}, base.Add(10*time.Second)); err != nil {
		t.Fatal(err)
	}

	ts.app.evaluateAlerts(base)                      // pending
	ts.app.evaluateAlerts(base.Add(6 * time.Minute)) // past window: fires
	st := ts.app.db.GetAlertState("host:" + host.ID + ":net_high")
	if st.State != "firing" {
		t.Fatalf("state after window = %q, want firing", st.State)
	}

	// Recovery: newest sample flat rx (delta 0) → rate under limit, resolves.
	if err := ts.app.db.InsertHostMetric(host.ID, contracts.HostMetrics{NetRx: 1000 + 1048576, MemTotal: 100, MemUsed: 1}, base.Add(7*time.Minute)); err != nil {
		t.Fatal(err)
	}
	ts.app.evaluateAlerts(base.Add(7 * time.Minute))
	st = ts.app.db.GetAlertState("host:" + host.ID + ":net_high")
	if st.State != "ok" {
		t.Fatalf("state after recovery = %q, want ok", st.State)
	}
}

func TestDispatchSendsAndRetries(t *testing.T) {
	ts := newTestServer(t)

	var hits int32
	sink := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer sink.Close()
	okHook, _ := ts.app.db.CreateWebhook("global", "", sink.URL, "", "", "")
	ts.app.db.EnqueueDelivery(okHook.ID, `{"x":1}`, time.Now().UTC())

	failSink := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer failSink.Close()
	badHook, _ := ts.app.db.CreateWebhook("global", "", failSink.URL, "", "", "")
	ts.app.db.EnqueueDelivery(badHook.ID, `{"x":2}`, time.Now().UTC())

	n := ts.app.dispatchDue(time.Now().UTC())
	if n != 2 {
		t.Fatalf("attempted %d deliveries, want 2", n)
	}
	if atomic.LoadInt32(&hits) != 1 {
		t.Errorf("ok sink hit %d times, want 1", hits)
	}
	if got := countRows(t, ts, `SELECT COUNT(*) FROM webhook_deliveries WHERE status='sent'`); got != 1 {
		t.Errorf("sent deliveries = %d, want 1", got)
	}
	if got := countRows(t, ts, `SELECT COUNT(*) FROM webhook_deliveries WHERE status='failed'`); got != 1 {
		t.Errorf("failed deliveries = %d, want 1", got)
	}
}
