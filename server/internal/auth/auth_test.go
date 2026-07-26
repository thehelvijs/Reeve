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

// The prefix is printed to operators and pasted into installers, so pin the
// literal: changing it is a rename, not an implementation detail.
func TestAgentTokenPrefixIsReeveBranded(t *testing.T) {
	if AgentTokenPrefix != "rva_" {
		t.Errorf("AgentTokenPrefix = %q, want rva_", AgentTokenPrefix)
	}
	agent, _ := NewAgentToken()
	if !strings.HasPrefix(agent, AgentTokenPrefix) {
		t.Errorf("agent token missing prefix: %q", agent)
	}
}
