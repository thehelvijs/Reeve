package main

import (
	"net"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
	"time"

	"github.com/thehelvijs/Reeve/contracts"
)

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
	if !push.AutoUpdateVetoed {
		t.Error("REEVE_AUTO_UPDATE=false was not reported to the server")
	}

	// The race check above already ran the collectors for real; this second call
	// only reads back a config flag, so it must not shell out again.
	speedUpGather(t)
	push = gather("test", config{AutoUpdate: true})
	if push.AutoUpdateVetoed {
		t.Error("a host with auto-update on reported a veto")
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

// A loopback server means the route is loopback, and 127.0.0.1 is no use to
// anyone else's browser, so the collector reports nothing rather than that.
func TestLocalIPForIgnoresALoopbackRoute(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	defer srv.Close()

	if got := localIPFor(srv.URL); got != "" {
		t.Errorf("localIPFor(loopback server) = %q, want an empty string", got)
	}
}

// Against an off-machine server the reported address is this host's own
// routable one, which is what a browser on the same network can reach.
func TestLocalIPForReportsTheOutboundAddress(t *testing.T) {
	// TEST-NET-1, per RFC 5737. A UDP "connect" sends nothing, so this only
	// asks the kernel which source address it would route from.
	got := localIPFor("http://192.0.2.1:8080")
	if got == "" {
		t.Skip("no route off this machine")
	}
	ip := net.ParseIP(got)
	if ip == nil {
		t.Fatalf("localIPFor = %q, which is not an IP", got)
	}
	if ip.IsLoopback() || ip.IsUnspecified() {
		t.Errorf("localIPFor = %q, want a routable address", got)
	}
}

func TestLocalIPForRejectsUnusableServerURLs(t *testing.T) {
	for _, url := range []string{"", "not a url", "://missing-scheme", "http://"} {
		if got := localIPFor(url); got != "" {
			t.Errorf("localIPFor(%q) = %q, want an empty string", url, got)
		}
	}
}
