package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/thehelvijs/Reeve/contracts"
	"github.com/thehelvijs/Reeve/server/internal/store"
)

type rollupResponse struct {
	ServerVersion string         `json:"server_version"`
	Counts        map[string]int `json:"counts"`
	Paused        bool           `json:"paused"`
	Stalled       []string       `json:"stalled"`
}

func timeMinus(t *testing.T, secs int) time.Time {
	t.Helper()
	return time.Now().UTC().Add(-time.Duration(secs) * time.Second)
}

// reportVersion puts a host on a given agent version through the real push path.
func reportVersion(t *testing.T, ts *testServer, hostID, version string) {
	t.Helper()
	push := contracts.Push{
		ProtocolVersion: contracts.PushProtocolVersion,
		AgentVersion:    version,
		SentAt:          time.Now().UTC(),
	}
	if err := ts.app.db.ApplyPush(hostID, push, time.Now().UTC()); err != nil {
		t.Fatalf("apply push: %v", err)
	}
}

func TestAgentUpdateRollupCountsHosts(t *testing.T) {
	ts := newTestServer(t)
	c := ts.client(t)
	signup(t, ts, c, "admin@example.com", "password123")

	current, _ := ts.app.db.CreateHost("current", "linux", "", "hash-c", 60)
	reportVersion(t, ts, current.ID, ts.app.cfg.Version)
	behind, _ := ts.app.db.CreateHost("behind", "linux", "", "hash-b", 60)
	reportVersion(t, ts, behind.ID, "0.0.1")

	resp, body := ts.do(t, c, http.MethodGet, "/api/v1/admin/agent-updates", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var out rollupResponse
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.ServerVersion != ts.app.cfg.Version {
		t.Errorf("server_version = %q, want %q", out.ServerVersion, ts.app.cfg.Version)
	}
	if out.Counts[updateStateUpToDate] != 1 {
		t.Errorf("up_to_date = %d, want 1", out.Counts[updateStateUpToDate])
	}
	if out.Counts[updateStateOutdated] != 1 {
		t.Errorf("outdated = %d, want 1", out.Counts[updateStateOutdated])
	}
	if len(out.Counts) != 6 {
		t.Errorf("counts has %d keys, want all 6 update states present: %+v", len(out.Counts), out.Counts)
	}
	if out.Paused {
		t.Error("rollup reports paused with nothing stalled")
	}
}

// The list endpoint is the only place an operator sees these fields; a
// dropped field or a wrong JSON tag would pass every other test in this file.
func TestListHostsReportsAutoUpdateAndState(t *testing.T) {
	ts := newTestServer(t)
	c := ts.client(t)
	signup(t, ts, c, "admin@example.com", "password123")
	h, _ := ts.app.db.CreateHost("web-1", "linux", "", "hash-1", 60)
	reportVersion(t, ts, h.ID, "0.0.1")

	before, rawList := hostFromList(t, ts, c, h.ID)
	if before.AutoUpdate != store.AutoUpdateDefault {
		t.Errorf("auto_update = %q, want %q", before.AutoUpdate, store.AutoUpdateDefault)
	}
	if before.UpdateState != updateStateOutdated {
		t.Errorf("update_state = %q, want %q", before.UpdateState, updateStateOutdated)
	}
	// Guards the wire key itself, not just the Go struct round trip.
	for _, key := range []string{`"auto_update":"default"`, `"update_state":"outdated"`} {
		if !strings.Contains(string(rawList), key) {
			t.Errorf("raw list response missing %s: %s", key, rawList)
		}
	}

	resp, body := ts.do(t, c, http.MethodPut,
		"/api/v1/admin/hosts/"+h.ID+"/auto-update", map[string]string{"policy": "off"}, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var putResp hostView
	if err := json.Unmarshal(body, &putResp); err != nil {
		t.Fatalf("decode put response: %v", err)
	}
	if putResp.AutoUpdate != store.AutoUpdateOff || putResp.UpdateState != updateStateDisabled {
		t.Errorf("put response auto_update/update_state = %q/%q, want off/disabled", putResp.AutoUpdate, putResp.UpdateState)
	}

	after, _ := hostFromList(t, ts, c, h.ID)
	if after.UpdateState != updateStateDisabled {
		t.Errorf("list update_state after policy off = %q, want %q", after.UpdateState, updateStateDisabled)
	}
}

func hostFromList(t *testing.T, ts *testServer, c *http.Client, id string) (hostView, []byte) {
	t.Helper()
	_, body := ts.do(t, c, http.MethodGet, "/api/v1/hosts", nil, nil)
	var hosts []hostView
	if err := json.Unmarshal(body, &hosts); err != nil {
		t.Fatalf("decode hosts: %v", err)
	}
	for _, h := range hosts {
		if h.ID == id {
			return h, body
		}
	}
	t.Fatalf("host %s missing from list", id)
	return hostView{}, nil
}

func TestSetHostAutoUpdatePolicy(t *testing.T) {
	ts := newTestServer(t)
	c := ts.client(t)
	signup(t, ts, c, "admin@example.com", "password123")
	h, _ := ts.app.db.CreateHost("web-1", "linux", "", "hash-1", 60)

	resp, _ := ts.do(t, c, http.MethodPut,
		"/api/v1/admin/hosts/"+h.ID+"/auto-update", map[string]string{"policy": "off"}, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	got, _ := ts.app.db.GetHost(h.ID)
	if got.AutoUpdate != store.AutoUpdateOff {
		t.Errorf("policy = %q, want off", got.AutoUpdate)
	}

	resp, _ = ts.do(t, c, http.MethodPut,
		"/api/v1/admin/hosts/"+h.ID+"/auto-update", map[string]string{"policy": "sometimes"}, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("bad policy status = %d, want 400", resp.StatusCode)
	}
}

func TestUpdateNowRefusesDisabledHost(t *testing.T) {
	ts := newTestServer(t)
	c := ts.client(t)
	signup(t, ts, c, "admin@example.com", "password123")
	h, _ := ts.app.db.CreateHost("web-1", "linux", "", "hash-1", 60)
	ts.app.db.SetHostAutoUpdate(h.ID, store.AutoUpdateOff)

	resp, _ := ts.do(t, c, http.MethodPost, "/api/v1/admin/hosts/"+h.ID+"/update-now", nil, nil)
	if resp.StatusCode != http.StatusConflict {
		t.Errorf("status = %d, want 409", resp.StatusCode)
	}
}

// A host already on the server's version must never hold a slot: an offline
// one would occupy it until the stall window pauses the whole fleet.
func TestUpdateNowRefusesUpToDateHost(t *testing.T) {
	ts := newTestServer(t)
	c := ts.client(t)
	signup(t, ts, c, "admin@example.com", "password123")
	h, _ := ts.app.db.CreateHost("web-1", "linux", "", "hash-1", 60)
	reportVersion(t, ts, h.ID, ts.app.cfg.Version)

	resp, _ := ts.do(t, c, http.MethodPost, "/api/v1/admin/hosts/"+h.ID+"/update-now", nil, nil)
	if resp.StatusCode != http.StatusConflict {
		t.Errorf("status = %d, want 409", resp.StatusCode)
	}
	got, _ := ts.app.db.GetHost(h.ID)
	if got.UpdateStartedAt != nil {
		t.Error("refused update-now still stamped a slot")
	}
}

func TestUpdateNowStampsASlot(t *testing.T) {
	ts := newTestServer(t)
	c := ts.client(t)
	signup(t, ts, c, "admin@example.com", "password123")
	h, _ := ts.app.db.CreateHost("web-1", "linux", "", "hash-1", 60)

	resp, _ := ts.do(t, c, http.MethodPost, "/api/v1/admin/hosts/"+h.ID+"/update-now", nil, nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", resp.StatusCode)
	}
	got, _ := ts.app.db.GetHost(h.ID)
	if got.UpdateStartedAt == nil {
		t.Error("update-now did not stamp a rollout slot")
	}
}

func TestResumeClearsStalledHosts(t *testing.T) {
	ts := newTestServer(t)
	c := ts.client(t)
	signup(t, ts, c, "admin@example.com", "password123")
	ts.app.db.SetSetting(settingAgentUpdateStallSecs, "1")
	h, _ := ts.app.db.CreateHost("web-1", "linux", "", "hash-1", 60)
	reportVersion(t, ts, h.ID, "0.0.1")
	ts.app.db.StartHostUpdate(h.ID, timeMinus(t, 10))

	resp, body := ts.do(t, c, http.MethodGet, "/api/v1/admin/agent-updates", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("rollup status = %d, want 200: %s", resp.StatusCode, body)
	}
	var out rollupResponse
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("decode rollup: %v", err)
	}
	if !out.Paused {
		t.Fatalf("rollup should be paused with a stalled host, got %s", body)
	}

	resp, _ = ts.do(t, c, http.MethodPost, "/api/v1/admin/agent-updates/resume", nil, nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("resume status = %d, want 204", resp.StatusCode)
	}
	got, _ := ts.app.db.GetHost(h.ID)
	if got.UpdateStartedAt != nil {
		t.Error("resume did not release the stalled slot")
	}
}

func TestAgentUpdateEndpointsAreAdminOnly(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.client(t)
	signup(t, ts, admin, "admin@example.com", "password123")
	basic := ts.client(t)
	signup(t, ts, basic, "dev@example.com", "password123")
	h, _ := ts.app.db.CreateHost("web-1", "linux", "", "hash-1", 60)

	cases := []struct {
		method string
		path   string
		body   any
	}{
		{http.MethodGet, "/api/v1/admin/agent-updates", nil},
		{http.MethodPost, "/api/v1/admin/agent-updates/resume", nil},
		{http.MethodPut, "/api/v1/admin/hosts/" + h.ID + "/auto-update", map[string]string{"policy": "off"}},
		{http.MethodPost, "/api/v1/admin/hosts/" + h.ID + "/update-now", nil},
	}
	for _, tc := range cases {
		resp, _ := ts.do(t, basic, tc.method, tc.path, tc.body, nil)
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("%s %s status = %d, want 403", tc.method, tc.path, resp.StatusCode)
		}
	}
}

func TestSettingsCarryAgentUpdateSection(t *testing.T) {
	ts := newTestServer(t)
	c := ts.client(t)
	signup(t, ts, c, "admin@example.com", "password123")

	resp, v := putSettings(t, ts, c, map[string]any{
		"agent_update": map[string]any{"enabled": false, "concurrency": 5, "stall_secs": 600},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if v.AgentUpdate.Enabled || v.AgentUpdate.Concurrency != 5 || v.AgentUpdate.StallSecs != 600 {
		t.Errorf("agent_update = %+v, want {false 5 600}", v.AgentUpdate)
	}

	resp, _ = putSettings(t, ts, c, map[string]any{
		"agent_update": map[string]any{"enabled": true, "concurrency": 0, "stall_secs": 600},
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("concurrency 0 status = %d, want 400", resp.StatusCode)
	}
}
