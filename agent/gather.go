package main

import (
	"context"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/thehelvijs/Reeve/agent/collect"
	"github.com/thehelvijs/Reeve/contracts"
)

// commandTimeout bounds every helper the agent shells out to. Without it a hung
// docker or systemctl would stall the push loop for as long as it hangs. A var
// so tests can shorten it.
var commandTimeout = 20 * time.Second

// maxLogReaders caps concurrent `docker logs` calls: a host with many
// containers should not fork one process per container at once.
const maxLogReaders = 4

// gather collects a full telemetry snapshot from the host. Every collector is
// best-effort: a missing command or file yields an empty section rather than a
// failure, so one broken source never blocks the push. The collectors run
// concurrently, so a tick costs the slowest source rather than their sum:
// `docker stats --no-stream` alone takes over a second, and the CPU sample
// spends 200ms inside its own measurement window.
func gather(version string, cfg config) contracts.Push {
	push := contracts.Push{
		AgentVersion:     version,
		AutoUpdateVetoed: !cfg.AutoUpdate,
		IPAddress:        localIPFor(cfg.ServerURL),
		SentAt:           time.Now().UTC(),
	}

	var services []contracts.ServiceState
	var containers []contracts.ContainerState
	var stats []contracts.ContainerSample
	var crons []contracts.CronState
	var metrics contracts.HostMetrics
	var journalErrors, dockerErrors []contracts.LogEvent

	var wg sync.WaitGroup
	run := func(fn func()) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fn()
		}()
	}

	run(func() {
		if out, err := runCmd("systemctl", "list-units", "--type=service", "--all", "--no-legend", "--plain", "--no-pager"); err == nil {
			services = collect.ParseSystemctl(out)
		}
	})
	run(func() {
		if out, err := runCmd("docker", "stats", "--no-stream", "--format", "{{json .}}"); err == nil {
			stats = collect.ParseDockerStats(out)
		}
	})
	run(func() { crons = gatherCron() })
	run(func() { metrics = collect.SampleHostMetrics() })
	run(func() { journalErrors = gatherLogErrors() })
	// The container list feeds both the inventory and the per-container log
	// scan, so one `docker ps` serves both.
	run(func() {
		out, err := runCmd("docker", "ps", "-a", "--format", "{{json .}}")
		if err != nil {
			return
		}
		containers = collect.ParseDockerPS(out)
		dockerErrors = gatherDockerLogErrors(containers)
	})
	wg.Wait()

	push.Services = services
	push.Containers = containers
	push.ContainerStats = stats
	push.CronJobs = crons
	push.Metrics = metrics
	push.LogEvents = append(journalErrors, dockerErrors...)
	return push
}

// localIPFor reports this host's address on the route to serverURL.
//
// A UDP "connection" sends nothing: the kernel just resolves the route and
// picks the source address it would use to reach that host. That is the one
// address guaranteed to be reachable from the server's network, which is also
// where the browser following a /go/ redirect lives, so docker0, tailscale0
// and VPN interfaces sort themselves out with no per-host configuration.
//
// Best-effort like every other collector: an empty string when it cannot tell.
func localIPFor(serverURL string) string {
	u, err := url.Parse(serverURL)
	if err != nil || u.Host == "" {
		return ""
	}
	host := u.Hostname()
	port := u.Port()
	if port == "" {
		port = "80"
		if u.Scheme == "https" {
			port = "443"
		}
	}
	conn, err := net.DialTimeout("udp", net.JoinHostPort(host, port), commandTimeout)
	if err != nil {
		return ""
	}
	defer conn.Close()
	addr, _, err := net.SplitHostPort(conn.LocalAddr().String())
	if err != nil {
		return ""
	}
	if ip := net.ParseIP(addr); ip == nil || ip.IsUnspecified() || ip.IsLoopback() {
		return ""
	}
	return addr
}

func gatherCron() []contracts.CronState {
	var jobs []contracts.CronState
	if content, err := os.ReadFile("/etc/crontab"); err == nil {
		jobs = append(jobs, collect.ParseCrontab(string(content), true)...)
	}
	matches, _ := filepath.Glob("/etc/cron.d/*")
	for _, f := range matches {
		if content, err := os.ReadFile(f); err == nil {
			jobs = append(jobs, collect.ParseCrontab(string(content), true)...)
		}
	}
	return jobs
}

func gatherLogErrors() []contracts.LogEvent {
	out, err := runCmd("journalctl", "-p", "err", "--since", "-1min", "--no-pager", "-q")
	if err != nil {
		return nil
	}
	return collect.ScanLogErrors("journald", strings.Split(out, "\n"), time.Now().UTC(), nil)
}

// gatherDockerLogErrors scans the recent logs of every running container,
// reading them concurrently since each is an independent docker call.
func gatherDockerLogErrors(containers []contracts.ContainerState) []contracts.LogEvent {
	perContainer := make([][]contracts.LogEvent, len(containers))
	slots := make(chan struct{}, maxLogReaders)
	var wg sync.WaitGroup
	for i, c := range containers {
		if c.State != "running" || c.ID == "" {
			continue
		}
		wg.Add(1)
		go func(i int, id string) {
			defer wg.Done()
			slots <- struct{}{}
			defer func() { <-slots }()
			logs, err := runCmdCombined("docker", "logs", "--since", "90s", "--tail", "500", id)
			if err != nil {
				return
			}
			perContainer[i] = collect.ScanLogErrors("docker:"+id, strings.Split(logs, "\n"), time.Now().UTC(), nil)
		}(i, c.ID)
	}
	wg.Wait()

	var events []contracts.LogEvent
	for _, e := range perContainer {
		events = append(events, e...)
	}
	return events
}

func runCmd(name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, name, args...).Output()
	return string(out), err
}

func runCmdCombined(name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	return string(out), err
}
