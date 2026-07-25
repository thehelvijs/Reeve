package main

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/thehelvijs/Reeve/signing"
)

func TestSelfArch(t *testing.T) {
	cases := map[string]string{
		"amd64":   "amd64",
		"arm64":   "arm64",
		"386":     "386",
		"riscv64": "riscv64",
		"arm":     "armv7",
		"mips":    "mips",
	}
	for goarch, want := range cases {
		if got := selfArch(goarch); got != want {
			t.Errorf("selfArch(%q) = %q, want %q", goarch, got, want)
		}
	}
}

func TestNeedsUpdate(t *testing.T) {
	sumA := strings.Repeat("a", 64)
	sumB := strings.Repeat("b", 64)
	tests := []struct {
		name       string
		localSum   string
		remoteLine string
		want       bool
	}{
		{"differ", sumA, sumB + "  agent-linux-amd64\n", true},
		{"equal", sumB, sumB + "  agent-linux-amd64\n", false},
		{"empty", sumA, "", false},
		{"whitespace-only", sumA, "   \n", false},
		{"malformed-short-hex", sumA, "abcd  agent-linux-amd64\n", false},
		{"malformed-non-hex", sumA, "not-a-checksum  agent-linux-amd64\n", false},
		{"malformed-uppercase-hex", sumA, strings.ToUpper(sumB) + "  agent-linux-amd64\n", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := needsUpdate(tt.localSum, tt.remoteLine); got != tt.want {
				t.Errorf("needsUpdate(%q, %q) = %v, want %v", tt.localSum, tt.remoteLine, got, tt.want)
			}
		})
	}
}

func TestIsSHA256Hex(t *testing.T) {
	valid := strings.Repeat("a", 64)
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{"valid", valid, true},
		{"empty", "", false},
		{"too-short", valid[:63], false},
		{"too-long", valid + "a", false},
		{"uppercase", strings.ToUpper(valid), false},
		{"non-hex-char", strings.Repeat("g", 64), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isSHA256Hex(tt.in); got != tt.want {
				t.Errorf("isSHA256Hex(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

// The committed public key must parse: a build that cannot read it refuses to
// self-update at all.
func TestReleasePublicKeyIsUsable(t *testing.T) {
	pub, err := releasePublicKey()
	if err != nil {
		t.Fatalf("releasePublicKey: %v", err)
	}
	if len(pub.Key) != ed25519.PublicKeySize {
		t.Fatalf("public key length = %d, want %d", len(pub.Key), ed25519.PublicKeySize)
	}
}

// updateStub serves a checksum, a binary, and optionally a signature, standing
// in for the server the agent updates from.
func updateStub(t *testing.T, binary []byte, sig string, serveSig bool) *httptest.Server {
	t.Helper()
	sum := sha256.Sum256(binary)
	mux := http.NewServeMux()
	mux.HandleFunc("/dl/agent-linux-"+selfArch(runtime.GOARCH)+".sha256", func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, hex.EncodeToString(sum[:])+"  agent\n")
	})
	mux.HandleFunc("/dl/agent-linux-"+selfArch(runtime.GOARCH)+".minisig", func(w http.ResponseWriter, _ *http.Request) {
		if !serveSig {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		io.WriteString(w, sig)
	})
	mux.HandleFunc("/dl/agent-linux-"+selfArch(runtime.GOARCH), func(w http.ResponseWriter, _ *http.Request) {
		w.Write(binary)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// An update the agent cannot verify must be refused, and the running binary
// left alone. This is the phone-home-to-RCE path, so it is the important test.
func TestCheckAndUpdateRefusesUnverifiedBinary(t *testing.T) {
	otherKey, _, err := signing.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	payload := []byte("malicious replacement binary")
	path := filepath.Join(t.TempDir(), "payload")
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		t.Fatalf("write payload: %v", err)
	}
	foreignSig, err := signing.SignFile(otherKey, path, "version:9.9.9")
	if err != nil {
		t.Fatalf("SignFile: %v", err)
	}

	cases := map[string]struct {
		sig      string
		serveSig bool
		wantErr  string
	}{
		"signed by another key": {foreignSig, true, "refusing unverified binary"},
		"garbage signature":     {"not a signature at all\n", true, "refusing unverified binary"},
		"no signature served":   {"", false, "fetch signature"},
	}
	for name, tc := range cases {
		srv := updateStub(t, payload, tc.sig, tc.serveSig)
		err := config{ServerURL: srv.URL}.checkAndUpdate()
		if err == nil {
			t.Fatalf("%s: checkAndUpdate = nil, want a refusal", name)
		}
		if !strings.Contains(err.Error(), tc.wantErr) {
			t.Errorf("%s: error = %v, want it to mention %q", name, err, tc.wantErr)
		}
	}
}
