package store

import (
	"testing"
	"time"

	"github.com/thehelvijs/Reeve/contracts"
)

func TestHostDisksRoundTripAndReplace(t *testing.T) {
	db := openTemp(t)
	h := seedHost(t, db)
	now := time.Now().UTC().Truncate(time.Second)

	first := []contracts.DiskUsage{
		{Mount: "/", Device: "/dev/sda3", FSType: "ext4", Used: 16 << 30, Total: 30 << 30},
		{Mount: "/mnt/media", Device: "/dev/sdb1", FSType: "ext4", Used: 1 << 40, Total: 2 << 40},
	}
	if err := db.ReplaceHostDisks(h, first, now); err != nil {
		t.Fatal(err)
	}
	got, ok := db.LatestHostDisks(h)
	if !ok {
		t.Fatal("no snapshot after storing one")
	}
	if len(got.Disks) != 2 || got.Disks[1].Mount != "/mnt/media" || got.Disks[1].Total != 2<<40 {
		t.Fatalf("snapshot did not round-trip: %+v", got.Disks)
	}
	if got.TS.IsZero() {
		t.Error("timestamp did not round-trip")
	}

	if err := db.ReplaceHostDisks(h, []contracts.DiskUsage{{Mount: "/", Total: 1}}, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	got, _ = db.LatestHostDisks(h)
	if len(got.Disks) != 1 {
		t.Errorf("second snapshot did not replace the first: %+v", got.Disks)
	}
	var rows int
	db.SQL().QueryRow(`SELECT COUNT(*) FROM host_disks WHERE host_id = ?`, h).Scan(&rows)
	if rows != 1 {
		t.Errorf("rows = %d, want 1: the snapshot replaces, it does not accumulate", rows)
	}
}

// A push carries the disks inside its metrics, so ApplyPush has to land them.
func TestApplyPushStoresDisks(t *testing.T) {
	db := openTemp(t)
	h := seedHost(t, db)
	push := contracts.Push{Metrics: contracts.HostMetrics{
		Disks: []contracts.DiskUsage{{Mount: "/data", Device: "/dev/sdc1", FSType: "xfs", Used: 5, Total: 10}},
	}}
	if err := db.ApplyPush(h, push, time.Now().UTC()); err != nil {
		t.Fatalf("ApplyPush: %v", err)
	}
	got, ok := db.LatestHostDisks(h)
	if !ok || len(got.Disks) != 1 || got.Disks[0].Mount != "/data" {
		t.Errorf("disks = %+v, ok = %v, want the pushed filesystem", got.Disks, ok)
	}
}

func TestHostDisksNilStoresEmptyList(t *testing.T) {
	db := openTemp(t)
	h := seedHost(t, db)
	if err := db.ReplaceHostDisks(h, nil, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	got, ok := db.LatestHostDisks(h)
	if !ok || got.Disks == nil || len(got.Disks) != 0 {
		t.Errorf("disks = %+v, want an empty list rather than null", got.Disks)
	}
}

func TestLatestHostDisksUnknownHost(t *testing.T) {
	db := openTemp(t)
	if _, ok := db.LatestHostDisks("nope"); ok {
		t.Error("reported a snapshot for a host that has none")
	}
}

// "Clear metrics" must not leave a stale filesystem list behind.
func TestDeleteHostMetricsDropsDisks(t *testing.T) {
	db := openTemp(t)
	h := seedHost(t, db)
	db.ReplaceHostDisks(h, []contracts.DiskUsage{{Mount: "/", Total: 1}}, time.Now().UTC())
	if err := db.DeleteHostMetrics(h); err != nil {
		t.Fatal(err)
	}
	if _, ok := db.LatestHostDisks(h); ok {
		t.Error("clearing metrics left the filesystem snapshot behind")
	}
}
