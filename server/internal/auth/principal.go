package auth

import "github.com/thehelvijs/Reeve/server/internal/store"

// Principal is the authenticated identity behind a request.
type Principal struct {
	UserID string
	Email  string
	Role   string
	// ViaToken is true when authenticated by API token rather than a session.
	ViaToken bool
}

// IsAdmin reports whether the principal has the admin role.
func (p Principal) IsAdmin() bool {
	return p.Role == store.RoleAdmin
}

// PrincipalFromUser builds a Principal from a user row.
func PrincipalFromUser(u store.User, viaToken bool) Principal {
	return Principal{UserID: u.ID, Email: u.Email, Role: u.Role, ViaToken: viaToken}
}
