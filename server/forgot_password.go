package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/thehelvijs/Reeve/server/internal/mail"
)

// resetTokenTTL keeps a mailed reset link short-lived.
const resetTokenTTL = time.Hour

// newResetToken returns a URL-safe token and the hash to store for it.
func newResetToken() (token, hash string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	token = hex.EncodeToString(b)
	return token, hashResetToken(token), nil
}

func hashResetToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// baseURL is the origin to put in emailed links: the configured public URL when
// set, otherwise whatever host the request arrived on.
func (a *app) baseURL(r *http.Request) string {
	if a.cfg.PublicURL != "" {
		return strings.TrimSuffix(a.cfg.PublicURL, "/")
	}
	return requestOrigin(r)
}

// requestOrigin is the address this request was sent to, as the caller spelled
// it. That is the address that host can reach the server on, which is why it is
// the default when nobody configured a public URL.
func requestOrigin(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}

// publicBase is the origin for work with no request of its own to read, such as
// an alert the dispatcher sends: the configured public URL, else the last
// address an admin's browser actually reached this server on. Empty until one of
// the two is known, and a message with no link beats one pointing at localhost.
func (a *app) publicBase() string {
	if a.cfg.PublicURL != "" {
		return strings.TrimSuffix(a.cfg.PublicURL, "/")
	}
	origin, _ := a.seenOrigin.Load().(string)
	return origin
}

// reachableFromOtherHosts reports whether an agent on another machine could dial
// base. A loopback origin means the admin reached the UI over localhost, so the
// resolved URL is only valid inside this box.
func reachableFromOtherHosts(base string) bool {
	u, err := url.Parse(base)
	if err != nil {
		return false
	}
	host := u.Hostname()
	if host == "" || host == "localhost" {
		return false
	}
	if ip := net.ParseIP(host); ip != nil {
		return !ip.IsLoopback() && !ip.IsUnspecified()
	}
	return true
}

// handleForgotPassword mails a reset link. It answers 204 whether or not the
// address exists, so the endpoint cannot be used to enumerate accounts.
func (a *app) handleForgotPassword(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email string `json:"email"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	email := strings.TrimSpace(strings.ToLower(in.Email))

	ipKey := "forgot-ip:" + a.clientIP(r)
	if a.throttled(w, ipKey) {
		return
	}
	a.loginThrottle().Fail(ipKey)

	w.WriteHeader(http.StatusNoContent)

	u, err := a.db.GetUserByEmail(email)
	if err != nil || !u.Active {
		return
	}
	if _, ok := a.mailConfig(); !ok {
		log.Printf("password reset requested for %s but no mail relay is configured", email)
		return
	}
	token, hash, err := newResetToken()
	if err != nil {
		log.Printf("password reset: token: %v", err)
		return
	}
	now := time.Now().UTC()
	if _, err := a.db.CreatePasswordReset(u.ID, hash, now.Add(resetTokenTTL), now); err != nil {
		log.Printf("password reset: store: %v", err)
		return
	}
	link := a.baseURL(r) + "/reset?token=" + url.QueryEscape(token)
	body := "Someone asked to reset the Reeve password for " + u.Email + ".\n\n" +
		"Open this link within the hour to choose a new one:\n\n" + link + "\n\n" +
		"If that wasn't you, ignore this message; the password is unchanged.\n"
	if err := a.sendMail(mail.Message{To: u.Email, Subject: "Reset your Reeve password", Body: body}); err != nil {
		log.Printf("password reset: send to %s: %v", u.Email, err)
	}
}

// handleResetPassword consumes a mailed token and sets the new password.
func (a *app) handleResetPassword(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	ipKey := "reset-ip:" + a.clientIP(r)
	if a.throttled(w, ipKey) {
		return
	}
	if len(in.Password) < minPasswordLen {
		writeError(w, http.StatusBadRequest, "weak_password", "password must be at least 8 characters")
		return
	}

	now := time.Now().UTC()
	pr, err := a.db.GetPasswordResetByHash(hashResetToken(in.Token), now)
	if err != nil {
		a.loginThrottle().Fail(ipKey)
		writeError(w, http.StatusBadRequest, "invalid_token", "that reset link is invalid or has expired")
		return
	}
	u, err := a.db.GetUserByID(pr.UserID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_token", "that reset link is invalid or has expired")
		return
	}
	if err := setPassword(a.db, u.ID, in.Password); err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not set password")
		return
	}
	if err := a.db.UsePasswordReset(pr.ID, u.ID, now); err != nil {
		log.Printf("password reset: mark used: %v", err)
	}
	a.loginThrottle().Reset("login-email:" + u.Email)
	a.loginThrottle().Reset(ipKey)
	w.WriteHeader(http.StatusNoContent)
}
