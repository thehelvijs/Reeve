package main

import (
	"log"
	"time"

	"github.com/thehelvijs/Reeve/server/internal/store"
)

const (
	settingAgentUpdateEnabled     = "agent_update.enabled"
	settingAgentUpdateConcurrency = "agent_update.concurrency"
	settingAgentUpdateStallSecs   = "agent_update.stall_secs"
)

const (
	defaultAgentUpdateConcurrency = 3
	defaultAgentUpdateStallSecs   = 900
	minAgentUpdateConcurrency     = 1
	maxAgentUpdateConcurrency     = 100
	minAgentUpdateStallSecs       = 1
	maxAgentUpdateStallSecs       = 86400
)

const (
	updateStateUpToDate = "up_to_date"
	updateStateOutdated = "outdated"
	updateStateUpdating = "updating"
	updateStateStalled  = "stalled"
	updateStateDisabled = "disabled"
	updateStateUnknown  = "unknown"
)

// agentUpdateConfig is the fleet-wide rollout policy.
type agentUpdateConfig struct {
	Enabled     bool
	Concurrency int
	StallSecs   int
}

func (a *app) agentUpdateConfig() agentUpdateConfig {
	return agentUpdateConfig{
		Enabled:     a.db.GetBoolSetting(settingAgentUpdateEnabled, true),
		Concurrency: a.db.GetIntSetting(settingAgentUpdateConcurrency, defaultAgentUpdateConcurrency),
		StallSecs:   a.db.GetIntSetting(settingAgentUpdateStallSecs, defaultAgentUpdateStallSecs),
	}
}

// updateContext is the fleet-wide state a host view needs to derive its state.
type updateContext struct {
	ServerVersion string
	FleetDefault  bool
	Stall         time.Duration
}

func (a *app) updateContext() updateContext {
	cfg := a.agentUpdateConfig()
	return updateContext{
		ServerVersion: a.cfg.Version,
		FleetDefault:  cfg.Enabled,
		Stall:         time.Duration(cfg.StallSecs) * time.Second,
	}
}

// effectiveAutoUpdate resolves the fleet default against a host's override.
func effectiveAutoUpdate(policy string, fleetDefault bool) bool {
	if policy == store.AutoUpdateOn {
		return true
	}
	if policy == store.AutoUpdateOff {
		return false
	}
	return fleetDefault
}

// versionComparable reports whether a version string names a real release. A
// dev build has no published checksum to chase, so it is never called outdated.
func versionComparable(v string) bool {
	return v != "" && v != "dev"
}

func updateStateFor(h store.Host, uc updateContext, now time.Time) string {
	if h.AutoUpdateVetoed || !effectiveAutoUpdate(h.AutoUpdate, uc.FleetDefault) {
		return updateStateDisabled
	}
	if !versionComparable(uc.ServerVersion) || !versionComparable(h.AgentVersion) {
		return updateStateUnknown
	}
	if h.AgentVersion == uc.ServerVersion {
		return updateStateUpToDate
	}
	if h.UpdateStartedAt == nil {
		return updateStateOutdated
	}
	if now.Sub(*h.UpdateStartedAt) < uc.Stall {
		return updateStateUpdating
	}
	return updateStateStalled
}

// releaseSlot drops a rollout slot the host is no longer waiting on, in the row
// and in the caller's copy. A slot nobody is waiting on halts every other host.
func (a *app) releaseSlot(h *store.Host) {
	if h.UpdateStartedAt == nil {
		return
	}
	if err := a.db.ClearHostUpdateSlot(h.ID); err != nil {
		log.Printf("agent update: clear slot for host %s: %v", h.ID, err)
		return
	}
	h.UpdateStartedAt = nil
}

// decideCheckNow answers one agent's push: may it self-update right now? The
// reported version and veto come from the push, which is newer than the row.
// It derives the same updateStateFor an operator sees, so ingest and the host
// view can never disagree about why a host was or wasn't granted a slot.
func (a *app) decideCheckNow(h store.Host, reportedVersion string, vetoed bool, now time.Time) bool {
	cfg := a.agentUpdateConfig()
	uc := updateContext{
		ServerVersion: a.cfg.Version,
		FleetDefault:  cfg.Enabled,
		Stall:         time.Duration(cfg.StallSecs) * time.Second,
	}
	h.AgentVersion = reportedVersion
	h.AutoUpdateVetoed = vetoed

	state := updateStateFor(h, uc, now)
	if state != updateStateUpdating && state != updateStateStalled {
		a.releaseSlot(&h)
	}

	switch state {
	case updateStateUpdating:
		return true
	case updateStateOutdated:
		cutoff := now.Add(-uc.Stall)
		granted, err := a.db.TryStartHostUpdate(h.ID, now, cutoff, cfg.Concurrency, cfg.Enabled)
		if err != nil {
			log.Printf("agent update: try start slot for host %s: %v", h.ID, err)
			return false
		}
		return granted
	default:
		return false
	}
}
