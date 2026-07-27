package store

import (
	"testing"
	"time"

	"github.com/thehelvijs/Reeve/contracts"
)

func seedHost(t *testing.T, db *DB) string {
	t.Helper()
	h, err := db.CreateHost("h", "linux", "", "hash-"+NewID(), 60)
	if err != nil {
		t.Fatalf("create host: %v", err)
	}
	return h.ID
}

func TestRollupRawTo5m(t *testing.T) {
	db := openTemp(t)
	host := seedHost(t, db)
	base := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)

	// Six raw samples inside the 10:00–10:05 bucket, cpu 0,10,20,30,40,50 -> avg 25.
	for i := 0; i < 6; i++ {
		m := contracts.HostMetrics{CPUPct: float64(i * 10), MemUsed: 100, MemTotal: 200}
		if err := db.InsertHostMetric(host, m, base.Add(time.Duration(i*50)*time.Second)); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}

	// now well past the bucket end.
	if err := db.RollupAndPrune(base.Add(10*time.Minute), DefaultRetention); err != nil {
		t.Fatalf("rollup: %v", err)
	}
	pts, err := db.QueryHostMetrics(host, "5m", base.Add(-time.Hour))
	if err != nil {
		t.Fatalf("query 5m: %v", err)
	}
	if len(pts) != 1 {
		t.Fatalf("got %d 5m points, want 1: %+v", len(pts), pts)
	}
	if pts[0].CPUPct < 24.9 || pts[0].CPUPct > 25.1 {
		t.Errorf("5m avg cpu = %f, want 25", pts[0].CPUPct)
	}
}

func TestRollupPreservesLoadAverage(t *testing.T) {
	db := openTemp(t)
	host := seedHost(t, db)
	base := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)

	for i := 0; i < 6; i++ {
		m := contracts.HostMetrics{CPUPct: 50, Load1: 2, Load5: 3, Load15: 4}
		if err := db.InsertHostMetric(host, m, base.Add(time.Duration(i*50)*time.Second)); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}
	if err := db.RollupAndPrune(base.Add(10*time.Minute), DefaultRetention); err != nil {
		t.Fatalf("rollup: %v", err)
	}
	pts, err := db.QueryHostMetrics(host, "5m", base.Add(-time.Hour))
	if err != nil || len(pts) != 1 {
		t.Fatalf("query 5m = %d pts, %v", len(pts), err)
	}
	if pts[0].Load1 != 2 || pts[0].Load5 != 3 || pts[0].Load15 != 4 {
		t.Errorf("rolled-up load = %v/%v/%v, want 2/3/4", pts[0].Load1, pts[0].Load5, pts[0].Load15)
	}
}

func TestRollupPreservesGPU(t *testing.T) {
	db := openTemp(t)
	host := seedHost(t, db)
	base := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)

	for i := 0; i < 6; i++ {
		m := contracts.HostMetrics{CPUPct: 50, GPUUtil: 20, GPUMemUsed: 1000, GPUMemTotal: 4000}
		if err := db.InsertHostMetric(host, m, base.Add(time.Duration(i*50)*time.Second)); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}
	if err := db.RollupAndPrune(base.Add(10*time.Minute), DefaultRetention); err != nil {
		t.Fatalf("rollup: %v", err)
	}
	pts, err := db.QueryHostMetrics(host, "5m", base.Add(-time.Hour))
	if err != nil || len(pts) != 1 {
		t.Fatalf("query 5m = %d pts, %v", len(pts), err)
	}
	if pts[0].GPUUtil != 20 || pts[0].GPUMemUsed != 1000 || pts[0].GPUMemTotal != 4000 {
		t.Errorf("rolled-up gpu = %v/%v/%v, want 20/1000/4000", pts[0].GPUUtil, pts[0].GPUMemUsed, pts[0].GPUMemTotal)
	}
}

func TestRollupSkipsIncompleteBucket(t *testing.T) {
	db := openTemp(t)
	host := seedHost(t, db)
	base := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	db.InsertHostMetric(host, contracts.HostMetrics{CPUPct: 5}, base.Add(30*time.Second))

	// now is still inside the 10:00–10:05 bucket -> not rolled up yet.
	db.RollupAndPrune(base.Add(2*time.Minute), DefaultRetention)
	pts, _ := db.QueryHostMetrics(host, "5m", base.Add(-time.Hour))
	if len(pts) != 0 {
		t.Errorf("incomplete bucket was rolled up: %+v", pts)
	}
}

func TestRollupIdempotent(t *testing.T) {
	db := openTemp(t)
	host := seedHost(t, db)
	base := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	for i := 0; i < 3; i++ {
		db.InsertHostMetric(host, contracts.HostMetrics{CPUPct: 10}, base.Add(time.Duration(i*30)*time.Second))
	}
	db.RollupAndPrune(base.Add(10*time.Minute), DefaultRetention)
	db.RollupAndPrune(base.Add(11*time.Minute), DefaultRetention)
	pts, _ := db.QueryHostMetrics(host, "5m", base.Add(-time.Hour))
	if len(pts) != 1 {
		t.Errorf("re-running rollup duplicated buckets: %d", len(pts))
	}
}

func TestPruneRemovesOldRaw(t *testing.T) {
	db := openTemp(t)
	host := seedHost(t, db)
	now := time.Date(2026, 1, 3, 10, 0, 0, 0, time.UTC)
	// One raw sample 50h old (beyond the 48h raw window) and one recent.
	db.InsertHostMetric(host, contracts.HostMetrics{CPUPct: 1}, now.Add(-50*time.Hour))
	db.InsertHostMetric(host, contracts.HostMetrics{CPUPct: 2}, now.Add(-1*time.Minute))

	db.RollupAndPrune(now, DefaultRetention)
	pts, _ := db.QueryHostMetrics(host, "raw", now.Add(-100*time.Hour))
	if len(pts) != 1 {
		t.Errorf("expected 1 raw point after prune, got %d", len(pts))
	}
}

// A byte counter that does not divide evenly is what a real machine reports, and
// AVG over it is a float. The integer columns hold it as one — SQLite converts a
// REAL to INTEGER only when that is lossless — so the read has to survive it.
// Every metrics range 500'd on a host whose rollup had ever run over odd bytes.
func TestRollupOfUnevenBytesStaysReadable(t *testing.T) {
	db := openTemp(t)
	host := seedHost(t, db)
	base := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)

	// Offsets chosen so every integer column's mean lands on a third, never a
	// whole number: 0+1+3 over 3 samples.
	for _, off := range []uint64{0, 1, 3} {
		m := contracts.HostMetrics{
			CPUPct: 10, MemUsed: 1000 + off, MemTotal: 4000 + off,
			DiskUsed: 700 + off, DiskTotal: 2000 + off, DiskRead: 11 + off, DiskWrite: 7 + off,
			NetRx: 101 + off, NetTx: 53 + off, UptimeSecs: 60 + off,
			GPUMemUsed: 5 + off, GPUMemTotal: 100 + off,
		}
		if err := db.InsertHostMetric(host, m, base.Add(time.Duration(off*40)*time.Second)); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}
	if err := db.RollupAndPrune(base.Add(2*time.Hour), DefaultRetention); err != nil {
		t.Fatalf("rollup: %v", err)
	}

	for _, res := range []string{"raw", "5m", "1h"} {
		pts, err := db.QueryHostMetrics(host, res, base.Add(-time.Hour))
		if err != nil {
			t.Fatalf("query %s: %v", res, err)
		}
		if len(pts) == 0 {
			t.Fatalf("query %s returned no points", res)
		}
		if pts[0].MemUsed == 0 || pts[0].NetRx == 0 {
			t.Errorf("%s point lost its byte counters: %+v", res, pts[0])
		}
	}
}
