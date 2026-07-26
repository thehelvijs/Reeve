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

func TestUpdateSlotLifecycle(t *testing.T) {
	db := openTemp(t)
	h, _ := db.CreateHost("web-1", "linux", "", "hash-1", 60)
	now := time.Date(2026, 7, 26, 10, 0, 0, 0, time.UTC)
	cutoff := now.Add(-15 * time.Minute)

	if err := db.StartHostUpdate(h.ID, now); err != nil {
		t.Fatalf("start update: %v", err)
	}
	live, err := db.CountLiveUpdateSlots(cutoff)
	if err != nil {
		t.Fatalf("count live: %v", err)
	}
	if live != 1 {
		t.Errorf("live slots = %d, want 1", live)
	}
	stalled, err := db.CountStalledUpdates(cutoff)
	if err != nil {
		t.Fatalf("count stalled: %v", err)
	}
	if stalled != 0 {
		t.Errorf("stalled = %d, want 0", stalled)
	}

	if err := db.ClearHostUpdateSlot(h.ID); err != nil {
		t.Fatalf("clear slot: %v", err)
	}
	live, _ = db.CountLiveUpdateSlots(cutoff)
	if live != 0 {
		t.Errorf("live slots after clear = %d, want 0", live)
	}
}

func TestStalledSlotsAreCountedNamedAndCleared(t *testing.T) {
	db := openTemp(t)
	h, _ := db.CreateHost("web-1", "linux", "", "hash-1", 60)
	started := time.Date(2026, 7, 26, 10, 0, 0, 0, time.UTC)
	cutoff := started.Add(15 * time.Minute)

	db.StartHostUpdate(h.ID, started)

	stalled, err := db.CountStalledUpdates(cutoff)
	if err != nil {
		t.Fatalf("count stalled: %v", err)
	}
	if stalled != 1 {
		t.Errorf("stalled = %d, want 1", stalled)
	}
	names, err := db.ListStalledHostNames(cutoff)
	if err != nil {
		t.Fatalf("list stalled: %v", err)
	}
	if len(names) != 1 || names[0] != "web-1" {
		t.Errorf("stalled names = %v, want [web-1]", names)
	}

	if err := db.ClearStalledUpdates(cutoff); err != nil {
		t.Fatalf("clear stalled: %v", err)
	}
	stalled, _ = db.CountStalledUpdates(cutoff)
	if stalled != 0 {
		t.Errorf("stalled after clear = %d, want 0", stalled)
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

	live, err := db.CountLiveUpdateSlots(base.Add(time.Second))
	if err != nil {
		t.Fatalf("count live: %v", err)
	}
	if live != 1 {
		t.Errorf("live slots = %d, want 1 (only the later stamp)", live)
	}
}

func TestTryStartHostUpdateGrantsUnderCap(t *testing.T) {
	db := openTemp(t)
	h, _ := db.CreateHost("web-1", "linux", "", "hash-1", 60)
	now := time.Date(2026, 7, 26, 10, 0, 0, 0, time.UTC)
	cutoff := now.Add(-15 * time.Minute)

	ok, err := db.TryStartHostUpdate(h.ID, now, cutoff, 1)
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
	ok, err := db.TryStartHostUpdate(b.ID, now, cutoff, 1)
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

	ok, err := db.TryStartHostUpdate(next.ID, later, cutoff, 10)
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
			ok, err := db.TryStartHostUpdate(id, now, cutoff, concurrencyCap)
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
	live, err := db.CountLiveUpdateSlots(cutoff)
	if err != nil {
		t.Fatalf("count live: %v", err)
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
