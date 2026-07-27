package main

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"testing/fstest"
	"time"

	"github.com/thehelvijs/Reeve/server/internal/store"
)

// publishAgent replaces the server's embedded builds with one known binary and
// returns its checksum: the value an agent running the current build reports.
// Call it before anything else touches updateContext, which memoises the set.
func publishAgent(t *testing.T, a *app) string {
	t.Helper()
	body := []byte("published-agent-binary")
	a.agentFS = fstest.MapFS{agentBinaryPrefix + "amd64": &fstest.MapFile{Data: body}}
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

// pacedSlot stamps the kind of slot the rollout grants, as opposed to the forced
// one StartHostUpdate records for an operator pressing Update. Tests about a
// dangling paced slot have to say which they mean.
func pacedSlot(t *testing.T, ts *testServer, hostID string, at time.Time) {
	t.Helper()
	if _, err := ts.app.db.SQL().Exec(
		`UPDATE hosts SET update_started_at = ?, update_forced = 0 WHERE id = ?`,
		at.UTC().Format(time.RFC3339), hostID); err != nil {
		t.Fatalf("paced slot: %v", err)
	}
}

func TestEffectiveAutoUpdateResolvesOverride(t *testing.T) {
	cases := []struct {
		policy       string
		fleetDefault bool
		want         bool
	}{
		{store.AutoUpdateDefault, true, true},
		{store.AutoUpdateDefault, false, false},
		{store.AutoUpdateOn, false, true},
		{store.AutoUpdateOff, true, false},
	}
	for _, c := range cases {
		if got := effectiveAutoUpdate(c.policy, c.fleetDefault); got != c.want {
			t.Errorf("effectiveAutoUpdate(%q, %v) = %v, want %v", c.policy, c.fleetDefault, got, c.want)
		}
	}
}

func TestUpdateStateFor(t *testing.T) {
	now := time.Date(2026, 7, 26, 10, 0, 0, 0, time.UTC)
	fresh := now.Add(-time.Minute)
	old := now.Add(-time.Hour)
	uc := updateContext{Published: map[string]bool{"cur": true}, FleetDefault: true, Stall: 15 * time.Minute}

	cases := []struct {
		name string
		host store.Host
		want string
	}{
		{"running the published build", store.Host{AgentChecksum: "cur", AutoUpdate: store.AutoUpdateDefault}, updateStateUpToDate},
		{"behind", store.Host{AgentChecksum: "old", AutoUpdate: store.AutoUpdateDefault}, updateStateOutdated},
		{"holding a live slot", store.Host{AgentChecksum: "old", AutoUpdate: store.AutoUpdateDefault, UpdateStartedAt: &fresh}, updateStateUpdating},
		{"slot past the stall window", store.Host{AgentChecksum: "old", AutoUpdate: store.AutoUpdateDefault, UpdateStartedAt: &old}, updateStateStalled},
		{"vetoed on the host", store.Host{AgentChecksum: "old", AutoUpdate: store.AutoUpdateDefault, AutoUpdateVetoed: true}, updateStateDisabled},
		{"policy off, no slot", store.Host{AgentChecksum: "old", AutoUpdate: store.AutoUpdateOff}, updateStateDisabled},
		{"policy off, operator forced a slot", store.Host{AgentChecksum: "old", AutoUpdate: store.AutoUpdateOff, UpdateStartedAt: &fresh, UpdateForced: true}, updateStateUpdating},
		{"policy off, dangling paced slot", store.Host{AgentChecksum: "old", AutoUpdate: store.AutoUpdateOff, UpdateStartedAt: &old}, updateStateDisabled},
		{"vetoed even with a forced slot", store.Host{AgentChecksum: "old", AutoUpdate: store.AutoUpdateOff, UpdateStartedAt: &fresh, UpdateForced: true, AutoUpdateVetoed: true}, updateStateDisabled},
		{"never reported", store.Host{AgentChecksum: "", AutoUpdate: store.AutoUpdateDefault}, updateStateUnknown},
	}
	for _, c := range cases {
		if got := updateStateFor(c.host, uc, now); got != c.want {
			t.Errorf("%s: state = %q, want %q", c.name, got, c.want)
		}
	}
}

// The version string decides nothing, in either direction. A rebuild at the
// same version is a different binary and has to roll out; a host running the
// published build is current even when the server's own version string differs
// (a `docker` or `dev` build stamp), which otherwise parked it on "updating"
// forever chasing an update the agent had already decided it did not need.
func TestUpdateStateForIgnoresVersionString(t *testing.T) {
	now := time.Date(2026, 7, 26, 10, 0, 0, 0, time.UTC)
	uc := updateContext{Published: map[string]bool{"cur": true}, FleetDefault: true, Stall: 15 * time.Minute}

	rebuilt := store.Host{AgentVersion: "0.1.0", AgentChecksum: "old", AutoUpdate: store.AutoUpdateDefault}
	if got := updateStateFor(rebuilt, uc, now); got != updateStateOutdated {
		t.Errorf("same version, older binary: state = %q, want %q", got, updateStateOutdated)
	}

	current := store.Host{AgentVersion: "0.1.0", AgentChecksum: "cur", AutoUpdate: store.AutoUpdateDefault}
	if got := updateStateFor(current, uc, now); got != updateStateUpToDate {
		t.Errorf("different version string, published binary: state = %q, want %q", got, updateStateUpToDate)
	}
}

// A server shipping no agent builds has nothing to update anyone to, and must
// never park a host on "updating" waiting for an update that cannot happen.
func TestUpdateStateUnknownWhenServerPublishesNothing(t *testing.T) {
	now := time.Date(2026, 7, 26, 10, 0, 0, 0, time.UTC)
	uc := updateContext{FleetDefault: true, Stall: 15 * time.Minute}
	h := store.Host{AgentChecksum: "old", AutoUpdate: store.AutoUpdateDefault}
	if got := updateStateFor(h, uc, now); got != updateStateUnknown {
		t.Errorf("state = %q, want %q", got, updateStateUnknown)
	}
}

func TestPublishedChecksumsHashesEmbeddedBuilds(t *testing.T) {
	a := &app{}
	want := publishAgent(t, a)
	got := a.publishedChecksums()
	if !got[want] {
		t.Errorf("published set %v does not contain the embedded build %s", got, want)
	}
	if len(got) != 1 {
		t.Errorf("published set has %d entries, want 1", len(got))
	}
}

func TestDecideCheckNowGrantsAndCaps(t *testing.T) {
	ts := newTestServer(t)
	publishAgent(t, ts.app)
	ts.app.db.SetSetting(settingAgentUpdateConcurrency, "1")
	now := time.Now().UTC()

	a, _ := ts.app.db.CreateHost("host-a", "linux", "", "hash-a", 60)
	b, _ := ts.app.db.CreateHost("host-b", "linux", "", "hash-b", 60)

	if !ts.app.decideCheckNow(a, "stale-sum", false, now) {
		t.Fatal("first outdated host was refused a slot")
	}
	if ts.app.decideCheckNow(b, "stale-sum", false, now) {
		t.Error("second host got a slot with concurrency 1")
	}
}

func TestDecideCheckNowRefusesVetoAndPolicyOff(t *testing.T) {
	ts := newTestServer(t)
	publishAgent(t, ts.app)
	now := time.Now().UTC()

	vetoed, _ := ts.app.db.CreateHost("vetoed", "linux", "", "hash-v", 60)
	if ts.app.decideCheckNow(vetoed, "stale-sum", true, now) {
		t.Error("a host that vetoed locally was told to update")
	}

	off, _ := ts.app.db.CreateHost("off", "linux", "", "hash-o", 60)
	ts.app.db.SetHostAutoUpdate(off.ID, store.AutoUpdateOff)
	off, _ = ts.app.db.GetHost(off.ID)
	if ts.app.decideCheckNow(off, "stale-sum", false, now) {
		t.Error("a host with policy off was told to update")
	}
}

// A veto is the owner refusing an update on their box; no host policy override
// may outrank it. This is the top-precedence rule the design calls out.
func TestDecideCheckNowVetoBeatsHostOnPolicy(t *testing.T) {
	ts := newTestServer(t)
	publishAgent(t, ts.app)
	h, _ := ts.app.db.CreateHost("vetoed-on", "linux", "", "hash-vo", 60)
	ts.app.db.SetHostAutoUpdate(h.ID, store.AutoUpdateOn)
	h, _ = ts.app.db.GetHost(h.ID)

	if ts.app.decideCheckNow(h, "stale-sum", true, time.Now().UTC()) {
		t.Error("a host pinned on but vetoed locally was told to update")
	}
}

// The slot release for a host confirming the target version must run ahead of
// the veto check, or a vetoed host's stale slot wedges the fleet forever.
func TestDecideCheckNowReleasesSlotEvenWhenVetoed(t *testing.T) {
	ts := newTestServer(t)
	now := time.Now().UTC()
	h, _ := ts.app.db.CreateHost("web-2", "linux", "", "hash-2", 60)
	ts.app.db.StartHostUpdate(h.ID, now)
	h, _ = ts.app.db.GetHost(h.ID)

	if ts.app.decideCheckNow(h, publishAgent(t, ts.app), true, now) {
		t.Error("a vetoed host reporting the target version was told to update")
	}
	got, _ := ts.app.db.GetHost(h.ID)
	if got.UpdateStartedAt != nil {
		t.Error("the slot was not released for a vetoed host reporting the target version")
	}
}

func TestDecideCheckNowHonoursHostOnAgainstFleetOff(t *testing.T) {
	ts := newTestServer(t)
	publishAgent(t, ts.app)
	ts.app.db.SetSetting(settingAgentUpdateEnabled, "false")
	on, _ := ts.app.db.CreateHost("pinned-on", "linux", "", "hash-p", 60)
	ts.app.db.SetHostAutoUpdate(on.ID, store.AutoUpdateOn)
	on, _ = ts.app.db.GetHost(on.ID)

	if !ts.app.decideCheckNow(on, "stale-sum", false, time.Now().UTC()) {
		t.Error("a host pinned on was refused while the fleet default was off")
	}
}

func TestStallHaltsTheFleet(t *testing.T) {
	ts := newTestServer(t)
	publishAgent(t, ts.app)
	ts.app.db.SetSetting(settingAgentUpdateStallSecs, "1")
	now := time.Now().UTC()

	canary, _ := ts.app.db.CreateHost("canary", "linux", "", "hash-c", 60)
	next, _ := ts.app.db.CreateHost("next", "linux", "", "hash-n", 60)

	if !ts.app.decideCheckNow(canary, "stale-sum", false, now) {
		t.Fatal("canary was refused the first slot")
	}
	later := now.Add(5 * time.Second)
	if ts.app.decideCheckNow(next, "stale-sum", false, later) {
		t.Error("a new slot was granted while a host was stalled")
	}

	if err := ts.app.db.ClearStalledUpdates(later); err != nil {
		t.Fatalf("clear stalled: %v", err)
	}
	if !ts.app.decideCheckNow(next, "stale-sum", false, later) {
		t.Error("resuming did not free a slot")
	}
}

func TestDecideCheckNowReleasesSlotOnMatchingVersion(t *testing.T) {
	ts := newTestServer(t)
	now := time.Now().UTC()
	h, _ := ts.app.db.CreateHost("web-1", "linux", "", "hash-1", 60)
	ts.app.db.StartHostUpdate(h.ID, now)
	h, _ = ts.app.db.GetHost(h.ID)

	if ts.app.decideCheckNow(h, publishAgent(t, ts.app), false, now) {
		t.Error("an up-to-date host was told to update again")
	}
	got, _ := ts.app.db.GetHost(h.ID)
	if got.UpdateStartedAt != nil {
		t.Error("the slot was not released when the host reported the target version")
	}
}

// A host that has not reported a checksum is not chased on its own, but a slot
// an operator stamped by hand is a real update: if the state read "unknown"
// while the slot existed, the next push would release it and the button an
// operator just pressed would do nothing.
func TestUpdateStateForHostWithNoChecksum(t *testing.T) {
	now := time.Date(2026, 7, 26, 10, 0, 0, 0, time.UTC)
	fresh := now.Add(-time.Minute)
	old := now.Add(-time.Hour)
	uc := updateContext{Published: map[string]bool{"cur": true}, FleetDefault: true, Stall: 15 * time.Minute}

	silent := store.Host{AgentVersion: "0.1.0", AutoUpdate: store.AutoUpdateDefault}
	if got := updateStateFor(silent, uc, now); got != updateStateUnknown {
		t.Errorf("no checksum, no slot: state = %q, want %q", got, updateStateUnknown)
	}

	granted := silent
	granted.UpdateStartedAt = &fresh
	if got := updateStateFor(granted, uc, now); got != updateStateUpdating {
		t.Errorf("no checksum, fresh slot: state = %q, want %q", got, updateStateUpdating)
	}

	granted.UpdateStartedAt = &old
	if got := updateStateFor(granted, uc, now); got != updateStateStalled {
		t.Errorf("no checksum, slot past the window: state = %q, want %q", got, updateStateStalled)
	}
}

// The manual grant has to survive the next push, which is the only thing that
// turns it into an ack telling the agent to check.
func TestDecideCheckNowKeepsAManualSlotForASilentAgent(t *testing.T) {
	ts := newTestServer(t)
	publishAgent(t, ts.app)
	now := time.Now().UTC()
	h, _ := ts.app.db.CreateHost("silent", "linux", "", "hash-s", 60)
	if err := ts.app.db.StartHostUpdate(h.ID, now); err != nil {
		t.Fatalf("start update: %v", err)
	}
	h, _ = ts.app.db.GetHost(h.ID)

	if !ts.app.decideCheckNow(h, "", false, now) {
		t.Error("a host holding an operator's slot was not told to check")
	}
	got, _ := ts.app.db.GetHost(h.ID)
	if got.UpdateStartedAt == nil {
		t.Error("the operator's slot was released on the next push")
	}
}

// An agent that has never reported a checksum cannot be judged, and that is the
// one state an operator cannot fix any other way from the UI.
func TestDecideCheckNowLeavesASilentAgentAloneWithoutASlot(t *testing.T) {
	ts := newTestServer(t)
	publishAgent(t, ts.app)
	h, _ := ts.app.db.CreateHost("silent", "linux", "", "hash-s2", 60)

	if ts.app.decideCheckNow(h, "", false, time.Now().UTC()) {
		t.Error("a host that cannot say what it runs was chased on its own")
	}
	got, _ := ts.app.db.GetHost(h.ID)
	if got.UpdateStartedAt != nil {
		t.Error("a slot was stamped for a host the server cannot judge")
	}
}
