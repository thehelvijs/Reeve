package main

import (
	"net/http"
	"os"

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
		avatar = "/api/v1/users/" + u.ID + "/avatar"
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
