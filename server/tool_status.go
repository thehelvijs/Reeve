package main

import (
	"time"

	"github.com/thehelvijs/Reeve/contracts"
	"github.com/thehelvijs/Reeve/server/internal/store"
)

// toolStatus derives a tool's live status from agent telemetry; tools without an
// agent signal report "unknown".
func (a *app) toolStatus(t store.Tool, hosts map[string]store.Host, now time.Time) contracts.ToolStatus {
	return a.agentToolStatus(t, hosts, now)
}

// agentToolStatus derives status from the tool's agent source and host
// telemetry. Tools without a monitored source report "unknown".
func (a *app) agentToolStatus(t store.Tool, hosts map[string]store.Host, now time.Time) contracts.ToolStatus {
	if !agentMonitored(t) {
		return contracts.StatusUnknown
	}
	host, ok := hosts[t.HostID]
	if !ok {
		return contracts.StatusUnknown
	}
	if !host.Online(now) {
		return contracts.StatusAgentOffline
	}
	switch t.SourceType {
	case "systemd":
		active, ok := a.db.LookupServiceState(t.HostID, t.SourceRef)
		if !ok {
			return contracts.StatusUnknown
		}
		if active == "active" {
			return contracts.StatusUp
		}
		return contracts.StatusDown
	case "docker":
		state, health, ok := a.db.LookupContainerState(t.HostID, t.SourceRef)
		if !ok {
			return contracts.StatusUnknown
		}
		if state == "running" && (health == "" || health == "healthy") {
			return contracts.StatusUp
		}
		return contracts.StatusDown
	case "cron":
		if a.db.LookupCronExists(t.HostID, t.SourceRef) {
			return contracts.StatusUp
		}
		return contracts.StatusUnknown
	}
	return contracts.StatusUnknown
}

// agentMonitored reports whether a tool derives status from an agent source on
// a host (systemd/docker/cron).
func agentMonitored(t store.Tool) bool {
	return t.HostID != "" && t.SourceType != "manual" && t.SourceRef != ""
}

// hostMap loads all hosts keyed by id for status derivation.
func (a *app) hostMap() map[string]store.Host {
	hosts, err := a.db.ListHosts()
	if err != nil {
		return map[string]store.Host{}
	}
	m := make(map[string]store.Host, len(hosts))
	for _, h := range hosts {
		m[h.ID] = h
	}
	return m
}
