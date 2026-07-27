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
	HostID    string `json:"host_id"`
	Type      string `json:"type"`
	Label     string `json:"label"`
	CanReveal bool   `json:"can_reveal"`
}

// loadHost resolves the {id} path segment. Every signed-in user may see that a
// host exists, so this needs no visibility check; revealing a secret is what is
// gated, not knowing the machine is there.
func (a *app) loadHost(w http.ResponseWriter, r *http.Request) (store.Host, bool) {
	h, err := a.db.GetHost(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "host not found")
		return store.Host{}, false
	}
	return h, true
}

func (a *app) handleListCredentials(w http.ResponseWriter, r *http.Request) {
	p, _ := rbac.FromContext(r.Context())
	h, ok := a.loadHost(w, r)
	if !ok {
		return
	}
	creds, err := a.db.ListCredentials(h.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not list credentials")
		return
	}
	canReveal := a.canReveal(h.ID, p)
	out := make([]credentialView, 0, len(creds))
	for _, c := range creds {
		out = append(out, credentialView{ID: c.ID, HostID: c.HostID, Type: c.Type, Label: c.Label, CanReveal: canReveal})
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *app) handleCreateCredential(w http.ResponseWriter, r *http.Request) {
	p, _ := rbac.FromContext(r.Context())
	h, ok := a.loadHost(w, r)
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
	ct, nonce, err := a.sealSecret(h.ID, in.Secret)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not encrypt secret")
		return
	}
	c, err := a.db.CreateCredential(h.ID, in.Type, in.Label, ct, nonce, p.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not store credential")
		return
	}
	writeJSON(w, http.StatusCreated, credentialView{ID: c.ID, HostID: c.HostID, Type: c.Type, Label: c.Label, CanReveal: true})
}

func (a *app) handleUpdateCredential(w http.ResponseWriter, r *http.Request) {
	c, ok := a.loadCredential(w, r)
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
	ct, nonce, err := a.sealSecret(c.HostID, in.Secret)
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
	c, ok := a.loadCredential(w, r)
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
	if !a.canReveal(c.HostID, p) {
		writeError(w, http.StatusForbidden, "forbidden", "you do not have access to this credential")
		return
	}
	ct, nonce, err := a.db.GetCredentialSecret(cid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not read credential")
		return
	}
	plain, err := a.cipher.Open(ct, nonce, credentialAAD(c.HostID))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not decrypt credential")
		return
	}
	var secret map[string]string
	if err := json.Unmarshal(plain, &secret); err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not decode credential")
		return
	}

	a.db.RecordReveal(c.ID, c.HostID, p.UserID, a.clientIP(r))
	writeJSON(w, http.StatusOK, map[string]any{
		"id": c.ID, "type": c.Type, "label": c.Label, "secret": secret,
	})
}

// canReveal reports whether the principal may reveal a host's credentials. A
// host has no creator to inherit access from, unlike a tool, so it is an admin
// or an explicit standing grant and nothing else. A lookup that fails denies,
// and says so in the log: a silent false is indistinguishable from a real
// denial when someone is trying to explain one.
func (a *app) canReveal(hostID string, p auth.Principal) bool {
	if p.IsAdmin() {
		return true
	}
	ok, err := a.db.HasCredentialAccess(p.UserID, hostID)
	if err != nil {
		log.Printf("credentials: access lookup for user %s on host %s: %v", p.UserID, hostID, err)
		return false
	}
	return ok
}

// loadCredential fetches a credential by id. Callers sit behind the admin
// middleware, which is the whole authorization check now that credentials
// belong to admin-managed hosts.
func (a *app) loadCredential(w http.ResponseWriter, r *http.Request) (store.Credential, bool) {
	c, err := a.db.GetCredential(r.PathValue("cid"))
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "credential not found")
		return store.Credential{}, false
	}
	return c, true
}

// credentialAAD binds a sealed secret to the host it opens, so a row's
// ciphertext cannot be moved to a machine it was never meant for. Two
// credentials on the same host remain interchangeable to a database writer,
// which changes a label and nothing about who may reveal what.
func credentialAAD(hostID string) []byte {
	return []byte("reeve/credential/" + hostID)
}

func (a *app) sealSecret(hostID string, secret map[string]string) (ct, nonce []byte, err error) {
	plain, err := json.Marshal(secret)
	if err != nil {
		return nil, nil, err
	}
	return a.cipher.Seal(plain, credentialAAD(hostID))
}
