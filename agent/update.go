package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/thehelvijs/Reeve/signing"
)

// selfArch maps runtime.GOARCH to the arch suffix the server publishes
// binaries under. GOARM is not available at runtime, so arm defaults to
// armv7; armv6 hosts will not auto-update correctly.
func selfArch(goarch string) string {
	switch goarch {
	case "amd64", "arm64", "386", "riscv64":
		return goarch
	case "arm":
		return "armv7"
	default:
		return goarch
	}
}

// needsUpdate reports whether remoteLine's checksum differs from localSum.
// A missing or malformed remoteLine fails safe by returning false.
func needsUpdate(localSum, remoteLine string) bool {
	fields := strings.Fields(remoteLine)
	if len(fields) == 0 {
		return false
	}
	remoteSum := fields[0]
	if remoteSum == "" {
		return false
	}
	if !isSHA256Hex(remoteSum) {
		return false
	}
	return remoteSum != localSum
}

// isSHA256Hex reports whether s is a well-formed lowercase hex sha256 digest.
func isSHA256Hex(s string) bool {
	if len(s) != 64 {
		return false
	}
	for _, r := range s {
		isDigit := r >= '0' && r <= '9'
		isLower := r >= 'a' && r <= 'f'
		if !isDigit && !isLower {
			return false
		}
	}
	return true
}

func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// maxChecksumBytes, maxSignatureBytes, and maxBinaryBytes bound fetchBody so a
// compromised or misbehaving server can't make this root process buffer an
// unbounded body.
const (
	maxChecksumBytes  = 4096
	maxSignatureBytes = 4096
	maxBinaryBytes    = 128 << 20
)

func fetchBody(client *http.Client, url string, maxBytes int64) ([]byte, error) {
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("GET %s returned %d", url, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > maxBytes {
		return nil, fmt.Errorf("GET %s exceeded %d byte limit", url, maxBytes)
	}
	return body, nil
}

// checkAndUpdate compares the running binary's checksum to the server's
// published checksum for this arch, and if different, downloads, verifies the
// release signature, and atomically swaps the binary in before exiting so
// systemd relaunches it.
//
// The checksum only detects change: it comes from the same origin as the binary,
// so it proves nothing about authorship. The Ed25519 signature over the release
// key is what makes this safe, and an update that cannot be verified is refused.
func (c config) checkAndUpdate() error {
	pub, err := releasePublicKey()
	if err != nil {
		return fmt.Errorf("self-update: %w", err)
	}
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("self-update: locate executable: %w", err)
	}
	localSum, err := sha256File(exe)
	if err != nil {
		return fmt.Errorf("self-update: hash running binary: %w", err)
	}

	arch := selfArch(runtime.GOARCH)
	base := strings.TrimSuffix(c.ServerURL, "/")
	client := &http.Client{Timeout: 30 * time.Second}

	sumBody, err := fetchBody(client, base+"/dl/agent-linux-"+arch+".sha256", maxChecksumBytes)
	if err != nil {
		return fmt.Errorf("self-update: fetch checksum: %w", err)
	}
	remoteLine := string(sumBody)
	if !needsUpdate(localSum, remoteLine) {
		return nil
	}
	remoteSum := strings.Fields(remoteLine)[0]

	sigBody, err := fetchBody(client, base+"/dl/agent-linux-"+arch+".minisig", maxSignatureBytes)
	if err != nil {
		return fmt.Errorf("self-update: fetch signature: %w", err)
	}

	binBody, err := fetchBody(client, base+"/dl/agent-linux-"+arch, maxBinaryBytes)
	if err != nil {
		return fmt.Errorf("self-update: fetch binary: %w", err)
	}
	// Verified before the bytes are ever written to disk, let alone executed.
	if err := signing.VerifyBytes(pub, binBody, string(sigBody)); err != nil {
		return fmt.Errorf("self-update: refusing unverified binary: %w", err)
	}

	dir := filepath.Dir(exe)
	tmp, err := os.CreateTemp(dir, ".agent-update-*")
	if err != nil {
		return fmt.Errorf("self-update: create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	_, writeErr := tmp.Write(binBody)
	closeErr := tmp.Close()
	if writeErr != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("self-update: write temp file: %w", writeErr)
	}
	if closeErr != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("self-update: close temp file: %w", closeErr)
	}

	downloadedSum, err := sha256File(tmpPath)
	if err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("self-update: hash downloaded binary: %w", err)
	}
	if downloadedSum != remoteSum {
		os.Remove(tmpPath)
		return fmt.Errorf("self-update: downloaded binary checksum %s does not match published %s", downloadedSum, remoteSum)
	}

	if err := os.Chmod(tmpPath, 0o755); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("self-update: chmod temp file: %w", err)
	}
	if err := os.Rename(tmpPath, exe); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("self-update: replace running binary: %w", err)
	}

	log.Printf("agent updated %s -> %s, restarting", localSum[:12], downloadedSum[:12])
	os.Exit(0)
	return nil
}
