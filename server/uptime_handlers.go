package main

import (
	"net/http"
	"time"

	"github.com/thehelvijs/Reeve/server/internal/rbac"
)

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
	pct = roundTo2dp(pct)
	return uptimeResponse{Range: rng, UptimePct: pct}, nil
}

func roundTo2dp(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}
