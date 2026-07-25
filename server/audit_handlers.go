package main

import (
	"net/http"

	"github.com/thehelvijs/Reeve/server/internal/store"
)

func (a *app) auditFilter(r *http.Request) store.AuditFilter {
	q := r.URL.Query()
	return store.AuditFilter{
		UserID: q.Get("user"),
		ToolID: q.Get("tool"),
		From:   q.Get("from"),
		To:     q.Get("to"),
	}
}

func (a *app) handleRevealAudit(w http.ResponseWriter, r *http.Request) {
	events, err := a.db.ListRevealAudit(a.auditFilter(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not read audit")
		return
	}
	writeJSON(w, http.StatusOK, events)
}

func (a *app) handleGrantAudit(w http.ResponseWriter, r *http.Request) {
	events, err := a.db.ListGrantAudit(a.auditFilter(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not read audit")
		return
	}
	writeJSON(w, http.StatusOK, events)
}
