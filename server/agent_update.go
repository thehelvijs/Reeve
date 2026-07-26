package main

import (
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

// decideCheckNow answers one agent's push: may it self-update right now? The
// reported version and veto come from the push, which is newer than the row.
func (a *app) decideCheckNow(h store.Host, reportedVersion string, vetoed bool, now time.Time) bool {
	if reportedVersion == a.cfg.Version && h.UpdateStartedAt != nil {
		a.db.ClearHostUpdateSlot(h.ID)
	}
	if vetoed {
		return false
	}
	cfg := a.agentUpdateConfig()
	if !effectiveAutoUpdate(h.AutoUpdate, cfg.Enabled) {
		return false
	}
	if !versionComparable(a.cfg.Version) || !versionComparable(reportedVersion) {
		return false
	}
	if reportedVersion == a.cfg.Version {
		return false
	}

	stall := time.Duration(cfg.StallSecs) * time.Second
	cutoff := now.Add(-stall)
	if h.UpdateStartedAt != nil && now.Sub(*h.UpdateStartedAt) < stall {
		return true
	}
	stalled, err := a.db.CountStalledUpdates(cutoff)
	if err != nil || stalled > 0 {
		return false
	}
	live, err := a.db.CountLiveUpdateSlots(cutoff)
	if err != nil || live >= cfg.Concurrency {
		return false
	}
	if err := a.db.StartHostUpdate(h.ID, now); err != nil {
		return false
	}
	return true
}
