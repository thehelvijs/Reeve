package main

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"sync"
	"time"

	"github.com/thehelvijs/Reeve/contracts"
)

// commandRunTimeout bounds one remote action. A var so tests can shorten it.
var commandRunTimeout = 30 * time.Second

// runArgv executes a resolved argv. It is a var so tests can substitute a fake:
// nothing in the suite may reboot, power off, or restart a real unit on the
// machine running it. Only this one function ever touches the system.
var runArgv = func(ctx context.Context, argv []string) ([]byte, error) {
	return exec.CommandContext(ctx, argv[0], argv[1:]...).CombinedOutput()
}

// safeTarget is what a unit or container name may look like. The server also
// checks the target against what this host reported, but this process runs as
// root on a machine its operator owns and does not get to assume the server is
// honest. Note the absence of '/', whitespace and every shell metacharacter.
var safeTarget = regexp.MustCompile(`^[A-Za-z0-9_.@:-]{1,128}$`)

// argvFor maps an action to the exact argv to execute. Returning a slice, never
// a string, is what keeps a shell out of this path entirely.
func argvFor(action, target string) ([]string, error) {
	kind, known := contracts.CommandActions[action]
	if !known {
		return nil, fmt.Errorf("unknown action %q", action)
	}
	if kind == contracts.TargetNone {
		if action == contracts.ActionReboot {
			return []string{"systemctl", "reboot"}, nil
		}
		return []string{"systemctl", "poweroff"}, nil
	}
	if !safeTarget.MatchString(target) {
		return nil, fmt.Errorf("refusing target %q", target)
	}
	verb := map[string]string{
		contracts.ActionServiceStart: "start", contracts.ActionServiceStop: "stop",
		contracts.ActionServiceRestart: "restart", contracts.ActionContainerStart: "start",
		contracts.ActionContainerStop: "stop", contracts.ActionContainerRestart: "restart",
	}[action]
	if kind == contracts.TargetService {
		return []string{"systemctl", verb, target}, nil
	}
	return []string{"docker", verb, target}, nil
}

// controller runs the commands an ack delivers and holds their results for the
// next push. Only one run is in flight at a time, so a slow command drops the
// next batch rather than stacking overlapping executions.
type controller struct {
	allowed bool
	mu      sync.Mutex
	results []contracts.CommandResult
	running bool
}

func newController(allowed bool) *controller {
	return &controller{allowed: allowed}
}

// takeResults returns and clears the results waiting to be reported.
func (c *controller) takeResults() []contracts.CommandResult {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := c.results
	c.results = nil
	return out
}

func (c *controller) addResult(r contracts.CommandResult) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.results) >= contracts.MaxPushCommandResults {
		return
	}
	c.results = append(c.results, r)
}

// begin claims the single run slot, reporting whether the caller got it.
func (c *controller) begin() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.running {
		return false
	}
	c.running = true
	return true
}

func (c *controller) end() {
	c.mu.Lock()
	c.running = false
	c.mu.Unlock()
}

// run executes one command. reportNow is called with the result of a power
// action before it is invoked, because rebooting kills this process before it
// could ever report; that push is the only acknowledgement the server will get.
func (c *controller) run(cmd contracts.Command, reportNow func(contracts.CommandResult)) contracts.CommandResult {
	argv, err := argvFor(cmd.Action, cmd.Target)
	if err != nil {
		return contracts.CommandResult{ID: cmd.ID, OK: false, Output: err.Error(), FinishedAt: time.Now().UTC()}
	}
	if contracts.IsPowerAction(cmd.Action) {
		reportNow(contracts.CommandResult{
			ID: cmd.ID, OK: true, Output: cmd.Action + " invoked", FinishedAt: time.Now().UTC(),
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), commandRunTimeout)
	defer cancel()
	out, err := runArgv(ctx, argv)
	body := string(out)
	if len(body) > contracts.MaxCommandOutput {
		body = body[:contracts.MaxCommandOutput]
	}
	res := contracts.CommandResult{ID: cmd.ID, OK: err == nil, Output: body, FinishedAt: time.Now().UTC()}
	if err != nil && body == "" {
		res.Output = err.Error()
	}
	return res
}

// handle runs every command in an ack, unless this machine has opted out. The
// local veto wins over anything the server says.
func (c *controller) handle(cmds []contracts.Command, reportNow func(contracts.CommandResult)) {
	if !c.allowed || len(cmds) == 0 {
		return
	}
	if !c.begin() {
		return
	}
	defer c.end()
	for _, cmd := range cmds {
		res := c.run(cmd, reportNow)
		if !contracts.IsPowerAction(cmd.Action) {
			c.addResult(res)
		}
	}
}
