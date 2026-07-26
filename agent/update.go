package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/thehelvijs/Reeve/signing"
)

// buildArch is the arch suffix this binary was published under, stamped at
// build time via -ldflags. GOARM is not readable at runtime, so an armv6 build
// cannot tell itself apart from an armv7 one; without this stamp it would
// fetch the armv7 binary and replace itself with something it cannot execute.
var buildArch = ""

// selfArch is the arch suffix to fetch updates under: the build-time stamp
// when the release builder set one, otherwise derived from GOARCH.
func selfArch(goarch string) string {
	if buildArch != "" {
		return buildArch
	}
	switch goarch {
	case "amd64", "arm64", "386", "riscv64":
		return goarch
	case "arm":
		return "armv7"
	default:
		return goarch
	}
}

// errArmVariantUnknown stops an unstamped 32-bit arm build from guessing. A
// wrong guess replaces the running binary with one the CPU cannot run, and the
// service then restart-loops with no agent left to fix it.
var errArmVariantUnknown = errors.New(
	"this build carries no arch stamp and cannot tell armv6 from armv7; reinstall from the server to get a stamped build")

// signedVersion pulls the version out of a verified trusted comment, which the
// release builder writes as "version:<v>".
func signedVersion(trustedComment string) string {
	v, ok := strings.CutPrefix(strings.TrimSpace(trustedComment), "version:")
	if !ok {
		return ""
	}
	return strings.TrimSpace(v)
}

// olderThan reports whether candidate is a strictly lower version than current.
// Both are dot-separated numbers with an optional "+build" suffix; anything
// that does not parse that way compares as not-older, since refusing an update
// on a version string nobody can read would strand the fleet.
func olderThan(candidate, current string) bool {
	c, okC := versionParts(candidate)
	r, okR := versionParts(current)
	if !okC || !okR {
		return false
	}
	for i := 0; i < len(c) || i < len(r); i++ {
		cv, rv := 0, 0
		if i < len(c) {
			cv = c[i]
		}
		if i < len(r) {
			rv = r[i]
		}
		if cv != rv {
			return cv < rv
		}
	}
	return false
}

// versionParts splits "1.2.3+abc" into [1 2 3], reporting false for anything
// that is not a dot-separated run of numbers.
func versionParts(v string) ([]int, bool) {
	base, _, _ := strings.Cut(v, "+")
	if base == "" {
		return nil, false
	}
	var out []int
	for _, field := range strings.Split(base, ".") {
		n, err := strconv.Atoi(field)
		if err != nil {
			return nil, false
		}
		out = append(out, n)
	}
	return out, true
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

// selfChecksum is the sha256 of the running binary, hashed once. The update
// path replaces the file and exits, so it cannot change under a live process.
// Empty when it cannot be read, which the server reads as "cannot judge".
var selfChecksum = sync.OnceValue(func() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	sum, err := sha256File(exe)
	if err != nil {
		return ""
	}
	return sum
})

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

	if buildArch == "" && runtime.GOARCH == "arm" {
		return fmt.Errorf("self-update: %w", errArmVariantUnknown)
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
	comment, err := signing.VerifyBytes(pub, binBody, string(sigBody))
	if err != nil {
		return fmt.Errorf("self-update: refusing unverified binary: %w", err)
	}
	// A signature proves who built the binary, not that it is the newest one
	// they built. Without this, replaying an old release's bytes and signature
	// walks a host back onto a version whose bugs are public.
	if offered := signedVersion(comment); olderThan(offered, version) {
		return fmt.Errorf("self-update: refusing to move from %s back to %s", version, offered)
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
