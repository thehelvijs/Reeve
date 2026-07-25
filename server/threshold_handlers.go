package main

import (
	"net/http"
	"strconv"
)

var thresholdMetrics = []string{"cpu", "mem", "disk", "temp", "load", "net"}

type thresholdDTO struct {
	Enabled   bool    `json:"enabled"`
	Threshold float64 `json:"threshold"`
}

type thresholdsPayload struct {
	WindowSecs int                     `json:"window_secs"`
	Thresholds map[string]thresholdDTO `json:"thresholds"`
}

func (a *app) thresholdsFor(hostID string) map[string]thresholdDTO {
	out := map[string]thresholdDTO{}
	set, err := a.db.LoadThresholds()
	if err != nil {
		return out
	}
	for _, m := range thresholdMetrics {
		if th, ok := set.Effective(hostID, m); ok {
			out[m] = thresholdDTO{Enabled: th.Enabled, Threshold: th.Value}
		}
	}
	return out
}

func (a *app) handleGetThresholds(w http.ResponseWriter, _ *http.Request) {
	window := 300
	if v, ok := a.db.GetSetting("threshold.window_secs"); ok {
		if n, err := strconv.Atoi(v); err == nil {
			window = n
		}
	}
	writeJSON(w, http.StatusOK, thresholdsPayload{WindowSecs: window, Thresholds: a.thresholdsFor("")})
}

func (a *app) handlePutThresholds(w http.ResponseWriter, r *http.Request) {
	var in thresholdsPayload
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if in.WindowSecs > 0 {
		a.db.SetSetting("threshold.window_secs", strconv.Itoa(in.WindowSecs))
	}
	for _, m := range thresholdMetrics {
		if dto, ok := in.Thresholds[m]; ok {
			if err := a.db.SetThreshold("", m, dto.Enabled, dto.Threshold); err != nil {
				writeError(w, http.StatusInternalServerError, "internal", "could not save thresholds")
				return
			}
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *app) handleGetHostThresholds(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := a.db.GetHost(id); err != nil {
		writeError(w, http.StatusNotFound, "not_found", "host not found")
		return
	}
	writeJSON(w, http.StatusOK, thresholdsPayload{Thresholds: a.thresholdsFor(id)})
}

func (a *app) handlePutHostThresholds(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := a.db.GetHost(id); err != nil {
		writeError(w, http.StatusNotFound, "not_found", "host not found")
		return
	}
	var in thresholdsPayload
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	for _, m := range thresholdMetrics {
		if dto, ok := in.Thresholds[m]; ok {
			if err := a.db.SetThreshold(id, m, dto.Enabled, dto.Threshold); err != nil {
				writeError(w, http.StatusInternalServerError, "internal", "could not save thresholds")
				return
			}
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *app) handleResetHostThresholds(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := a.db.GetHost(id); err != nil {
		writeError(w, http.StatusNotFound, "not_found", "host not found")
		return
	}
	if err := a.db.DeleteHostThresholds(id); err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not reset thresholds")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
