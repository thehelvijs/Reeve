package main

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/thehelvijs/Reeve/server/internal/auth"
	"github.com/thehelvijs/Reeve/server/internal/rbac"
	"github.com/thehelvijs/Reeve/server/internal/store"
)

// setPassword hashes password for the user and signs every existing session of
// theirs out, so a stolen or shared session cannot outlive the change. Every
// path that sets a password routes through here, so the length floor lives here
// rather than in each caller.
func setPassword(db *store.DB, userID, password string) error {
	if len(password) < minPasswordLen {
		return fmt.Errorf("password must be at least %d characters", minPasswordLen)
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}
	if err := db.SetUserPassword(userID, hash); err != nil {
		return err
	}
	return db.DeleteSessionsForUser(userID)
}

// resetPasswordCLI is the out-of-band recovery path for a locked-out admin:
// `server -reset-password you@example.com -password hunter2`. It runs before
// the HTTP server starts and needs no master key.
func resetPasswordCLI(dbPath, email, password string) error {
	email = strings.TrimSpace(strings.ToLower(email))
	if password == "" {
		return errors.New("-reset-password also needs -password <new password>")
	}
	db, err := store.Open(dbPath)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	u, err := db.GetUserByEmail(email)
	if err != nil {
		return fmt.Errorf("no account with email %q", email)
	}
	if err := setPassword(db, u.ID, password); err != nil {
		return fmt.Errorf("set password: %w", err)
	}
	fmt.Printf("password for %s reset; all of that account's sessions were signed out\n", u.Email)
	return nil
}

// handleAdminResetPassword lets an admin set another account's password when a
// user is locked out and no mail server is configured.
func (a *app) handleAdminResetPassword(w http.ResponseWriter, r *http.Request) {
	actor, _ := rbac.FromContext(r.Context())
	id := r.PathValue("id")
	var in struct {
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if id == actor.UserID {
		writeError(w, http.StatusBadRequest, "self_reset", "change your own password from your profile")
		return
	}
	if len(in.Password) < minPasswordLen {
		writeError(w, http.StatusBadRequest, "weak_password", "password must be at least 8 characters")
		return
	}
	u, err := a.db.GetUserByID(id)
	if err != nil {
		a.notFoundOrInternal(w, err)
		return
	}
	if err := setPassword(a.db, u.ID, in.Password); err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not set password")
		return
	}
	// Clear the email-keyed lockout so the user can sign in immediately.
	a.loginThrottle().Reset("login-email:" + u.Email)
	writeJSON(w, http.StatusOK, map[string]string{"email": u.Email})
}
