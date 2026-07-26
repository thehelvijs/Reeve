package main

import (
	"net/http"
	"strings"
	"time"

	"github.com/thehelvijs/Reeve/contracts"
	"github.com/thehelvijs/Reeve/server/internal/rbac"
	"github.com/thehelvijs/Reeve/server/internal/store"
)

// commandHistoryLimit is how many past commands the host page shows.
const commandHistoryLimit = 20

type commandInput struct {
	Action string `json:"action"`
	Target string `json:"target"`
}

// commandView carries the requester's identity resolved, so the history reads
// as "who did this" without the client joining against the user list.
type commandView struct {
	ID              string `json:"id"`
	HostID          string `json:"host_id"`
	Action          string `json:"action"`
	Target          string `json:"target"`
	Status          string `json:"status"`
	Output          string `json:"output"`
	RequestedBy     string `json:"requested_by"`
	RequestedByName string `json:"requested_by_name"`
	RequestedAt     string `json:"requested_at"`
	FinishedAt      string `json:"finished_at,omitempty"`
}

// toCommandView resolves the actor best-effort. A deleted account keeps its id
// rather than dropping the row: the history must still say who ran this.
func (a *app) toCommandView(c store.HostCommand) commandView {
	name := c.RequestedBy
	if u, err := a.db.GetUserByID(c.RequestedBy); err == nil {
		name = u.Email
	}
	v := commandView{
		ID: c.ID, HostID: c.HostID, Action: c.Action, Target: c.Target,
		Status: c.Status, Output: c.Output, RequestedBy: c.RequestedBy,
		RequestedByName: name, RequestedAt: c.RequestedAt.Format(time.RFC3339),
	}
	if c.FinishedAt != nil {
		v.FinishedAt = c.FinishedAt.Format(time.RFC3339)
	}
	return v
}

func (a *app) handleCreateCommand(w http.ResponseWriter, r *http.Request) {
	p, _ := rbac.FromContext(r.Context())
	h, ok := a.loadHost(w, r)
	if !ok {
		return
	}
	var in commandInput
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	kind, known := contracts.CommandActions[in.Action]
	if !known {
		writeError(w, http.StatusBadRequest, "unknown_action", "that is not an action this agent can be asked to run")
		return
	}
	// An agent that has not said it will run commands never gets one queued:
	// the alternative is a command sitting pending until it expires, with
	// nothing on screen explaining why.
	if !h.ControlEnabled {
		writeError(w, http.StatusConflict, "control_unavailable",
			"this host's agent has not reported control support — it predates the feature, or the machine runs it with REEVE_ALLOW_CONTROL=false")
		return
	}

	target := strings.TrimSpace(in.Target)
	switch kind {
	case contracts.TargetNone:
		target = ""
	case contracts.TargetService:
		if !a.db.HostReportsService(h.ID, target) {
			writeError(w, http.StatusBadRequest, "unknown_target", "this host has not reported that systemd unit")
			return
		}
	case contracts.TargetContainer:
		if !a.db.HostReportsContainer(h.ID, target) {
			writeError(w, http.StatusBadRequest, "unknown_target", "this host has not reported that container")
			return
		}
	}

	c, err := a.db.EnqueueCommand(h.ID, in.Action, target, p.UserID, time.Now().UTC())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not queue the command")
		return
	}
	writeJSON(w, http.StatusCreated, a.toCommandView(c))
}

func (a *app) handleListCommands(w http.ResponseWriter, r *http.Request) {
	h, ok := a.loadHost(w, r)
	if !ok {
		return
	}
	cmds, err := a.db.ListHostCommands(h.ID, commandHistoryLimit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not list commands")
		return
	}
	out := make([]commandView, 0, len(cmds))
	for _, c := range cmds {
		out = append(out, a.toCommandView(c))
	}
	writeJSON(w, http.StatusOK, out)
}
