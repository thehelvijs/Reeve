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
	ServerVersion string              `json:"server_version"`
	Counts        map[string]int      `json:"counts"`
	Paused        bool                `json:"paused"`
	Stalled       []store.StalledHost `json:"stalled"`
}

// fetchRollup reads the rollup and asserts the one invariant the whole banner
// rests on: the UI only ever says "paused" about hosts it also renders stalled.
// Without it a single response can claim the fleet is wedged while every host
// on the page looks fine.
func fetchRollup(t *testing.T, ts *testServer, c *http.Client) rollupResponse {
	t.Helper()
	resp, body := ts.do(t, c, http.MethodGet, "/api/admin/agent-updates", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("rollup status = %d, want 200: %s", resp.StatusCode, body)
	}
	var out rollupResponse
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("decode rollup: %v", err)
	}
	if out.Paused && out.Counts[updateStateStalled] == 0 {
		t.Fatalf("rollup is paused but counts.stalled is 0, so the banner names hosts the page renders as fine: %s", body)
	}
	if !out.Paused && len(out.Stalled) > 0 {
		t.Fatalf("rollup names stalled hosts but is not paused: %s", body)
	}
	for _, h := range out.Stalled {
		if h.ID == "" || h.Name == "" {
			t.Fatalf("stalled entry missing id or name, the banner cannot link it: %s", body)
		}
	}
	return out
}

func timeMinus(t *testing.T, secs int) time.Time {
	t.Helper()
	return time.Now().UTC().Add(-time.Duration(secs) * time.Second)
}

// reportBuild puts a host on a given agent build through the real push path.
func reportBuild(t *testing.T, ts *testServer, hostID, checksum string) {
	t.Helper()
	push := contracts.Push{
		AgentVersion:  "0.1.0",
		AgentChecksum: checksum,
		SentAt:        time.Now().UTC(),
	}
	if err := ts.app.db.ApplyPush(hostID, push, time.Now().UTC()); err != nil {
		t.Fatalf("apply push: %v", err)
	}
}

func TestAgentUpdateRollupCountsHosts(t *testing.T) {
	ts := newTestServer(t)
	publishAgent(t, ts.app)
	c := ts.client(t)
	signup(t, ts, c, "admin@example.com", "password123")

	current, _ := ts.app.db.CreateHost("current", "linux", "", "hash-c", 60)
	reportBuild(t, ts, current.ID, publishAgent(t, ts.app))
	behind, _ := ts.app.db.CreateHost("behind", "linux", "", "hash-b", 60)
	reportBuild(t, ts, behind.ID, "old-sum")

	out := fetchRollup(t, ts, c)
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
	publishAgent(t, ts.app)
	c := ts.client(t)
	signup(t, ts, c, "admin@example.com", "password123")
	h, _ := ts.app.db.CreateHost("web-1", "linux", "", "hash-1", 60)
	reportBuild(t, ts, h.ID, "old-sum")

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
		"/api/admin/hosts/"+h.ID+"/auto-update", map[string]string{"policy": "off"}, nil)
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
	_, body := ts.do(t, c, http.MethodGet, "/api/hosts", nil, nil)
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
		"/api/admin/hosts/"+h.ID+"/auto-update", map[string]string{"policy": "off"}, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	got, _ := ts.app.db.GetHost(h.ID)
	if got.AutoUpdate != store.AutoUpdateOff {
		t.Errorf("policy = %q, want off", got.AutoUpdate)
	}

	resp, _ = ts.do(t, c, http.MethodPut,
		"/api/admin/hosts/"+h.ID+"/auto-update", map[string]string{"policy": "sometimes"}, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("bad policy status = %d, want 400", resp.StatusCode)
	}
}

// "Never update" is a rollout policy, not a prohibition: an operator standing at
// the page and pressing Update is asking for this one host, now, and update-now
// already bypasses the concurrency cap and a paused rollout for the same reason.
func TestUpdateNowReachesAPolicyOffHost(t *testing.T) {
	ts := newTestServer(t)
	publishAgent(t, ts.app)
	c := ts.client(t)
	signup(t, ts, c, "admin@example.com", "password123")
	h, _ := ts.app.db.CreateHost("web-1", "linux", "", "hash-1", 60)
	reportBuild(t, ts, h.ID, "old-sum")
	ts.app.db.SetHostAutoUpdate(h.ID, store.AutoUpdateOff)

	resp, body := ts.do(t, c, http.MethodPost, "/api/admin/hosts/"+h.ID+"/update-now", nil, nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want 204: %s", resp.StatusCode, body)
	}
	got, _ := ts.app.db.GetHost(h.ID)
	if got.UpdateStartedAt == nil {
		t.Fatal("no slot stamped for a host the operator explicitly asked to update")
	}
	if !got.UpdateForced {
		t.Error("the slot was not marked forced, so the next push would release it as dangling")
	}
	// And the grant has to survive that next push, which is what carries it.
	if !ts.app.decideCheckNow(got, "old-sum", false, time.Now().UTC()) {
		t.Error("the host was not told to check on the push after the operator's grant")
	}
}

// The machine's own veto is the one refusal that stands: the agent ignores the
// ack, so granting a slot would strand it holding one it will never clear.
func TestUpdateNowRefusesAVetoedHost(t *testing.T) {
	ts := newTestServer(t)
	publishAgent(t, ts.app)
	c := ts.client(t)
	signup(t, ts, c, "admin@example.com", "password123")
	hostID, token := enrollHost(t, ts, c, "vetoed-host")
	push := samplePush()
	push.AutoUpdateVetoed = true
	ts.do(t, nil, http.MethodPost, "/api/ingest", push, map[string]string{"Authorization": "Bearer " + token})

	resp, _ := ts.do(t, c, http.MethodPost, "/api/admin/hosts/"+hostID+"/update-now", nil, nil)
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
	reportBuild(t, ts, h.ID, publishAgent(t, ts.app))

	resp, _ := ts.do(t, c, http.MethodPost, "/api/admin/hosts/"+h.ID+"/update-now", nil, nil)
	if resp.StatusCode != http.StatusConflict {
		t.Errorf("status = %d, want 409", resp.StatusCode)
	}
	got, _ := ts.app.db.GetHost(h.ID)
	if got.UpdateStartedAt != nil {
		t.Error("refused update-now still stamped a slot")
	}
}

// Only a server with nothing to serve is a real refusal. A host that has never
// reported a checksum is exactly what the button is for: the agent compares its
// own binary against the published one itself, so it does not need the server to
// know what it is running.
func TestUpdateNowRefusesOnlyWhenTheServerShipsNothing(t *testing.T) {
	t.Run("host never reported a checksum, server has builds", func(t *testing.T) {
		ts := newTestServer(t)
		publishAgent(t, ts.app)
		c := adminClient(t, ts)
		h, _ := ts.app.db.CreateHost("silent", "linux", "", "hash-s", 60)
		reportBuild(t, ts, h.ID, "")

		resp, body := ts.do(t, c, http.MethodPost, "/api/admin/hosts/"+h.ID+"/update-now", nil, nil)
		if resp.StatusCode != http.StatusNoContent {
			t.Fatalf("status = %d, want 204: %s", resp.StatusCode, body)
		}
		got, _ := ts.app.db.GetHost(h.ID)
		if got.UpdateStartedAt == nil {
			t.Error("no slot stamped: this is the one state an operator cannot fix any other way")
		}
	})

	t.Run("server ships no agent builds", func(t *testing.T) {
		ts := newTestServer(t)
		c := adminClient(t, ts)
		ts.app.agentFS = nil
		h, _ := ts.app.db.CreateHost("web-1", "linux", "", "hash-1", 60)
		reportBuild(t, ts, h.ID, "old-sum")

		resp, body := ts.do(t, c, http.MethodPost, "/api/admin/hosts/"+h.ID+"/update-now", nil, nil)
		if resp.StatusCode != http.StatusConflict {
			t.Fatalf("status = %d, want 409: %s", resp.StatusCode, body)
		}
		got, _ := ts.app.db.GetHost(h.ID)
		if got.UpdateStartedAt != nil {
			t.Error("a server with no agent builds stamped a slot that can never clear")
		}
	})
}

func TestUpdateNowStampsASlot(t *testing.T) {
	ts := newTestServer(t)
	publishAgent(t, ts.app)
	c := ts.client(t)
	signup(t, ts, c, "admin@example.com", "password123")
	h, _ := ts.app.db.CreateHost("web-1", "linux", "", "hash-1", 60)
	reportBuild(t, ts, h.ID, "old-sum")

	resp, _ := ts.do(t, c, http.MethodPost, "/api/admin/hosts/"+h.ID+"/update-now", nil, nil)
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
	publishAgent(t, ts.app)
	c := ts.client(t)
	signup(t, ts, c, "admin@example.com", "password123")
	ts.app.db.SetSetting(settingAgentUpdateStallSecs, "1")
	h, _ := ts.app.db.CreateHost("web-1", "linux", "", "hash-1", 60)
	reportBuild(t, ts, h.ID, "old-sum")
	pacedSlot(t, ts, h.ID, timeMinus(t, 10))

	out := fetchRollup(t, ts, c)
	if !out.Paused {
		t.Fatalf("rollup should be paused with a stalled host, got %+v", out)
	}
	if len(out.Stalled) != 1 || out.Stalled[0].ID != h.ID {
		t.Fatalf("stalled = %+v, want the web-1 row with its id", out.Stalled)
	}

	resp, _ := ts.do(t, c, http.MethodPost, "/api/admin/agent-updates/resume", nil, nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("resume status = %d, want 204", resp.StatusCode)
	}
	got, _ := ts.app.db.GetHost(h.ID)
	if got.UpdateStartedAt != nil {
		t.Error("resume did not release the stalled slot")
	}
}

// The two ways a host granted a slot can become ineligible: its owner vetoes
// locally, or an admin sets it to never update. Either way the slot must go, or
// the rollout stays wedged on a host the UI renders as "updates off" and the
// operator is told the fleet is broken by a host that looks fine.
func TestAnIneligibleHostReleasesItsSlot(t *testing.T) {
	cases := []struct {
		name        string
		disable     func(t *testing.T, ts *testServer, c *http.Client, hostID string)
		pushVetoed  bool
		wantVersion string
	}{
		{
			name:       "the host vetoes locally",
			disable:    func(*testing.T, *testServer, *http.Client, string) {},
			pushVetoed: true,
		},
		{
			name: "an admin sets it to never update",
			disable: func(t *testing.T, ts *testServer, c *http.Client, hostID string) {
				resp, body := ts.do(t, c, http.MethodPut,
					"/api/admin/hosts/"+hostID+"/auto-update", map[string]string{"policy": "off"}, nil)
				if resp.StatusCode != http.StatusOK {
					t.Fatalf("set policy off = %d: %s", resp.StatusCode, body)
				}
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ts := newTestServer(t)
			publishAgent(t, ts.app)
			c := adminClient(t, ts)
			ts.app.db.SetSetting(settingAgentUpdateStallSecs, "1")
			hostID, token := enrollHost(t, ts, c, "canary")

			p := samplePush()
			resp, body := ts.do(t, nil, http.MethodPost, "/api/ingest", p,
				map[string]string{"Authorization": "Bearer " + token})
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("first ingest = %d: %s", resp.StatusCode, body)
			}
			granted, _ := ts.app.db.GetHost(hostID)
			if granted.UpdateStartedAt == nil {
				t.Fatal("the canary was never granted a slot, so this test proves nothing")
			}

			tc.disable(t, ts, c, hostID)

			// Past the one-second stall window, still on the old version.
			time.Sleep(1100 * time.Millisecond)
			p.AutoUpdateVetoed = tc.pushVetoed
			resp, body = ts.do(t, nil, http.MethodPost, "/api/ingest", p,
				map[string]string{"Authorization": "Bearer " + token})
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("second ingest = %d: %s", resp.StatusCode, body)
			}

			out := fetchRollup(t, ts, c)
			if out.Paused {
				t.Errorf("rollout paused by a host that can no longer update: %+v", out)
			}
			if out.Counts[updateStateDisabled] != 1 {
				t.Errorf("disabled = %d, want 1: %+v", out.Counts[updateStateDisabled], out.Counts)
			}
			after, _ := ts.app.db.GetHost(hostID)
			if after.UpdateStartedAt != nil {
				t.Error("an ineligible host kept its rollout slot")
			}

			// The real cost of the wedge: another host must still be grantable.
			otherID, otherToken := enrollHost(t, ts, c, "other")
			resp, body = ts.do(t, nil, http.MethodPost, "/api/ingest", samplePush(),
				map[string]string{"Authorization": "Bearer " + otherToken})
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("other ingest = %d: %s", resp.StatusCode, body)
			}
			var ack contracts.PushAck
			json.Unmarshal(body, &ack)
			if !ack.CheckNow {
				t.Error("the fleet was still halted behind the ineligible host")
			}
			other, _ := ts.app.db.GetHost(otherID)
			if other.UpdateStartedAt == nil {
				t.Error("no slot was stamped for the next host")
			}
		})
	}
}

// A host on the `default` policy holding a stale slot while an admin turns
// the fleet toggle off must not wedge the rollup: updateStateFor already
// counts it disabled, so ListStalledHosts must agree and not name it, or the
// rollup would report paused with counts.stalled at 0 - the same
// self-contradiction the per-host `off`/veto fix closed, reached this time
// through the fleet default instead of the per-host policy.
func TestFleetToggleOffDoesNotStallOnADefaultPolicyHost(t *testing.T) {
	ts := newTestServer(t)
	publishAgent(t, ts.app)
	c := ts.client(t)
	signup(t, ts, c, "admin@example.com", "password123")
	ts.app.db.SetSetting(settingAgentUpdateStallSecs, "1")

	h, _ := ts.app.db.CreateHost("web-1", "linux", "", "hash-1", 60)
	reportBuild(t, ts, h.ID, "old-sum")
	// A slot the paced rollout granted, not one an operator forced: the two are
	// treated differently, and this scenario is about the paced one going stale.
	pacedSlot(t, ts, h.ID, timeMinus(t, 10))

	ts.app.db.SetSetting(settingAgentUpdateEnabled, "false")

	out := fetchRollup(t, ts, c)
	if out.Paused {
		t.Errorf("rollup paused by a default-policy host while the fleet toggle is off: %+v", out)
	}
	if len(out.Stalled) != 0 {
		t.Errorf("stalled = %+v, want none", out.Stalled)
	}
	if out.Counts[updateStateDisabled] != 1 {
		t.Errorf("disabled = %d, want 1: %+v", out.Counts[updateStateDisabled], out.Counts)
	}

	// The real cost of the wedge: another host must still be grantable.
	other, _ := ts.app.db.CreateHost("other", "linux", "", "hash-o", 60)
	ts.app.db.SetHostAutoUpdate(other.ID, store.AutoUpdateOn)
	reportBuild(t, ts, other.ID, "old-sum")
	other, _ = ts.app.db.GetHost(other.ID)
	if !ts.app.decideCheckNow(other, "0.0.1", false, time.Now().UTC()) {
		t.Error("the fleet was still halted behind the ineligible default-policy host")
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
		{http.MethodGet, "/api/admin/agent-updates", nil},
		{http.MethodPost, "/api/admin/agent-updates/resume", nil},
		{http.MethodPut, "/api/admin/hosts/" + h.ID + "/auto-update", map[string]string{"policy": "off"}},
		{http.MethodPost, "/api/admin/hosts/" + h.ID + "/update-now", nil},
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
