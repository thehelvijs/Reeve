package main

import (
	"math"
	"net/http"
	"time"

	"github.com/thehelvijs/Reeve/server/internal/rbac"
	"github.com/thehelvijs/Reeve/server/internal/store"
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

// handleServerMetrics returns the Reeve server's own metric history,
// recorded under the reserved self-monitoring host.
func (a *app) handleServerMetrics(w http.ResponseWriter, r *http.Request) {
	spec, ok := rangeSpec[r.URL.Query().Get("range")]
	if !ok {
		spec = rangeSpec["24h"]
	}
	since := time.Now().UTC().Add(-spec.window)
	host, err := a.db.QueryHostMetrics(store.ServerHostID, spec.resolution, since)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not read metrics")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"resolution": spec.resolution,
		"host":       host,
		"containers": []any{},
	})
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
	containers, err := a.db.QueryContainerStats(id, spec.resolution, since)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not read container metrics")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"resolution": spec.resolution,
		"host":       host,
		"containers": containers,
	})
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
