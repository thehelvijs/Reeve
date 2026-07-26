package main

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func newDownloadApp(agentFS, scriptFS fstest.MapFS) *app {
	return &app{agentFS: agentFS, scriptFS: scriptFS}
}

func TestServeAgentBinaryKnownArch(t *testing.T) {
	bin := []byte("FAKEELF-amd64")
	a := newDownloadApp(
		fstest.MapFS{"agent-linux-amd64": {Data: bin}},
		fstest.MapFS{},
	)
	req := httptest.NewRequest(http.MethodGet, "/dl/agent-linux-amd64", nil)
	req.SetPathValue("arch", "amd64")
	rr := httptest.NewRecorder()
	a.handleAgentBinary(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
	if rr.Body.String() != string(bin) {
		t.Errorf("body mismatch")
	}
}

func TestServeAgentBinaryUnknownArch404(t *testing.T) {
	a := newDownloadApp(fstest.MapFS{}, fstest.MapFS{})
	req := httptest.NewRequest(http.MethodGet, "/dl/agent-linux-sparc", nil)
	req.SetPathValue("arch", "sparc")
	rr := httptest.NewRecorder()
	a.handleAgentBinary(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rr.Code)
	}
}

func TestServeAgentChecksumMatchesBytes(t *testing.T) {
	bin := []byte("FAKEELF-arm64")
	sum := sha256.Sum256(bin)
	a := newDownloadApp(
		fstest.MapFS{"agent-linux-arm64": {Data: bin}},
		fstest.MapFS{},
	)
	req := httptest.NewRequest(http.MethodGet, "/dl/agent-linux-arm64.sha256", nil)
	req.SetPathValue("arch", "arm64")
	rr := httptest.NewRecorder()
	a.handleAgentChecksum(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
	if !strings.HasPrefix(rr.Body.String(), hex.EncodeToString(sum[:])) {
		t.Errorf("checksum body = %q", rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "agent-linux-arm64") {
		t.Errorf("checksum missing filename: %q", rr.Body.String())
	}
}

func TestRoutesDownloadDispatch(t *testing.T) {
	bin := []byte("FAKEELF-amd64")
	a := newDownloadApp(
		fstest.MapFS{"agent-linux-amd64": {Data: bin}},
		fstest.MapFS{"scripts/install.sh": {Data: []byte("#!/usr/bin/env bash\necho hi\n")}},
	)
	mux := a.routes()

	cases := []struct {
		path       string
		wantStatus int
		wantBody   string
	}{
		{"/dl/agent-linux-amd64", http.StatusOK, string(bin)},
		{"/dl/agent-linux-amd64.sha256", http.StatusOK, ""},
		{"/dl/agent-linux-sparc", http.StatusNotFound, ""},
		{"/dl/bogus-file", http.StatusNotFound, ""},
		{"/install.sh", http.StatusOK, ""},
	}
	for _, tc := range cases {
		req := httptest.NewRequest(http.MethodGet, tc.path, nil)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)
		if rr.Code != tc.wantStatus {
			t.Errorf("%s: status = %d, want %d", tc.path, rr.Code, tc.wantStatus)
		}
		if tc.wantBody != "" && rr.Body.String() != tc.wantBody {
			t.Errorf("%s: body = %q, want %q", tc.path, rr.Body.String(), tc.wantBody)
		}
	}
}

func TestServeInstallScript(t *testing.T) {
	a := newDownloadApp(fstest.MapFS{}, fstest.MapFS{
		"scripts/install.sh": {Data: []byte("#!/usr/bin/env bash\necho hi\n")},
	})
	req := httptest.NewRequest(http.MethodGet, "/install.sh", nil)
	rr := httptest.NewRecorder()
	a.handleInstallScript(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "bash") {
		t.Errorf("install.sh body wrong: %q", rr.Body.String())
	}
	if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "text") {
		t.Errorf("content-type = %q", ct)
	}
}

// The SSH uninstall push copies this script over and runs it, so a build that
// forgot to stage it must 404 rather than serve an empty file the remote shell
// would happily execute.
func TestServeUninstallScript(t *testing.T) {
	a := newDownloadApp(fstest.MapFS{}, fstest.MapFS{
		"scripts/uninstall.sh": {Data: []byte("#!/usr/bin/env bash\nsystemctl stop reeve-agent\n")},
	})
	req := httptest.NewRequest(http.MethodGet, "/uninstall.sh", nil)
	rr := httptest.NewRecorder()
	a.handleUninstallScript(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "reeve-agent") {
		t.Errorf("uninstall.sh body wrong: %q", rr.Body.String())
	}
	if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "shellscript") {
		t.Errorf("content-type = %q, want a shellscript type", ct)
	}

	missing := newDownloadApp(fstest.MapFS{}, fstest.MapFS{})
	rr = httptest.NewRecorder()
	missing.handleUninstallScript(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("unstaged script = %d, want 404", rr.Code)
	}
}

// The agent asks for a signature next to the binary; without one it refuses to
// update, so serving and 404ing both have to behave predictably.
func TestServeAgentSignature(t *testing.T) {
	sig := "untrusted comment: sig\nAAAA\ntrusted comment: version:1.2.3\nBBBB\n"
	a := newDownloadApp(
		fstest.MapFS{
			"agent-linux-amd64":         {Data: []byte("FAKEELF-amd64")},
			"agent-linux-amd64.minisig": {Data: []byte(sig)},
		},
		fstest.MapFS{},
	)
	mux := a.routes()

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/dl/agent-linux-amd64.minisig", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	if rr.Body.String() != sig {
		t.Errorf("body = %q, want the stored signature", rr.Body.String())
	}

	// An arch with no signature, and an unsupported arch, are both 404.
	for _, path := range []string{"/dl/agent-linux-arm64.minisig", "/dl/agent-linux-sparc.minisig"} {
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, path, nil))
		if rr.Code != http.StatusNotFound {
			t.Errorf("%s status = %d, want 404", path, rr.Code)
		}
	}
}
