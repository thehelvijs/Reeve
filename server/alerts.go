package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/thehelvijs/Reeve/contracts"
	"github.com/thehelvijs/Reeve/server/internal/store"
)

// metricLabels renders each metric scalar in alert messages.
var metricLabels = map[string]string{"cpu": "CPU", "mem": "memory", "disk": "disk", "temp": "temperature", "load": "load"}

const (
	downDebounce = 60 * time.Second
	logWindow    = 5 * time.Minute
	logThreshold = 1
)

// alertSubject describes what an alert is about, for the event row and payload.
type alertSubject struct {
	key      string
	altype   string
	toolID   string
	toolName string
	hostID   string
	hostName string
	message  string
	severity string
	debounce time.Duration
}

// evaluateAlerts runs one alerting pass: detect down/offline/log-error
// conditions, apply debounce, fire/resolve events, enqueue webhook deliveries.
// `now` is injected so tests are deterministic.
func (a *app) evaluateAlerts(now time.Time) {
	hosts, err := a.db.ListHosts()
	if err != nil {
		log.Printf("alerts: list hosts: %v", err)
		return
	}
	hostMap := make(map[string]store.Host, len(hosts))
	for _, h := range hosts {
		hostMap[h.ID] = h
	}
	thresholds, err := a.db.LoadThresholds()
	if err != nil {
		log.Printf("alerts: load thresholds: %v", err)
		return
	}
	window := a.thresholdWindow()

	for _, h := range hosts {
		offline := h.LastSeenAt != nil && !h.Online(now)
		a.transition(alertSubject{
			key: "host:" + h.ID + ":agent_offline", altype: "agent_offline",
			hostID: h.ID, hostName: h.Name, message: "agent for host " + h.Name + " is offline",
			severity: "error", debounce: downDebounce,
		}, offline, now)

		if !offline && h.LastSeenAt != nil {
			a.evaluateHostThresholds(h, thresholds, window, now)
		}
	}

	tools, err := a.db.ListMonitoredTools()
	if err != nil {
		log.Printf("alerts: list tools: %v", err)
		return
	}
	for _, t := range tools {
		host, ok := hostMap[t.HostID]
		if !ok || !host.Online(now) {
			continue // an offline host is covered by its host-level alert
		}
		down := a.toolStatus(t, hostMap, now) == contracts.StatusDown
		a.transition(alertSubject{
			key: "tool:" + t.ID + ":down", altype: "down",
			toolID: t.ID, toolName: t.Name, hostID: t.HostID, hostName: host.Name,
			message: t.Name + " is down", severity: "error", debounce: downDebounce,
		}, down, now)

		if t.LogAlertEnabled {
			cnt, _ := a.db.CountLogEventsForTool(t.HostID, t.SourceRef, now.Add(-logWindow))
			a.transition(alertSubject{
				key: "tool:" + t.ID + ":log_error", altype: "log_error",
				toolID: t.ID, toolName: t.Name, hostID: t.HostID, hostName: host.Name,
				message: t.Name + " is logging errors", severity: "warning", debounce: 0,
			}, cnt >= logThreshold, now)
		}
	}
}

func (a *app) thresholdWindow() time.Duration {
	if v, ok := a.db.GetSetting("threshold.window_secs"); ok {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return time.Duration(n) * time.Second
		}
	}
	return 5 * time.Minute
}

// hostMetricValues derives the single scalar per metric (cpu, mem%, disk%, temp
// max, load) so static thresholds and anomaly rules can never disagree.
func hostMetricValues(m store.MetricPoint) map[string]float64 {
	values := map[string]float64{}
	values["cpu"] = m.CPUPct
	if m.MemTotal > 0 {
		values["mem"] = float64(m.MemUsed) / float64(m.MemTotal) * 100
	}
	if m.DiskTotal > 0 {
		values["disk"] = float64(m.DiskUsed) / float64(m.DiskTotal) * 100
	}
	if len(m.Temps) > 0 {
		var maxT float64
		for _, v := range m.Temps {
			if v > maxT {
				maxT = v
			}
		}
		values["temp"] = maxT
	}
	values["load"] = m.Load1
	return values
}

// evaluateHostThresholds fires <metric>_high subjects for a host whose latest
// sample breaches an enabled threshold for the sustained window. The threshold
// set and window come from the pass, so neither is re-read per host.
func (a *app) evaluateHostThresholds(h store.Host, thresholds store.ThresholdSet, window time.Duration, now time.Time) {
	values := map[string]float64{}
	if m, haveMetric := a.db.LatestHostMetric(h.ID); haveMetric {
		values = hostMetricValues(m)
	}
	if rate, ok := a.db.HostNetRate(h.ID); ok {
		values["net"] = rate
	}
	for _, metric := range thresholdMetrics {
		val, present := values[metric]
		th, thOK := thresholds.Effective(h.ID, metric)
		bad := present && thOK && th.Enabled && val >= th.Value
		a.transition(alertSubject{
			key: "host:" + h.ID + ":" + metric + "_high", altype: metric + "_high",
			hostID: h.ID, hostName: h.Name, message: thresholdMessage(h.Name, metric, val, th.Value),
			severity: "warning", debounce: window,
		}, bad, now)
	}
}

// thresholdMessage renders the alert text for one breached metric.
func thresholdMessage(hostName, metric string, val, limit float64) string {
	switch metric {
	case "temp":
		return fmt.Sprintf("%s temperature %.0f°C over %.0f°C", hostName, val, limit)
	case "load":
		return fmt.Sprintf("%s load %.2f over %.2f", hostName, val, limit)
	case "net":
		return fmt.Sprintf("%s network %s/s over %s/s", hostName, fmtRate(val), fmtRate(limit))
	default:
		return fmt.Sprintf("%s %s %.0f%% over %.0f%%", hostName, metricLabels[metric], val, limit)
	}
}

// transition advances one subject's state machine and fires/resolves as needed.
func (a *app) transition(s alertSubject, bad bool, now time.Time) {
	st := a.db.GetAlertState(s.key)
	if bad {
		if st.State == "firing" {
			return
		}
		if st.State == "ok" {
			st.State = "pending"
			st.PendingSince = &now
			a.db.PutAlertState(st)
		}
		if st.PendingSince != nil && now.Sub(*st.PendingSince) >= s.debounce {
			a.fire(s, st, now)
		}
		return
	}
	// good
	switch st.State {
	case "firing":
		a.db.ResolveAlertEvent(st.OpenEventID, now)
		a.enqueue(s, "resolved", now)
		a.db.PutAlertState(store.AlertState{SubjectKey: s.key, State: "ok"})
	case "pending":
		a.db.PutAlertState(store.AlertState{SubjectKey: s.key, State: "ok"})
	}
}

func (a *app) fire(s alertSubject, st store.AlertState, now time.Time) {
	ev, err := a.db.CreateAlertEvent(store.AlertEvent{
		SubjectKey: s.key, ToolID: s.toolID, HostID: s.hostID,
		Type: s.altype, Severity: severityOrError(s.severity), Message: s.message,
	}, now)
	if err != nil {
		log.Printf("alerts: create event: %v", err)
		return
	}
	st.State = "firing"
	st.OpenEventID = ev.ID
	st.PendingSince = nil
	a.db.PutAlertState(st)
	a.enqueue(s, "fired", now)
}

// enqueue writes a delivery for each resolved channel, formatted per kind and
// filtered by the channel's min_severity.
func (a *app) enqueue(s alertSubject, phase string, now time.Time) {
	severity := severityOrError(s.severity)
	generic, _ := json.Marshal(map[string]any{
		"event":     phase,
		"type":      s.altype,
		"severity":  severity,
		"tool":      s.toolName,
		"tool_id":   s.toolID,
		"host":      s.hostName,
		"host_id":   s.hostID,
		"message":   s.message,
		"timestamp": now.UTC().Format(time.RFC3339),
	})
	hooks, _ := a.db.ResolveChannels(s.toolID, s.hostID, severity)
	for _, h := range hooks {
		a.db.EnqueueDelivery(h.ID, string(generic), now)
	}
}

// severityOrError defaults an unset severity to error so no alert is silently
// downgraded when a subject leaves it blank.
func severityOrError(s string) string {
	if s == "" {
		return "error"
	}
	return s
}

// fmtRate renders a bytes/sec value in human-readable binary units.
func fmtRate(bytesPerSec float64) string {
	const unit = 1024
	if bytesPerSec < unit {
		return fmt.Sprintf("%.0f B", bytesPerSec)
	}
	div, exp := float64(unit), 0
	for n := bytesPerSec / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	units := []string{"KiB", "MiB", "GiB", "TiB", "PiB"}
	return fmt.Sprintf("%.1f %s", bytesPerSec/div, units[exp])
}
