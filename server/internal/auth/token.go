package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
)

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
