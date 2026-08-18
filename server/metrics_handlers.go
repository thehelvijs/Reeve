package main

import (
	"math"
	"net/http"
	"time"

	"github.com/thehelvijs/Reeve/contracts"
	"github.com/thehelvijs/Reeve/server/internal/rbac"
)

// rangeSpec maps a UI range to a lookback window and the resolution that keeps
// the series small enough to chart quickly.
var rangeSpec = map[string]struct {
	window     time.Duration
	resolution string
}{
	"1h":  {time.Hour, "raw"},
	"12h": {12 * time.Hour, "raw"},
	"24h": {24 * time.Hour, "raw"},
	"7d":  {7 * 24 * time.Hour, "5m"},
	"30d": {30 * 24 * time.Hour, "1h"},
}

func (a *app) handleHostMetrics(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := a.db.GetHost(id); err != nil {
		writeError(w, http.StatusNotFound, "not_found", "host not found")
		return
	}
	spec, ok := rangeSpec[r.URL.Query().Get("range")]
	if !ok {
		spec = rangeSpec["24h"]
	}
	since := time.Now().UTC().Add(-spec.window)

	host, err := a.db.QueryHostMetrics(id, spec.resolution, since)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not read metrics")
		return
	}
	// One point draws nothing, and a coarser tier holds fewer points rather than
	// more, so telling the reader to pick a longer range is advice that cannot
	// work on a host that started reporting minutes ago. Show everything kept
	// instead, and say which resolution it came back at.
	resolution := spec.resolution
	if len(host) < 2 && resolution != "raw" {
		if all, err := a.db.QueryHostMetrics(id, "raw", time.Time{}); err == nil && len(all) > len(host) {
			host, resolution, since = all, "raw", time.Time{}
		}
	}
	containers, err := a.db.QueryContainerStats(id, resolution, since)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not read container metrics")
		return
	}
	// Processes are the latest snapshot, not a series, so the range does not
	// apply to them; a host that has never reported any sends an empty list.
	procs, _ := a.db.LatestHostProcesses(id)
	if procs.Procs == nil {
		procs.Procs = []contracts.ProcessSample{}
	}
	disks, _ := a.db.LatestHostDisks(id)
	if disks.Disks == nil {
		disks.Disks = []contracts.DiskUsage{}
	}
	out := map[string]any{
		"resolution": resolution,
		"host":       host,
		"disks":      disks,
	}
	// The whole-machine series is for everyone; naming the containers and
	// processes behind it is the same inventory the dedicated endpoints gate, so
	// it travels with them rather than leaking around the side.
	if p, _ := rbac.FromContext(r.Context()); p.IsAdmin() {
		out["containers"] = containers
		out["processes"] = procs
	}
	writeJSON(w, http.StatusOK, out)
}

// topProcessUsage is how many commands one window reports, per dimension.
const topProcessUsage = 25

// handleHostProcessUsage answers "what has this machine actually been spending
// itself on", averaged over a window rather than sampled at the last push.
func (a *app) handleHostProcessUsage(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := a.db.GetHost(id); err != nil {
		writeError(w, http.StatusNotFound, "not_found", "host not found")
		return
	}
	spec, ok := rangeSpec[r.URL.Query().Get("window")]
	if !ok {
		spec = rangeSpec["24h"]
	}
	usage, err := a.db.ProcessUsageSince(id, time.Now().UTC().Add(-spec.window), topProcessUsage)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not read process usage")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"window": spec.window.String(), "processes": usage})
}

type uptimeResponse struct {
	Range     string  `json:"range"`
	UptimePct float64 `json:"uptime_pct"`
}

func (a *app) handleHostUptime(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := a.db.GetHost(id); err != nil {
		writeError(w, http.StatusNotFound, "not_found", "host not found")
		return
	}
	resp, err := a.uptimeFor("host_id", id, []string{"agent_offline"}, r)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not compute uptime")
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (a *app) handleToolUptime(w http.ResponseWriter, r *http.Request) {
	p, _ := rbac.FromContext(r.Context())
	t, ok := a.loadVisibleTool(w, r, p)
	if !ok {
		return
	}
	resp, err := a.uptimeFor("tool_id", t.ID, []string{"down"}, r)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not compute uptime")
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// uptimeFor resolves the range query param to a lookback window and returns
// the availability percentage for the subject over that window.
func (a *app) uptimeFor(filterCol, id string, types []string, r *http.Request) (uptimeResponse, error) {
	rng := r.URL.Query().Get("range")
	spec, ok := rangeSpec[rng]
	if !ok {
		rng = "24h"
		spec = rangeSpec["24h"]
	}
	now := time.Now().UTC()
	since := now.Add(-spec.window)
	downtime, err := a.db.DowntimeSecs(filterCol, id, types, since, now)
	if err != nil {
		return uptimeResponse{}, err
	}
	pct := 100 * (1 - downtime/spec.window.Seconds())
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	pct = math.Round(pct*100) / 100
	return uptimeResponse{Range: rng, UptimePct: pct}, nil
}
