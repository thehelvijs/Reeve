package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/thehelvijs/Reeve/server/internal/auth"
)

// revealStored returns the one stored credential for a host, decrypted.
func revealStored(t *testing.T, ts *testServer, hostID string) (ctype string, secret map[string]string) {
	t.Helper()
	creds, err := ts.app.db.ListCredentials(hostID)
	if err != nil {
		t.Fatal(err)
	}
	if len(creds) != 1 {
		t.Fatalf("stored credentials = %d, want 1", len(creds))
	}
	ct, nonce, err := ts.app.db.GetCredentialSecret(creds[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	plain, err := ts.app.cipher.Open(ct, nonce, credentialAAD(creds[0].HostID))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(plain, &secret); err != nil {
		t.Fatal(err)
	}
	return creds[0].Type, secret
}

func TestInstallSavesAPasswordCredential(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	host := newHost(t, ts, admin, "box")
	me, err := ts.app.db.GetUserByEmail("boss@example.com")
	if err != nil {
		t.Fatal(err)
	}
	p := auth.Principal{UserID: me.ID, Email: me.Email, Role: me.Role}

	in := sshTargetInput{Username: "helvijs", Password: "hunter2", SudoPassword: "hunter2"}
	if err := ts.app.saveInstallCredential(host, in, p); err != nil {
		t.Fatalf("save: %v", err)
	}

	ctype, secret := revealStored(t, ts, host)
	if ctype != "ssh_password" {
		t.Errorf("type = %q, want ssh_password", ctype)
	}
	if secret["username"] != "helvijs" || secret["password"] != "hunter2" {
		t.Errorf("secret = %v", secret)
	}
	// The sudo password is the same one; keeping a second copy of it is noise.
	if _, ok := secret["sudo_password"]; ok {
		t.Error("an identical sudo password was stored twice")
	}
	// It is ciphertext on disk, like every other credential.
	var raw []byte
	ts.app.db.SQL().QueryRow(`SELECT ciphertext FROM credentials LIMIT 1`).Scan(&raw)
	if bytes.Contains(raw, []byte("hunter2")) {
		t.Error("the password is in the database in plaintext")
	}
}

// The admin who ran the install gets an explicit grant. They can already reveal
// by role; the grant is what survives losing it, and what the audit reads back.
func TestInstallGrantsTheInstallerAccess(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	host := newHost(t, ts, admin, "box")
	me, _ := ts.app.db.GetUserByEmail("boss@example.com")
	p := auth.Principal{UserID: me.ID, Email: me.Email, Role: me.Role}

	if err := ts.app.saveInstallCredential(host, sshTargetInput{Username: "u", Password: "pw"}, p); err != nil {
		t.Fatal(err)
	}
	ok, err := ts.app.db.HasCredentialAccess(me.ID, host)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("the installing admin has no standing grant on the credential they saved")
	}
	if n := countRows(t, ts, `SELECT COUNT(*) FROM grant_audit WHERE host_id = ? AND action='grant'`, host); n != 1 {
		t.Errorf("grant audit rows = %d, want 1", n)
	}
}

func TestInstallSavesAKeyCredential(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	host := newHost(t, ts, admin, "box")
	me, _ := ts.app.db.GetUserByEmail("boss@example.com")
	p := auth.Principal{UserID: me.ID, Email: me.Email, Role: me.Role}

	in := sshTargetInput{
		Username: "root", PrivateKey: "-----BEGIN OPENSSH PRIVATE KEY-----\nabc\n",
		Passphrase: "phrase", SudoPassword: "sudopw",
	}
	if err := ts.app.saveInstallCredential(host, in, p); err != nil {
		t.Fatal(err)
	}
	ctype, secret := revealStored(t, ts, host)
	if ctype != "ssh_key" {
		t.Errorf("type = %q, want ssh_key", ctype)
	}
	if secret["private_key"] == "" || secret["passphrase"] != "phrase" {
		t.Errorf("secret = %v", secret)
	}
	// A sudo password that differs from the login is needed to use this, so it
	// is kept alongside.
	if secret["sudo_password"] != "sudopw" {
		t.Errorf("sudo password not stored: %v", secret)
	}
}

// A key-agent or NOPASSWD install has no secret worth keeping, and must not
// leave an empty credential behind for someone to find and trust.
func TestInstallWithNothingToSaveStoresNothing(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	host := newHost(t, ts, admin, "box")
	me, _ := ts.app.db.GetUserByEmail("boss@example.com")
	p := auth.Principal{UserID: me.ID, Email: me.Email, Role: me.Role}

	if err := ts.app.saveInstallCredential(host, sshTargetInput{Username: "root"}, p); err != nil {
		t.Fatal(err)
	}
	creds, _ := ts.app.db.ListCredentials(host)
	if len(creds) != 0 {
		t.Errorf("stored %d credentials for an install with no secret", len(creds))
	}
}

// Saving is the default, so an absent field must mean save. The checkbox in the
// modal is phrased as the opt-out for exactly this reason.
func TestSkipCredentialSaveDefaultsToSaving(t *testing.T) {
	var in sshTargetInput
	if err := json.Unmarshal([]byte(`{"address":"h","username":"u","password":"p"}`), &in); err != nil {
		t.Fatal(err)
	}
	if in.SkipCredentialSave {
		t.Error("an absent skip_credential_save must mean the credential is saved")
	}
	if err := json.Unmarshal([]byte(`{"skip_credential_save":true}`), &in); err != nil {
		t.Fatal(err)
	}
	if !in.SkipCredentialSave {
		t.Error("skip_credential_save:true was not honoured")
	}
}

// A failed install must not leave the credential behind: nothing proved it
// works, and the host is not enrolled.
func TestFailedInstallSavesNoCredential(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	host := newHost(t, ts, admin, "box")
	ts.app.cfg.PublicURL = ts.srv.URL

	body := sshBody()
	body["address"] = "127.0.0.1"
	body["port"] = 1 // nothing listening
	resp, _ := ts.do(t, admin, http.MethodPost, "/api/admin/hosts/"+host+"/ssh-install", body, nil)
	if resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502", resp.StatusCode)
	}
	creds, _ := ts.app.db.ListCredentials(host)
	if len(creds) != 0 {
		t.Errorf("a failed install stored %d credentials", len(creds))
	}
}
