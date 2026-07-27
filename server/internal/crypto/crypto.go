// Package crypto encrypts credentials at rest with AES-256-GCM. The master key
// is supplied by the operator via the environment and never persisted.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
)

// MasterKeyEnv is the environment variable holding the base64 32-byte key.
const MasterKeyEnv = "REEVE_MASTER_KEY"

const keyLen = 32

// Cipher seals and opens secrets with a fixed master key, and authenticates the
// audit chain with a key derived from the same secret.
type Cipher struct {
	aead   cipher.AEAD
	macKey []byte
}

// macKeyLabel domain-separates the audit MAC key from the encryption key, so the
// two uses of the master secret cannot be played off against each other.
const macKeyLabel = "reeve/audit-mac/v1"

// New builds a Cipher from a raw 32-byte key.
func New(key []byte) (*Cipher, error) {
	if len(key) != keyLen {
		return nil, fmt.Errorf("master key must be %d bytes, got %d", keyLen, len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(macKeyLabel))
	return &Cipher{aead: aead, macKey: mac.Sum(nil)}, nil
}

// MAC authenticates one audit-chain link. A plain hash would let anyone who can
// write to the database recompute the whole chain after editing it, which is the
// attack the chain exists to catch.
func (c *Cipher) MAC(data []byte) string {
	mac := hmac.New(sha256.New, c.macKey)
	mac.Write(data)
	return hex.EncodeToString(mac.Sum(nil))
}

// NewFromEnv builds a Cipher from REEVE_MASTER_KEY (base64). It returns an
// error when the key is missing or not 32 bytes, so the server can refuse to
// start rather than run without encryption.
func NewFromEnv() (*Cipher, error) {
	raw := os.Getenv(MasterKeyEnv)
	if raw == "" {
		return nil, fmt.Errorf("%s is not set", MasterKeyEnv)
	}
	key, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("%s is not valid base64: %w", MasterKeyEnv, err)
	}
	return New(key)
}

// Seal encrypts plaintext, returning the ciphertext and the random nonce used.
// The nonce is stored alongside the ciphertext; it need not be secret.
//
// aad binds the ciphertext to where it is stored. It is authenticated but not
// encrypted, and Open must be given the same bytes. Without it, GCM proves only
// that a blob was sealed with this key, so anyone who can write to the database
// could move one row's ciphertext into another row and have it decrypt happily
// under the wrong identity.
func (c *Cipher) Seal(plaintext, aad []byte) (ciphertext, nonce []byte, err error) {
	nonce = make([]byte, c.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, nil, err
	}
	ciphertext = c.aead.Seal(nil, nonce, plaintext, aad)
	return ciphertext, nonce, nil
}

// Open decrypts ciphertext with its nonce and the aad it was sealed under. It
// fails if any of the three was tampered with, if the aad does not match, or if
// the key is wrong (GCM authentication).
func (c *Cipher) Open(ciphertext, nonce, aad []byte) ([]byte, error) {
	if len(nonce) != c.aead.NonceSize() {
		return nil, errors.New("invalid nonce length")
	}
	return c.aead.Open(nil, nonce, ciphertext, aad)
}
