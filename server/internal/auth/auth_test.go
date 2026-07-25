package auth

import (
	"strings"
	"testing"
)

func TestHashVerifyPassword(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if strings.Contains(hash, "correct horse") {
		t.Fatal("hash leaks the plaintext")
	}
	if !VerifyPassword("correct horse battery staple", hash) {
		t.Error("correct password rejected")
	}
	if VerifyPassword("wrong", hash) {
		t.Error("wrong password accepted")
	}
}

func TestHashSaltsDiffer(t *testing.T) {
	h1, _ := HashPassword("same")
	h2, _ := HashPassword("same")
	if h1 == h2 {
		t.Error("two hashes of the same password are identical (missing salt)")
	}
}

func TestVerifyRejectsMalformed(t *testing.T) {
	for _, bad := range []string{"", "notahash", "$argon2id$v=19$bad"} {
		if VerifyPassword("x", bad) {
			t.Errorf("verify accepted malformed hash %q", bad)
		}
	}
}

func TestNewTokenPrefixAndHash(t *testing.T) {
	tok, hash := NewToken()
	if !strings.HasPrefix(tok, TokenPrefix) {
		t.Errorf("token missing prefix: %q", tok)
	}
	if HashToken(tok) != hash {
		t.Error("HashToken not deterministic with NewToken")
	}
	if strings.Contains(hash, tok) {
		t.Error("hash contains the raw token")
	}
	tok2, _ := NewToken()
	if tok == tok2 {
		t.Error("two tokens collided")
	}
}

// The prefixes are printed to operators and pasted into installers, so pin the
// literals: changing them is a rename, not an implementation detail.
func TestTokenPrefixesAreReeveBranded(t *testing.T) {
	if TokenPrefix != "rvt_" {
		t.Errorf("TokenPrefix = %q, want rvt_", TokenPrefix)
	}
	if AgentTokenPrefix != "rva_" {
		t.Errorf("AgentTokenPrefix = %q, want rva_", AgentTokenPrefix)
	}
	agent, _ := NewAgentToken()
	if !strings.HasPrefix(agent, AgentTokenPrefix) {
		t.Errorf("agent token missing prefix: %q", agent)
	}
}
