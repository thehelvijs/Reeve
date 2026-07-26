package main

import (
	"bufio"
	"encoding/json"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

// relay is a stub SMTP server that captures the messages the app sends.
type relay struct {
	host     string
	port     int
	messages chan string
}

// newRelay starts a stub SMTP listener that accepts every message it is handed.
func newRelay(t *testing.T) *relay {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	host, portStr, _ := net.SplitHostPort(ln.Addr().String())
	port, _ := strconv.Atoi(portStr)
	r := &relay{host: host, port: port, messages: make(chan string, 8)}
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go r.serve(conn)
		}
	}()
	return r
}

func (r *relay) serve(conn net.Conn) {
	defer conn.Close()
	br := bufio.NewReader(conn)
	bw := bufio.NewWriter(conn)
	reply := func(s string) {
		bw.WriteString(s + "\r\n")
		bw.Flush()
	}
	reply("220 stub ESMTP")
	var data strings.Builder
	inData := false
	for {
		line, err := br.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")
		if inData {
			if line == "." {
				inData = false
				r.messages <- data.String()
				data.Reset()
				reply("250 queued")
				continue
			}
			data.WriteString(line + "\n")
			continue
		}
		switch {
		case strings.HasPrefix(line, "EHLO"), strings.HasPrefix(line, "HELO"):
			reply("250-stub")
			reply("250 AUTH PLAIN")
		case strings.HasPrefix(line, "AUTH"):
			reply("235 authenticated")
		case line == "DATA":
			inData = true
			reply("354 send it")
		case line == "QUIT":
			reply("221 bye")
			return
		default:
			reply("250 ok")
		}
	}
}

// next returns the next captured message, failing the test if none arrives.
func (r *relay) next(t *testing.T) string {
	t.Helper()
	select {
	case m := <-r.messages:
		return m
	case <-time.After(5 * time.Second):
		t.Fatal("no message reached the relay")
		return ""
	}
}

func (r *relay) empty(t *testing.T) {
	t.Helper()
	select {
	case m := <-r.messages:
		t.Fatalf("unexpected message sent:\n%s", m)
	case <-time.After(300 * time.Millisecond):
	}
}

// configureRelay points the instance at the stub relay via the settings API.
func configureRelay(t *testing.T, ts *testServer, c *http.Client, r *relay) {
	t.Helper()
	body := map[string]any{"smtp": map[string]any{
		"enabled": true, "host": r.host, "port": r.port,
		"username": "reeve", "password": "smtp-secret",
		"from": "reeve@example.com", "tls": "none",
	}}
	resp, v := putSettings(t, ts, c, body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("configure relay status = %d, want 200", resp.StatusCode)
	}
	if !v.SMTP.Enabled || v.SMTP.Host != r.host || !v.SMTP.PasswordSet {
		t.Fatalf("smtp settings after save = %+v", v.SMTP)
	}
}

func TestSMTPSettingsRoundTripAndSecrecy(t *testing.T) {
	ts := newTestServer(t)
	c := ts.client(t)
	signup(t, ts, c, "boss@example.com", "password123")
	r := newRelay(t)
	configureRelay(t, ts, c, r)

	// The password is never returned, and never stored in the clear.
	_, data := ts.do(t, c, http.MethodGet, "/api/admin/settings", nil, nil)
	if strings.Contains(string(data), "smtp-secret") {
		t.Error("settings response leaked the smtp password")
	}
	raw, _ := ts.app.db.GetSetting(settingMailPassword)
	if strings.Contains(raw, "smtp-secret") {
		t.Error("smtp password stored in plaintext")
	}
	if got, ok := ts.app.sealedSetting(settingMailPassword); !ok || got != "smtp-secret" {
		t.Errorf("sealed smtp password = %q, %v", got, ok)
	}

	// Saving again without a password keeps the stored one.
	putSettings(t, ts, c, map[string]any{"smtp": map[string]any{
		"enabled": true, "host": r.host, "port": r.port, "username": "reeve",
		"from": "reeve@example.com", "tls": "none",
	}})
	if got, _ := ts.app.sealedSetting(settingMailPassword); got != "smtp-secret" {
		t.Errorf("password after a passwordless save = %q, want it unchanged", got)
	}
}

func TestSMTPSettingsRejectIncompleteWhenEnabled(t *testing.T) {
	ts := newTestServer(t)
	c := ts.client(t)
	signup(t, ts, c, "boss@example.com", "password123")

	cases := map[string]map[string]any{
		"no host":  {"enabled": true, "port": 587, "from": "v@example.com", "tls": "starttls"},
		"no from":  {"enabled": true, "host": "smtp.example.com", "port": 587, "tls": "starttls"},
		"bad port": {"enabled": true, "host": "smtp.example.com", "port": 0, "from": "v@example.com", "tls": "starttls"},
		"bad tls":  {"enabled": true, "host": "smtp.example.com", "port": 587, "from": "v@example.com", "tls": "ssl"},
	}
	for name, smtp := range cases {
		resp, _ := putSettings(t, ts, c, map[string]any{"smtp": smtp})
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("%s status = %d, want 400", name, resp.StatusCode)
		}
	}
	// Disabled relays may be saved half-filled; nothing depends on them yet.
	resp, _ := putSettings(t, ts, c, map[string]any{"smtp": map[string]any{"enabled": false, "host": "", "port": 587, "tls": "starttls"}})
	if resp.StatusCode != http.StatusOK {
		t.Errorf("disabled incomplete relay status = %d, want 200", resp.StatusCode)
	}
}

func TestTestEmailSendsToCallingAdmin(t *testing.T) {
	ts := newTestServer(t)
	c := ts.client(t)
	signup(t, ts, c, "boss@example.com", "password123")
	r := newRelay(t)
	configureRelay(t, ts, c, r)

	resp, _ := ts.do(t, c, http.MethodPost, "/api/admin/settings/test-email", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("test email status = %d, want 200", resp.StatusCode)
	}
	if msg := r.next(t); !strings.Contains(msg, "To: boss@example.com") {
		t.Errorf("test email not addressed to the admin:\n%s", msg)
	}
}

func TestTestEmailFailsWithoutRelay(t *testing.T) {
	ts := newTestServer(t)
	c := ts.client(t)
	signup(t, ts, c, "boss@example.com", "password123")
	resp, _ := ts.do(t, c, http.MethodPost, "/api/admin/settings/test-email", nil, nil)
	if resp.StatusCode != http.StatusBadGateway {
		t.Errorf("status without a relay = %d, want 502", resp.StatusCode)
	}
}

var tokenRe = regexp.MustCompile(`/reset\?token=([0-9a-f]{64})`)

// resetTokenFrom pulls the reset token out of the mailed link.
func resetTokenFrom(t *testing.T, msg string) string {
	t.Helper()
	m := tokenRe.FindStringSubmatch(msg)
	if m == nil {
		t.Fatalf("no reset link in message:\n%s", msg)
	}
	tok, err := url.QueryUnescape(m[1])
	if err != nil {
		t.Fatalf("unescape token: %v", err)
	}
	return tok
}

func forgot(t *testing.T, ts *testServer, email, ip string) *http.Response {
	t.Helper()
	resp, _ := ts.do(t, nil, http.MethodPost, "/api/auth/forgot",
		map[string]string{"email": email}, fromIP(ip))
	return resp
}

func doReset(t *testing.T, ts *testServer, token, password, ip string) *http.Response {
	t.Helper()
	resp, _ := ts.do(t, nil, http.MethodPost, "/api/auth/reset",
		map[string]string{"token": token, "password": password}, fromIP(ip))
	return resp
}

func TestForgotPasswordEndToEnd(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.client(t)
	signup(t, ts, admin, "boss@example.com", "password123")
	userClient := ts.client(t)
	signup(t, ts, userClient, "dev@example.com", "password123")
	r := newRelay(t)
	configureRelay(t, ts, admin, r)

	if resp := forgot(t, ts, "dev@example.com", "10.1.0.1"); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("forgot status = %d, want 204", resp.StatusCode)
	}
	token := resetTokenFrom(t, r.next(t))

	if resp := doReset(t, ts, token, "new-password", "10.1.0.1"); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("reset status = %d, want 204", resp.StatusCode)
	}
	if resp := login(t, ts, "dev@example.com", "new-password", "10.1.0.2"); resp.StatusCode != http.StatusOK {
		t.Errorf("login with the reset password status = %d, want 200", resp.StatusCode)
	}
	// The old session and the used token are both dead.
	resp, _ := ts.do(t, userClient, http.MethodGet, "/api/me", nil, nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("session after reset status = %d, want 401", resp.StatusCode)
	}
	if resp := doReset(t, ts, token, "another-password", "10.1.0.3"); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("replayed token status = %d, want 400", resp.StatusCode)
	}
}

// The endpoint must not reveal whether an address has an account.
func TestForgotPasswordDoesNotEnumerateAccounts(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.client(t)
	signup(t, ts, admin, "boss@example.com", "password123")
	r := newRelay(t)
	configureRelay(t, ts, admin, r)

	if resp := forgot(t, ts, "nobody@example.com", "10.2.0.1"); resp.StatusCode != http.StatusNoContent {
		t.Errorf("unknown address status = %d, want 204", resp.StatusCode)
	}
	r.empty(t)
}

func TestForgotPasswordThrottledPerIP(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.client(t)
	signup(t, ts, admin, "boss@example.com", "password123")
	r := newRelay(t)
	configureRelay(t, ts, admin, r)

	for i := 0; i < 5; i++ {
		if resp := forgot(t, ts, "boss@example.com", "10.3.0.1"); resp.StatusCode != http.StatusNoContent {
			t.Fatalf("attempt %d status = %d, want 204", i+1, resp.StatusCode)
		}
		r.next(t)
	}
	if resp := forgot(t, ts, "boss@example.com", "10.3.0.1"); resp.StatusCode != http.StatusTooManyRequests {
		t.Errorf("attempt 6 status = %d, want 429", resp.StatusCode)
	}
}

func TestResetRejectsBadTokens(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.client(t)
	signup(t, ts, admin, "boss@example.com", "password123")

	if resp := doReset(t, ts, "deadbeef", "new-password", "10.4.0.1"); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("unknown token status = %d, want 400", resp.StatusCode)
	}
	if resp := doReset(t, ts, "", "new-password", "10.4.0.2"); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("empty token status = %d, want 400", resp.StatusCode)
	}
	if resp := doReset(t, ts, "deadbeef", "", "10.4.0.3"); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("empty password status = %d, want 400", resp.StatusCode)
	}
}

func TestExpiredResetTokenIsRejected(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.client(t)
	signup(t, ts, admin, "boss@example.com", "password123")
	u, err := ts.app.db.GetUserByEmail("boss@example.com")
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	token, hash, err := newResetToken()
	if err != nil {
		t.Fatalf("newResetToken: %v", err)
	}
	past := time.Now().UTC().Add(-time.Minute)
	if _, err := ts.app.db.CreatePasswordReset(u.ID, hash, past, past.Add(-time.Hour)); err != nil {
		t.Fatalf("CreatePasswordReset: %v", err)
	}
	if resp := doReset(t, ts, token, "new-password", "10.5.0.1"); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expired token status = %d, want 400", resp.StatusCode)
	}
}

// The login screen only offers "forgot password" once mail actually works.
func TestAuthStatusReportsPasswordResetAvailability(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.client(t)
	signup(t, ts, admin, "boss@example.com", "password123")

	readStatus := func() map[string]bool {
		_, data := ts.do(t, nil, http.MethodGet, "/api/auth/status", nil, nil)
		var s map[string]bool
		json.Unmarshal(data, &s)
		return s
	}
	if readStatus()["password_reset_enabled"] {
		t.Error("password reset advertised with no relay configured")
	}
	configureRelay(t, ts, admin, newRelay(t))
	if !readStatus()["password_reset_enabled"] {
		t.Error("password reset not advertised after configuring a relay")
	}
}
