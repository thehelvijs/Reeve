package main

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/thehelvijs/Reeve/contracts"
	"github.com/thehelvijs/Reeve/server/internal/auth"
	"github.com/thehelvijs/Reeve/server/internal/rbac"
	"github.com/thehelvijs/Reeve/server/internal/store"
)

const sessionCookie = "lv_session"

func (a *app) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "version": a.cfg.Version})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, contracts.ErrorResponse{Code: code, Message: msg})
}

// decodeJSON reads a JSON body into dst, rejecting unknown fields and oversized
// payloads.
func decodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return errors.New("invalid request body")
	}
	return nil
}

// resolvePrincipal authenticates a request by session cookie and, on success,
// injects the principal into the context. It never rejects; gating is left to
// RequireAuth / RequireAdmin so public routes stay open.
func (a *app) resolvePrincipal(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !dynamicRequest(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		if p, ok := a.principalFromRequest(r); ok {
			r = r.WithContext(rbac.WithPrincipal(r.Context(), p))
		}
		next.ServeHTTP(w, r)
	})
}

func (a *app) principalFromRequest(r *http.Request) (auth.Principal, bool) {
	if c, err := r.Cookie(sessionCookie); err == nil {
		sess, err := a.db.GetSession(c.Value)
		if err == nil {
			if u, err := a.activeUser(sess.UserID); err == nil {
				return auth.PrincipalFromUser(u, false), true
			}
		}
	}
	return auth.Principal{}, false
}

// activeUser returns the user only if the account is active.
func (a *app) activeUser(id string) (store.User, error) {
	u, err := a.db.GetUserByID(id)
	if err != nil {
		return store.User{}, err
	}
	if !u.Active {
		return store.User{}, errors.New("user deactivated")
	}
	return u, nil
}

// statusRecorder captures the response status for the access log.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

// dynamicRequest reports whether a path is served by the server's own routes
// rather than the embedded SPA bundle: API calls and agent-facing downloads.
// Those are the requests worth an access-log line, and the only ones that can
// carry a principal, so a static asset costs no session lookup.
func dynamicRequest(path string) bool {
	if strings.HasPrefix(path, "/api/") || strings.HasPrefix(path, "/dl/") {
		return true
	}
	return path == "/install.sh" || path == "/uninstall.sh"
}

// logRequests writes one access-log line per meaningful request when enabled.
// It runs inside resolvePrincipal so the authenticated user is available.
func (a *app) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !a.cfg.LogRequests || !dynamicRequest(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		start := time.Now()
		next.ServeHTTP(rec, r)
		who := "-"
		if p, ok := rbac.FromContext(r.Context()); ok {
			who = p.Email
		}
		log.Printf("http: %s %s %d %s ip=%s user=%s",
			r.Method, r.URL.Path, rec.status, time.Since(start).Round(time.Millisecond), clientIP(r), who)
	})
}
