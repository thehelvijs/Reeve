package main

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
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
	// Published is the set of sha256 sums of the agent builds this server
	// serves. Empty when it ships no agents, which makes every host unknown
	// rather than outdated: there is nothing to update to.
	Published    map[string]bool
	FleetDefault bool
	Stall        time.Duration
}

func (a *app) updateContext() updateContext {
	cfg := a.agentUpdateConfig()
	return updateContext{
		Published:    a.publishedChecksums(),
		FleetDefault: cfg.Enabled,
		Stall:        time.Duration(cfg.StallSecs) * time.Second,
	}
}

// publishedChecksums hashes every embedded agent build once. The files are
// baked into the binary, so the answer cannot change while the server runs.
func (a *app) publishedChecksums() map[string]bool {
	a.sumsOnce.Do(func() {
		a.sums = map[string]bool{}
		if a.agentFS == nil {
			return
		}
		for arch := range supportedArches {
			f, err := a.agentFS.Open(agentBinaryPrefix + arch)
			if err != nil {
				continue
			}
			h := sha256.New()
			_, err = io.Copy(h, f)
			f.Close()
			if err != nil {
				continue
			}
			a.sums[hex.EncodeToString(h.Sum(nil))] = true
		}
	})
	return a.sums
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

func updateStateFor(h store.Host, uc updateContext, now time.Time) string {
	if h.AutoUpdateVetoed || !effectiveAutoUpdate(h.AutoUpdate, uc.FleetDefault) {
		return updateStateDisabled
	}
	// A server with no builds has nothing to chase, whatever a host reports.
	if len(uc.Published) == 0 {
		return updateStateUnknown
	}
	if h.AgentChecksum != "" && uc.Published[h.AgentChecksum] {
		return updateStateUpToDate
	}
	// A host that has not said what it runs is not chased on its own: an agent
	// that can never answer would be granted a slot on every push and hold it
	// until the stall window paused the fleet. Once an operator stamps a slot by
	// hand it is a real update, and the state has to say so, or the next push
	// would release the slot as belonging to a host with nothing to do.
	if h.UpdateStartedAt == nil {
		if h.AgentChecksum == "" {
			return updateStateUnknown
		}
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
// reported checksum and veto come from the push, which is newer than the row.
// It derives the same updateStateFor an operator sees, so ingest and the host
// view can never disagree about why a host was or wasn't granted a slot.
func (a *app) decideCheckNow(h store.Host, reportedChecksum string, vetoed bool, now time.Time) bool {
	uc := a.updateContext()
	h.AgentChecksum = reportedChecksum
	h.AutoUpdateVetoed = vetoed

	state := updateStateFor(h, uc, now)
	if state != updateStateUpdating && state != updateStateStalled {
		a.releaseSlot(&h)
	}

	switch state {
	case updateStateUpdating:
		return true
	case updateStateOutdated:
		cfg := a.agentUpdateConfig()
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
