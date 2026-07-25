package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/thehelvijs/Reeve/contracts"
)

const sessionCookie = "lv_session"

func (a *app) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "version": a.cfg.Version})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, contracts.ErrorResponse{Code: code, Message: msg})
}

// decodeJSON reads a JSON body into dst, rejecting unknown fields and oversized
// payloads.
func decodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return errors.New("invalid request body")
	}
	return nil
}
