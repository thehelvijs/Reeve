package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/thehelvijs/Reeve/agent/collect"
	"github.com/thehelvijs/Reeve/contracts"
)

// gather collects a full telemetry snapshot from the host. Every collector is
// best-effort: a missing command or file yields an empty section rather than a
// failure, so one broken source never blocks the push.
func gather(version string, cfg config) contracts.Push {
	push := contracts.Push{
		ProtocolVersion: contracts.PushProtocolVersion,
		AgentVersion:    version,
		SentAt:          time.Now().UTC(),
	}

	if out, err := run("systemctl", "list-units", "--type=service", "--all", "--no-legend", "--plain", "--no-pager"); err == nil {
		push.Services = collect.ParseSystemctl(out)
	}
	if out, err := run("docker", "ps", "-a", "--format", "{{json .}}"); err == nil {
		push.Containers = collect.ParseDockerPS(out)
	}
	if out, err := run("docker", "stats", "--no-stream", "--format", "{{json .}}"); err == nil {
		push.ContainerStats = collect.ParseDockerStats(out)
	}
	push.CronJobs = gatherCron()
	push.Metrics = gatherMetrics()
	push.LogEvents = append(gatherLogErrors(), gatherDockerLogErrors()...)
	return push
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

func gatherMetrics() contracts.HostMetrics {
	return collect.SampleHostMetrics()
}

func gatherLogErrors() []contracts.LogEvent {
	out, err := run("journalctl", "-p", "err", "--since", "-1min", "--no-pager", "-q")
	if err != nil {
		return nil
	}
	var lines []string
	for _, l := range splitLines(out) {
		lines = append(lines, l)
	}
	return collect.ScanLogErrors("journald", lines, time.Now().UTC(), nil)
}

func gatherDockerLogErrors() []contracts.LogEvent {
	out, err := run("docker", "ps", "--format", "{{.ID}}")
	if err != nil {
		return nil
	}
	var events []contracts.LogEvent
	for _, id := range splitLines(out) {
		if id == "" {
			continue
		}
		logs, err := runCombined("docker", "logs", "--since", "90s", "--tail", "500", id)
		if err != nil {
			continue
		}
		events = append(events, collect.ScanLogErrors("docker:"+id, splitLines(logs), time.Now().UTC(), nil)...)
	}
	return events
}

func run(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).Output()
	return string(out), err
}

func runCombined(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).CombinedOutput()
	return string(out), err
}

func splitLines(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}
