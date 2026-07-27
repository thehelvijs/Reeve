package main

import (
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/thehelvijs/Reeve/server/internal/rbac"
	"github.com/thehelvijs/Reeve/server/internal/store"
)

type adminUserView struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	Role        string `json:"role"`
	Active      bool   `json:"active"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url"`
}

func (a *app) adminUserView(u store.User) adminUserView {
	avatar := ""
	if u.AvatarPath != "" {
		avatar = "/api/users/" + u.ID + "/avatar"
	}
	return adminUserView{ID: u.ID, Email: u.Email, Role: u.Role, Active: u.Active, DisplayName: u.DisplayName, AvatarURL: avatar}
}

func (a *app) handleListUsers(w http.ResponseWriter, _ *http.Request) {
	users, err := a.db.ListUsers()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not list users")
		return
	}
	out := make([]adminUserView, 0, len(users))
	for _, u := range users {
		out = append(out, a.adminUserView(u))
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *app) handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	actor, _ := rbac.FromContext(r.Context())
	id := r.PathValue("id")
	var in struct {
		Role   *string `json:"role"`
		Active *bool   `json:"active"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if in.Role == nil && in.Active == nil {
		writeError(w, http.StatusBadRequest, "no_changes", "role or active is required")
		return
	}
	// An admin cannot lock themselves out of their own account.
	if id == actor.UserID {
		if (in.Role != nil && *in.Role != store.RoleAdmin) || (in.Active != nil && !*in.Active) {
			writeError(w, http.StatusBadRequest, "self_lockout", "cannot demote or deactivate your own account")
			return
		}
	}
	if in.Role != nil {
		if *in.Role != store.RoleAdmin && *in.Role != store.RoleBasic {
			writeError(w, http.StatusBadRequest, "invalid_role", "role must be admin or basic")
			return
		}
		if err := a.db.SetUserRole(id, *in.Role); err != nil {
			a.notFoundOrInternal(w, err)
			return
		}
	}
	if in.Active != nil {
		if err := a.db.SetUserActive(id, *in.Active); err != nil {
			a.notFoundOrInternal(w, err)
			return
		}
	}
	u, err := a.db.GetUserByID(id)
	if err != nil {
		a.notFoundOrInternal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, a.adminUserView(u))
}

// handleDeleteUser lets an admin hard-delete another account. Admins delete
// their own account through the profile endpoint, and the last admin is
// protected there; here we block self-deletion and deleting the last admin.
func (a *app) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	actor, _ := rbac.FromContext(r.Context())
	id := r.PathValue("id")
	if id == actor.UserID {
		writeError(w, http.StatusBadRequest, "self_delete", "delete your own account from your profile")
		return
	}
	target, err := a.db.GetUserByID(id)
	if err != nil {
		a.notFoundOrInternal(w, err)
		return
	}
	if target.Role == store.RoleAdmin {
		n, err := a.db.CountAdmins()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal", "could not read users")
			return
		}
		if n <= 1 {
			writeError(w, http.StatusBadRequest, "last_admin", "cannot delete the only admin account")
			return
		}
	}
	if err := a.db.DeleteUser(id, actor.UserID); err != nil {
		a.notFoundOrInternal(w, err)
		return
	}
	if target.AvatarPath != "" {
		os.Remove(target.AvatarPath)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *app) notFoundOrInternal(w http.ResponseWriter, err error) {
	if err == store.ErrNotFound {
		writeError(w, http.StatusNotFound, "not_found", "user not found")
		return
	}
	writeError(w, http.StatusInternalServerError, "internal", "operation failed")
}

func (a *app) auditFilter(r *http.Request) store.AuditFilter {
	q := r.URL.Query()
	return store.AuditFilter{
		UserID: q.Get("user"),
		HostID: q.Get("host"),
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

// handleVerifyAudit recomputes both audit chains on demand. It is a read-only
// check an admin can run before trusting what the audit says.
func (a *app) handleVerifyAudit(w http.ResponseWriter, _ *http.Request) {
	results, err := a.db.VerifyAuditChain()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not verify audit")
		return
	}
	ok := true
	for _, r := range results {
		if !r.OK {
			ok = false
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": ok, "tables": results})
}

func (a *app) handleGrantAudit(w http.ResponseWriter, r *http.Request) {
	events, err := a.db.ListGrantAudit(a.auditFilter(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not read audit")
		return
	}
	writeJSON(w, http.StatusOK, events)
}

func (a *app) handleServerInfo(w http.ResponseWriter, _ *http.Request) {
	tools, _ := a.db.CountTools()
	hosts, _ := a.db.CountHosts()
	users, _ := a.db.CountUsers()

	var dbSize int64
	if fi, err := os.Stat(a.cfg.DBPath); err == nil {
		dbSize = fi.Size()
	}

	now := time.Now().UTC()
	writeJSON(w, http.StatusOK, map[string]any{
		"version":     a.cfg.Version,
		"started_at":  a.startedAt.Format(time.RFC3339),
		"uptime_secs": int64(now.Sub(a.startedAt).Seconds()),
		"go": map[string]any{
			"version":       runtime.Version(),
			"os":            runtime.GOOS,
			"arch":          runtime.GOARCH,
			"num_cpu":       runtime.NumCPU(),
			"num_goroutine": runtime.NumGoroutine(),
		},
		"db": map[string]any{
			"path":           a.cfg.DBPath,
			"size_bytes":     dbSize,
			"restore_staged": a.restoreStaged(),
		},
		"counts": map[string]any{
			"tools": tools,
			"hosts": hosts,
			"users": users,
		},
		"host": a.sampleHost(),
	})
}
