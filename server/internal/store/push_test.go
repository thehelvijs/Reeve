package store

import (
	"testing"
	"time"

	"github.com/thehelvijs/Reeve/contracts"
)

func TestApplyPushStoresEverySection(t *testing.T) {
	db := openTemp(t)
	host := seedHost(t, db)
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

	push := contracts.Push{
		AgentVersion:    "1.2.3",
		Services:        []contracts.ServiceState{{Unit: "web.service", ActiveState: "active", SubState: "running"}},
		Containers:      []contracts.ContainerState{{ID: "c1", Name: "web", Image: "nginx", State: "running", Health: "healthy"}},
		CronJobs:        []contracts.CronState{{Name: "backup.sh", Schedule: "0 * * * *"}},
		ContainerStats:  []contracts.ContainerSample{{ContainerID: "c1", CPUPct: 7, MemUsed: 100, MemLimit: 200}},
		LogEvents:       []contracts.LogEvent{{Source: "journald", Level: "error", Message: "boom", At: now}},
		Metrics:         contracts.HostMetrics{CPUPct: 42, MemUsed: 5, MemTotal: 10},
	}
	if err := db.ApplyPush(host, push, now); err != nil {
		t.Fatalf("ApplyPush: %v", err)
	}

	services, err := db.ListServiceStatus(host)
	if err != nil || len(services) != 1 || services[0].Unit != "web.service" {
		t.Fatalf("services = %+v, err = %v", services, err)
	}
	containers, err := db.ListContainerStatus(host)
	if err != nil || len(containers) != 1 || containers[0].Name != "web" {
		t.Fatalf("containers = %+v, err = %v", containers, err)
	}
	crons, err := db.ListCronJobs(host)
	if err != nil || len(crons) != 1 || crons[0].Name != "backup.sh" {
		t.Fatalf("cron jobs = %+v, err = %v", crons, err)
	}
	m, ok := db.LatestHostMetric(host)
	if !ok || m.CPUPct != 42 {
		t.Fatalf("latest metric = %+v, ok = %v", m, ok)
	}
	stats, err := db.QueryContainerStats(host, "raw", now.Add(-time.Hour))
	if err != nil || len(stats) != 1 || stats[0].CPUPct != 7 {
		t.Fatalf("container stats = %+v, err = %v", stats, err)
	}
	var logCount int
	if err := db.SQL().QueryRow(`SELECT COUNT(*) FROM log_events WHERE host_id = ?`, host).Scan(&logCount); err != nil {
		t.Fatalf("count log events: %v", err)
	}
	if logCount != 1 {
		t.Errorf("log events = %d, want 1", logCount)
	}

	h, err := db.GetHost(host)
	if err != nil {
		t.Fatalf("GetHost: %v", err)
	}
	if h.AgentVersion != "1.2.3" {
		t.Errorf("agent version = %q, want 1.2.3", h.AgentVersion)
	}
	if h.LastSeenAt == nil || !h.LastSeenAt.Equal(now) {
		t.Errorf("last seen = %v, want %v", h.LastSeenAt, now)
	}
}

// A push replaces the previous snapshot rather than accumulating rows, and a
// failing section rolls the whole push back.
func TestApplyPushReplacesSnapshotAndRollsBack(t *testing.T) {
	db := openTemp(t)
	host := seedHost(t, db)
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

	first := contracts.Push{
		AgentVersion: "1",
		Services: []contracts.ServiceState{
			{Unit: "a.service", ActiveState: "active"}, {Unit: "b.service", ActiveState: "active"},
		},
	}
	if err := db.ApplyPush(host, first, now); err != nil {
		t.Fatalf("first ApplyPush: %v", err)
	}
	second := contracts.Push{
		AgentVersion: "2",
		Services:     []contracts.ServiceState{{Unit: "a.service", ActiveState: "failed"}},
	}
	if err := db.ApplyPush(host, second, now.Add(time.Minute)); err != nil {
		t.Fatalf("second ApplyPush: %v", err)
	}
	services, err := db.ListServiceStatus(host)
	if err != nil || len(services) != 1 || services[0].ActiveState != "failed" {
		t.Fatalf("services after replace = %+v, err = %v", services, err)
	}

	// A container sample for a host that does not exist violates the foreign
	// key, so the earlier sections of that push must not survive.
	bad := contracts.Push{
		AgentVersion:   "3",
		Services:       []contracts.ServiceState{{Unit: "c.service", ActiveState: "active"}},
		ContainerStats: []contracts.ContainerSample{{ContainerID: "c1"}},
	}
	if err := db.ApplyPush("no-such-host", bad, now.Add(2*time.Minute)); err == nil {
		t.Fatal("expected ApplyPush to fail for an unknown host")
	}
	var orphans int
	if err := db.SQL().QueryRow(
		`SELECT COUNT(*) FROM service_status WHERE host_id = ?`, "no-such-host").Scan(&orphans); err != nil {
		t.Fatalf("count orphans: %v", err)
	}
	if orphans != 0 {
		t.Errorf("orphan service rows = %d, want 0 after rollback", orphans)
	}
}

func TestLatestHostMetricsBatchesEveryHost(t *testing.T) {
	db := openTemp(t)
	withSamples := seedHost(t, db)
	empty := seedHost(t, db)
	base := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

	if err := db.InsertHostMetric(withSamples, contracts.HostMetrics{CPUPct: 10}, base); err != nil {
		t.Fatalf("insert old: %v", err)
	}
	if err := db.InsertHostMetric(withSamples, contracts.HostMetrics{CPUPct: 90}, base.Add(time.Minute)); err != nil {
		t.Fatalf("insert new: %v", err)
	}

	latest, err := db.LatestHostMetrics()
	if err != nil {
		t.Fatalf("LatestHostMetrics: %v", err)
	}
	got, ok := latest[withSamples]
	if !ok {
		t.Fatal("host with samples missing from the batch")
	}
	if got.CPUPct != 90 {
		t.Errorf("cpu = %v, want the newest sample (90)", got.CPUPct)
	}
	if !got.TS.Equal(base.Add(time.Minute)) {
		t.Errorf("ts = %v, want %v", got.TS, base.Add(time.Minute))
	}
	if _, ok := latest[empty]; ok {
		t.Error("host without samples should be absent from the batch")
	}

	single, ok := db.LatestHostMetric(withSamples)
	if !ok || single.CPUPct != got.CPUPct || !single.TS.Equal(got.TS) {
		t.Errorf("single lookup = %+v, want the same point as the batch %+v", single, got)
	}
}
