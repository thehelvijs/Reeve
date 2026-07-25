package main

import (
	"net/http"
	"strings"
	"time"

	"github.com/thehelvijs/Reeve/server/internal/auth"
	"github.com/thehelvijs/Reeve/server/internal/rbac"
	"github.com/thehelvijs/Reeve/server/internal/store"
)

type requestView struct {
	ID          string `json:"id"`
	ToolID      string `json:"tool_id"`
	RequesterID string `json:"requester_id"`
	Status      string `json:"status"`
	Note        string `json:"note"`
	CreatedAt   string `json:"created_at"`
}

func toRequestView(r store.AccessRequest) requestView {
	return requestView{
		ID: r.ID, ToolID: r.ToolID, RequesterID: r.RequesterID, Status: r.Status,
		Note: r.Note, CreatedAt: r.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func (a *app) handleCreateAccessRequest(w http.ResponseWriter, r *http.Request) {
	p, _ := rbac.FromContext(r.Context())
	t, ok := a.loadVisibleTool(w, r, p)
	if !ok {
		return
	}
	var in struct {
		Note string `json:"note"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	req, err := a.db.CreateAccessRequest(t.ID, p.UserID, strings.TrimSpace(in.Note))
	if err == store.ErrDuplicateRequest {
		writeError(w, http.StatusConflict, "duplicate_request", "you already have an open request for this tool")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not create request")
		return
	}
	// In-app notification is the inbox; also fire a webhook if any is configured.
	a.notifyToolEvent(t.ID, map[string]any{
		"event": "access_request", "tool": t.Name, "tool_id": t.ID,
		"requester": p.Email, "note": req.Note,
		"timestamp": req.CreatedAt.UTC().Format(time.RFC3339),
	}, time.Now().UTC())
	writeJSON(w, http.StatusCreated, toRequestView(req))
}

func (a *app) handleListAccessRequests(w http.ResponseWriter, r *http.Request) {
	p, _ := rbac.FromContext(r.Context())
	var reqs []store.AccessRequest
	var err error
	if r.URL.Query().Get("box") == "inbox" {
		reqs, err = a.pendingForApprover(p)
	} else {
		reqs, err = a.db.ListRequestsByRequester(p.UserID)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not list requests")
		return
	}
	out := make([]requestView, 0, len(reqs))
	for _, req := range reqs {
		out = append(out, toRequestView(req))
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *app) handleApproveAccessRequest(w http.ResponseWriter, r *http.Request) {
	a.decideAccessRequest(w, r, store.RequestApproved)
}

func (a *app) handleDenyAccessRequest(w http.ResponseWriter, r *http.Request) {
	a.decideAccessRequest(w, r, store.RequestDenied)
}

func (a *app) decideAccessRequest(w http.ResponseWriter, r *http.Request, status string) {
	p, _ := rbac.FromContext(r.Context())
	req, err := a.db.GetAccessRequest(r.PathValue("rid"))
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "request not found")
		return
	}
	t, err := a.db.GetTool(req.ToolID)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "tool not found")
		return
	}
	if !p.IsAdmin() && t.CreatorID != p.UserID {
		writeError(w, http.StatusForbidden, "forbidden", "only the tool creator or an admin can decide this")
		return
	}

	// Approval may target the requester (default) or a chosen group.
	ptype, pid := "user", req.RequesterID
	if status == store.RequestApproved {
		var in struct {
			PrincipalType string `json:"principal_type"`
			PrincipalID   string `json:"principal_id"`
		}
		decodeJSON(r, &in)
		if in.PrincipalType == "group" && in.PrincipalID != "" {
			ptype, pid = "group", in.PrincipalID
		}
	}

	if err := a.db.DecideAccessRequest(req.ID, status, p.UserID); err != nil {
		writeError(w, http.StatusConflict, "already_decided", "request is no longer pending")
		return
	}
	if status == store.RequestApproved {
		a.db.GrantCredentialAccess(t.ID, ptype, pid, p.UserID)
		a.db.RecordGrant(t.ID, ptype, pid, "grant", p.UserID)
	}
	w.WriteHeader(http.StatusNoContent)
}

// Standing-grant management (independent of requests).

type accessGrantView struct {
	PrincipalType string `json:"principal_type"`
	PrincipalID   string `json:"principal_id"`
	GrantedBy     string `json:"granted_by"`
}

func (a *app) handleListToolAccess(w http.ResponseWriter, r *http.Request) {
	p, _ := rbac.FromContext(r.Context())
	t, ok := a.loadEditableTool(w, r, p)
	if !ok {
		return
	}
	grants, err := a.db.ListCredentialAccess(t.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not list access")
		return
	}
	out := make([]accessGrantView, 0, len(grants))
	for _, g := range grants {
		out = append(out, accessGrantView{PrincipalType: g.PrincipalType, PrincipalID: g.PrincipalID, GrantedBy: g.GrantedBy})
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *app) handleGrantToolAccess(w http.ResponseWriter, r *http.Request) {
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
	pid := r.PathValue("pid")
	a.db.GrantCredentialAccess(t.ID, ptype, pid, p.UserID)
	a.db.RecordGrant(t.ID, ptype, pid, "grant", p.UserID)
	w.WriteHeader(http.StatusNoContent)
}

func (a *app) handleRevokeToolAccess(w http.ResponseWriter, r *http.Request) {
	p, _ := rbac.FromContext(r.Context())
	t, ok := a.loadEditableTool(w, r, p)
	if !ok {
		return
	}
	ptype, pid := r.PathValue("ptype"), r.PathValue("pid")
	a.db.RevokeCredentialAccess(t.ID, ptype, pid)
	a.db.RecordGrant(t.ID, ptype, pid, "revoke", p.UserID)
	w.WriteHeader(http.StatusNoContent)
}

// pendingForApprover returns pending requests an approver may act on: all for an
// admin, or those on tools they created.
func (a *app) pendingForApprover(p auth.Principal) ([]store.AccessRequest, error) {
	if p.IsAdmin() {
		return a.db.ListAllPendingRequests()
	}
	toolIDs, err := a.db.ToolIDsByCreator(p.UserID)
	if err != nil {
		return nil, err
	}
	return a.db.ListPendingRequestsForTools(toolIDs)
}
