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

// The whole point of the accumulator: an average over a window, from pushes
// folded into buckets rather than one row kept per push.
func TestProcessUsageAveragesAndPeaks(t *testing.T) {
	db := openTemp(t)
	h := seedHost(t, db)
	base := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)

	pushes := []struct {
		at  time.Time
		cpu float64
		mem uint64
	}{
		{base, 10, 100},
		{base.Add(15 * time.Second), 30, 300},
		{base.Add(30 * time.Second), 20, 200},
	}
	for _, p := range pushes {
		err := db.ApplyPush(h, contracts.Push{
			Processes: []contracts.ProcessSample{{PID: 1, Command: "plex", CPUPct: p.cpu, MemRSS: p.mem}},
		}, p.at)
		if err != nil {
			t.Fatalf("apply push: %v", err)
		}
	}

	got, err := db.ProcessUsageSince(h, base.Add(-time.Hour), 10)
	if err != nil {
		t.Fatalf("ProcessUsageSince: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("usage = %+v, want one command", got)
	}
	u := got[0]
	if u.Samples != 3 {
		t.Errorf("samples = %d, want 3", u.Samples)
	}
	if u.CPUAvg != 20 {
		t.Errorf("cpu_avg = %v, want the mean 20", u.CPUAvg)
	}
	if u.CPUMax != 30 {
		t.Errorf("cpu_max = %v, want the worst push 30", u.CPUMax)
	}
	if u.MemAvg != 200 {
		t.Errorf("mem_avg = %v, want the mean 200", u.MemAvg)
	}
	if u.MemMax != 300 {
		t.Errorf("mem_max = %d, want the worst push 300", u.MemMax)
	}

	var rows int
	db.SQL().QueryRow(`SELECT COUNT(*) FROM process_usage WHERE host_id = ?`, h).Scan(&rows)
	if rows != 1 {
		t.Errorf("rows = %d, want 1: three pushes inside one bucket accumulate into it", rows)
	}
}

// A window must not average in buckets older than itself.
func TestProcessUsageWindowExcludesOlderBuckets(t *testing.T) {
	db := openTemp(t)
	h := seedHost(t, db)
	now := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)

	db.ApplyPush(h, contracts.Push{
		Processes: []contracts.ProcessSample{{PID: 1, Command: "old", CPUPct: 99}},
	}, now.Add(-2*time.Hour))
	db.ApplyPush(h, contracts.Push{
		Processes: []contracts.ProcessSample{{PID: 2, Command: "recent", CPUPct: 5}},
	}, now)

	got, _ := db.ProcessUsageSince(h, now.Add(-time.Hour), 10)
	if len(got) != 1 || got[0].Command != "recent" {
		t.Errorf("usage = %+v, want only the command inside the window", got)
	}
}

// A memory hog that never burns CPU is exactly what someone hunting waste is
// after, so it must survive a list that is ordered by CPU.
func TestProcessUsageKeepsTopByMemory(t *testing.T) {
	db := openTemp(t)
	h := seedHost(t, db)
	now := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)

	procs := []contracts.ProcessSample{
		{PID: 1, Command: "busy", CPUPct: 50, MemRSS: 1},
		{PID: 2, Command: "fat", CPUPct: 0, MemRSS: 1 << 30},
	}
	db.ApplyPush(h, contracts.Push{Processes: procs}, now)

	got, _ := db.ProcessUsageSince(h, now.Add(-time.Hour), 1)
	names := map[string]bool{}
	for _, u := range got {
		names[u.Command] = true
	}
	if !names["busy"] || !names["fat"] {
		t.Errorf("usage = %+v, want the top command of each dimension", got)
	}
}

func TestPruneDropsOldProcessUsage(t *testing.T) {
	db := openTemp(t)
	h := seedHost(t, db)
	now := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)

	db.ApplyPush(h, contracts.Push{
		Processes: []contracts.ProcessSample{{PID: 1, Command: "ancient", CPUPct: 1}},
	}, now.Add(-30*24*time.Hour))
	db.ApplyPush(h, contracts.Push{
		Processes: []contracts.ProcessSample{{PID: 2, Command: "fresh", CPUPct: 1}},
	}, now)

	if err := db.RollupAndPrune(now, DefaultRetention); err != nil {
		t.Fatalf("RollupAndPrune: %v", err)
	}
	got, _ := db.ProcessUsageSince(h, now.Add(-365*24*time.Hour), 10)
	if len(got) != 1 || got[0].Command != "fresh" {
		t.Errorf("usage = %+v, want only what is inside the retention window", got)
	}
}

func TestDeleteHostMetricsDropsProcessUsage(t *testing.T) {
	db := openTemp(t)
	h := seedHost(t, db)
	db.ApplyPush(h, contracts.Push{
		Processes: []contracts.ProcessSample{{PID: 1, Command: "x", CPUPct: 1}},
	}, time.Now().UTC())
	if err := db.DeleteHostMetrics(h); err != nil {
		t.Fatal(err)
	}
	got, _ := db.ProcessUsageSince(h, time.Now().UTC().Add(-time.Hour), 10)
	if len(got) != 0 {
		t.Errorf("clearing metrics left usage behind: %+v", got)
	}
}
