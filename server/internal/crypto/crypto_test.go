package crypto

import (
	"bytes"
	"encoding/base64"
	"testing"
)

func testKey() string {
	return base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{7}, 32))
}

func TestSealOpenRoundTrip(t *testing.T) {
	c, err := New(mustKey(t, testKey()))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	plain := []byte("ssh: root / hunter2")
	aad := []byte("reeve/credential/host-1")
	ct, nonce, err := c.Seal(plain, aad)
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	if bytes.Contains(ct, plain) {
		t.Fatal("ciphertext contains plaintext")
	}
	got, err := c.Open(ct, nonce, aad)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if !bytes.Equal(got, plain) {
		t.Fatalf("round-trip mismatch: %q != %q", got, plain)
	}
}

func TestUniqueNoncePerSeal(t *testing.T) {
	c, _ := New(mustKey(t, testKey()))
	_, n1, _ := c.Seal([]byte("x"), nil)
	_, n2, _ := c.Seal([]byte("x"), nil)
	if bytes.Equal(n1, n2) {
		t.Fatal("nonce reused across seals")
	}
}

func TestTamperFails(t *testing.T) {
	c, _ := New(mustKey(t, testKey()))
	ct, nonce, _ := c.Seal([]byte("secret"), nil)
	ct[0] ^= 0xFF
	if _, err := c.Open(ct, nonce, nil); err == nil {
		t.Fatal("Open accepted tampered ciphertext")
	}
}

func TestWrongKeyFails(t *testing.T) {
	c1, _ := New(mustKey(t, testKey()))
	ct, nonce, _ := c1.Seal([]byte("secret"), nil)

	otherKey := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{9}, 32))
	c2, _ := New(mustKey(t, otherKey))
	if _, err := c2.Open(ct, nonce, nil); err == nil {
		t.Fatal("Open accepted ciphertext under wrong key")
	}
}

func TestNewFromEnvMissingKeyFails(t *testing.T) {
	t.Setenv("REEVE_MASTER_KEY", "")
	if _, err := NewFromEnv(); err == nil {
		t.Fatal("expected error when master key is unset")
	}
}

func TestNewFromEnvWrongLengthFails(t *testing.T) {
	t.Setenv("REEVE_MASTER_KEY", base64.StdEncoding.EncodeToString([]byte("tooshort")))
	if _, err := NewFromEnv(); err == nil {
		t.Fatal("expected error when master key is not 32 bytes")
	}
}

func TestNewFromEnvValid(t *testing.T) {
	t.Setenv("REEVE_MASTER_KEY", testKey())
	if _, err := NewFromEnv(); err != nil {
		t.Fatalf("NewFromEnv with valid key: %v", err)
	}
}

func mustKey(t *testing.T, b64 string) []byte {
	t.Helper()
	k, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		t.Fatalf("decode key: %v", err)
	}
	return k
}

func TestOpenRejectsAnotherRowsAAD(t *testing.T) {
	c, _ := New(mustKey(t, testKey()))
	ct, nonce, _ := c.Seal([]byte("root / hunter2"), []byte("reeve/credential/host-1"))
	if _, err := c.Open(ct, nonce, []byte("reeve/credential/host-2")); err == nil {
		t.Fatal("a credential sealed for host-1 opened as host-2")
	}
	if _, err := c.Open(ct, nonce, nil); err == nil {
		t.Fatal("a bound credential opened with no aad at all")
	}
	got, err := c.Open(ct, nonce, []byte("reeve/credential/host-1"))
	if err != nil {
		t.Fatalf("Open with the right aad: %v", err)
	}
	if string(got) != "root / hunter2" {
		t.Fatalf("round-trip mismatch: %q", got)
	}
}
