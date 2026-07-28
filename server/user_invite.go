package main

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/thehelvijs/Reeve/server/internal/auth"
	"github.com/thehelvijs/Reeve/server/internal/mail"
	"github.com/thehelvijs/Reeve/server/internal/store"
)

// inviteTokenTTL is generous where a password reset is not: an invite sits in an
// inbox until the person gets to it, and nobody asked for it a minute ago.
const inviteTokenTTL = 7 * 24 * time.Hour

// inviteView answers what the admin needs to know next: the account exists, and
// either it was mailed or here is the link to hand over.
type inviteView struct {
	User       adminUserView `json:"user"`
	Emailed    bool          `json:"emailed"`
	InviteLink string        `json:"invite_link,omitempty"`
}

// handleInviteUser creates an account nobody can sign into yet and issues the
// link that sets its first password. The password hash is over random bytes that
// are never stored or shown, so the invite link (or a federated sign-in on the
// same address) is the only way in.
func (a *app) handleInviteUser(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email       string `json:"email"`
		Role        string `json:"role"`
		DisplayName string `json:"display_name"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	in.Email = strings.TrimSpace(strings.ToLower(in.Email))
	if in.Email == "" || !strings.Contains(in.Email, "@") {
		writeError(w, http.StatusBadRequest, "invalid_email", "a valid email is required")
		return
	}
	if in.Role == "" {
		in.Role = store.RoleBasic
	}
	if in.Role != store.RoleAdmin && in.Role != store.RoleBasic {
		writeError(w, http.StatusBadRequest, "invalid_role", "role must be admin or basic")
		return
	}
	if _, err := a.db.GetUserByEmail(in.Email); err == nil {
		writeError(w, http.StatusConflict, "email_taken", "an account with that email already exists")
		return
	}

	unusable := make([]byte, 32)
	if _, err := rand.Read(unusable); err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not create account")
		return
	}
	hash, err := auth.HashPassword(hex.EncodeToString(unusable))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not create account")
		return
	}
	u, err := a.db.CreateUser(in.Email, hash, in.Role)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not create account")
		return
	}
	if name := strings.TrimSpace(in.DisplayName); name != "" {
		if err := a.db.SetUserDisplayName(u.ID, name); err != nil {
			log.Printf("invite: display name for %s: %v", u.Email, err)
		} else {
			u.DisplayName = name
		}
	}

	token, tokenHash, err := newResetToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not create invite")
		return
	}
	now := time.Now().UTC()
	if _, err := a.db.CreatePasswordReset(u.ID, tokenHash, now.Add(inviteTokenTTL), now); err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not create invite")
		return
	}
	link := a.baseURL(r) + "/reset?token=" + url.QueryEscape(token)

	out := inviteView{User: a.adminUserView(u)}
	if _, ok := a.mailConfig(); ok {
		body := "You have an account on Reeve, the infrastructure catalog at " + a.baseURL(r) + ".\n\n" +
			"Open this link within seven days to choose your password:\n\n" + link + "\n\n" +
			"Your sign-in address is " + u.Email + ".\n"
		if err := a.sendMail(mail.Message{To: u.Email, Subject: "Your Reeve account", Body: body}); err != nil {
			log.Printf("invite: send to %s: %v", u.Email, err)
		} else {
			out.Emailed = true
		}
	}
	// No relay, or the relay refused it: the admin passes the link on themselves,
	// otherwise the account they just made is one nobody can reach.
	if !out.Emailed {
		out.InviteLink = link
	}
	writeJSON(w, http.StatusCreated, out)
}
