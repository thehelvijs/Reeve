package store

import (
	"testing"
	"time"

	"github.com/thehelvijs/Reeve/contracts"
)

func TestHostProcessesRoundTripAndOverwrite(t *testing.T) {
	db := openTemp(t)
	h, err := db.CreateHost("h", "linux", "", "hh", 60)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Second)

	first := []contracts.ProcessSample{
		{PID: 1, User: "root", Command: "dockerd", CPUPct: 34.2, MemRSS: 1 << 30},
		{PID: 2, User: "plex", Command: "Plex Transcoder", CPUPct: 28.7, MemRSS: 1 << 20},
	}
	if err := db.ReplaceHostProcesses(h.ID, first, now); err != nil {
		t.Fatal(err)
	}
	got, ok := db.LatestHostProcesses(h.ID)
	if !ok {
		t.Fatal("no snapshot after storing one")
	}
	if len(got.Procs) != 2 || got.Procs[0].Command != "dockerd" || got.Procs[0].MemRSS != 1<<30 {
		t.Fatalf("snapshot did not round-trip: %+v", got.Procs)
	}
	if got.TS.IsZero() {
		t.Error("timestamp did not round-trip")
	}

	second := []contracts.ProcessSample{{PID: 9, User: "pg", Command: "postgres", CPUPct: 1}}
	if err := db.ReplaceHostProcesses(h.ID, second, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	got, _ = db.LatestHostProcesses(h.ID)
	if len(got.Procs) != 1 || got.Procs[0].Command != "postgres" {
		t.Errorf("second snapshot did not replace the first: %+v", got.Procs)
	}

	var rows int
	db.SQL().QueryRow(`SELECT COUNT(*) FROM host_processes WHERE host_id = ?`, h.ID).Scan(&rows)
	if rows != 1 {
		t.Errorf("rows = %d, want 1: the snapshot replaces, it does not accumulate", rows)
	}
}

func TestHostProcessesNilStoresEmptyList(t *testing.T) {
	db := openTemp(t)
	h, _ := db.CreateHost("h", "linux", "", "hh", 60)
	if err := db.ReplaceHostProcesses(h.ID, nil, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	got, ok := db.LatestHostProcesses(h.ID)
	if !ok {
		t.Fatal("a nil slice must still store a row")
	}
	if got.Procs == nil || len(got.Procs) != 0 {
		t.Errorf("procs = %+v, want an empty list rather than null", got.Procs)
	}
}

func TestLatestHostProcessesUnknownHost(t *testing.T) {
	db := openTemp(t)
	if _, ok := db.LatestHostProcesses("nope"); ok {
		t.Error("reported a snapshot for a host that has none")
	}
}

// The admin "Clear metrics" action must not leave a stale process table behind.
func TestDeleteHostMetricsDropsProcesses(t *testing.T) {
	db := openTemp(t)
	h, _ := db.CreateHost("h", "linux", "", "hh", 60)
	db.ReplaceHostProcesses(h.ID, []contracts.ProcessSample{{PID: 1, Command: "x"}}, time.Now().UTC())
	if err := db.DeleteHostMetrics(h.ID); err != nil {
		t.Fatal(err)
	}
	if _, ok := db.LatestHostProcesses(h.ID); ok {
		t.Error("clearing metrics left the process snapshot behind")
	}
}
