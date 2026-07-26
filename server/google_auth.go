package main

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/thehelvijs/Reeve/server/internal/auth"
	"github.com/thehelvijs/Reeve/server/internal/oauth"
	"github.com/thehelvijs/Reeve/server/internal/store"
)

// Settings keys for Google sign-in. The client secret is sealed with the master
// key; the rest are readable config.
const (
	settingGoogleEnabled  = "oauth.google.enabled"
	settingGoogleClientID = "oauth.google.client_id"
	settingGoogleSecret   = "oauth.google.client_secret"
	settingGoogleDomains  = "oauth.google.allowed_domains"
)

// oauthStateCookie carries the CSRF state across the redirect to Google.
const oauthStateCookie = "reeve_oauth_state"

// oauthStateTTL is how long a started sign-in may take to come back.
const oauthStateTTL = 10 * time.Minute

// googleCallbackPath must match the redirect URI registered with Google.
const googleCallbackPath = "/api/auth/google/callback"

type googleAuthView struct {
	Enabled        bool   `json:"enabled"`
	ClientID       string `json:"client_id"`
	AllowedDomains string `json:"allowed_domains"`
	SecretSet      bool   `json:"secret_set"`
	RedirectURL    string `json:"redirect_url"`
}

// googleInput is the write shape; an empty client_secret keeps the stored one.
type googleInput struct {
	Enabled        bool   `json:"enabled"`
	ClientID       string `json:"client_id"`
	ClientSecret   string `json:"client_secret"`
	AllowedDomains string `json:"allowed_domains"`
}

func (a *app) googleAuthView() googleAuthView {
	_, hasSecret := a.sealedSetting(settingGoogleSecret)
	return googleAuthView{
		Enabled:        a.db.GetBoolSetting(settingGoogleEnabled, false),
		ClientID:       a.settingOr(settingGoogleClientID, ""),
		AllowedDomains: a.settingOr(settingGoogleDomains, ""),
		SecretSet:      hasSecret,
		RedirectURL:    a.googleRedirectURL(),
	}
}

// saveGoogleSettings validates and persists the OAuth client. An enabled client
// must be complete so the login screen never offers a button that cannot work.
func (a *app) saveGoogleSettings(in googleInput) error {
	in.ClientID = strings.TrimSpace(in.ClientID)
	if in.Enabled {
		secret := in.ClientSecret
		if secret == "" {
			secret, _ = a.sealedSetting(settingGoogleSecret)
		}
		cfg := oauth.Google(in.ClientID, secret, "placeholder")
		if err := cfg.Validate(); err != nil {
			return err
		}
	}
	writes := map[string]string{
		settingGoogleEnabled:  strconv.FormatBool(in.Enabled),
		settingGoogleClientID: in.ClientID,
		settingGoogleDomains:  strings.Join(oauth.ParseDomains(in.AllowedDomains), ","),
	}
	for k, v := range writes {
		if err := a.db.SetSetting(k, v); err != nil {
			return err
		}
	}
	if in.ClientSecret != "" {
		return a.setSealedSetting(settingGoogleSecret, in.ClientSecret)
	}
	return nil
}

// googleConfig builds the OAuth client, reporting false when Google sign-in is
// off or incompletely configured. The redirect URI comes from the configured
// public URL, never from the request host: Google matches it exactly against
// what the operator registered, so it has to be stable.
func (a *app) googleConfig() (oauth.Config, bool) {
	if !a.db.GetBoolSetting(settingGoogleEnabled, false) {
		return oauth.Config{}, false
	}
	if a.cfg.PublicURL == "" {
		return oauth.Config{}, false
	}
	secret, _ := a.sealedSetting(settingGoogleSecret)
	cfg := oauth.Google(a.settingOr(settingGoogleClientID, ""), secret, a.googleRedirectURL())
	if e := a.googleEndpoints; e != nil {
		cfg.AuthURL, cfg.TokenURL, cfg.UserInfoURL = e.Auth, e.Token, e.UserInfo
	}
	if cfg.Validate() != nil {
		return oauth.Config{}, false
	}
	return cfg, true
}

// googleRedirectURL is the callback an operator registers with Google.
func (a *app) googleRedirectURL() string {
	if a.cfg.PublicURL == "" {
		return ""
	}
	return strings.TrimSuffix(a.cfg.PublicURL, "/") + googleCallbackPath
}

// handleGoogleStart redirects to Google, pinning a random state in a cookie.
func (a *app) handleGoogleStart(w http.ResponseWriter, r *http.Request) {
	cfg, ok := a.googleConfig()
	if !ok {
		a.redirectLoginError(w, r, "google_not_configured")
		return
	}
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		a.redirectLoginError(w, r, "state_failed")
		return
	}
	state := hex.EncodeToString(b)
	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookie,
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		Secure:   a.cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(oauthStateTTL.Seconds()),
	})
	http.Redirect(w, r, cfg.AuthCodeURL(state), http.StatusFound)
}

// handleGoogleCallback completes the flow: verify state, exchange the code,
// then sign in (provisioning an account when the instance allows it).
func (a *app) handleGoogleCallback(w http.ResponseWriter, r *http.Request) {
	cfg, ok := a.googleConfig()
	if !ok {
		a.redirectLoginError(w, r, "google_not_configured")
		return
	}
	a.clearOAuthStateCookie(w)

	cookie, err := r.Cookie(oauthStateCookie)
	state := r.URL.Query().Get("state")
	if err != nil || cookie.Value == "" || state == "" || cookie.Value != state {
		a.redirectLoginError(w, r, "state_mismatch")
		return
	}
	if provErr := r.URL.Query().Get("error"); provErr != "" {
		a.redirectLoginError(w, r, "provider_denied")
		return
	}

	token, err := cfg.Exchange(r.Context(), r.URL.Query().Get("code"))
	if err != nil {
		log.Printf("google sign-in: %v", err)
		a.redirectLoginError(w, r, "exchange_failed")
		return
	}
	profile, err := cfg.UserInfo(r.Context(), token)
	if err != nil {
		log.Printf("google sign-in: %v", err)
		a.redirectLoginError(w, r, "profile_failed")
		return
	}

	domains := oauth.ParseDomains(a.settingOr(settingGoogleDomains, ""))
	if !oauth.DomainAllowed(profile.Email, domains) {
		a.redirectLoginError(w, r, "domain_not_allowed")
		return
	}

	u, err := a.db.GetUserByEmail(profile.Email)
	if err == nil {
		if !u.Active {
			a.redirectLoginError(w, r, "account_disabled")
			return
		}
		if !a.linkOAuthSubject(u, profile.Subject) {
			a.redirectLoginError(w, r, "subject_mismatch")
			return
		}
		a.startSession(w, u.ID)
		http.Redirect(w, r, "/dashboard", http.StatusFound)
		return
	}

	// Provisioning is allowed when signup is open, or when the admin narrowed
	// sign-in to specific domains — that allow-list is the intent to let them in.
	if !a.db.GetBoolSetting(settingSignupEnabled, true) && len(domains) == 0 {
		a.redirectLoginError(w, r, "no_account")
		return
	}
	u, err = a.provisionOAuthUser(profile)
	if err != nil {
		log.Printf("google sign-in: provision %s: %v", profile.Email, err)
		a.redirectLoginError(w, r, "provision_failed")
		return
	}
	a.startSession(w, u.ID)
	http.Redirect(w, r, "/dashboard", http.StatusFound)
}

// linkOAuthSubject ties an account to the provider's subject id on its first
// federated sign-in and, from then on, requires every later sign-in to present
// the same one. Matching on the email alone would hand the account to whoever
// holds that address next, which the domain's owner decides, not this server.
func (a *app) linkOAuthSubject(u store.User, subject string) bool {
	if subject == "" {
		return false
	}
	if u.OAuthSubject == subject {
		return true
	}
	if u.OAuthSubject != "" {
		log.Printf("google sign-in: %s presented subject %s but the account is bound to another", u.Email, subject)
		return false
	}
	if err := a.db.SetUserOAuthSubject(u.ID, subject); err != nil {
		log.Printf("google sign-in: bind subject for %s: %v", u.Email, err)
		return false
	}
	return true
}

// provisionOAuthUser creates an account for a federated identity. Its password
// hash is over unguessable random bytes, so the account has no usable password
// until someone sets one.
func (a *app) provisionOAuthUser(p oauth.Profile) (store.User, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return store.User{}, err
	}
	hash, err := auth.HashPassword(hex.EncodeToString(b))
	if err != nil {
		return store.User{}, err
	}
	count, err := a.db.CountUsers()
	if err != nil {
		return store.User{}, err
	}
	role := store.RoleBasic
	if count == 0 {
		role = store.RoleAdmin
	}
	u, err := a.db.CreateUser(p.Email, hash, role)
	if err != nil {
		return store.User{}, err
	}
	if err := a.db.SetUserOAuthSubject(u.ID, p.Subject); err != nil {
		return store.User{}, err
	}
	u.OAuthSubject = p.Subject
	if p.Name != "" {
		if err := a.db.SetUserDisplayName(u.ID, p.Name); err == nil {
			u.DisplayName = p.Name
		}
	}
	return u, nil
}

func (a *app) clearOAuthStateCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookie,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   a.cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

// redirectLoginError sends the browser back to the login screen with a code the
// UI turns into a message; OAuth failures arrive as top-level navigations, so
// there is no JSON client to answer.
func (a *app) redirectLoginError(w http.ResponseWriter, r *http.Request, code string) {
	http.Redirect(w, r, "/login?oauth_error="+url.QueryEscape(code), http.StatusFound)
}
