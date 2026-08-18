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
	"github.com/thehelvijs/Reeve/server/internal/store"
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

	resp, data := ts.do(t, admin, http.MethodGet, "/api/hosts", nil, nil)
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

// Re-issuing hands back a token that works and retires the one it replaced,
// which is what makes it usable on a host created before anyone needed an agent
// on it — the server's own row included.
func TestReissueEnrollTokenReplacesTheOldOne(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	ts.app.cfg.PublicURL = ts.srv.URL
	hostID, oldToken := enrollHost(t, ts, admin, "h")

	resp, data := ts.do(t, admin, http.MethodPost, "/api/admin/hosts/"+hostID+"/enroll-token", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("enroll-token = %d: %s", resp.StatusCode, data)
	}
	var out struct {
		InstallCommand string `json:"install_command"`
		RunCommand     string `json:"run_command"`
	}
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	newToken := tokenFromCommand(t, out.InstallCommand)
	if newToken == oldToken {
		t.Fatal("re-issue returned the same token")
	}
	if !strings.Contains(out.RunCommand, newToken) {
		t.Error("the docker command carries a different token than the installer")
	}

	push := samplePush()
	fresh, _ := ts.do(t, nil, http.MethodPost, "/api/ingest", push,
		map[string]string{"Authorization": "Bearer " + newToken})
	if fresh.StatusCode != http.StatusOK {
		t.Errorf("push with the re-issued token = %d, want 200", fresh.StatusCode)
	}
	stale, _ := ts.do(t, nil, http.MethodPost, "/api/ingest", push,
		map[string]string{"Authorization": "Bearer " + oldToken})
	if stale.StatusCode != http.StatusUnauthorized {
		t.Errorf("push with the replaced token = %d, want 401", stale.StatusCode)
	}
}

func tokenFromCommand(t *testing.T, cmd string) string {
	t.Helper()
	_, rest, ok := strings.Cut(cmd, "REEVE_AGENT_TOKEN=")
	if !ok {
		t.Fatalf("no token in %q", cmd)
	}
	token, _, _ := strings.Cut(rest, " ")
	return token
}

func TestAgentInstallCommandShape(t *testing.T) {
	a := bareApp(t, config{PublicURL: "http://10.0.0.2:8080"})
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

// With no public URL configured the enroll commands use the address the admin's
// browser reached the UI on, which on a LAN is the server's LAN address.
func TestAgentCommandsFallBackToRequestHost(t *testing.T) {
	a := bareApp(t, config{Addr: "127.0.0.1:8080"})
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

// Deleting a host has to take its telemetry with it. Every one of these tables
// hangs off hosts(id) ON DELETE CASCADE, which only fires because the store
// opens SQLite with foreign_keys(ON) — without that pragma the rows would
// silently outlive the host they describe.
func TestDeleteHostTakesItsTelemetryWithIt(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.client(t)
	signup(t, ts, admin, "admin@example.com", "password123")

	h, err := ts.app.db.CreateHost("doomed", "linux", "", "tok", 60)
	if err != nil {
		t.Fatalf("create host: %v", err)
	}
	keep, err := ts.app.db.CreateHost("survivor", "linux", "", "tok2", 60)
	if err != nil {
		t.Fatalf("create second host: %v", err)
	}
	sql := ts.app.db.SQL()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	for _, h := range []string{h.ID, keep.ID} {
		sql.Exec(`INSERT INTO metric_samples(host_id, ts, resolution, cpu_pct) VALUES (?,?,'raw',9)`, h, now)
		sql.Exec(`INSERT INTO log_events(id, host_id, level, message, at) VALUES (?,?,'error','boom',?)`, "le-"+h, h, now)
		sql.Exec(`INSERT INTO service_status(host_id, unit, updated_at) VALUES (?,'nginx.service',?)`, h, now)
		sql.Exec(`INSERT INTO container_status(host_id, container_id, updated_at) VALUES (?,'abc',?)`, h, now)
		sql.Exec(`INSERT INTO cron_jobs(host_id, name, updated_at) VALUES (?,'backup',?)`, h, now)
		sql.Exec(`INSERT INTO container_stats(host_id, container_id, ts, resolution) VALUES (?,'abc',?,'raw')`, h, now)
		sql.Exec(`INSERT INTO alert_thresholds(host_id, metric, threshold) VALUES (?,'cpu',50)`, h)
	}

	resp, data := ts.do(t, admin, http.MethodDelete, "/api/admin/hosts/"+h.ID, nil, nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete host = %d: %s", resp.StatusCode, data)
	}
	if _, err := ts.app.db.GetHost(h.ID); err == nil {
		t.Error("host still resolves after delete")
	}

	// alert_thresholds has no foreign key, so the handler clears it explicitly;
	// the rest ride the cascade. Either way nothing may be left behind.
	for _, table := range []string{
		"metric_samples", "log_events", "service_status", "container_status",
		"cron_jobs", "container_stats", "alert_thresholds",
	} {
		if n := countRows(t, ts, `SELECT COUNT(*) FROM `+table+` WHERE host_id = ?`, h.ID); n != 0 {
			t.Errorf("%s kept %d rows for the deleted host", table, n)
		}
		if n := countRows(t, ts, `SELECT COUNT(*) FROM `+table+` WHERE host_id = ?`, keep.ID); n != 1 {
			t.Errorf("%s lost the other host's row (%d left)", table, n)
		}
	}

	// The global thresholds live in the same table under host_id '' and must
	// survive, or deleting one host would disarm alerting for the whole fleet.
	if n := countRows(t, ts, `SELECT COUNT(*) FROM alert_thresholds WHERE host_id = ''`); n == 0 {
		t.Error("deleting a host wiped the global thresholds")
	}

	resp, _ = ts.do(t, admin, http.MethodDelete, "/api/admin/hosts/"+h.ID, nil, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("second delete = %d, want 404", resp.StatusCode)
	}
}

// tools.host_id carries no foreign key, so a linked tool outlives its host with
// a dangling reference. It has to keep listing rather than 500 — the tool is
// still a real service someone catalogued.
func TestDeleteHostLeavesLinkedToolsListable(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.client(t)
	signup(t, ts, admin, "admin@example.com", "password123")

	h, err := ts.app.db.CreateHost("doomed", "linux", "", "tok", 60)
	if err != nil {
		t.Fatalf("create host: %v", err)
	}
	tool := createTool(t, ts, admin, toolInput{Name: "Orphan", HostID: h.ID})

	if resp, _ := ts.do(t, admin, http.MethodDelete, "/api/admin/hosts/"+h.ID, nil, nil); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete host = %d", resp.StatusCode)
	}

	resp, data := ts.do(t, admin, http.MethodGet, "/api/tools", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list tools after host delete = %d: %s", resp.StatusCode, data)
	}
	if !bytes.Contains(data, []byte("Orphan")) {
		t.Errorf("the tool vanished with its host: %s", data)
	}
	resp, data = ts.do(t, admin, http.MethodGet, "/api/tools/"+tool.ID, nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("tool detail after host delete = %d: %s", resp.StatusCode, data)
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
	resp, data := ts.do(t, admin, http.MethodPatch, "/api/admin/hosts/"+h.ID, body, nil)
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
	_, list := ts.do(t, admin, http.MethodGet, "/api/hosts", nil, nil)
	if !bytes.Contains(list, []byte("56.946")) {
		t.Errorf("host list missing latitude: %s", list)
	}
}

func TestUpdateHostPinColor(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.client(t)
	signup(t, ts, admin, "admin@example.com", "password123")

	h, err := ts.app.db.CreateHost("pin-box", "linux", "", "tok", 60)
	if err != nil {
		t.Fatalf("create host: %v", err)
	}
	path := "/api/admin/hosts/" + h.ID

	// A hex colour is stored and served back, to the admin list and to the
	// anonymous portal, which is what draws the pin.
	resp, data := ts.do(t, admin, http.MethodPatch, path,
		map[string]any{"physical_location": "Riga", "latitude": 56.9, "longitude": 24.1, "pin_color": "#ff8800"}, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("patch = %d: %s", resp.StatusCode, data)
	}
	if !bytes.Contains(data, []byte(`"pin_color":"#ff8800"`)) {
		t.Errorf("response missing the colour: %s", data)
	}
	if _, list := ts.do(t, admin, http.MethodGet, "/api/hosts", nil, nil); !bytes.Contains(list, []byte("#ff8800")) {
		t.Errorf("host list missing the colour: %s", list)
	}

	// Anything that is not #rrggbb is refused here: the value is interpolated
	// into the marker's inline style on the page.
	for _, bad := range []string{"red", "#fff", "#ff8800; content: url(x)", "javascript:alert(1)", "#gggggg"} {
		body := map[string]any{"physical_location": "Riga", "pin_color": bad}
		if resp, data := ts.do(t, admin, http.MethodPatch, path, body, nil); resp.StatusCode != http.StatusBadRequest {
			t.Errorf("pin_color %q = %d, want 400: %s", bad, resp.StatusCode, data)
		}
	}

	// The refusals left the stored colour alone.
	_, after := ts.do(t, admin, http.MethodGet, "/api/hosts", nil, nil)
	if !bytes.Contains(after, []byte("#ff8800")) {
		t.Errorf("a rejected colour overwrote the stored one: %s", after)
	}

	// Empty clears it back to the brand accent.
	if resp, data := ts.do(t, admin, http.MethodPatch, path,
		map[string]any{"physical_location": "Riga", "pin_color": ""}, nil); resp.StatusCode != http.StatusOK {
		t.Fatalf("clearing the colour = %d: %s", resp.StatusCode, data)
	}
	if _, list := ts.do(t, admin, http.MethodGet, "/api/hosts", nil, nil); bytes.Contains(list, []byte("#ff8800")) {
		t.Errorf("colour survived being cleared: %s", list)
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
	resp, _ := ts.do(t, basic, http.MethodDelete, "/api/admin/hosts/"+host.ID+"/metrics", nil, nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("basic clear = %d, want 403", resp.StatusCode)
	}

	resp2, data := ts.do(t, admin, http.MethodDelete, "/api/admin/hosts/"+host.ID+"/metrics", nil, nil)
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

	resp3, _ := ts.do(t, admin, http.MethodDelete, "/api/admin/hosts/nope/metrics", nil, nil)
	if resp3.StatusCode != http.StatusNotFound {
		t.Errorf("clear unknown host = %d, want 404", resp3.StatusCode)
	}
}

// TestNonAdminCannotSeeWhatAHostRuns pins the split agreed for infrastructure
// visibility: status stays open to every account, the unit/container/cron and
// process lists do not.
func TestNonAdminCannotSeeWhatAHostRuns(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	host := newHost(t, ts, admin, "box")
	user := ts.client(t)
	signup(t, ts, user, "basic@example.com", "password123")

	for _, path := range []string{"/api/hosts/" + host + "/inventory", "/api/hosts/" + host + "/process-usage"} {
		if resp, _ := ts.do(t, user, http.MethodGet, path, nil, nil); resp.StatusCode != http.StatusForbidden {
			t.Errorf("basic user GET %s = %d, want 403", path, resp.StatusCode)
		}
		if resp, _ := ts.do(t, admin, http.MethodGet, path, nil, nil); resp.StatusCode != http.StatusOK {
			t.Errorf("admin GET %s = %d, want 200", path, resp.StatusCode)
		}
	}

	// The dashboard still works: the host list and its metrics stay readable.
	for _, path := range []string{"/api/hosts", "/api/hosts/" + host + "/metrics"} {
		if resp, _ := ts.do(t, user, http.MethodGet, path, nil, nil); resp.StatusCode != http.StatusOK {
			t.Errorf("basic user GET %s = %d, want 200", path, resp.StatusCode)
		}
	}
}

func TestNonAdminDoesNotSeeOtherPeoplesEmails(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	user := ts.client(t)
	signup(t, ts, user, "basic@example.com", "password123")

	_, data := ts.do(t, user, http.MethodGet, "/api/principals", nil, nil)
	var view principalsView
	if err := json.Unmarshal(data, &view); err != nil {
		t.Fatalf("decode principals: %v", err)
	}
	if len(view.Users) < 2 {
		t.Fatalf("users = %d, want at least 2", len(view.Users))
	}
	for _, u := range view.Users {
		if u.Email != "" && u.Email != "basic@example.com" {
			t.Errorf("basic user sees %q, want only their own address", u.Email)
		}
		if u.DisplayName == "" {
			t.Errorf("user %s has no label left for the picker", u.ID)
		}
	}

	_, data = ts.do(t, admin, http.MethodGet, "/api/principals", nil, nil)
	json.Unmarshal(data, &view)
	for _, u := range view.Users {
		if u.Email == "" {
			t.Errorf("admin sees an empty address for %s", u.ID)
		}
	}
}

// A host's name is a label somebody typed once, the server's own row starts on a
// default nobody chose, and what a machine is for is written nowhere else.
func TestUpdateHostIdentity(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	if err := ts.app.db.EnsureServerHost("linux"); err != nil {
		t.Fatal(err)
	}

	for _, id := range []string{store.ServerHostID, mustCreateHost(t, ts, "old-name")} {
		resp, data := ts.do(t, admin, http.MethodPut, "/api/admin/hosts/"+id+"/identity",
			map[string]string{"name": "  Renamed " + id + "  ", "description": "  runs the backups  "}, nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("update %s = %d: %s", id, resp.StatusCode, data)
		}
		var view hostView
		if err := json.Unmarshal(data, &view); err != nil {
			t.Fatal(err)
		}
		if view.Name != "Renamed "+id || view.Description != "runs the backups" {
			t.Errorf("view = %q / %q, want the trimmed values", view.Name, view.Description)
		}
		got, err := ts.app.db.GetHost(id)
		if err != nil || got.Name != "Renamed "+id || got.Description != "runs the backups" {
			t.Errorf("stored = %q / %q (%v)", got.Name, got.Description, err)
		}

		// A description can be cleared; a name cannot.
		cleared, cdata := ts.do(t, admin, http.MethodPut, "/api/admin/hosts/"+id+"/identity",
			map[string]string{"name": "Renamed " + id, "description": ""}, nil)
		if cleared.StatusCode != http.StatusOK {
			t.Fatalf("clear description = %d: %s", cleared.StatusCode, cdata)
		}
		if after, _ := ts.app.db.GetHost(id); after.Description != "" {
			t.Errorf("description = %q, want it cleared", after.Description)
		}
		blank, _ := ts.do(t, admin, http.MethodPut, "/api/admin/hosts/"+id+"/identity",
			map[string]string{"name": "   "}, nil)
		if blank.StatusCode != http.StatusBadRequest {
			t.Errorf("blank name = %d, want 400", blank.StatusCode)
		}
	}

	missing, _ := ts.do(t, admin, http.MethodPut, "/api/admin/hosts/nope/identity",
		map[string]string{"name": "x"}, nil)
	if missing.StatusCode != http.StatusNotFound {
		t.Errorf("update of an absent host = %d, want 404", missing.StatusCode)
	}
}

func mustCreateHost(t *testing.T, ts *testServer, name string) string {
	t.Helper()
	h, err := ts.app.db.CreateHost(name, "linux", "", "hash-"+name, 60)
	if err != nil {
		t.Fatal(err)
	}
	return h.ID
}
