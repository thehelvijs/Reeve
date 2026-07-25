package main

import (
	"net/http"

	"github.com/thehelvijs/Reeve/server/internal/rbac"
)

type visibilityGrantView struct {
	PrincipalType string `json:"principal_type"`
	PrincipalID   string `json:"principal_id"`
}

func (a *app) handleListToolVisibility(w http.ResponseWriter, r *http.Request) {
	p, _ := rbac.FromContext(r.Context())
	t, ok := a.loadEditableTool(w, r, p)
	if !ok {
		return
	}
	grants, err := a.db.ListToolVisibility(t.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not list visibility")
		return
	}
	out := make([]visibilityGrantView, 0, len(grants))
	for _, g := range grants {
		out = append(out, visibilityGrantView{PrincipalType: g.PrincipalType, PrincipalID: g.PrincipalID})
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *app) handleAddToolVisibility(w http.ResponseWriter, r *http.Request) {
	p, _ := rbac.FromContext(r.Context())
	t, ok := a.loadEditableTool(w, r, p)
	if !ok {
		return
	}
	ptype := r.PathValue("ptype")
	if ptype != "user" && ptype != "group" {
		writeError(w, http.StatusBadRequest, "invalid_principal", "principal type must be user or group")
		return
	}
	if err := a.db.AddToolVisibility(t.ID, ptype, r.PathValue("pid")); err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not add visibility")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *app) handleRemoveToolVisibility(w http.ResponseWriter, r *http.Request) {
	p, _ := rbac.FromContext(r.Context())
	t, ok := a.loadEditableTool(w, r, p)
	if !ok {
		return
	}
	if err := a.db.RemoveToolVisibility(t.ID, r.PathValue("ptype"), r.PathValue("pid")); err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not remove visibility")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
