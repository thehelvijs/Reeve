// Package crypto encrypts credentials at rest with AES-256-GCM. The master key
// is supplied by the operator via the environment and never persisted.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
)

// MasterKeyEnv is the environment variable holding the base64 32-byte key.
const MasterKeyEnv = "REEVE_MASTER_KEY"

const keyLen = 32

// Cipher seals and opens secrets with a fixed master key.
type Cipher struct {
	aead cipher.AEAD
}

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
	return &Cipher{aead: aead}, nil
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
func (c *Cipher) Seal(plaintext []byte) (ciphertext, nonce []byte, err error) {
	nonce = make([]byte, c.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, nil, err
	}
	ciphertext = c.aead.Seal(nil, nonce, plaintext, nil)
	return ciphertext, nonce, nil
}

// Open decrypts ciphertext with its nonce. It fails if either was tampered with
// or the key is wrong (GCM authentication).
func (c *Cipher) Open(ciphertext, nonce []byte) ([]byte, error) {
	if len(nonce) != c.aead.NonceSize() {
		return nil, errors.New("invalid nonce length")
	}
	return c.aead.Open(nil, nonce, ciphertext, nil)
}
