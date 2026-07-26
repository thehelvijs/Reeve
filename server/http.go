package main

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/thehelvijs/Reeve/contracts"
	"github.com/thehelvijs/Reeve/server/internal/auth"
	"github.com/thehelvijs/Reeve/server/internal/rbac"
	"github.com/thehelvijs/Reeve/server/internal/store"
)

const sessionCookie = "reeve_session"

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

// clientIP is the caller's address: the left-most X-Forwarded-For entry when
// the operator has declared a reverse proxy in front, otherwise the peer
// address. Honoring the header unconditionally would let any caller forge both
// the login throttle key and the source IP recorded against a reveal.
func (a *app) clientIP(r *http.Request) string {
	if a.cfg.TrustProxyHeaders {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			return strings.TrimSpace(strings.Split(xff, ",")[0])
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// csrfSafeMethods do not change state, so they need no origin check.
var csrfSafeMethods = map[string]bool{
	http.MethodGet: true, http.MethodHead: true, http.MethodOptions: true,
}

// requireSameOrigin rejects a state-changing request that carries a session
// cookie but does not come from this origin. The cookie is the only ambient
// credential the browser attaches on its own, so gating on its presence covers
// every CSRF-reachable route while leaving token-authenticated callers (the
// agent's ingest push, scripts) alone.
func (a *app) requireSameOrigin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if csrfSafeMethods[r.Method] {
			next.ServeHTTP(w, r)
			return
		}
		if _, err := r.Cookie(sessionCookie); err != nil {
			next.ServeHTTP(w, r)
			return
		}
		if !a.originAllowed(r) {
			writeError(w, http.StatusForbidden, "bad_origin",
				"cross-origin request rejected; send an Origin header matching this server")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// originAllowed reports whether the request's Origin names this server. A
// missing Origin fails: a browser always sends one on a state-changing fetch,
// so its absence alongside a session cookie is not a case worth trusting.
func (a *app) originAllowed(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return false
	}
	if a.cfg.PublicURL != "" && origin == strings.TrimSuffix(a.cfg.PublicURL, "/") {
		return true
	}
	for _, allowed := range a.cfg.AllowedOrigins {
		if origin == allowed {
			return true
		}
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return origin == scheme+"://"+r.Host
}

// contentSecurityPolicy locks the page down to its own bundle. Inline styles
// stay allowed because Leaflet, uPlot and React Flow all set element styles at
// runtime; script injection, the vector that matters for a page that renders
// decrypted secrets, is blocked outright.
const contentSecurityPolicy = "default-src 'self'; " +
	"script-src 'self'; " +
	"style-src 'self' 'unsafe-inline'; " +
	"img-src 'self' data: https://*.basemaps.cartocdn.com; " +
	"font-src 'self'; " +
	"connect-src 'self'; " +
	"object-src 'none'; " +
	"base-uri 'self'; " +
	"form-action 'self'; " +
	"frame-ancestors 'none'"

// securityHeaders sets the response headers the browser needs to keep a
// credential-revealing page out of a frame and off a sniffed content type.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Content-Security-Policy", contentSecurityPolicy)
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
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
			r.Method, r.URL.Path, rec.status, time.Since(start).Round(time.Millisecond), a.clientIP(r), who)
	})
}
