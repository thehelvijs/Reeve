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
	ID            string `json:"id"`
	HostID        string `json:"host_id"`
	HostName      string `json:"host_name"`
	RequesterID   string `json:"requester_id"`
	RequesterName string `json:"requester_name"`
	Status        string `json:"status"`
	Note          string `json:"note"`
	CreatedAt     string `json:"created_at"`
}

// toRequestView resolves the asker best-effort, the way command history does:
// approving credential access to an id nobody can read is not a decision.
func (a *app) toRequestView(r store.AccessRequest) requestView {
	name := ""
	if h, err := a.db.GetHost(r.HostID); err == nil {
		name = h.Name
	}
	requester := r.RequesterID
	if u, err := a.db.GetUserByID(r.RequesterID); err == nil {
		requester = u.Email
		if u.DisplayName != "" {
			requester = u.DisplayName + " (" + u.Email + ")"
		}
	}
	return requestView{
		ID: r.ID, HostID: r.HostID, HostName: name, RequesterID: r.RequesterID,
		RequesterName: requester,
		Status:        r.Status, Note: r.Note, CreatedAt: r.CreatedAt.Format(time.RFC3339),
	}
}

func (a *app) handleCreateAccessRequest(w http.ResponseWriter, r *http.Request) {
	p, _ := rbac.FromContext(r.Context())
	h, ok := a.loadHost(w, r)
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
	req, err := a.db.CreateAccessRequest(h.ID, p.UserID, strings.TrimSpace(in.Note))
	if err == store.ErrDuplicateRequest {
		writeError(w, http.StatusConflict, "duplicate_request", "you already have an open request for this host")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not create request")
		return
	}
	// The in-app notification is the inbox; this reaches whoever watches the
	// host a request is against, plus every global channel.
	a.notifyEvent("", h.ID, eventAccessRequest, map[string]any{
		"event": eventAccessRequest, "type": eventAccessRequest, "host": h.Name, "host_id": h.ID,
		"requester": p.Email, "note": req.Note,
		"timestamp": req.CreatedAt.UTC().Format(time.RFC3339),
	}, time.Now().UTC())
	writeJSON(w, http.StatusCreated, a.toRequestView(req))
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
		out = append(out, a.toRequestView(req))
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
	if _, err := a.db.GetHost(req.HostID); err != nil {
		writeError(w, http.StatusNotFound, "not_found", "host not found")
		return
	}
	if !p.IsAdmin() {
		writeError(w, http.StatusForbidden, "forbidden", "only an admin can decide this")
		return
	}

	// Approval may target the requester (default) or a chosen group. The body
	// is optional, so only a body that is present and malformed is an error.
	ptype, pid := "user", req.RequesterID
	if status == store.RequestApproved && r.ContentLength > 0 {
		var in struct {
			PrincipalType string `json:"principal_type"`
			PrincipalID   string `json:"principal_id"`
		}
		if err := decodeJSON(r, &in); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		if in.PrincipalType == "group" && in.PrincipalID != "" {
			if !a.principalExists(w, "group", in.PrincipalID) {
				return
			}
			ptype, pid = "group", in.PrincipalID
		}
	}

	if err := a.db.DecideAccessRequest(req.ID, status, p.UserID); err != nil {
		writeError(w, http.StatusConflict, "already_decided", "request is no longer pending")
		return
	}
	if status == store.RequestApproved {
		a.db.GrantCredentialAccess(req.HostID, ptype, pid, p.UserID)
		a.db.RecordGrant(req.HostID, ptype, pid, "grant", p.UserID)
	}
	w.WriteHeader(http.StatusNoContent)
}

// Standing-grant management (independent of requests).

type accessGrantView struct {
	PrincipalType string `json:"principal_type"`
	PrincipalID   string `json:"principal_id"`
	GrantedBy     string `json:"granted_by"`
}

func (a *app) handleListHostAccess(w http.ResponseWriter, r *http.Request) {
	h, ok := a.loadHost(w, r)
	if !ok {
		return
	}
	grants, err := a.db.ListCredentialAccess(h.ID)
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

func (a *app) handleGrantHostAccess(w http.ResponseWriter, r *http.Request) {
	p, _ := rbac.FromContext(r.Context())
	h, ok := a.loadHost(w, r)
	if !ok {
		return
	}
	ptype, pid := r.PathValue("ptype"), r.PathValue("pid")
	if !validPrincipalType(w, ptype) || !a.principalExists(w, ptype, pid) {
		return
	}
	a.db.GrantCredentialAccess(h.ID, ptype, pid, p.UserID)
	a.db.RecordGrant(h.ID, ptype, pid, "grant", p.UserID)
	w.WriteHeader(http.StatusNoContent)
}

func (a *app) handleRevokeHostAccess(w http.ResponseWriter, r *http.Request) {
	p, _ := rbac.FromContext(r.Context())
	h, ok := a.loadHost(w, r)
	if !ok {
		return
	}
	ptype, pid := r.PathValue("ptype"), r.PathValue("pid")
	if !validPrincipalType(w, ptype) {
		return
	}
	a.db.RevokeCredentialAccess(h.ID, ptype, pid)
	a.db.RecordGrant(h.ID, ptype, pid, "revoke", p.UserID)
	w.WriteHeader(http.StatusNoContent)
}

// validPrincipalType answers 400 for anything but a user or a group. Every
// route that takes a {ptype} segment checks it, so a revoke cannot record an
// audit row naming a principal kind that does not exist.
func validPrincipalType(w http.ResponseWriter, ptype string) bool {
	if ptype == "user" || ptype == "group" {
		return true
	}
	writeError(w, http.StatusBadRequest, "invalid_principal", "principal type must be user or group")
	return false
}

// principalExists answers 404 for a grant aimed at nobody, so an access list
// cannot fill up with rows that can never match a caller.
func (a *app) principalExists(w http.ResponseWriter, ptype, pid string) bool {
	var err error
	if ptype == "user" {
		_, err = a.db.GetUserByID(pid)
	} else {
		_, err = a.db.GetGroup(pid)
	}
	if err != nil {
		writeError(w, http.StatusNotFound, "unknown_principal", "no such "+ptype)
		return false
	}
	return true
}

// pendingForApprover returns the pending requests an approver may act on.
// Credentials belong to hosts, and a host has no owner but an admin, so a
// non-admin has an empty inbox rather than one built from what they created.
func (a *app) pendingForApprover(p auth.Principal) ([]store.AccessRequest, error) {
	if p.IsAdmin() {
		return a.db.ListAllPendingRequests()
	}
	return nil, nil
}
