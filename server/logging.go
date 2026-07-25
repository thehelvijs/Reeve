package main

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/thehelvijs/Reeve/server/internal/rbac"
)

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
