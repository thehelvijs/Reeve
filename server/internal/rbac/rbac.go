// Package rbac carries the authenticated principal through the request context
// and gates handlers by authentication and role.
package rbac

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/thehelvijs/Reeve/contracts"
	"github.com/thehelvijs/Reeve/server/internal/auth"
)

type ctxKey struct{}

// WithPrincipal returns a context carrying the principal.
func WithPrincipal(ctx context.Context, p auth.Principal) context.Context {
	return context.WithValue(ctx, ctxKey{}, p)
}

// FromContext extracts the principal, if any, from the context.
func FromContext(ctx context.Context) (auth.Principal, bool) {
	p, ok := ctx.Value(ctxKey{}).(auth.Principal)
	return p, ok
}

// RequireAuth rejects requests with no authenticated principal.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := FromContext(r.Context()); !ok {
			writeError(w, http.StatusUnauthorized, "unauthenticated", "authentication required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireAdmin rejects requests whose principal is not an admin.
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, ok := FromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "unauthenticated", "authentication required")
			return
		}
		if !p.IsAdmin() {
			writeError(w, http.StatusForbidden, "forbidden", "admin role required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(contracts.ErrorResponse{Code: code, Message: msg})
}
