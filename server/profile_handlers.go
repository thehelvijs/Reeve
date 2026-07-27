package main

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/thehelvijs/Reeve/server/internal/auth"
	"github.com/thehelvijs/Reeve/server/internal/rbac"
	"github.com/thehelvijs/Reeve/server/internal/store"
)

const maxDisplayNameLen = 60

// avatarDir returns the directory that holds avatar files, defaulting to an
// "avatars" folder beside the database file when unconfigured.
func (a *app) avatarDir() string {
	if a.cfg.AvatarDir != "" {
		return a.cfg.AvatarDir
	}
	return filepath.Join(filepath.Dir(a.cfg.DBPath), "avatars")
}

// handleUpdateMe changes the caller's own display name.
func (a *app) handleUpdateMe(w http.ResponseWriter, r *http.Request) {
	p, _ := rbac.FromContext(r.Context())
	var in struct {
		DisplayName *string `json:"display_name"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if in.DisplayName == nil {
		writeError(w, http.StatusBadRequest, "no_changes", "display_name is required")
		return
	}
	name := strings.TrimSpace(*in.DisplayName)
	if len(name) > maxDisplayNameLen {
		writeError(w, http.StatusBadRequest, "name_too_long", "display name must be at most 60 characters")
		return
	}
	if err := a.db.SetUserDisplayName(p.UserID, name); err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not update profile")
		return
	}
	a.writeMe(w, p.UserID)
}

// handleChangePassword verifies the current password before setting a new one.
func (a *app) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	p, _ := rbac.FromContext(r.Context())
	var in struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	u, err := a.db.GetUserByID(p.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not load account")
		return
	}
	if !auth.VerifyPassword(in.CurrentPassword, u.PasswordHash) {
		writeError(w, http.StatusBadRequest, "wrong_password", "current password is incorrect")
		return
	}
	if len(in.NewPassword) < minPasswordLen {
		writeError(w, http.StatusBadRequest, "weak_password", "password must be at least 8 characters")
		return
	}
	if err := setPassword(a.db, p.UserID, in.NewPassword); err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not update password")
		return
	}
	// Every session was just dropped, including this one; issue a fresh cookie
	// so the caller stays signed in and other devices do not.
	a.startSession(w, p.UserID)
	w.WriteHeader(http.StatusNoContent)
}

// handleUploadAvatar stores the caller's avatar image on disk.
func (a *app) handleUploadAvatar(w http.ResponseWriter, r *http.Request) {
	p, _ := rbac.FromContext(r.Context())
	data, ext, ok := readImageUpload(w, r, "avatar")
	if !ok {
		return
	}
	u, err := a.db.GetUserByID(p.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not load account")
		return
	}
	dest, err := writeImageFile(a.avatarDir(), p.UserID, ext, data, u.AvatarPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not store avatar")
		return
	}
	if err := a.db.SetUserAvatarPath(p.UserID, dest); err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not store avatar")
		return
	}
	a.writeMe(w, p.UserID)
}

// handleDeleteAvatar removes the caller's stored avatar.
func (a *app) handleDeleteAvatar(w http.ResponseWriter, r *http.Request) {
	p, _ := rbac.FromContext(r.Context())
	u, err := a.db.GetUserByID(p.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not load account")
		return
	}
	if u.AvatarPath != "" {
		os.Remove(u.AvatarPath)
		if err := a.db.SetUserAvatarPath(p.UserID, ""); err != nil {
			writeError(w, http.StatusInternalServerError, "internal", "could not update profile")
			return
		}
	}
	a.writeMe(w, p.UserID)
}

// handleGetAvatar serves a user's avatar image to any authenticated caller.
func (a *app) handleGetAvatar(w http.ResponseWriter, r *http.Request) {
	u, err := a.db.GetUserByID(r.PathValue("id"))
	if err != nil || u.AvatarPath == "" {
		writeError(w, http.StatusNotFound, "not_found", "no avatar")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	http.ServeFile(w, r, u.AvatarPath)
}

// handleDeleteMe deletes the caller's own account. The last admin cannot delete
// themselves; any other account's owned tools reassign to the oldest admin.
func (a *app) handleDeleteMe(w http.ResponseWriter, r *http.Request) {
	p, _ := rbac.FromContext(r.Context())
	if p.Role == store.RoleAdmin {
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
	if err := a.deleteUser(p.UserID); err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not delete account")
		return
	}
	if c, err := r.Cookie(sessionCookie); err == nil {
		a.db.DeleteSession(auth.HashToken(c.Value))
	}
	a.clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

// deleteUser reassigns the user's owned resources to the oldest other admin and
// removes their account and avatar file.
func (a *app) deleteUser(id string) error {
	heir, err := a.db.OldestAdminExcluding(id)
	if err != nil {
		return err
	}
	u, err := a.db.GetUserByID(id)
	if err != nil {
		return err
	}
	if err := a.db.DeleteUser(id, heir.ID); err != nil {
		return err
	}
	if u.AvatarPath != "" {
		os.Remove(u.AvatarPath)
	}
	return nil
}

func (a *app) writeMe(w http.ResponseWriter, id string) {
	u, err := a.db.GetUserByID(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not load account")
		return
	}
	writeJSON(w, http.StatusOK, a.userView(u))
}

// formFile pulls a single uploaded file, translating a missing part into a
// clean error rather than the multipart package's opaque one.
func formFile(r *http.Request, field string) (io.ReadCloser, string, error) {
	f, hdr, err := r.FormFile(field)
	if err != nil {
		return nil, "", errors.New("missing file")
	}
	return f, hdr.Filename, nil
}
