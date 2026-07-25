package store

import (
	"testing"
	"time"

	"github.com/thehelvijs/Reeve/contracts"
)

func TestContainerStatsRoundTrip(t *testing.T) {
	db := openTemp(t)
	hostID := seedHost(t, db)
	ts := time.Now().UTC()

	if err := db.ReplaceContainerStatus(hostID, []contracts.ContainerState{
		{ID: "c1", Name: "web", Image: "nginx", State: "running", Health: "healthy"},
	}); err != nil {
		t.Fatalf("ReplaceContainerStatus: %v", err)
	}
	if err := db.InsertContainerStats(hostID, []contracts.ContainerSample{
		{ContainerID: "c1", CPUPct: 5, MemUsed: 1000, MemLimit: 5000},
	}, ts); err != nil {
		t.Fatalf("InsertContainerStats: %v", err)
	}

	points, err := db.QueryContainerStats(hostID, "raw", ts.Add(-time.Hour))
	if err != nil {
		t.Fatalf("QueryContainerStats: %v", err)
	}
	if len(points) != 1 {
		t.Fatalf("points = %d, want 1", len(points))
	}
	p := points[0]
	if p.Name != "web" {
		t.Errorf("Name = %q, want %q", p.Name, "web")
	}
	if p.ContainerID != "c1" {
		t.Errorf("ContainerID = %q, want %q", p.ContainerID, "c1")
	}
	if p.CPUPct != 5 {
		t.Errorf("CPUPct = %v, want 5", p.CPUPct)
	}
	if p.MemUsed != 1000 {
		t.Errorf("MemUsed = %d, want 1000", p.MemUsed)
	}
	if p.MemLimit != 5000 {
		t.Errorf("MemLimit = %d, want 5000", p.MemLimit)
	}
}

func TestContainerStatsNameFallback(t *testing.T) {
	db := openTemp(t)
	hostID := seedHost(t, db)
	ts := time.Now().UTC()

	if err := db.InsertContainerStats(hostID, []contracts.ContainerSample{
		{ContainerID: "c9", CPUPct: 1, MemUsed: 100, MemLimit: 200},
	}, ts); err != nil {
		t.Fatalf("InsertContainerStats: %v", err)
	}

	points, err := db.QueryContainerStats(hostID, "raw", ts.Add(-time.Hour))
	if err != nil {
		t.Fatalf("QueryContainerStats: %v", err)
	}
	if len(points) != 1 {
		t.Fatalf("points = %d, want 1", len(points))
	}
	if points[0].Name != "" {
		t.Errorf("Name = %q, want empty (no matching container_status)", points[0].Name)
	}
}
