package signing

import (
	"crypto/ed25519"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "artifact")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	return path
}

func TestSignAndVerifyRoundTrip(t *testing.T) {
	sk, pub, err := GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	path := writeFile(t, "agent binary bytes")

	sig, err := SignFile(sk, path, "version:1.2.3")
	if err != nil {
		t.Fatalf("SignFile: %v", err)
	}
	if _, err := VerifyFile(pub, path, sig); err != nil {
		t.Fatalf("VerifyFile: %v", err)
	}
	if _, err := VerifyBytes(pub, []byte("agent binary bytes"), sig); err != nil {
		t.Errorf("VerifyBytes: %v", err)
	}
}

// The whole point: content that does not match the signature must be refused.
func TestVerifyRejectsTamperedContent(t *testing.T) {
	sk, pub, _ := GenerateKey()
	path := writeFile(t, "agent binary bytes")
	sig, err := SignFile(sk, path, "version:1.2.3")
	if err != nil {
		t.Fatalf("SignFile: %v", err)
	}
	if _, err := VerifyBytes(pub, []byte("agent binary bytez"), sig); err == nil {
		t.Error("VerifyBytes accepted tampered content")
	}
	if err := os.WriteFile(path, []byte("malicious payload"), 0o600); err != nil {
		t.Fatalf("rewrite: %v", err)
	}
	if _, err := VerifyFile(pub, path, sig); err == nil {
		t.Error("VerifyFile accepted a swapped file")
	}
}

// A signature from any other key must not verify, or an attacker could just
// sign their own build.
func TestVerifyRejectsOtherKeys(t *testing.T) {
	sk, _, _ := GenerateKey()
	_, otherPub, _ := GenerateKey()
	path := writeFile(t, "agent binary bytes")
	sig, _ := SignFile(sk, path, "version:1.2.3")

	if _, err := VerifyFile(otherPub, path, sig); err == nil {
		t.Error("a signature verified against an unrelated public key")
	}
}

// The trusted comment carries the version, so tampering with it must fail even
// though the file itself is untouched.
func TestVerifyRejectsEditedTrustedComment(t *testing.T) {
	sk, pub, _ := GenerateKey()
	path := writeFile(t, "agent binary bytes")
	sig, _ := SignFile(sk, path, "version:1.2.3")
	edited := strings.Replace(sig, "version:1.2.3", "version:9.9.9", 1)

	if _, err := VerifyFile(pub, path, edited); err == nil {
		t.Error("an edited trusted comment still verified")
	}
}

func TestVerifyRejectsMalformedSignatures(t *testing.T) {
	sk, pub, _ := GenerateKey()
	path := writeFile(t, "agent binary bytes")
	good, _ := SignFile(sk, path, "version:1.2.3")
	lines := strings.Split(strings.TrimRight(good, "\n"), "\n")

	cases := map[string]string{
		"empty":            "",
		"truncated":        lines[0] + "\n" + lines[1] + "\n",
		"not base64":       lines[0] + "\n!!!!\n" + lines[2] + "\n" + lines[3] + "\n",
		"no trusted line":  lines[0] + "\n" + lines[1] + "\nnot a trusted comment\n" + lines[3] + "\n",
		"short signature":  lines[0] + "\n" + base64.StdEncoding.EncodeToString([]byte("ED")) + "\n" + lines[2] + "\n" + lines[3] + "\n",
		"bad global":       lines[0] + "\n" + lines[1] + "\n" + lines[2] + "\n!!!!\n",
		"wrong global len": lines[0] + "\n" + lines[1] + "\n" + lines[2] + "\n" + base64.StdEncoding.EncodeToString([]byte("short")) + "\n",
	}
	for name, sig := range cases {
		if _, err := VerifyFile(pub, path, sig); err == nil {
			t.Errorf("%s: VerifyFile = nil, want an error", name)
		}
	}
}

func TestPublicKeyRoundTripAndParsing(t *testing.T) {
	_, pub, _ := GenerateKey()
	encoded := EncodePublicKey(pub)
	if !strings.HasPrefix(encoded, untrustedPrefix) {
		t.Errorf("encoded key missing the comment line:\n%s", encoded)
	}
	parsed, err := ParsePublicKey(encoded)
	if err != nil {
		t.Fatalf("ParsePublicKey: %v", err)
	}
	if parsed.KeyID != pub.KeyID || !parsed.Key.Equal(pub.Key) {
		t.Error("public key did not survive a round trip")
	}
	// A bare key line (no comment) is accepted too.
	bare := strings.Split(strings.TrimSpace(encoded), "\n")[1]
	if _, err := ParsePublicKey(bare); err != nil {
		t.Errorf("ParsePublicKey on a bare line: %v", err)
	}
	for name, bad := range map[string]string{
		"empty":        "",
		"comment only": untrustedPrefix + "nothing here\n",
		"not base64":   "!!!!\n",
		"short":        base64.StdEncoding.EncodeToString([]byte("Edshort")) + "\n",
		"wrong alg": base64.StdEncoding.EncodeToString(
			append([]byte("XX12345678"), make([]byte, ed25519.PublicKeySize)...)) + "\n",
	} {
		if _, err := ParsePublicKey(bad); err == nil {
			t.Errorf("%s: ParsePublicKey = nil error, want a rejection", name)
		}
	}
}

func TestSecretKeyRoundTrip(t *testing.T) {
	sk, pub, _ := GenerateKey()
	parsed, err := ParseSecretKey(EncodeSecretKey(sk))
	if err != nil {
		t.Fatalf("ParseSecretKey: %v", err)
	}
	if parsed.KeyID != sk.KeyID || !parsed.Key.Equal(sk.Key) {
		t.Fatal("secret key did not survive a round trip")
	}
	// And it still produces signatures the original public key accepts.
	path := writeFile(t, "bytes")
	sig, err := SignFile(parsed, path, "v1")
	if err != nil {
		t.Fatalf("SignFile: %v", err)
	}
	if _, err := VerifyFile(pub, path, sig); err != nil {
		t.Errorf("VerifyFile after a key round trip: %v", err)
	}
	for name, bad := range map[string]string{
		"empty":      "",
		"not base64": secretKeyPrefix + "!!!!\n",
		"short":      secretKeyPrefix + base64.StdEncoding.EncodeToString([]byte("short")) + "\n",
	} {
		if _, err := ParseSecretKey(bad); err == nil {
			t.Errorf("%s: ParseSecretKey = nil error, want a rejection", name)
		}
	}
}

func TestSignRejectsMultilineTrustedComment(t *testing.T) {
	sk, _, _ := GenerateKey()
	path := writeFile(t, "bytes")
	if _, err := SignFile(sk, path, "v1\nuntrusted comment: injected"); err == nil {
		t.Error("SignFile accepted a multi-line trusted comment")
	}
}

func TestSignMissingFile(t *testing.T) {
	sk, _, _ := GenerateKey()
	if _, err := SignFile(sk, filepath.Join(t.TempDir(), "nope"), "v1"); err == nil {
		t.Error("SignFile on a missing file returned nil error")
	}
}
