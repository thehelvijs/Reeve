package main

import (
	"embed"
	"errors"

	"github.com/thehelvijs/Reeve/signing"
)

// releasePubKeyFile holds the public half of the release signing key. Self-update
// verifies every downloaded binary against it, so a compromised or spoofed
// server cannot push code to a monitored host.
//
//go:embed release_pubkey.txt
var releasePubKeyFile embed.FS

// errNoReleaseKey means this build carries no usable public key. Self-update
// then refuses to run rather than installing something unverifiable.
var errNoReleaseKey = errors.New("no release signing key is embedded in this build")

// releasePublicKey parses the embedded public key.
func releasePublicKey() (signing.PublicKey, error) {
	data, err := releasePubKeyFile.ReadFile("release_pubkey.txt")
	if err != nil {
		return signing.PublicKey{}, errNoReleaseKey
	}
	pub, err := signing.ParsePublicKey(string(data))
	if err != nil {
		return signing.PublicKey{}, errNoReleaseKey
	}
	return pub, nil
}
