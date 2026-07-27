package main

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/thehelvijs/Reeve/server/internal/auth"
	"github.com/thehelvijs/Reeve/server/internal/rbac"
	"github.com/thehelvijs/Reeve/server/internal/store"
)

const settingSignupEnabled = "signup_enabled"
const minPasswordLen = 8

type credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type userView struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	Role        string `json:"role"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url"`
}

// userView renders a store user as the API DTO, deriving the avatar URL from
// whether a file is stored (never leaking the filesystem path).
func (a *app) userView(u store.User) userView {
	avatar := ""
	if u.AvatarPath != "" {
		avatar = "/api/users/" + u.ID + "/avatar"
	}
	return userView{ID: u.ID, Email: u.Email, Role: u.Role, DisplayName: u.DisplayName, AvatarURL: avatar}
}

// handleAuthStatus reports whether first-run admin setup is needed and whether
// signup is currently open. Public (drives the login/setup screens).
func (a *app) handleAuthStatus(w http.ResponseWriter, _ *http.Request) {
	count, err := a.db.CountUsers()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not read users")
		return
	}
	_, mailReady := a.mailConfig()
	_, googleReady := a.googleConfig()
	writeJSON(w, http.StatusOK, map[string]bool{
		"setup_required":         count == 0,
		"signup_enabled":         a.db.GetBoolSetting(settingSignupEnabled, true),
		"password_reset_enabled": mailReady,
		"google_enabled":         googleReady,
	})
}

func (a *app) handleSignup(w http.ResponseWriter, r *http.Request) {
	var in credentials
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	in.Email = strings.TrimSpace(strings.ToLower(in.Email))
	if !strings.Contains(in.Email, "@") || in.Email == "" {
		writeError(w, http.StatusBadRequest, "invalid_email", "a valid email is required")
		return
	}
	if len(in.Password) < minPasswordLen {
		writeError(w, http.StatusBadRequest, "weak_password", "password must be at least 8 characters")
		return
	}
	// Open signup is a write endpoint anyone can reach, so it gets the same
	// per-source ceiling as login.
	signupKey := "signup-ip:" + a.clientIP(r)
	if a.throttled(w, signupKey) {
		return
	}
	a.loginThrottle().Fail(signupKey)

	count, err := a.db.CountUsers()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not read users")
		return
	}
	// The first account bootstraps an admin and is always allowed; afterward,
	// signup honors the admin-controlled toggle.
	role := store.RoleBasic
	if count == 0 {
		role = store.RoleAdmin
	} else if !a.db.GetBoolSetting(settingSignupEnabled, true) {
		writeError(w, http.StatusForbidden, "signup_disabled", "sign-up is disabled")
		return
	}

	if _, err := a.db.GetUserByEmail(in.Email); err == nil {
		writeError(w, http.StatusConflict, "email_taken", "an account with that email already exists")
		return
	}

	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not hash password")
		return
	}
	u, err := a.db.CreateUser(in.Email, hash, role)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not create user")
		return
	}
	a.loginThrottle().Reset(signupKey)
	a.startSession(w, u.ID)
	writeJSON(w, http.StatusCreated, a.userView(u))
}

func (a *app) handleLogin(w http.ResponseWriter, r *http.Request) {
	var in credentials
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	in.Email = strings.TrimSpace(strings.ToLower(in.Email))

	keys := []string{"login-ip:" + a.clientIP(r), "login-email:" + in.Email}
	if a.throttled(w, keys...) {
		return
	}

	u, err := a.db.GetUserByEmail(in.Email)
	if err != nil || !u.Active || !auth.VerifyPassword(in.Password, u.PasswordHash) {
		for _, k := range keys {
			a.loginThrottle().Fail(k)
		}
		writeError(w, http.StatusUnauthorized, "invalid_credentials", auth.ErrInvalidCredentials.Error())
		return
	}
	for _, k := range keys {
		a.loginThrottle().Reset(k)
	}
	a.startSession(w, u.ID)
	writeJSON(w, http.StatusOK, a.userView(u))
}

// throttled reports whether any key is locked out, answering 429 with a
// Retry-After when it is. Both the source IP and the target email are keyed so
// neither a single host nor a single account can be ground down.
func (a *app) throttled(w http.ResponseWriter, keys ...string) bool {
	for _, k := range keys {
		retry, locked := a.loginThrottle().RetryAfter(k)
		if !locked {
			continue
		}
		secs := int(math.Ceil(retry.Seconds()))
		w.Header().Set("Retry-After", strconv.Itoa(secs))
		writeError(w, http.StatusTooManyRequests, "too_many_attempts",
			fmt.Sprintf("too many failed attempts, try again in %ds", secs))
		return true
	}
	return false
}

func (a *app) handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil {
		a.db.DeleteSession(auth.HashToken(c.Value))
	}
	a.clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (a *app) handleMe(w http.ResponseWriter, r *http.Request) {
	p, _ := rbac.FromContext(r.Context())
	u, err := a.db.GetUserByID(p.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not load account")
		return
	}
	writeJSON(w, http.StatusOK, a.userView(u))
}

func (a *app) startSession(w http.ResponseWriter, userID string) {
	token, hash := auth.NewSessionToken()
	sess, err := a.db.CreateSession(userID, hash, a.cfg.SessionTTL)
	if err != nil {
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   a.cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
		Expires:  sess.ExpiresAt,
	})
}

func (a *app) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   a.cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}
