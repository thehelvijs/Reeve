// Package auth provides password hashing, API-token generation, and the
// request principal used by RBAC.
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/thehelvijs/Reeve/server/internal/store"
	"golang.org/x/crypto/argon2"
)

// argon2id parameters. Tuned for interactive login on a LAN server.
const (
	argonTime    = 1
	argonMemory  = 64 * 1024
	argonThreads = 4
	argonKeyLen  = 32
	argonSaltLen = 16
)

// HashPassword returns an encoded argon2id hash (self-describing: params+salt).
func HashPassword(password string) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argonMemory, argonTime, argonThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

// VerifyPassword reports whether password matches the encoded hash.
func VerifyPassword(password, encoded string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return false
	}
	var mem, time, threads int
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &mem, &time, &threads); err != nil {
		return false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false
	}
	got := argon2.IDKey([]byte(password), salt, uint32(time), uint32(mem), uint8(threads), uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1
}

// ErrInvalidCredentials is returned by login for any auth failure, without
// distinguishing unknown user from bad password.
var ErrInvalidCredentials = errors.New("invalid email or password")

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

// TokenPrefix marks a Reeve API token in Authorization headers.
const TokenPrefix = "lvt_"

// AgentTokenPrefix marks a host agent enrollment token.
const AgentTokenPrefix = "lva_"

// NewToken returns a fresh random API token (shown once) and its storage hash.
func NewToken() (token, hash string) {
	return newPrefixedToken(TokenPrefix)
}

// NewAgentToken returns a fresh agent enrollment token and its storage hash.
func NewAgentToken() (token, hash string) {
	return newPrefixedToken(AgentTokenPrefix)
}

func newPrefixedToken(prefix string) (token, hash string) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	token = prefix + hex.EncodeToString(b)
	return token, HashToken(token)
}

// HashToken returns the stored, lookup-able hash of a token. A SHA-256 is used
// (not argon2) because tokens are high-entropy random values, not passwords.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
