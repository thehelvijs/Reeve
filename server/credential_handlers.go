package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/thehelvijs/Reeve/server/internal/auth"
	"github.com/thehelvijs/Reeve/server/internal/rbac"
	"github.com/thehelvijs/Reeve/server/internal/store"
)

var credentialTypes = map[string]bool{
	"ssh_password": true, "ssh_key": true, "api_token": true, "db": true, "kv": true,
}

type credentialInput struct {
	Type   string            `json:"type"`
	Label  string            `json:"label"`
	Secret map[string]string `json:"secret"`
}

type credentialView struct {
	ID        string `json:"id"`
	ToolID    string `json:"tool_id"`
	Type      string `json:"type"`
	Label     string `json:"label"`
	CanReveal bool   `json:"can_reveal"`
}

func (a *app) handleListCredentials(w http.ResponseWriter, r *http.Request) {
	p, _ := rbac.FromContext(r.Context())
	t, ok := a.loadVisibleTool(w, r, p)
	if !ok {
		return
	}
	creds, err := a.db.ListCredentials(t.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not list credentials")
		return
	}
	canReveal := a.canReveal(t, p)
	out := make([]credentialView, 0, len(creds))
	for _, c := range creds {
		out = append(out, credentialView{ID: c.ID, ToolID: c.ToolID, Type: c.Type, Label: c.Label, CanReveal: canReveal})
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *app) handleCreateCredential(w http.ResponseWriter, r *http.Request) {
	p, _ := rbac.FromContext(r.Context())
	t, ok := a.loadEditableTool(w, r, p)
	if !ok {
		return
	}
	var in credentialInput
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if !credentialTypes[in.Type] {
		writeError(w, http.StatusBadRequest, "invalid_type", "unknown credential type")
		return
	}
	if len(in.Secret) == 0 {
		writeError(w, http.StatusBadRequest, "empty_secret", "secret fields are required")
		return
	}
	ct, nonce, err := a.sealSecret(in.Secret)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not encrypt secret")
		return
	}
	c, err := a.db.CreateCredential(t.ID, in.Type, in.Label, ct, nonce, p.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not store credential")
		return
	}
	writeJSON(w, http.StatusCreated, credentialView{ID: c.ID, ToolID: c.ToolID, Type: c.Type, Label: c.Label, CanReveal: true})
}

func (a *app) handleUpdateCredential(w http.ResponseWriter, r *http.Request) {
	p, _ := rbac.FromContext(r.Context())
	c, _, ok := a.loadManageableCredential(w, r, p)
	if !ok {
		return
	}
	var in credentialInput
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if len(in.Secret) == 0 {
		writeError(w, http.StatusBadRequest, "empty_secret", "secret fields are required")
		return
	}
	ct, nonce, err := a.sealSecret(in.Secret)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not encrypt secret")
		return
	}
	if err := a.db.UpdateCredential(c.ID, in.Label, ct, nonce); err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not update credential")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *app) handleDeleteCredential(w http.ResponseWriter, r *http.Request) {
	p, _ := rbac.FromContext(r.Context())
	c, _, ok := a.loadManageableCredential(w, r, p)
	if !ok {
		return
	}
	if err := a.db.DeleteCredential(c.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not delete credential")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *app) handleRevealCredential(w http.ResponseWriter, r *http.Request) {
	p, _ := rbac.FromContext(r.Context())
	cid := r.PathValue("cid")
	c, err := a.db.GetCredential(cid)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "credential not found")
		return
	}
	t, err := a.db.GetTool(c.ToolID)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "credential not found")
		return
	}
	// Hide existence from users who can't even see the tool.
	if canSee, _ := a.db.CanSeeTool(p.UserID, p.IsAdmin(), t.ID); !canSee {
		writeError(w, http.StatusNotFound, "not_found", "credential not found")
		return
	}
	if !a.canReveal(t, p) {
		writeError(w, http.StatusForbidden, "forbidden", "you do not have access to this credential")
		return
	}
	ct, nonce, err := a.db.GetCredentialSecret(cid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not read credential")
		return
	}
	plain, err := a.cipher.Open(ct, nonce)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not decrypt credential")
		return
	}
	var secret map[string]string
	if err := json.Unmarshal(plain, &secret); err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not decode credential")
		return
	}

	a.db.RecordReveal(c.ID, t.ID, p.UserID, a.clientIP(r))
	writeJSON(w, http.StatusOK, map[string]any{
		"id": c.ID, "type": c.Type, "label": c.Label, "secret": secret,
	})
}

// canReveal reports whether the principal may reveal a tool's credentials. A
// lookup that fails denies, and says so in the log: a silent false is
// indistinguishable from a real denial when someone is trying to explain one.
func (a *app) canReveal(t store.Tool, p auth.Principal) bool {
	if p.IsAdmin() || t.CreatorID == p.UserID {
		return true
	}
	ok, err := a.db.HasCredentialAccess(p.UserID, t.ID)
	if err != nil {
		log.Printf("credentials: access lookup for user %s on tool %s: %v", p.UserID, t.ID, err)
		return false
	}
	return ok
}

// loadManageableCredential fetches a credential and requires the principal to be
// the tool creator or an admin.
func (a *app) loadManageableCredential(w http.ResponseWriter, r *http.Request, p auth.Principal) (store.Credential, store.Tool, bool) {
	cid := r.PathValue("cid")
	c, err := a.db.GetCredential(cid)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "credential not found")
		return store.Credential{}, store.Tool{}, false
	}
	t, err := a.db.GetTool(c.ToolID)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "credential not found")
		return store.Credential{}, store.Tool{}, false
	}
	if canSee, _ := a.db.CanSeeTool(p.UserID, p.IsAdmin(), t.ID); !canSee {
		writeError(w, http.StatusNotFound, "not_found", "credential not found")
		return store.Credential{}, store.Tool{}, false
	}
	if !p.IsAdmin() && t.CreatorID != p.UserID {
		writeError(w, http.StatusForbidden, "forbidden", "only the creator or an admin can manage credentials")
		return store.Credential{}, store.Tool{}, false
	}
	return c, t, true
}

func (a *app) sealSecret(secret map[string]string) (ct, nonce []byte, err error) {
	plain, err := json.Marshal(secret)
	if err != nil {
		return nil, nil, err
	}
	return a.cipher.Seal(plain)
}
