package contracts

import (
	"encoding/json"
	"testing"
	"time"
)

func TestPushRoundTrip(t *testing.T) {
	exit := 0
	last := time.Unix(1_700_000_000, 0).UTC()
	in := Push{
		ProtocolVersion: PushProtocolVersion,
		AgentVersion:    "0.1.0",
		SentAt:          last,
		Services:        []ServiceState{{Unit: "nginx.service", ActiveState: "active", SubState: "running"}},
		Containers:      []ContainerState{{ID: "abc", Name: "web", Image: "nginx", State: "running", Health: "healthy"}},
		CronJobs:        []CronState{{Name: "backup", Schedule: "0 3 * * *", LastRunAt: &last, LastExit: &exit}},
		Metrics:         HostMetrics{CPUPct: 12.5, MemUsed: 100, MemTotal: 200, UptimeSecs: 3600, Temps: map[string]float64{"cpu": 45}},
		ContainerStats:  []ContainerSample{{ContainerID: "abc", CPUPct: 3.2, MemUsed: 50, MemLimit: 100}},
		LogEvents:       []LogEvent{{Source: "web", Level: "error", Message: "boom", At: last}},
	}

	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out Push
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.ProtocolVersion != PushProtocolVersion {
		t.Errorf("protocol version = %d, want %d", out.ProtocolVersion, PushProtocolVersion)
	}
	if len(out.Services) != 1 || out.Services[0].Unit != "nginx.service" {
		t.Errorf("services did not round-trip: %+v", out.Services)
	}
	if out.Metrics.Temps["cpu"] != 45 {
		t.Errorf("temps did not round-trip: %+v", out.Metrics.Temps)
	}
	if out.CronJobs[0].LastExit == nil || *out.CronJobs[0].LastExit != 0 {
		t.Errorf("cron last exit did not round-trip: %+v", out.CronJobs[0])
	}
	if len(out.LogEvents) != 1 || out.LogEvents[0].Message != "boom" {
		t.Errorf("log events did not round-trip: %+v", out.LogEvents)
	}
}
