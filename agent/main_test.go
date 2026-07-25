package main

import (
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
