package main

import (
	"net/http"
	"time"

	"github.com/thehelvijs/Reeve/server/internal/store"
)

type agentUpdateRollup struct {
	ServerVersion string              `json:"server_version"`
	Counts        map[string]int      `json:"counts"`
	Paused        bool                `json:"paused"`
	Stalled       []store.StalledHost `json:"stalled"`
}

// handleGetAgentUpdates reports fleet agent versions and rollout health.
func (a *app) handleGetAgentUpdates(w http.ResponseWriter, _ *http.Request) {
	hosts, err := a.db.ListHosts()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not list hosts")
		return
	}
	now := time.Now().UTC()
	uc := a.updateContext()
	counts := map[string]int{
		updateStateUpToDate: 0,
		updateStateOutdated: 0,
		updateStateUpdating: 0,
		updateStateStalled:  0,
		updateStateDisabled: 0,
		updateStateUnknown:  0,
	}
	for _, h := range hosts {
		counts[updateStateFor(h, uc, now)]++
	}
	stalled, err := a.db.ListStalledHosts(now.Add(-uc.Stall))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not read rollout state")
		return
	}
	writeJSON(w, http.StatusOK, agentUpdateRollup{
		ServerVersion: uc.ServerVersion,
		Counts:        counts,
		Paused:        len(stalled) > 0,
		Stalled:       stalled,
	})
}

// handleResumeAgentUpdates releases every stalled slot so the rollout restarts.
func (a *app) handleResumeAgentUpdates(w http.ResponseWriter, _ *http.Request) {
	uc := a.updateContext()
	if err := a.db.ClearStalledUpdates(time.Now().UTC().Add(-uc.Stall)); err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not resume the rollout")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleSetHostAutoUpdate overrides the fleet default for one host.
func (a *app) handleSetHostAutoUpdate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var in struct {
		Policy string `json:"policy"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if !store.ValidAutoUpdatePolicy(in.Policy) {
		writeError(w, http.StatusBadRequest, "invalid_policy", "policy must be default, on, or off")
		return
	}
	if err := a.db.SetHostAutoUpdate(id, in.Policy); err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusNotFound, "not_found", "host not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", "could not save the policy")
		return
	}
	updated, err := a.db.GetHost(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not load host")
		return
	}
	writeJSON(w, http.StatusOK, hostToView(updated, time.Now().UTC(), a.updateContext()))
}

// handleHostUpdateNow grants one host a slot immediately, ignoring the
// concurrency cap and a paused rollout. It refuses a host that cannot update.
func (a *app) handleHostUpdateNow(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	h, err := a.db.GetHost(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "host not found")
		return
	}
	if h.AutoUpdateVetoed {
		writeError(w, http.StatusConflict, "update_vetoed", "auto-update is disabled on the host itself")
		return
	}
	if !effectiveAutoUpdate(h.AutoUpdate, a.agentUpdateConfig().Enabled) {
		writeError(w, http.StatusConflict, "update_disabled", "auto-update is off for this host")
		return
	}
	if !versionComparable(a.cfg.Version) || !versionComparable(h.AgentVersion) {
		writeError(w, http.StatusConflict, "version_unknown",
			"no comparable release version on the server or the host, so there is nothing to update to")
		return
	}
	if h.AgentVersion == a.cfg.Version {
		writeError(w, http.StatusConflict, "already_up_to_date", "host is already on the current version")
		return
	}
	if err := a.db.StartHostUpdate(id, time.Now().UTC()); err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not start the update")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
