package main

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/thehelvijs/Reeve/server/internal/store"
)

// downloadBackup pulls the backup bytes over the admin endpoint.
func downloadBackup(t *testing.T, ts *testServer, c *http.Client) (*http.Response, []byte) {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, ts.srv.URL+"/api/v1/admin/backup", nil)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	resp, err := c.Do(req)
	if err != nil {
		t.Fatalf("download: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return resp, body
}

// uploadRestore posts bytes as a staged restore.
func uploadRestore(t *testing.T, ts *testServer, c *http.Client, content []byte) (*http.Response, []byte) {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("backup", "restore.db")
	if err != nil {
		t.Fatalf("form file: %v", err)
	}
	fw.Write(content)
	mw.Close()

	req, err := http.NewRequest(http.MethodPost, ts.srv.URL+"/api/v1/admin/restore", &buf)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	resp, err := c.Do(req)
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return resp, data
}

func TestBackupDownloadIsAUsableDatabase(t *testing.T) {
	ts := newTestServer(t)
	c := ts.client(t)
	signup(t, ts, c, "boss@example.com", "password123")

	resp, body := downloadBackup(t, ts, c)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if cd := resp.Header.Get("Content-Disposition"); !bytes.Contains([]byte(cd), []byte("attachment")) {
		t.Errorf("Content-Disposition = %q, want an attachment", cd)
	}
	if !bytes.HasPrefix(body, []byte("SQLite format 3\x00")) {
		t.Fatal("downloaded file is not a SQLite database")
	}

	// The copy opens and carries the data the live instance had.
	path := filepath.Join(t.TempDir(), "copy.db")
	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatalf("write copy: %v", err)
	}
	db, err := store.Open(path)
	if err != nil {
		t.Fatalf("open copy: %v", err)
	}
	defer db.Close()
	if _, err := db.GetUserByEmail("boss@example.com"); err != nil {
		t.Errorf("user missing from the backup: %v", err)
	}
}

// A credential in a backup must still be ciphertext: the file travels off the
// host, the master key does not.
func TestBackupKeepsCredentialsEncrypted(t *testing.T) {
	ts := newTestServer(t)
	c := ts.client(t)
	signup(t, ts, c, "boss@example.com", "password123")
	tool := createTool(t, ts, c, toolInput{Name: "db-server", Address: "10.0.0.9", Port: 5432})
	body := map[string]any{
		"type":   "api_token",
		"label":  "prod token",
		"secret": map[string]string{"token": "super-secret-value"},
	}
	resp, _ := ts.do(t, c, http.MethodPost, "/api/v1/tools/"+tool.ID+"/credentials", body, nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create credential status = %d", resp.StatusCode)
	}

	_, backup := downloadBackup(t, ts, c)
	if bytes.Contains(backup, []byte("super-secret-value")) {
		t.Error("backup contains a credential in plaintext")
	}
}

func TestRestoreStagesAndAppliesOnStart(t *testing.T) {
	ts := newTestServer(t)
	c := ts.client(t)
	signup(t, ts, c, "boss@example.com", "password123")
	_, backup := downloadBackup(t, ts, c)

	// Change the live database after the backup so the restore is observable.
	signup(t, ts, ts.client(t), "later@example.com", "password123")

	resp, data := uploadRestore(t, ts, c, backup)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("upload status = %d, want 200: %s", resp.StatusCode, data)
	}
	var out map[string]any
	json.Unmarshal(data, &out)
	if out["staged"] != true {
		t.Errorf("response = %s, want staged true", data)
	}
	if !ts.app.restoreStaged() {
		t.Fatal("restore not staged on disk")
	}
	_, info := ts.do(t, c, http.MethodGet, "/api/v1/admin/server-info", nil, nil)
	if !bytes.Contains(info, []byte(`"restore_staged":true`)) {
		t.Error("server-info does not report the staged restore")
	}

	// The live database is untouched until the swap happens at startup.
	if _, err := ts.app.db.GetUserByEmail("later@example.com"); err != nil {
		t.Error("staging a restore already changed the live database")
	}
	ts.app.db.Close()
	if err := store.ApplyStagedRestore(ts.app.cfg.DBPath); err != nil {
		t.Fatalf("ApplyStagedRestore: %v", err)
	}
	db, err := store.Open(ts.app.cfg.DBPath)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer db.Close()
	if _, err := db.GetUserByEmail("boss@example.com"); err != nil {
		t.Error("restored database lost the account it had")
	}
	if _, err := db.GetUserByEmail("later@example.com"); err == nil {
		t.Error("restored database still has the post-backup account")
	}
}

// Staging garbage would brick the next startup, so uploads are validated first.
func TestRestoreRejectsNonDatabaseUploads(t *testing.T) {
	ts := newTestServer(t)
	c := ts.client(t)
	signup(t, ts, c, "boss@example.com", "password123")

	cases := map[string][]byte{
		"text":         []byte("this is not a database"),
		"empty":        {},
		"truncated db": []byte("SQLite format 3\x00"),
		"wrong magic":  append([]byte("SQLITE format 4\x00"), make([]byte, 512)...),
	}
	for name, content := range cases {
		resp, _ := uploadRestore(t, ts, c, content)
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("%s status = %d, want 400", name, resp.StatusCode)
		}
		if ts.app.restoreStaged() {
			t.Fatalf("%s was staged despite being rejected", name)
		}
	}
}

// A valid SQLite file that is not a Reeve database is also refused.
func TestRestoreRejectsForeignDatabase(t *testing.T) {
	ts := newTestServer(t)
	c := ts.client(t)
	signup(t, ts, c, "boss@example.com", "password123")

	foreign := filepath.Join(t.TempDir(), "foreign.db")
	db, err := store.Open(foreign)
	if err != nil {
		t.Fatalf("open foreign: %v", err)
	}
	if _, err := db.SQL().Exec(`DROP TABLE credentials`); err != nil {
		t.Fatalf("drop table: %v", err)
	}
	db.Close()
	content, err := os.ReadFile(foreign)
	if err != nil {
		t.Fatalf("read foreign: %v", err)
	}
	resp, data := uploadRestore(t, ts, c, content)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400: %s", resp.StatusCode, data)
	}
}

func TestRestoreCancelDiscardsStaged(t *testing.T) {
	ts := newTestServer(t)
	c := ts.client(t)
	signup(t, ts, c, "boss@example.com", "password123")
	_, backup := downloadBackup(t, ts, c)
	uploadRestore(t, ts, c, backup)
	if !ts.app.restoreStaged() {
		t.Fatal("precondition: restore not staged")
	}

	resp, _ := ts.do(t, c, http.MethodDelete, "/api/v1/admin/restore", nil, nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("cancel status = %d, want 204", resp.StatusCode)
	}
	if ts.app.restoreStaged() {
		t.Error("staged restore survived a cancel")
	}
	// Cancelling with nothing staged is not an error.
	resp, _ = ts.do(t, c, http.MethodDelete, "/api/v1/admin/restore", nil, nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("second cancel status = %d, want 204", resp.StatusCode)
	}
}

// The backup is the whole instance including credential ciphertext, so only
// admins may pull or push it.
func TestBackupAndRestoreAreAdminOnly(t *testing.T) {
	ts := newTestServer(t)
	signup(t, ts, ts.client(t), "boss@example.com", "password123")
	basic := ts.client(t)
	signup(t, ts, basic, "dev@example.com", "password123")

	if resp, _ := downloadBackup(t, ts, basic); resp.StatusCode != http.StatusForbidden {
		t.Errorf("basic download status = %d, want 403", resp.StatusCode)
	}
	if resp, _ := uploadRestore(t, ts, basic, []byte("x")); resp.StatusCode != http.StatusForbidden {
		t.Errorf("basic upload status = %d, want 403", resp.StatusCode)
	}
	anon := ts.client(t)
	if resp, _ := downloadBackup(t, ts, anon); resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("anonymous download status = %d, want 401", resp.StatusCode)
	}
}
