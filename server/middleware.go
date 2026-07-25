package main

import (
	"errors"
	"net/http"

	"github.com/thehelvijs/Reeve/server/internal/auth"
	"github.com/thehelvijs/Reeve/server/internal/rbac"
	"github.com/thehelvijs/Reeve/server/internal/store"
)

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
