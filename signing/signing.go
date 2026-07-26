// Package signing signs and verifies release artifacts with Ed25519 in
// minisign's file formats, so a published signature can also be checked with
// the standard `minisign -V` tool. The agent embeds a public key and refuses to
// install any update it cannot verify against it.
package signing

import (
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/crypto/blake2b"
)

// minisign wire constants. "Ed" tags a public key; "ED" tags a signature over
// the file's Blake2b-512 hash (minisign's prehashed mode).
const (
	algPublicKey = "Ed"
	algHashedSig = "ED"
	keyIDLen     = 8
)

const (
	untrustedPrefix = "untrusted comment: "
	trustedPrefix   = "trusted comment: "
)

// PublicKey is a parsed minisign public key.
type PublicKey struct {
	KeyID [keyIDLen]byte
	Key   ed25519.PublicKey
}

// SecretKey is a signing key. Its file format is this project's own (an
// unencrypted base64 seed): nothing else has to read it, and minisign's
// scrypt-wrapped format would add a KDF for no gain here.
type SecretKey struct {
	KeyID [keyIDLen]byte
	Key   ed25519.PrivateKey
}

const secretKeyPrefix = "reeve-release-secret-key-v1: "

// GenerateKey creates a signing keypair.
func GenerateKey() (SecretKey, PublicKey, error) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		return SecretKey{}, PublicKey{}, err
	}
	var id [keyIDLen]byte
	sum := blake2b.Sum256(pub)
	copy(id[:], sum[:keyIDLen])
	return SecretKey{KeyID: id, Key: priv}, PublicKey{KeyID: id, Key: pub}, nil
}

// EncodePublicKey renders a minisign-format public key file.
func EncodePublicKey(pub PublicKey) string {
	blob := append([]byte(algPublicKey), pub.KeyID[:]...)
	blob = append(blob, pub.Key...)
	return untrustedPrefix + "minisign public key\n" + base64.StdEncoding.EncodeToString(blob) + "\n"
}

// ParsePublicKey reads a minisign public key file, tolerating a bare key line.
func ParsePublicKey(s string) (PublicKey, error) {
	line := lastNonCommentLine(s)
	if line == "" {
		return PublicKey{}, errors.New("no public key line")
	}
	blob, err := base64.StdEncoding.DecodeString(line)
	if err != nil {
		return PublicKey{}, fmt.Errorf("public key is not base64: %w", err)
	}
	if len(blob) != 2+keyIDLen+ed25519.PublicKeySize {
		return PublicKey{}, errors.New("public key has the wrong length")
	}
	if string(blob[:2]) != algPublicKey {
		return PublicKey{}, fmt.Errorf("unsupported public key algorithm %q", blob[:2])
	}
	var pub PublicKey
	copy(pub.KeyID[:], blob[2:2+keyIDLen])
	pub.Key = ed25519.PublicKey(blob[2+keyIDLen:])
	return pub, nil
}

// EncodeSecretKey renders a secret key file.
func EncodeSecretKey(sk SecretKey) string {
	blob := append(sk.KeyID[:], sk.Key.Seed()...)
	return secretKeyPrefix + base64.StdEncoding.EncodeToString(blob) + "\n"
}

// ParseSecretKey reads a secret key file.
func ParseSecretKey(s string) (SecretKey, error) {
	line := ""
	for _, l := range strings.Split(s, "\n") {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		line = strings.TrimPrefix(l, secretKeyPrefix)
	}
	if line == "" {
		return SecretKey{}, errors.New("no secret key line")
	}
	blob, err := base64.StdEncoding.DecodeString(line)
	if err != nil {
		return SecretKey{}, fmt.Errorf("secret key is not base64: %w", err)
	}
	if len(blob) != keyIDLen+ed25519.SeedSize {
		return SecretKey{}, errors.New("secret key has the wrong length")
	}
	var sk SecretKey
	copy(sk.KeyID[:], blob[:keyIDLen])
	sk.Key = ed25519.NewKeyFromSeed(blob[keyIDLen:])
	return sk, nil
}

// hashFile returns the Blake2b-512 digest minisign's prehashed mode signs.
func hashFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	h, err := blake2b.New512(nil)
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(h, f); err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}

// SignFile returns the .minisig contents for path.
func SignFile(sk SecretKey, path, trustedComment string) (string, error) {
	digest, err := hashFile(path)
	if err != nil {
		return "", err
	}
	if strings.ContainsAny(trustedComment, "\r\n") {
		return "", errors.New("trusted comment must be a single line")
	}
	sig := ed25519.Sign(sk.Key, digest)

	blob := append([]byte(algHashedSig), sk.KeyID[:]...)
	blob = append(blob, sig...)
	// The global signature covers the signature and the trusted comment
	// together, which is what makes the comment trustworthy.
	global := ed25519.Sign(sk.Key, append(append([]byte{}, sig...), []byte(trustedComment)...))

	var b strings.Builder
	b.WriteString(untrustedPrefix + "signature from reeve release key\n")
	b.WriteString(base64.StdEncoding.EncodeToString(blob) + "\n")
	b.WriteString(trustedPrefix + trustedComment + "\n")
	b.WriteString(base64.StdEncoding.EncodeToString(global) + "\n")
	return b.String(), nil
}

// VerifyFile checks a .minisig against a file and the expected public key,
// returning the signed trusted comment.
func VerifyFile(pub PublicKey, path, signature string) (string, error) {
	digest, err := hashFile(path)
	if err != nil {
		return "", err
	}
	return VerifyDigest(pub, digest, signature)
}

// VerifyBytes checks a .minisig against in-memory content, returning the
// signed trusted comment.
func VerifyBytes(pub PublicKey, content []byte, signature string) (string, error) {
	h, err := blake2b.New512(nil)
	if err != nil {
		return "", err
	}
	h.Write(content)
	return VerifyDigest(pub, h.Sum(nil), signature)
}

// VerifyDigest checks a .minisig against an already-computed Blake2b-512
// digest. The trusted comment it returns is covered by the global signature,
// so a caller may act on what it says.
func VerifyDigest(pub PublicKey, digest []byte, signature string) (string, error) {
	sigBlob, trustedComment, global, err := parseSignature(signature)
	if err != nil {
		return "", err
	}
	if string(sigBlob[:2]) != algHashedSig {
		return "", fmt.Errorf("unsupported signature algorithm %q", sigBlob[:2])
	}
	var keyID [keyIDLen]byte
	copy(keyID[:], sigBlob[2:2+keyIDLen])
	if keyID != pub.KeyID {
		return "", errors.New("signature was made by a different key")
	}
	sig := sigBlob[2+keyIDLen:]
	if !ed25519.Verify(pub.Key, digest, sig) {
		return "", errors.New("signature does not match the file")
	}
	if !ed25519.Verify(pub.Key, append(append([]byte{}, sig...), []byte(trustedComment)...), global) {
		return "", errors.New("trusted comment signature does not match")
	}
	return trustedComment, nil
}

// parseSignature splits a .minisig into its signature blob, trusted comment,
// and global signature. Lines are located by what they are rather than by
// position, so a file without the leading untrusted comment fails on its own
// merits instead of misreading the line after it.
func parseSignature(s string) (sigBlob []byte, trustedComment string, global []byte, err error) {
	var payload []string
	for _, l := range strings.Split(s, "\n") {
		l = strings.TrimRight(l, "\r")
		if strings.TrimSpace(l) == "" || strings.HasPrefix(l, untrustedPrefix) {
			continue
		}
		if strings.HasPrefix(l, trustedPrefix) {
			if trustedComment != "" {
				return nil, "", nil, errors.New("signature file has more than one trusted comment")
			}
			trustedComment = strings.TrimPrefix(l, trustedPrefix)
			continue
		}
		payload = append(payload, strings.TrimSpace(l))
	}
	if trustedComment == "" {
		return nil, "", nil, errors.New("missing trusted comment")
	}
	if len(payload) != 2 {
		return nil, "", nil, errors.New("signature file is truncated")
	}
	sigBlob, err = base64.StdEncoding.DecodeString(payload[0])
	if err != nil {
		return nil, "", nil, fmt.Errorf("signature line is not base64: %w", err)
	}
	if len(sigBlob) != 2+keyIDLen+ed25519.SignatureSize {
		return nil, "", nil, errors.New("signature line has the wrong length")
	}
	global, err = base64.StdEncoding.DecodeString(payload[1])
	if err != nil {
		return nil, "", nil, fmt.Errorf("global signature is not base64: %w", err)
	}
	if len(global) != ed25519.SignatureSize {
		return nil, "", nil, errors.New("global signature has the wrong length")
	}
	return sigBlob, trustedComment, global, nil
}

// lastNonCommentLine returns the final line that is not a comment, which is
// where minisign keeps the key material.
func lastNonCommentLine(s string) string {
	out := ""
	for _, l := range strings.Split(s, "\n") {
		l = strings.TrimSpace(l)
		if l == "" || strings.HasPrefix(l, untrustedPrefix) || strings.HasPrefix(l, "#") {
			continue
		}
		out = l
	}
	return out
}
