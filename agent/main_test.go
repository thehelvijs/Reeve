package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestLoadConfigDefaultInterval(t *testing.T) {
	t.Setenv("REEVE_SERVER_URL", "http://server:8080")
	t.Setenv("REEVE_AGENT_TOKEN", "tok")
	t.Setenv("REEVE_PUSH_INTERVAL", "")

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	if cfg.Interval != 15*time.Second {
		t.Errorf("default interval = %s, want 15s", cfg.Interval)
	}
	if cfg.ServerURL != "http://server:8080" || cfg.Token != "tok" {
		t.Errorf("config not read from env: %+v", cfg)
	}
}

func TestLoadConfigBadInterval(t *testing.T) {
	t.Setenv("REEVE_PUSH_INTERVAL", "notaduration")
	if _, err := loadConfig(); err == nil {
		t.Fatal("expected error on bad interval")
	}
}

func TestVersionFlagPrintsAndExits(t *testing.T) {
	if os.Getenv("RUN_VERSION_FLAG") == "1" {
		os.Args = []string{"reeve-agent", "--version"}
		main()
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestVersionFlagPrintsAndExits")
	cmd.Env = append(os.Environ(), "RUN_VERSION_FLAG=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("subprocess failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "dev") {
		t.Errorf("expected version output to contain the default 'dev', got: %s", out)
	}
}

// speedUpGather makes gather's exec calls fail fast instead of running for real.
func speedUpGather(t *testing.T) {
	t.Helper()
	prev := commandTimeout
	commandTimeout = 10 * time.Millisecond
	t.Cleanup(func() { commandTimeout = prev })
}

func TestRunOnceDoesNotStampLastAckOnMalformedAck(t *testing.T) {
	speedUpGather(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `not json`)
	}))
	defer srv.Close()
	p := newPusher(config{ServerURL: srv.URL, Token: "t"})
	p.bufferDir = t.TempDir()

	var lastAck time.Time
	if runOnce(p, config{ServerURL: srv.URL, Token: "t", AutoUpdate: true}, "1.0.0", &lastAck) {
		t.Error("runOnce requested an update from a malformed ack")
	}
	if !lastAck.IsZero() {
		t.Error("a malformed ack was treated as contact")
	}
}

func TestRunOnceStampsLastAckOnRealAck(t *testing.T) {
	speedUpGather(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"check_now":true}`)
	}))
	defer srv.Close()
	p := newPusher(config{ServerURL: srv.URL, Token: "t"})
	p.bufferDir = t.TempDir()

	var lastAck time.Time
	before := time.Now()
	if !runOnce(p, config{ServerURL: srv.URL, Token: "t", AutoUpdate: true}, "1.0.0", &lastAck) {
		t.Error("runOnce did not request the update the server asked for")
	}
	if lastAck.Before(before) {
		t.Error("lastAck was not stamped with the current time")
	}
}

func TestRunOnceVetoBlocksUpdateNotContact(t *testing.T) {
	speedUpGather(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"check_now":true}`)
	}))
	defer srv.Close()
	p := newPusher(config{ServerURL: srv.URL, Token: "t"})
	p.bufferDir = t.TempDir()

	var lastAck time.Time
	if runOnce(p, config{ServerURL: srv.URL, Token: "t", AutoUpdate: false}, "1.0.0", &lastAck) {
		t.Error("runOnce requested an update on a host that vetoed auto-update")
	}
	if lastAck.IsZero() {
		t.Error("the veto blocked contact tracking, not just the update")
	}
}
