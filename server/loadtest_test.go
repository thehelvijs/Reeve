package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/thehelvijs/Reeve/contracts"
	"github.com/thehelvijs/Reeve/server/internal/crypto"
	"github.com/thehelvijs/Reeve/server/internal/store"
)

// TestLoadIngest validates the SQLite write path at the target scale (~100
// hosts pushing every 15s). It is skipped in the gate lane; run explicitly:
//
//	LOADTEST=1 go test ./server/ -run TestLoadIngest -v -timeout 300s
func TestLoadIngest(t *testing.T) {
	if os.Getenv("LOADTEST") == "" {
		t.Skip("set LOADTEST=1 to run the ingest load test")
	}

	cipher, _ := crypto.New(bytes.Repeat([]byte{1}, 32))
	db, err := store.Open(filepath.Join(t.TempDir(), "load.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()
	a := &app{db: db, cipher: cipher, cfg: config{SessionTTL: time.Hour}}

	const hosts = 100
	const rounds = 40 // 40 * 15s = 10 minutes of pushes

	hostIDs := make([]string, hosts)
	for i := range hostIDs {
		h, err := db.CreateHost(fmt.Sprintf("host-%d", i), "linux", "", fmt.Sprintf("hash-%d", i), 60)
		if err != nil {
			t.Fatalf("create host: %v", err)
		}
		hostIDs[i] = h.ID
	}

	push := realisticPush()
	start := time.Now()
	total := 0
	for r := 0; r < rounds; r++ {
		for _, id := range hostIDs {
			if err := a.storePush(id, push); err != nil {
				t.Fatalf("storePush: %v", err)
			}
			total++
		}
	}
	elapsed := time.Since(start)
	rate := float64(total) / elapsed.Seconds()
	required := float64(hosts) / 15.0 // sustained pushes/sec the deployment needs

	t.Logf("ingested %d pushes in %s = %.0f pushes/sec (need >= %.1f for %d hosts @15s)",
		total, elapsed.Round(time.Millisecond), rate, required, hosts)
	if rate < required {
		t.Errorf("throughput %.0f/s below required %.1f/s", rate, required)
	}
}

func realisticPush() contracts.Push {
	p := contracts.Push{AgentVersion: "load", SentAt: time.Now().UTC()}
	for i := 0; i < 50; i++ {
		p.Services = append(p.Services, contracts.ServiceState{
			Unit: fmt.Sprintf("svc-%d.service", i), ActiveState: "active", SubState: "running"})
	}
	for i := 0; i < 5; i++ {
		id := fmt.Sprintf("c%d", i)
		p.Containers = append(p.Containers, contracts.ContainerState{ID: id, Name: id, Image: "img", State: "running", Health: "healthy"})
		p.ContainerStats = append(p.ContainerStats, contracts.ContainerSample{ContainerID: id, CPUPct: 1.2, MemUsed: 1000, MemLimit: 5000})
	}
	for i := 0; i < 3; i++ {
		p.CronJobs = append(p.CronJobs, contracts.CronState{Name: fmt.Sprintf("job-%d", i), Schedule: "0 * * * *"})
	}
	p.Metrics = contracts.HostMetrics{CPUPct: 12, MemUsed: 1e9, MemTotal: 2e9, DiskUsed: 5e10, DiskTotal: 1e11, UptimeSecs: 3600}
	return p
}
