package contracts

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestPushRoundTrip(t *testing.T) {
	exit := 0
	last := time.Unix(1_700_000_000, 0).UTC()
	in := Push{
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

func TestPushCarriesAutoUpdateVeto(t *testing.T) {
	body, err := json.Marshal(Push{AutoUpdateVetoed: true})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(body), `"auto_update_vetoed":true`) {
		t.Errorf("push json = %s, want an auto_update_vetoed field", body)
	}
}

func TestPushAckRoundTrip(t *testing.T) {
	body, err := json.Marshal(PushAck{CheckNow: true})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(body) != `{"check_now":true}` {
		t.Errorf("ack json = %s, want {\"check_now\":true}", body)
	}
	var back PushAck
	if err := json.Unmarshal(body, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !back.CheckNow {
		t.Error("check_now did not survive the round trip")
	}
}

func TestPushTooLargeNamesTheOffendingSection(t *testing.T) {
	if got := (Push{}).TooLarge(); got != "" {
		t.Errorf("empty push TooLarge = %q, want none", got)
	}
	atCap := Push{Services: make([]ServiceState, MaxPushServices)}
	if got := atCap.TooLarge(); got != "" {
		t.Errorf("push at the cap TooLarge = %q, want none", got)
	}
	over := Push{
		Services:  make([]ServiceState, MaxPushServices+1),
		LogEvents: make([]LogEvent, MaxPushLogEvents+1),
	}
	if got := over.TooLarge(); got != "services" {
		t.Errorf("TooLarge = %q, want the first oversized section \"services\"", got)
	}
	if got := (Push{LogEvents: make([]LogEvent, MaxPushLogEvents+1)}).TooLarge(); got != "log_events" {
		t.Errorf("TooLarge = %q, want log_events", got)
	}
}
