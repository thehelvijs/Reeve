package main

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/thehelvijs/Reeve/contracts"
	"github.com/thehelvijs/Reeve/server/internal/store"
)

// controllableHost enrolls a host and pushes once as an agent that reports it
// will run commands, which is what makes the host commandable.
func controllableHost(t *testing.T, ts *testServer, admin *http.Client, name string) (id, token string) {
	t.Helper()
	id, token = enrollHost(t, ts, admin, name)
	p := samplePush()
	p.ControlEnabled = true
	ts.do(t, nil, http.MethodPost, "/api/ingest", p,
		map[string]string{"Authorization": "Bearer " + token})
	return id, token
}

func queue(t *testing.T, ts *testServer, c *http.Client, hostID, action, target string) (int, commandView) {
	t.Helper()
	resp, data := ts.do(t, c, http.MethodPost, "/api/admin/hosts/"+hostID+"/commands",
		commandInput{Action: action, Target: target}, nil)
	var out commandView
	json.Unmarshal(data, &out)
	return resp.StatusCode, out
}

func TestQueueAndDeliverACommand(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	host, token := controllableHost(t, ts, admin, "box")

	// samplePush reports nginx.service, so it is a target the host has named.
	code, cmd := queue(t, ts, admin, host, contracts.ActionServiceRestart, "nginx.service")
	if code != http.StatusCreated {
		t.Fatalf("queue = %d", code)
	}
	if cmd.Status != store.CommandPending {
		t.Errorf("status = %q, want pending", cmd.Status)
	}
	if cmd.RequestedByName != "boss@example.com" {
		t.Errorf("requested_by_name = %q, want the admin's email", cmd.RequestedByName)
	}

	// The next push collects it.
	p := samplePush()
	p.ControlEnabled = true
	_, data := ts.do(t, nil, http.MethodPost, "/api/ingest", p,
		map[string]string{"Authorization": "Bearer " + token})
	var ack contracts.PushAck
	json.Unmarshal(data, &ack)
	if len(ack.Commands) != 1 || ack.Commands[0].ID != cmd.ID {
		t.Fatalf("ack commands = %+v, want the queued one", ack.Commands)
	}
	if ack.Commands[0].Action != contracts.ActionServiceRestart || ack.Commands[0].Target != "nginx.service" {
		t.Errorf("delivered %+v", ack.Commands[0])
	}

	// A second push must not hand it out again. Decoded into a fresh value:
	// `commands` is omitempty, so reusing the first ack would silently keep the
	// old slice and the assertion would pass either way.
	_, data = ts.do(t, nil, http.MethodPost, "/api/ingest", p,
		map[string]string{"Authorization": "Bearer " + token})
	var second contracts.PushAck
	json.Unmarshal(data, &second)
	if len(second.Commands) != 0 {
		t.Errorf("a claimed command was delivered twice: %+v", second.Commands)
	}

	// Reporting the outcome closes it, and the history shows who asked.
	p2 := samplePush()
	p2.ControlEnabled = true
	p2.CommandResults = []contracts.CommandResult{
		{ID: cmd.ID, OK: true, Output: "done", FinishedAt: time.Now().UTC()},
	}
	ts.do(t, nil, http.MethodPost, "/api/ingest", p2,
		map[string]string{"Authorization": "Bearer " + token})

	_, data = ts.do(t, admin, http.MethodGet, "/api/admin/hosts/"+host+"/commands", nil, nil)
	var hist []commandView
	json.Unmarshal(data, &hist)
	if len(hist) != 1 || hist[0].Status != store.CommandDone || hist[0].Output != "done" {
		t.Fatalf("history = %s", data)
	}
	if hist[0].RequestedByName != "boss@example.com" {
		t.Errorf("history lost the requester: %+v", hist[0])
	}
}

// The action list is an allowlist, not a suggestion.
func TestQueueRejectsAnUnknownAction(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	host, _ := controllableHost(t, ts, admin, "box")

	for _, action := range []string{"rm -rf /", "exec", "", "run_script", "REBOOT"} {
		if code, _ := queue(t, ts, admin, host, action, "nginx.service"); code != http.StatusBadRequest {
			t.Errorf("action %q = %d, want 400", action, code)
		}
	}
	if n := countRows(t, ts, `SELECT COUNT(*) FROM host_commands`); n != 0 {
		t.Errorf("%d rejected commands were still queued", n)
	}
}

// A target the host never reported cannot be named: the server does not take
// the caller's word for what runs on someone else's machine.
func TestQueueRejectsAnUnreportedTarget(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	host, _ := controllableHost(t, ts, admin, "box")

	cases := []struct{ action, target string }{
		{contracts.ActionServiceRestart, "not-a-unit.service"},
		{contracts.ActionServiceStop, ""},
		{contracts.ActionContainerRestart, "no-such-container"},
		{contracts.ActionServiceRestart, "nginx.service; rm -rf /"},
	}
	for _, tc := range cases {
		if code, _ := queue(t, ts, admin, host, tc.action, tc.target); code != http.StatusBadRequest {
			t.Errorf("%s %q = %d, want 400", tc.action, tc.target, code)
		}
	}
	// The container samplePush reports is named "web", and it is accepted.
	if code, _ := queue(t, ts, admin, host, contracts.ActionContainerRestart, "web"); code != http.StatusCreated {
		t.Errorf("a reported container = %d, want 201", code)
	}
}

// Power actions take no target, so one is not required of them.
func TestQueuePowerActionsNeedNoTarget(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	host, _ := controllableHost(t, ts, admin, "box")

	for _, action := range []string{contracts.ActionReboot, contracts.ActionPoweroff} {
		code, cmd := queue(t, ts, admin, host, action, "")
		if code != http.StatusCreated {
			t.Fatalf("%s = %d, want 201", action, code)
		}
		if cmd.Target != "" {
			t.Errorf("%s carried target %q", action, cmd.Target)
		}
	}
}

// An agent that has not said it runs commands is never sent one, so the button
// fails immediately with a reason instead of the command expiring in silence.
func TestQueueRefusedWhenTheAgentHasNotReportedControl(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	host, token := enrollHost(t, ts, admin, "old-agent")
	// An agent from before the feature pushes without control_enabled.
	ts.do(t, nil, http.MethodPost, "/api/ingest", samplePush(),
		map[string]string{"Authorization": "Bearer " + token})

	code, _ := queue(t, ts, admin, host, contracts.ActionReboot, "")
	if code != http.StatusConflict {
		t.Errorf("queue for an agent without control = %d, want 409", code)
	}
	if n := countRows(t, ts, `SELECT COUNT(*) FROM host_commands`); n != 0 {
		t.Errorf("%d commands queued for a host that cannot run them", n)
	}
}

// A host that vetoed control locally is refused for the same reason.
func TestQueueRefusedWhenTheHostVetoes(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	host, token := controllableHost(t, ts, admin, "box")

	vetoed := samplePush()
	vetoed.ControlEnabled = false
	ts.do(t, nil, http.MethodPost, "/api/ingest", vetoed,
		map[string]string{"Authorization": "Bearer " + token})

	if code, _ := queue(t, ts, admin, host, contracts.ActionReboot, ""); code != http.StatusConflict {
		t.Errorf("queue after a local veto = %d, want 409", code)
	}
}

// A vetoed agent is also never handed a command already in the queue.
func TestVetoedAgentCollectsNothing(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	host, token := controllableHost(t, ts, admin, "box")
	queue(t, ts, admin, host, contracts.ActionServiceRestart, "nginx.service")

	vetoed := samplePush()
	vetoed.ControlEnabled = false
	_, data := ts.do(t, nil, http.MethodPost, "/api/ingest", vetoed,
		map[string]string{"Authorization": "Bearer " + token})
	var ack contracts.PushAck
	json.Unmarshal(data, &ack)
	if len(ack.Commands) != 0 {
		t.Errorf("a vetoed agent was handed %+v", ack.Commands)
	}
}

func TestCommandsAreAdminOnly(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	host, _ := controllableHost(t, ts, admin, "box")
	basic := ts.client(t)
	signup(t, ts, basic, "dev@example.com", "password123")

	if code, _ := queue(t, ts, basic, host, contracts.ActionReboot, ""); code != http.StatusForbidden {
		t.Errorf("basic user queueing = %d, want 403", code)
	}
	resp, _ := ts.do(t, basic, http.MethodGet, "/api/admin/hosts/"+host+"/commands", nil, nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("basic user reading history = %d, want 403", resp.StatusCode)
	}
}

// One host must not be able to close another's command by guessing its id.
func TestAResultFromTheWrongHostIsIgnored(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	victim, victimToken := controllableHost(t, ts, admin, "victim")
	_, attackerToken := controllableHost(t, ts, admin, "attacker")

	_, cmd := queue(t, ts, admin, victim, contracts.ActionServiceRestart, "nginx.service")
	p := samplePush()
	p.ControlEnabled = true
	ts.do(t, nil, http.MethodPost, "/api/ingest", p,
		map[string]string{"Authorization": "Bearer " + victimToken})

	forged := samplePush()
	forged.ControlEnabled = true
	forged.CommandResults = []contracts.CommandResult{{ID: cmd.ID, OK: true, Output: "pwned"}}
	ts.do(t, nil, http.MethodPost, "/api/ingest", forged,
		map[string]string{"Authorization": "Bearer " + attackerToken})

	got, err := ts.app.db.GetCommand(cmd.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != store.CommandSent || got.Output == "pwned" {
		t.Errorf("another host closed this command: %+v", got)
	}
}
