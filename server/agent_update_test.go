package main

import (
	"testing"
	"time"

	"github.com/thehelvijs/Reeve/server/internal/store"
)

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
	uc := updateContext{ServerVersion: "0.2.0", FleetDefault: true, Stall: 15 * time.Minute}

	cases := []struct {
		name string
		host store.Host
		want string
	}{
		{"matching version", store.Host{AgentVersion: "0.2.0", AutoUpdate: store.AutoUpdateDefault}, updateStateUpToDate},
		{"behind", store.Host{AgentVersion: "0.1.0", AutoUpdate: store.AutoUpdateDefault}, updateStateOutdated},
		{"holding a live slot", store.Host{AgentVersion: "0.1.0", AutoUpdate: store.AutoUpdateDefault, UpdateStartedAt: &fresh}, updateStateUpdating},
		{"slot past the stall window", store.Host{AgentVersion: "0.1.0", AutoUpdate: store.AutoUpdateDefault, UpdateStartedAt: &old}, updateStateStalled},
		{"vetoed on the host", store.Host{AgentVersion: "0.1.0", AutoUpdate: store.AutoUpdateDefault, AutoUpdateVetoed: true}, updateStateDisabled},
		{"policy off", store.Host{AgentVersion: "0.1.0", AutoUpdate: store.AutoUpdateOff}, updateStateDisabled},
		{"never reported", store.Host{AgentVersion: "", AutoUpdate: store.AutoUpdateDefault}, updateStateUnknown},
		{"dev build", store.Host{AgentVersion: "dev", AutoUpdate: store.AutoUpdateDefault}, updateStateUnknown},
	}
	for _, c := range cases {
		if got := updateStateFor(c.host, uc, now); got != c.want {
			t.Errorf("%s: state = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestUpdateStateUnknownWhenServerVersionIsDev(t *testing.T) {
	now := time.Date(2026, 7, 26, 10, 0, 0, 0, time.UTC)
	uc := updateContext{ServerVersion: "dev", FleetDefault: true, Stall: 15 * time.Minute}
	h := store.Host{AgentVersion: "0.1.0", AutoUpdate: store.AutoUpdateDefault}
	if got := updateStateFor(h, uc, now); got != updateStateUnknown {
		t.Errorf("state = %q, want %q", got, updateStateUnknown)
	}
}

func TestDecideCheckNowGrantsAndCaps(t *testing.T) {
	ts := newTestServer(t)
	ts.app.db.SetSetting(settingAgentUpdateConcurrency, "1")
	now := time.Now().UTC()

	a, _ := ts.app.db.CreateHost("host-a", "linux", "", "hash-a", 60)
	b, _ := ts.app.db.CreateHost("host-b", "linux", "", "hash-b", 60)

	if !ts.app.decideCheckNow(a, "0.0.1", false, now) {
		t.Fatal("first outdated host was refused a slot")
	}
	if ts.app.decideCheckNow(b, "0.0.1", false, now) {
		t.Error("second host got a slot with concurrency 1")
	}
}

func TestDecideCheckNowRefusesVetoAndPolicyOff(t *testing.T) {
	ts := newTestServer(t)
	now := time.Now().UTC()

	vetoed, _ := ts.app.db.CreateHost("vetoed", "linux", "", "hash-v", 60)
	if ts.app.decideCheckNow(vetoed, "0.0.1", true, now) {
		t.Error("a host that vetoed locally was told to update")
	}

	off, _ := ts.app.db.CreateHost("off", "linux", "", "hash-o", 60)
	ts.app.db.SetHostAutoUpdate(off.ID, store.AutoUpdateOff)
	off, _ = ts.app.db.GetHost(off.ID)
	if ts.app.decideCheckNow(off, "0.0.1", false, now) {
		t.Error("a host with policy off was told to update")
	}
}

// A veto is the owner refusing an update on their box; no host policy override
// may outrank it. This is the top-precedence rule the design calls out.
func TestDecideCheckNowVetoBeatsHostOnPolicy(t *testing.T) {
	ts := newTestServer(t)
	h, _ := ts.app.db.CreateHost("vetoed-on", "linux", "", "hash-vo", 60)
	ts.app.db.SetHostAutoUpdate(h.ID, store.AutoUpdateOn)
	h, _ = ts.app.db.GetHost(h.ID)

	if ts.app.decideCheckNow(h, "0.0.1", true, time.Now().UTC()) {
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

	if ts.app.decideCheckNow(h, ts.app.cfg.Version, true, now) {
		t.Error("a vetoed host reporting the target version was told to update")
	}
	got, _ := ts.app.db.GetHost(h.ID)
	if got.UpdateStartedAt != nil {
		t.Error("the slot was not released for a vetoed host reporting the target version")
	}
}

func TestDecideCheckNowHonoursHostOnAgainstFleetOff(t *testing.T) {
	ts := newTestServer(t)
	ts.app.db.SetSetting(settingAgentUpdateEnabled, "false")
	on, _ := ts.app.db.CreateHost("pinned-on", "linux", "", "hash-p", 60)
	ts.app.db.SetHostAutoUpdate(on.ID, store.AutoUpdateOn)
	on, _ = ts.app.db.GetHost(on.ID)

	if !ts.app.decideCheckNow(on, "0.0.1", false, time.Now().UTC()) {
		t.Error("a host pinned on was refused while the fleet default was off")
	}
}

func TestStallHaltsTheFleet(t *testing.T) {
	ts := newTestServer(t)
	ts.app.db.SetSetting(settingAgentUpdateStallSecs, "1")
	now := time.Now().UTC()

	canary, _ := ts.app.db.CreateHost("canary", "linux", "", "hash-c", 60)
	next, _ := ts.app.db.CreateHost("next", "linux", "", "hash-n", 60)

	if !ts.app.decideCheckNow(canary, "0.0.1", false, now) {
		t.Fatal("canary was refused the first slot")
	}
	later := now.Add(5 * time.Second)
	if ts.app.decideCheckNow(next, "0.0.1", false, later) {
		t.Error("a new slot was granted while a host was stalled")
	}

	if err := ts.app.db.ClearStalledUpdates(later); err != nil {
		t.Fatalf("clear stalled: %v", err)
	}
	if !ts.app.decideCheckNow(next, "0.0.1", false, later) {
		t.Error("resuming did not free a slot")
	}
}

func TestDecideCheckNowReleasesSlotOnMatchingVersion(t *testing.T) {
	ts := newTestServer(t)
	now := time.Now().UTC()
	h, _ := ts.app.db.CreateHost("web-1", "linux", "", "hash-1", 60)
	ts.app.db.StartHostUpdate(h.ID, now)
	h, _ = ts.app.db.GetHost(h.ID)

	if ts.app.decideCheckNow(h, ts.app.cfg.Version, false, now) {
		t.Error("an up-to-date host was told to update again")
	}
	got, _ := ts.app.db.GetHost(h.ID)
	if got.UpdateStartedAt != nil {
		t.Error("the slot was not released when the host reported the target version")
	}
}
