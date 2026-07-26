package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/thehelvijs/Reeve/server/internal/auth"
)

// sshBody is a complete, well-formed SSH target payload.
func sshBody() map[string]any {
	return map[string]any{
		"address":     "192.168.1.20",
		"port":        22,
		"username":    "deploy",
		"password":    "hunter2",
		"fingerprint": "SHA256:abcdefghijklmnopqrstuvwxyz0123456789ABCDEFG",
	}
}

// createHostFor makes a host and returns its id and one-time enrollment token.
func createHostFor(t *testing.T, ts *testServer, c *http.Client, name string) (id, token string) {
	t.Helper()
	resp, data := ts.do(t, c, http.MethodPost, "/api/admin/hosts", map[string]any{"name": name}, nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create host status = %d: %s", resp.StatusCode, data)
	}
	var out struct {
		Host struct {
			ID string `json:"id"`
		} `json:"host"`
		EnrollToken string `json:"enroll_token"`
	}
	json.Unmarshal(data, &out)
	if out.Host.ID == "" || out.EnrollToken == "" {
		t.Fatalf("no host id or token in %s", data)
	}
	return out.Host.ID, out.EnrollToken
}

func TestSSHInstallRejectsIncompleteTargets(t *testing.T) {
	ts := newTestServer(t)
	c := ts.client(t)
	signup(t, ts, c, "boss@example.com", "password123")
	ts.app.cfg.PublicURL = ts.srv.URL
	id, _ := createHostFor(t, ts, c, "db-1")

	cases := map[string]func(map[string]any){
		"no address":     func(b map[string]any) { delete(b, "address") },
		"no username":    func(b map[string]any) { delete(b, "username") },
		"no credentials": func(b map[string]any) { delete(b, "password") },
		"no fingerprint": func(b map[string]any) { delete(b, "fingerprint") },
	}
	for _, path := range []string{"ssh-install", "ssh-uninstall"} {
		for name, mutate := range cases {
			body := sshBody()
			mutate(body)
			resp, _ := ts.do(t, c, http.MethodPost, "/api/admin/hosts/"+id+"/"+path, body, nil)
			if resp.StatusCode != http.StatusBadRequest {
				t.Errorf("%s %s status = %d, want 400", path, name, resp.StatusCode)
			}
		}
	}
}

// A loopback origin resolves to an address the remote agent cannot dial, so the
// push is refused rather than enrolling a host that can never report.
func TestSSHInstallRejectsLoopbackServerURL(t *testing.T) {
	ts := newTestServer(t)
	c := ts.client(t)
	signup(t, ts, c, "boss@example.com", "password123")
	id, _ := createHostFor(t, ts, c, "db-1")
	ts.app.cfg.PublicURL = ""

	resp, data := ts.do(t, c, http.MethodPost, "/api/admin/hosts/"+id+"/ssh-install", sshBody(), nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400: %s", resp.StatusCode, data)
	}
	var out map[string]string
	json.Unmarshal(data, &out)
	if out["code"] != "unreachable_server_url" {
		t.Errorf("code = %q, want unreachable_server_url", out["code"])
	}
}

func TestReachableFromOtherHosts(t *testing.T) {
	for base, want := range map[string]bool{
		"http://192.168.1.50:8080":  true,
		"https://reeve.example.com": true,
		"http://10.0.0.2":           true,
		"http://[fd00::1]:8080":     true,
		"http://127.0.0.1:8080":     false,
		"http://localhost:8080":     false,
		"http://0.0.0.0:8080":       false,
		"http://[::1]:8080":         false,
		"":                          false,
	} {
		if got := reachableFromOtherHosts(base); got != want {
			t.Errorf("reachableFromOtherHosts(%q) = %v, want %v", base, got, want)
		}
	}
}

func TestSSHEndpointsAreAdminOnlyAndScopedToAHost(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.client(t)
	signup(t, ts, admin, "boss@example.com", "password123")
	ts.app.cfg.PublicURL = ts.srv.URL
	id, _ := createHostFor(t, ts, admin, "db-1")
	basic := ts.client(t)
	signup(t, ts, basic, "dev@example.com", "password123")

	for _, path := range []string{
		"/api/admin/hosts/" + id + "/ssh-install",
		"/api/admin/hosts/" + id + "/ssh-uninstall",
	} {
		if resp, _ := ts.do(t, basic, http.MethodPost, path, sshBody(), nil); resp.StatusCode != http.StatusForbidden {
			t.Errorf("basic user on %s status = %d, want 403", path, resp.StatusCode)
		}
		if resp, _ := ts.do(t, nil, http.MethodPost, path, sshBody(), nil); resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("anonymous on %s status = %d, want 401", path, resp.StatusCode)
		}
	}
	if resp, _ := ts.do(t, admin, http.MethodPost, "/api/admin/ssh-probe", map[string]any{"address": "10.0.0.1"}, nil); resp.StatusCode == http.StatusForbidden {
		t.Error("admin was refused the probe endpoint")
	}
	if resp, _ := ts.do(t, basic, http.MethodPost, "/api/admin/ssh-probe", map[string]any{"address": "10.0.0.1"}, nil); resp.StatusCode != http.StatusForbidden {
		t.Error("probe endpoint is not admin gated")
	}
	// Unknown host, valid payload.
	resp, _ := ts.do(t, admin, http.MethodPost, "/api/admin/hosts/nope/ssh-install", sshBody(), nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("unknown host status = %d, want 404", resp.StatusCode)
	}
}

func TestSSHProbeValidatesAddress(t *testing.T) {
	ts := newTestServer(t)
	c := ts.client(t)
	signup(t, ts, c, "boss@example.com", "password123")

	resp, _ := ts.do(t, c, http.MethodPost, "/api/admin/ssh-probe", map[string]any{"address": "  "}, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("blank address status = %d, want 400", resp.StatusCode)
	}
}

// The pushed binary must be the build for the host's own architecture, checksum
// included so the remote installer can confirm what landed.
func TestAgentBinaryForUname(t *testing.T) {
	ts := newTestServer(t)

	data, sum, err := ts.app.agentBinaryForUname("x86_64")
	if err != nil {
		t.Fatalf("agentBinaryForUname: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("no bytes returned for x86_64")
	}
	want := sha256.Sum256(data)
	if sum != hex.EncodeToString(want[:]) {
		t.Errorf("checksum = %q, want the sha256 of the returned bytes", sum)
	}
	// aarch64 maps to a different build than x86_64.
	armData, _, err := ts.app.agentBinaryForUname("aarch64")
	if err != nil {
		t.Fatalf("agentBinaryForUname(aarch64): %v", err)
	}
	if string(armData) == string(data) {
		t.Error("aarch64 and x86_64 returned the same binary")
	}
	if _, _, err := ts.app.agentBinaryForUname("sparc64"); err == nil {
		t.Error("an unsupported architecture was accepted")
	}
}

// Every push mints a fresh enrollment token, so a reinstall never depends on
// anyone having kept the old one, and the old one stops working.
func TestSSHInstallRotatesTheEnrollmentToken(t *testing.T) {
	ts := newTestServer(t)
	c := ts.client(t)
	signup(t, ts, c, "boss@example.com", "password123")
	ts.app.cfg.PublicURL = ts.srv.URL
	id, oldToken := createHostFor(t, ts, c, "db-1")

	if _, err := ts.app.db.GetHostByTokenHash(auth.HashToken(oldToken)); err != nil {
		t.Fatalf("precondition: original token does not resolve: %v", err)
	}
	// The SSH attempt fails (nothing listening), but the token must already be
	// rotated: the installer it would have run is the only holder of the new one.
	body := sshBody()
	body["address"] = "127.0.0.1"
	body["port"] = 1
	resp, _ := ts.do(t, c, http.MethodPost, "/api/admin/hosts/"+id+"/ssh-install", body, nil)
	if resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502 from an unreachable host", resp.StatusCode)
	}
	if _, err := ts.app.db.GetHostByTokenHash(auth.HashToken(oldToken)); err == nil {
		t.Error("the previous enrollment token still resolves after an install push")
	}
}
