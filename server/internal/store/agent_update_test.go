package store

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/thehelvijs/Reeve/contracts"
)

func TestHostDefaultsToInheritedPolicy(t *testing.T) {
	db := openTemp(t)
	h, err := db.CreateHost("web-1", "linux", "", "hash-1", 60)
	if err != nil {
		t.Fatalf("create host: %v", err)
	}
	if h.AutoUpdate != AutoUpdateDefault {
		t.Errorf("auto_update = %q, want %q", h.AutoUpdate, AutoUpdateDefault)
	}
	if h.AutoUpdateVetoed {
		t.Error("a fresh host must not report a veto")
	}
	if h.UpdateStartedAt != nil {
		t.Error("a fresh host must not hold a rollout slot")
	}
}

func TestSetHostAutoUpdatePersists(t *testing.T) {
	db := openTemp(t)
	h, _ := db.CreateHost("web-1", "linux", "", "hash-1", 60)
	if err := db.SetHostAutoUpdate(h.ID, AutoUpdateOff); err != nil {
		t.Fatalf("set policy: %v", err)
	}
	got, err := db.GetHost(h.ID)
	if err != nil {
		t.Fatalf("get host: %v", err)
	}
	if got.AutoUpdate != AutoUpdateOff {
		t.Errorf("auto_update = %q, want %q", got.AutoUpdate, AutoUpdateOff)
	}
}

func TestApplyPushRecordsVeto(t *testing.T) {
	db := openTemp(t)
	h, _ := db.CreateHost("web-1", "linux", "", "hash-1", 60)
	push := contracts.Push{
		ProtocolVersion:  contracts.PushProtocolVersion,
		AgentVersion:     "0.1.0",
		AutoUpdateVetoed: true,
	}
	if err := db.ApplyPush(h.ID, push, time.Now().UTC()); err != nil {
		t.Fatalf("apply push: %v", err)
	}
	got, _ := db.GetHost(h.ID)
	if !got.AutoUpdateVetoed {
		t.Error("veto reported in the push was not stored")
	}
}

// The owner may veto after the server already granted the host a slot. Keeping
// that slot halts every other host behind a row the UI renders "updates off".
func TestApplyPushWithAVetoReleasesTheSlot(t *testing.T) {
	db := openTemp(t)
	h, _ := db.CreateHost("web-1", "linux", "", "hash-1", 60)
	db.StartHostUpdate(h.ID, time.Now().UTC())

	vetoed := contracts.Push{ProtocolVersion: contracts.PushProtocolVersion, AgentVersion: "0.1.0", AutoUpdateVetoed: true}
	if err := db.ApplyPush(h.ID, vetoed, time.Now().UTC()); err != nil {
		t.Fatalf("apply push: %v", err)
	}
	got, _ := db.GetHost(h.ID)
	if got.UpdateStartedAt != nil {
		t.Fatal("a vetoing push kept its rollout slot")
	}

	// A push without a veto must leave a live slot alone, or every heartbeat
	// would cancel the update the host is in the middle of.
	db.StartHostUpdate(h.ID, time.Now().UTC())
	plain := contracts.Push{ProtocolVersion: contracts.PushProtocolVersion, AgentVersion: "0.1.0"}
	if err := db.ApplyPush(h.ID, plain, time.Now().UTC()); err != nil {
		t.Fatalf("apply push: %v", err)
	}
	got, _ = db.GetHost(h.ID)
	if got.UpdateStartedAt == nil {
		t.Error("an ordinary push released a live rollout slot")
	}
}

// "Never update" is how an admin takes a slow host out of the rollout. If the
// slot survived, the rollout would stay wedged on a host reading "updates off".
func TestSetHostAutoUpdateOffReleasesTheSlot(t *testing.T) {
	db := openTemp(t)
	h, _ := db.CreateHost("web-1", "linux", "", "hash-1", 60)
	db.StartHostUpdate(h.ID, time.Now().UTC())

	if err := db.SetHostAutoUpdate(h.ID, AutoUpdateOff); err != nil {
		t.Fatalf("set policy: %v", err)
	}
	got, _ := db.GetHost(h.ID)
	if got.UpdateStartedAt != nil {
		t.Fatal("setting the policy off kept the rollout slot")
	}

	// Switching back on must not resurrect or invent a slot.
	db.StartHostUpdate(h.ID, time.Now().UTC())
	if err := db.SetHostAutoUpdate(h.ID, AutoUpdateOn); err != nil {
		t.Fatalf("set policy on: %v", err)
	}
	got, _ = db.GetHost(h.ID)
	if got.UpdateStartedAt == nil {
		t.Error("setting the policy on released a live slot")
	}
}

// A slot on a host that can no longer update is dangling. It must neither be
// named as stalled nor block a grant, or one ineligible host halts the fleet.
func TestIneligibleHostsDoNotHoldTheRollout(t *testing.T) {
	started := time.Date(2026, 7, 26, 10, 0, 0, 0, time.UTC)
	later := started.Add(time.Hour)
	cutoff := later.Add(-15 * time.Minute)

	cases := []struct {
		name  string
		spoil func(db *DB, id string)
	}{
		{"policy off", func(db *DB, id string) {
			db.exec1(`UPDATE hosts SET auto_update = ? WHERE id = ?`, AutoUpdateOff, id)
		}},
		{"vetoed on the host", func(db *DB, id string) {
			db.exec1(`UPDATE hosts SET auto_update_vetoed = 1 WHERE id = ?`, id)
		}},
	}
	for _, c := range cases {
		db := openTemp(t)
		dangling, _ := db.CreateHost("dangling", "linux", "", "hash-d", 60)
		next, _ := db.CreateHost("next", "linux", "", "hash-n", 60)
		db.StartHostUpdate(dangling.ID, started)
		c.spoil(db, dangling.ID)

		stalled, err := db.ListStalledHosts(cutoff, true)
		if err != nil {
			t.Fatalf("%s: list stalled: %v", c.name, err)
		}
		if len(stalled) != 0 {
			t.Errorf("%s: stalled = %+v, want none", c.name, stalled)
		}
		ok, err := db.TryStartHostUpdate(next.ID, later, cutoff, 1, true)
		if err != nil {
			t.Fatalf("%s: try start: %v", c.name, err)
		}
		if !ok {
			t.Errorf("%s: a dangling slot blocked a grant", c.name)
		}
	}
}

// A host left on the `default` policy is also ineligible once the fleet
// toggle itself is off, same as a host explicitly set to `off`: its stale slot
// must not block a grant to another host or fabricate a stalled entry.
func TestFleetDefaultOffMakesADefaultPolicyHostIneligible(t *testing.T) {
	started := time.Date(2026, 7, 26, 10, 0, 0, 0, time.UTC)
	later := started.Add(time.Hour)
	cutoff := later.Add(-15 * time.Minute)

	db := openTemp(t)
	dangling, _ := db.CreateHost("dangling", "linux", "", "hash-d", 60)
	next, _ := db.CreateHost("next", "linux", "", "hash-n", 60)
	db.StartHostUpdate(dangling.ID, started)

	stalled, err := db.ListStalledHosts(cutoff, false)
	if err != nil {
		t.Fatalf("list stalled: %v", err)
	}
	if len(stalled) != 0 {
		t.Errorf("stalled = %+v, want none: the fleet default is off", stalled)
	}
	ok, err := db.TryStartHostUpdate(next.ID, later, cutoff, 1, false)
	if err != nil {
		t.Fatalf("try start: %v", err)
	}
	if !ok {
		t.Error("a default-policy host's stale slot blocked a grant while the fleet toggle was off")
	}

	// A host pinned `on` overrides the fleet default and must still block.
	pinned, _ := db.CreateHost("pinned", "linux", "", "hash-p", 60)
	db.SetHostAutoUpdate(pinned.ID, AutoUpdateOn)
	db.StartHostUpdate(pinned.ID, started)
	stalled, err = db.ListStalledHosts(cutoff, false)
	if err != nil {
		t.Fatalf("list stalled after pin: %v", err)
	}
	if len(stalled) != 1 || stalled[0].ID != pinned.ID {
		t.Errorf("stalled = %+v, want only the pinned-on host", stalled)
	}
}

// Resume is deliberately blind to policy, so it also sweeps dangling slots the
// blocking queries already ignore.
func TestClearStalledUpdatesSweepsIneligibleHostsToo(t *testing.T) {
	db := openTemp(t)
	h, _ := db.CreateHost("web-1", "linux", "", "hash-1", 60)
	started := time.Date(2026, 7, 26, 10, 0, 0, 0, time.UTC)
	db.StartHostUpdate(h.ID, started)
	db.exec1(`UPDATE hosts SET auto_update_vetoed = 1 WHERE id = ?`, h.ID)

	if err := db.ClearStalledUpdates(started.Add(time.Hour)); err != nil {
		t.Fatalf("clear stalled: %v", err)
	}
	got, _ := db.GetHost(h.ID)
	if got.UpdateStartedAt != nil {
		t.Error("resume left a dangling slot on an ineligible host")
	}
}

func TestUpdateSlotLifecycle(t *testing.T) {
	db := openTemp(t)
	h, _ := db.CreateHost("web-1", "linux", "", "hash-1", 60)
	other, _ := db.CreateHost("web-2", "linux", "", "hash-2", 60)
	now := time.Date(2026, 7, 26, 10, 0, 0, 0, time.UTC)
	cutoff := now.Add(-15 * time.Minute)

	if err := db.StartHostUpdate(h.ID, now); err != nil {
		t.Fatalf("start update: %v", err)
	}
	stalled, err := db.ListStalledHosts(cutoff, true)
	if err != nil {
		t.Fatalf("list stalled: %v", err)
	}
	if len(stalled) != 0 {
		t.Errorf("stalled = %v, want none: the slot is still live", stalled)
	}
	if ok, err := db.TryStartHostUpdate(other.ID, now, cutoff, 1, true); err != nil {
		t.Fatalf("try start: %v", err)
	} else if ok {
		t.Error("granted a slot while the live one held the cap of 1")
	}

	if err := db.ClearHostUpdateSlot(h.ID); err != nil {
		t.Fatalf("clear slot: %v", err)
	}
	ok, err := db.TryStartHostUpdate(other.ID, now, cutoff, 1, true)
	if err != nil {
		t.Fatalf("try start after clear: %v", err)
	}
	if !ok {
		t.Error("clearing the slot did not free the cap for another host")
	}
}

func TestStalledSlotsAreCountedNamedAndCleared(t *testing.T) {
	db := openTemp(t)
	h, _ := db.CreateHost("web-1", "linux", "", "hash-1", 60)
	started := time.Date(2026, 7, 26, 10, 0, 0, 0, time.UTC)
	cutoff := started.Add(15 * time.Minute)

	db.StartHostUpdate(h.ID, started)

	stalled, err := db.ListStalledHosts(cutoff, true)
	if err != nil {
		t.Fatalf("list stalled: %v", err)
	}
	if len(stalled) != 1 || stalled[0].Name != "web-1" || stalled[0].ID != h.ID {
		t.Errorf("stalled = %+v, want the web-1 row with its id", stalled)
	}

	if err := db.ClearStalledUpdates(cutoff); err != nil {
		t.Fatalf("clear stalled: %v", err)
	}
	stalled, _ = db.ListStalledHosts(cutoff, true)
	if len(stalled) != 0 {
		t.Errorf("stalled after clear = %+v, want none", stalled)
	}
}

// Sub-second stamps must still compare correctly as SQL strings, which
// RFC3339Nano's trimmed fractional part would break.
func TestSlotStampsCompareCorrectlyAcrossFractions(t *testing.T) {
	db := openTemp(t)
	early, _ := db.CreateHost("early", "linux", "", "hash-early", 60)
	late, _ := db.CreateHost("late", "linux", "", "hash-late", 60)
	base := time.Date(2026, 7, 26, 10, 0, 0, 0, time.UTC)

	db.StartHostUpdate(early.ID, base.Add(500*time.Millisecond))
	db.StartHostUpdate(late.ID, base.Add(2*time.Second))

	stalled, err := db.ListStalledHosts(base.Add(time.Second), true)
	if err != nil {
		t.Fatalf("list stalled: %v", err)
	}
	if len(stalled) != 1 || stalled[0].Name != "early" {
		t.Errorf("stalled = %+v, want only the earlier stamp", stalled)
	}
}

func TestTryStartHostUpdateGrantsUnderCap(t *testing.T) {
	db := openTemp(t)
	h, _ := db.CreateHost("web-1", "linux", "", "hash-1", 60)
	now := time.Date(2026, 7, 26, 10, 0, 0, 0, time.UTC)
	cutoff := now.Add(-15 * time.Minute)

	ok, err := db.TryStartHostUpdate(h.ID, now, cutoff, 1, true)
	if err != nil {
		t.Fatalf("try start: %v", err)
	}
	if !ok {
		t.Fatal("expected the slot to be granted under the cap")
	}
	got, _ := db.GetHost(h.ID)
	if got.UpdateStartedAt == nil {
		t.Fatal("slot was not stamped on grant")
	}
}

func TestTryStartHostUpdateRefusesAtCap(t *testing.T) {
	db := openTemp(t)
	a, _ := db.CreateHost("a", "linux", "", "hash-a", 60)
	b, _ := db.CreateHost("b", "linux", "", "hash-b", 60)
	now := time.Date(2026, 7, 26, 10, 0, 0, 0, time.UTC)
	cutoff := now.Add(-15 * time.Minute)

	db.StartHostUpdate(a.ID, now)
	ok, err := db.TryStartHostUpdate(b.ID, now, cutoff, 1, true)
	if err != nil {
		t.Fatalf("try start: %v", err)
	}
	if ok {
		t.Error("granted a slot at the concurrency cap")
	}
}

func TestTryStartHostUpdateRefusesWhileAnyHostIsStalled(t *testing.T) {
	db := openTemp(t)
	stalled, _ := db.CreateHost("stalled", "linux", "", "hash-s", 60)
	next, _ := db.CreateHost("next", "linux", "", "hash-n", 60)
	started := time.Date(2026, 7, 26, 10, 0, 0, 0, time.UTC)
	db.StartHostUpdate(stalled.ID, started)
	later := started.Add(time.Hour)
	cutoff := later.Add(-15 * time.Minute)

	ok, err := db.TryStartHostUpdate(next.ID, later, cutoff, 10, true)
	if err != nil {
		t.Fatalf("try start: %v", err)
	}
	if ok {
		t.Error("granted a new slot while a host was stalled")
	}
}

// This is Finding 2's regression test: a read-then-write gate lets two
// concurrent pushes both see spare capacity and both stamp, over-granting
// the canary batch. TryStartHostUpdate must fold the check into one write.
func TestTryStartHostUpdateIsAtomicUnderConcurrency(t *testing.T) {
	db := openTemp(t)
	const hostCount = 20
	const concurrencyCap = 3
	ids := make([]string, hostCount)
	for i := range ids {
		h, err := db.CreateHost(fmt.Sprintf("host-%d", i), "linux", "", fmt.Sprintf("hash-%d", i), 60)
		if err != nil {
			t.Fatalf("create host %d: %v", i, err)
		}
		ids[i] = h.ID
	}
	now := time.Date(2026, 7, 26, 10, 0, 0, 0, time.UTC)
	cutoff := now.Add(-15 * time.Minute)

	var wg sync.WaitGroup
	var granted int32
	for _, id := range ids {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			ok, err := db.TryStartHostUpdate(id, now, cutoff, concurrencyCap, true)
			if err != nil {
				t.Errorf("try start: %v", err)
				return
			}
			if ok {
				atomic.AddInt32(&granted, 1)
			}
		}(id)
	}
	wg.Wait()

	if granted != concurrencyCap {
		t.Errorf("granted = %d, want exactly the cap of %d", granted, concurrencyCap)
	}
	live := 0
	for _, id := range ids {
		h, err := db.GetHost(id)
		if err != nil {
			t.Fatalf("get host: %v", err)
		}
		if h.UpdateStartedAt != nil {
			live++
		}
	}
	if live != concurrencyCap {
		t.Errorf("live slots after the race = %d, want %d", live, concurrencyCap)
	}
}

func TestGetIntSettingDefaultsAndParses(t *testing.T) {
	db := openTemp(t)
	if got := db.GetIntSetting("agent_update.concurrency", 3); got != 3 {
		t.Errorf("unset = %d, want the default 3", got)
	}
	db.SetSetting("agent_update.concurrency", "7")
	if got := db.GetIntSetting("agent_update.concurrency", 3); got != 7 {
		t.Errorf("set = %d, want 7", got)
	}
	db.SetSetting("agent_update.concurrency", "not-a-number")
	if got := db.GetIntSetting("agent_update.concurrency", 3); got != 3 {
		t.Errorf("unparseable = %d, want the default 3", got)
	}
}
