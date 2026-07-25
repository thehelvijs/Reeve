package sshinstall

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"

	"golang.org/x/crypto/ssh"
)

// fakeHost is an in-process SSH server that records the commands it is asked to
// run and replies with canned output, so the install flow is exercised for real.
type fakeHost struct {
	addr        string
	fingerprint string
	password    string

	mu       sync.Mutex
	commands []string
	stdins   map[string]string
	failures map[string]bool
	replies  map[string]string
}

func newFakeHost(t *testing.T) *fakeHost {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("host key: %v", err)
	}
	signer, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatalf("signer: %v", err)
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	h := &fakeHost{
		addr:        ln.Addr().String(),
		fingerprint: ssh.FingerprintSHA256(signer.PublicKey()),
		password:    "hunter2",
		stdins:      map[string]string{},
		failures:    map[string]bool{},
		replies:     map[string]string{"uname -m": "x86_64\n", "mktemp -d /tmp/reeve-install.XXXXXX": "/tmp/reeve-install.abc123\n"},
	}
	cfg := &ssh.ServerConfig{
		PasswordCallback: func(_ ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			if string(pass) != h.password {
				return nil, errors.New("wrong password")
			}
			return &ssh.Permissions{}, nil
		},
	}
	cfg.AddHostKey(signer)

	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go h.serve(conn, cfg)
		}
	}()
	return h
}

func (h *fakeHost) port() int {
	_, p, _ := net.SplitHostPort(h.addr)
	n, _ := strconv.Atoi(p)
	return n
}

func (h *fakeHost) serve(conn net.Conn, cfg *ssh.ServerConfig) {
	defer conn.Close()
	sshConn, chans, reqs, err := ssh.NewServerConn(conn, cfg)
	if err != nil {
		return
	}
	defer sshConn.Close()
	go ssh.DiscardRequests(reqs)

	for newCh := range chans {
		if newCh.ChannelType() != "session" {
			newCh.Reject(ssh.UnknownChannelType, "only sessions")
			continue
		}
		ch, chReqs, err := newCh.Accept()
		if err != nil {
			return
		}
		go h.handleSession(ch, chReqs)
	}
}

func (h *fakeHost) handleSession(ch ssh.Channel, reqs <-chan *ssh.Request) {
	defer ch.Close()
	for req := range reqs {
		if req.Type != "exec" {
			req.Reply(false, nil)
			continue
		}
		var payload struct{ Command string }
		ssh.Unmarshal(req.Payload, &payload)
		req.Reply(true, nil)

		stdin, _ := io.ReadAll(ch)
		h.mu.Lock()
		h.commands = append(h.commands, payload.Command)
		h.stdins[payload.Command] = string(stdin)
		reply, hasReply := h.replies[payload.Command]
		fail := false
		for substr := range h.failures {
			if strings.Contains(payload.Command, substr) {
				fail = true
			}
		}
		h.mu.Unlock()

		if hasReply {
			io.WriteString(ch, reply)
		}
		status := uint32(0)
		if fail {
			io.WriteString(ch.Stderr(), "command failed\n")
			status = 1
		}
		ch.SendRequest("exit-status", false, ssh.Marshal(struct{ Status uint32 }{status}))
		return
	}
}

// ran returns the last command containing substr, so a match on the upload of a
// script does not shadow the command that ran it.
func (h *fakeHost) ran(substr string) string {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := ""
	for _, c := range h.commands {
		if strings.Contains(c, substr) {
			out = c
		}
	}
	return out
}

func (h *fakeHost) stdinFor(substr string) string {
	h.mu.Lock()
	defer h.mu.Unlock()
	for c, in := range h.stdins {
		if strings.Contains(c, substr) {
			return in
		}
	}
	return ""
}

func (h *fakeHost) failCommand(substr string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.failures[substr] = true
}

func provider(t *testing.T) BinaryProvider {
	t.Helper()
	return func(arch string) ([]byte, string, error) {
		if arch != "x86_64" {
			return nil, "", errors.New("unexpected arch " + arch)
		}
		return []byte("AGENTBYTES"), "abc123sum", nil
	}
}

func testTarget(h *fakeHost) Target {
	return Target{
		Address:     "127.0.0.1",
		Port:        h.port(),
		User:        "deploy",
		Password:    h.password,
		Fingerprint: h.fingerprint,
	}
}

var scripts = Scripts{Install: []byte("#!/bin/bash\necho installing\n"), Uninstall: []byte("#!/bin/bash\necho removing\n")}

func TestProbeReturnsFingerprint(t *testing.T) {
	h := newFakeHost(t)
	fp, keyType, err := Probe(context.Background(), "127.0.0.1", h.port())
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if fp != h.fingerprint {
		t.Errorf("fingerprint = %q, want %q", fp, h.fingerprint)
	}
	if keyType != "ssh-ed25519" {
		t.Errorf("key type = %q", keyType)
	}
}

func TestProbeUnreachable(t *testing.T) {
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	_, p, _ := net.SplitHostPort(ln.Addr().String())
	ln.Close()
	port, _ := strconv.Atoi(p)
	if _, _, err := Probe(context.Background(), "127.0.0.1", port); err == nil {
		t.Fatal("Probe on a closed port returned nil error")
	}
}

func TestInstallUploadsAndRunsInstaller(t *testing.T) {
	h := newFakeHost(t)
	out, err := Install(context.Background(), testTarget(h), provider(t), scripts,
		map[string]string{"REEVE_SERVER_URL": "http://reeve.lan:8080", "REEVE_AGENT_TOKEN": "tok-123"})
	if err != nil {
		t.Fatalf("Install: %v\n%s", err, out)
	}

	if h.ran("uname -m") == "" {
		t.Error("architecture was never detected")
	}
	if got := h.stdinFor("cat > /tmp/reeve-install.abc123/agent"); got != "AGENTBYTES" {
		t.Errorf("uploaded agent bytes = %q", got)
	}
	if got := h.stdinFor("cat > /tmp/reeve-install.abc123/install.sh"); !strings.Contains(got, "installing") {
		t.Errorf("uploaded install script = %q", got)
	}
	cmd := h.ran("bash /tmp")
	for _, want := range []string{
		"sudo -n ",
		"REEVE_AGENT_BINARY='/tmp/reeve-install.abc123/agent'",
		"REEVE_AGENT_SHA256='abc123sum'",
		"REEVE_AGENT_TOKEN='tok-123'",
		"REEVE_SERVER_URL='http://reeve.lan:8080'",
		"bash /tmp/reeve-install.abc123/install.sh",
	} {
		if !strings.Contains(cmd, want) {
			t.Errorf("install command %q missing %q", cmd, want)
		}
	}
	if h.ran("rm -rf /tmp/reeve-install.abc123") == "" {
		t.Error("staging directory was not cleaned up")
	}
}

func TestInstallUsesSudoPasswordWhenGiven(t *testing.T) {
	h := newFakeHost(t)
	target := testTarget(h)
	target.SudoPassword = "sudo-secret"
	if _, err := Install(context.Background(), target, provider(t), scripts, nil); err != nil {
		t.Fatalf("Install: %v", err)
	}
	cmd := h.ran("bash /tmp")
	if !strings.Contains(cmd, "sudo -S -p ''") {
		t.Errorf("command = %q, want sudo reading the password from stdin", cmd)
	}
	if got := h.stdinFor("bash /tmp"); got != "sudo-secret\n" {
		t.Errorf("sudo stdin = %q", got)
	}
}

func TestInstallAsRootSkipsSudo(t *testing.T) {
	h := newFakeHost(t)
	target := testTarget(h)
	target.User = "root"
	if _, err := Install(context.Background(), target, provider(t), scripts, nil); err != nil {
		t.Fatalf("Install: %v", err)
	}
	if cmd := h.ran("bash /tmp"); strings.Contains(cmd, "sudo") {
		t.Errorf("command = %q, want no sudo for root", cmd)
	}
}

// A mismatched fingerprint means the host is not the one the admin confirmed, so
// no credential may be sent to it.
func TestInstallRefusesWrongFingerprint(t *testing.T) {
	h := newFakeHost(t)
	target := testTarget(h)
	target.Fingerprint = "SHA256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	_, err := Install(context.Background(), target, provider(t), scripts, nil)
	if err == nil {
		t.Fatal("Install with a wrong fingerprint returned nil error")
	}
	if !strings.Contains(err.Error(), "fingerprint") {
		t.Errorf("error = %v, want it to name the fingerprint mismatch", err)
	}
	if h.ran("uname -m") != "" {
		t.Error("commands ran on a host whose key did not match")
	}
}

func TestInstallValidatesTarget(t *testing.T) {
	h := newFakeHost(t)
	cases := map[string]func(*Target){
		"no address":     func(t *Target) { t.Address = "" },
		"no user":        func(t *Target) { t.User = "" },
		"no credentials": func(t *Target) { t.Password = ""; t.PrivateKey = "" },
		"no fingerprint": func(t *Target) { t.Fingerprint = "" },
		"bad port":       func(t *Target) { t.Port = 99999 },
	}
	for name, mutate := range cases {
		target := testTarget(h)
		mutate(&target)
		if _, err := Install(context.Background(), target, provider(t), scripts, nil); err == nil {
			t.Errorf("%s: Install = nil error, want a rejection", name)
		}
	}
}

func TestInstallReportsRemoteFailureWithOutput(t *testing.T) {
	h := newFakeHost(t)
	h.failCommand("bash /tmp/reeve-install.abc123/install.sh")
	out, err := Install(context.Background(), testTarget(h), provider(t), scripts, nil)
	if err == nil {
		t.Fatal("Install returned nil despite a failing installer")
	}
	if !strings.Contains(out, "command failed") {
		t.Errorf("output = %q, want the remote stderr included", out)
	}
}

func TestInstallSurfacesUnsupportedArch(t *testing.T) {
	h := newFakeHost(t)
	h.mu.Lock()
	h.replies["uname -m"] = "sparc64\n"
	h.mu.Unlock()
	_, err := Install(context.Background(), testTarget(h), func(arch string) ([]byte, string, error) {
		return nil, "", errors.New("unsupported host architecture " + arch)
	}, scripts, nil)
	if err == nil || !strings.Contains(err.Error(), "sparc64") {
		t.Fatalf("error = %v, want it to name the unsupported arch", err)
	}
}

// A value carrying a quote could break out of the remote command, so it is
// refused rather than escaped.
func TestInstallRejectsUnsafeEnvValues(t *testing.T) {
	h := newFakeHost(t)
	_, err := Install(context.Background(), testTarget(h), provider(t), scripts,
		map[string]string{"REEVE_AGENT_TOKEN": "tok'; rm -rf /"})
	if err == nil {
		t.Fatal("Install accepted a token containing a quote")
	}
	if h.ran("bash /tmp") != "" {
		t.Error("installer ran despite an unsafe environment value")
	}
}

func TestUninstallRunsUninstaller(t *testing.T) {
	h := newFakeHost(t)
	out, err := Uninstall(context.Background(), testTarget(h), scripts)
	if err != nil {
		t.Fatalf("Uninstall: %v\n%s", err, out)
	}
	if got := h.stdinFor("cat > /tmp/reeve-install.abc123/uninstall.sh"); !strings.Contains(got, "removing") {
		t.Errorf("uploaded uninstall script = %q", got)
	}
	cmd := h.ran("bash /tmp/reeve-install.abc123/uninstall.sh")
	if !strings.Contains(cmd, "sudo -n") {
		t.Errorf("uninstall command = %q, want it under sudo", cmd)
	}
	if h.ran("rm -rf /tmp/reeve-install.abc123") == "" {
		t.Error("staging directory was not cleaned up")
	}
}

func TestUninstallRefusesWrongFingerprint(t *testing.T) {
	h := newFakeHost(t)
	target := testTarget(h)
	target.Fingerprint = "SHA256:nope"
	if _, err := Uninstall(context.Background(), target, scripts); err == nil {
		t.Fatal("Uninstall with a wrong fingerprint returned nil error")
	}
}

func TestInstallWithPrivateKey(t *testing.T) {
	h := newFakeHost(t)
	target := testTarget(h)
	target.Password = ""
	target.PrivateKey = "not a key"
	_, err := Install(context.Background(), target, provider(t), scripts, nil)
	if err == nil || !strings.Contains(err.Error(), "private key") {
		t.Fatalf("error = %v, want a private key parse failure", err)
	}
}

func TestEnvAssignmentsIsDeterministicAndQuoted(t *testing.T) {
	got, err := envAssignments(map[string]string{"B": "two", "A": "one"})
	if err != nil {
		t.Fatalf("envAssignments: %v", err)
	}
	if got != " A='one' B='two'" {
		t.Errorf("envAssignments = %q", got)
	}
	if _, err := envAssignments(map[string]string{"A": "has'quote"}); err == nil {
		t.Error("a quoted value was accepted")
	}
	if _, err := envAssignments(map[string]string{"A": "has\nnewline"}); err == nil {
		t.Error("a newline value was accepted")
	}
}
