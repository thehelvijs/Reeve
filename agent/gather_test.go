package main

import (
	"reflect"
	"runtime"
	"testing"
	"time"

	"github.com/thehelvijs/Reeve/contracts"
)

func TestSplitLines(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"", []string{}},
		{"one", []string{"one"}},
		{"one\n", []string{"one"}},
		{"one\ntwo", []string{"one", "two"}},
		{"one\ntwo\n", []string{"one", "two"}},
		{"\n", []string{""}},
		{"a\n\nb", []string{"a", "", "b"}},
	}
	for _, tc := range cases {
		if got := splitLines(tc.in); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("splitLines(%q) = %#v, want %#v", tc.in, got, tc.want)
		}
	}
}

// Stopped containers have no recent logs to scan, so no docker call is made for
// them; a list with none running must not shell out at all.
func TestGatherDockerLogErrorsSkipsStoppedContainers(t *testing.T) {
	prev := commandTimeout
	commandTimeout = 50 * time.Millisecond
	defer func() { commandTimeout = prev }()

	events := gatherDockerLogErrors([]contracts.ContainerState{
		{ID: "c1", State: "exited"},
		{ID: "", State: "running"},
	})
	if events != nil {
		t.Errorf("events = %+v, want none", events)
	}
}

// The collectors write into one push from several goroutines; run the real
// gather under -race so a future collector cannot introduce a data race there.
func TestGatherCollectsConcurrently(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("gather reads /proc")
	}
	push := gather("test", config{})
	if push.ProtocolVersion != contracts.PushProtocolVersion {
		t.Errorf("protocol version = %d, want %d", push.ProtocolVersion, contracts.PushProtocolVersion)
	}
	if push.AgentVersion != "test" {
		t.Errorf("agent version = %q, want test", push.AgentVersion)
	}
	if push.Metrics.MemTotal == 0 {
		t.Error("host metrics missing: MemTotal is zero")
	}
}

func TestRunCmdHonorsTimeout(t *testing.T) {
	prev := commandTimeout
	commandTimeout = 100 * time.Millisecond
	defer func() { commandTimeout = prev }()

	start := time.Now()
	if _, err := runCmd("sleep", "10"); err == nil {
		t.Fatal("expected a hung command to fail")
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("runCmd took %s, want it killed near the timeout", elapsed)
	}
}
