package main

import (
	"net/http"
	"time"

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
