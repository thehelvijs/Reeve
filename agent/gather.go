package main

import (
	"context"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
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

// topProcsPerDimension is how many processes each of the CPU and memory
// rankings contributes to a push.
const topProcsPerDimension = 25

// procs and host carry the previous tick's CPU counters, which is what makes a
// percentage a real average over the interval rather than over the process's
// whole lifetime. Ticks are serial, so they need no lock.
var procs = collect.NewProcSampler("/proc")
var host = collect.NewHostSampler()

// containerStatsInterval paces `docker stats`, which is the most expensive
// thing a tick does: it costs the daemon a full snapshot of every container and
// takes over a second. Per-container usage is graphed, not alerted on, so it
// does not need every push.
const containerStatsInterval = time.Minute

// lastLogScan and lastStats are when each throttled collector last ran, so a
// window is exactly the ground it has not covered yet. A fixed window wider
// than the push interval re-reads and re-sends the same log lines every tick:
// at 15s ticks the old 60s journal window sent each error four times, and the
// 90s docker window six.
var lastLogScan, lastStats time.Time

// maxLogWindow bounds the catch-up after a long outage: an agent that has been
// unable to push for a day must not then ask journald for a day of logs.
const maxLogWindow = 10 * time.Minute

// sinceLast returns how far back a collector should read to cover the ground
// since last, clamped to maxLogWindow, and never shorter than one interval.
func sinceLast(last, now time.Time, interval time.Duration) time.Duration {
	window := interval
	if !last.IsZero() && now.Sub(last) > window {
		window = now.Sub(last)
	}
	if window > maxLogWindow {
		return maxLogWindow
	}
	if window < time.Second {
		return time.Second
	}
	return window
}

// durationArg renders a window as whole seconds, the one form both
// `journalctl --since` and `docker logs --since` read the same way.
func durationArg(d time.Duration) string {
	return strconv.Itoa(int(d.Seconds())) + "s"
}

// control runs the actions an ack delivers. Replaced in main once the config
// is read; the zero value refuses everything, which is the safe default if a
// push somehow happens first.
var control = newController(false)

// gather collects a full telemetry snapshot from the host. Every collector is
// best-effort: a missing command or file yields an empty section rather than a
// failure, so one broken source never blocks the push. The collectors run
// concurrently, so a tick costs the slowest source rather than their sum, and
// the two most expensive sources are paced: `docker stats` runs once a minute,
// and the log scans read only the ground since the last tick.
func gather(version string, cfg config) contracts.Push {
	push := contracts.Push{
		AgentVersion:     version,
		AgentChecksum:    selfChecksum(),
		AutoUpdateVetoed: !cfg.AutoUpdate,
		ControlEnabled:   cfg.AllowControl,
		CommandResults:   control.takeResults(),
		IPAddress:        localIPFor(cfg.ServerURL),
		SentAt:           time.Now().UTC(),
	}

	var services []contracts.ServiceState
	var containers []contracts.ContainerState
	var stats []contracts.ContainerSample
	var crons []contracts.CronState
	var metrics contracts.HostMetrics
	var processes []contracts.ProcessSample
	var journalErrors, dockerErrors []contracts.LogEvent

	now := time.Now()
	logWindow := sinceLast(lastLogScan, now, cfg.Interval)
	lastLogScan = now
	wantStats := now.Sub(lastStats) >= containerStatsInterval
	if wantStats {
		lastStats = now
	}

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
	if wantStats {
		run(func() {
			if out, err := runCmd("docker", "stats", "--no-stream", "--format", "{{json .}}"); err == nil {
				stats = collect.ParseDockerStats(out)
			}
		})
	}
	run(func() { crons = gatherCron() })
	run(func() { metrics = host.Sample() })
	run(func() {
		processes = collect.TopProcs(procs.Sample(time.Now()), topProcsPerDimension)
	})
	run(func() { journalErrors = gatherLogErrors(logWindow) })
	// The container list feeds both the inventory and the per-container log
	// scan, so one `docker ps` serves both.
	run(func() {
		out, err := runCmd("docker", "ps", "-a", "--format", "{{json .}}")
		if err != nil {
			return
		}
		containers = collect.ParseDockerPS(out)
		dockerErrors = gatherDockerLogErrors(containers, logWindow)
	})
	wg.Wait()

	// Which container a process belongs to is a join across two collectors that
	// ran in parallel, so it waits until both have landed.
	procs.AttachContainers(processes, containers)

	push.Services = services
	push.Containers = containers
	push.ContainerStats = stats
	push.CronJobs = crons
	push.Metrics = metrics
	push.Processes = processes
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

func gatherLogErrors(window time.Duration) []contracts.LogEvent {
	out, err := runCmd("journalctl", "-p", "err", "--since", "-"+durationArg(window), "--no-pager", "-q")
	if err != nil {
		return nil
	}
	return collect.ScanLogErrors("journald", strings.Split(out, "\n"), time.Now().UTC(), nil)
}

// gatherDockerLogErrors scans the recent logs of every running container,
// reading them concurrently since each is an independent docker call.
func gatherDockerLogErrors(containers []contracts.ContainerState, window time.Duration) []contracts.LogEvent {
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
			logs, err := runCmdCombined("docker", "logs", "--since", durationArg(window), "--tail", "500", id)
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
