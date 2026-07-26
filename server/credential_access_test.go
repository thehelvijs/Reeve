package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
)

func createCred(t *testing.T, ts *testServer, c *http.Client, toolID string, in credentialInput) credentialView {
	t.Helper()
	resp, data := ts.do(t, c, http.MethodPost, "/api/v1/tools/"+toolID+"/credentials", in, nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create cred = %d: %s", resp.StatusCode, data)
	}
	var cv credentialView
	json.Unmarshal(data, &cv)
	return cv
}

func revealSecret(t *testing.T, ts *testServer, c *http.Client, cid string) (int, map[string]string) {
	t.Helper()
	resp, data := ts.do(t, c, http.MethodPost, "/api/v1/credentials/"+cid+"/reveal", nil, nil)
	var out struct {
		Secret map[string]string `json:"secret"`
	}
	json.Unmarshal(data, &out)
	return resp.StatusCode, out.Secret
}

func TestCredentialStoredEncrypted(t *testing.T) {
	ts := newTestServer(t)
	c := ts.client(t)
	signup(t, ts, c, "boss@example.com", "password123")
	tool := createTool(t, ts, c, toolInput{Name: "Box"})
	createCred(t, ts, c, tool.ID, credentialInput{
		Type: "ssh_password", Label: "root", Secret: map[string]string{"username": "root", "password": "hunter2"},
	})

	// The ciphertext in the DB must not contain the plaintext password.
	var ct []byte
	err := ts.app.db.SQL().QueryRow(`SELECT ciphertext FROM credentials LIMIT 1`).Scan(&ct)
	if err != nil {
		t.Fatalf("read ciphertext: %v", err)
	}
	if bytes.Contains(ct, []byte("hunter2")) {
		t.Fatal("plaintext password found in stored ciphertext")
	}
}

func TestCreatorCanRevealAndAudited(t *testing.T) {
	ts := newTestServer(t)
	c := ts.client(t)
	signup(t, ts, c, "boss@example.com", "password123")
	tool := createTool(t, ts, c, toolInput{Name: "Box"})
	cred := createCred(t, ts, c, tool.ID, credentialInput{
		Type: "api_token", Label: "grafana", Secret: map[string]string{"token": "abc123"},
	})

	code, secret := revealSecret(t, ts, c, cred.ID)
	if code != http.StatusOK || secret["token"] != "abc123" {
		t.Fatalf("reveal = %d secret=%v", code, secret)
	}
	// Audit row written.
	var n int
	ts.app.db.SQL().QueryRow(`SELECT COUNT(*) FROM reveal_audit WHERE credential_id = ?`, cred.ID).Scan(&n)
	if n != 1 {
		t.Errorf("reveal audit rows = %d, want 1", n)
	}
}

func TestNonAccessUserCannotReveal(t *testing.T) {
	ts := newTestServer(t)
	owner := ts.client(t)
	signup(t, ts, owner, "boss@example.com", "password123")
	tool := createTool(t, ts, owner, toolInput{Name: "Public"})
	cred := createCred(t, ts, owner, tool.ID, credentialInput{
		Type: "api_token", Secret: map[string]string{"token": "secret"},
	})

	other := ts.client(t)
	signup(t, ts, other, "dev@example.com", "password123")

	// Can see the tool + credential metadata, but not reveal.
	_, data := ts.do(t, other, http.MethodGet, "/api/v1/tools/"+tool.ID+"/credentials", nil, nil)
	var metas []credentialView
	json.Unmarshal(data, &metas)
	if len(metas) != 1 || metas[0].CanReveal {
		t.Fatalf("metadata leak or wrong can_reveal: %+v", metas)
	}
	code, _ := revealSecret(t, ts, other, cred.ID)
	if code != http.StatusForbidden {
		t.Errorf("outsider reveal = %d, want 403", code)
	}
}

func TestRequestApproveRevealFlow(t *testing.T) {
	ts := newTestServer(t)
	owner := ts.client(t)
	signup(t, ts, owner, "boss@example.com", "password123")
	tool := createTool(t, ts, owner, toolInput{Name: "Box"})
	cred := createCred(t, ts, owner, tool.ID, credentialInput{
		Type: "api_token", Secret: map[string]string{"token": "xyz"},
	})

	dev := ts.client(t)
	_, devUser := signup(t, ts, dev, "dev@example.com", "password123")

	// Dev requests access.
	resp, data := ts.do(t, dev, http.MethodPost, "/api/v1/tools/"+tool.ID+"/access-requests",
		map[string]string{"note": "need it"}, nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("request = %d: %s", resp.StatusCode, data)
	}
	var req requestView
	json.Unmarshal(data, &req)

	// Duplicate open request blocked.
	resp, _ = ts.do(t, dev, http.MethodPost, "/api/v1/tools/"+tool.ID+"/access-requests", map[string]string{}, nil)
	if resp.StatusCode != http.StatusConflict {
		t.Errorf("duplicate request = %d, want 409", resp.StatusCode)
	}

	// It shows in the owner's inbox.
	_, data = ts.do(t, owner, http.MethodGet, "/api/v1/access-requests?box=inbox", nil, nil)
	var inbox []requestView
	json.Unmarshal(data, &inbox)
	if len(inbox) != 1 || inbox[0].RequesterID != devUser.ID {
		t.Fatalf("inbox mismatch: %+v", inbox)
	}

	// Before approval, dev cannot reveal.
	if code, _ := revealSecret(t, ts, dev, cred.ID); code != http.StatusForbidden {
		t.Fatalf("pre-approval reveal = %d, want 403", code)
	}

	// Owner approves.
	resp, _ = ts.do(t, owner, http.MethodPost, "/api/v1/access-requests/"+req.ID+"/approve", map[string]string{}, nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("approve = %d", resp.StatusCode)
	}

	// Now dev can reveal.
	if code, secret := revealSecret(t, ts, dev, cred.ID); code != http.StatusOK || secret["token"] != "xyz" {
		t.Fatalf("post-approval reveal = %d secret=%v", code, secret)
	}

	// grant_audit recorded.
	var n int
	ts.app.db.SQL().QueryRow(`SELECT COUNT(*) FROM grant_audit WHERE tool_id = ? AND action='grant'`, tool.ID).Scan(&n)
	if n != 1 {
		t.Errorf("grant audit rows = %d, want 1", n)
	}
}

func TestRevokeRemovesAccess(t *testing.T) {
	ts := newTestServer(t)
	owner := ts.client(t)
	signup(t, ts, owner, "boss@example.com", "password123")
	tool := createTool(t, ts, owner, toolInput{Name: "Box"})
	cred := createCred(t, ts, owner, tool.ID, credentialInput{Type: "kv", Secret: map[string]string{"k": "v"}})

	dev := ts.client(t)
	_, devUser := signup(t, ts, dev, "dev@example.com", "password123")
	ts.do(t, owner, http.MethodPut, "/api/v1/tools/"+tool.ID+"/access/user/"+devUser.ID, nil, nil)

	if code, _ := revealSecret(t, ts, dev, cred.ID); code != http.StatusOK {
		t.Fatalf("granted reveal = %d, want 200", code)
	}
	// Revoke.
	ts.do(t, owner, http.MethodDelete, "/api/v1/tools/"+tool.ID+"/access/user/"+devUser.ID, nil, nil)
	if code, _ := revealSecret(t, ts, dev, cred.ID); code != http.StatusForbidden {
		t.Errorf("post-revoke reveal = %d, want 403", code)
	}
}

// A grant naming nobody can never match a caller, so it is a typo, not a
// grant: recording it would only pad the access list and the audit trail.
func TestGrantRejectsAnUnknownPrincipal(t *testing.T) {
	ts := newTestServer(t)
	owner := ts.client(t)
	signup(t, ts, owner, "boss@example.com", "password123")
	tool := createTool(t, ts, owner, toolInput{Name: "Box"})

	cases := map[string]struct {
		path string
		want int
	}{
		"unknown user":   {"/api/v1/tools/" + tool.ID + "/access/user/nobody", http.StatusNotFound},
		"unknown group":  {"/api/v1/tools/" + tool.ID + "/access/group/nogroup", http.StatusNotFound},
		"bad type":       {"/api/v1/tools/" + tool.ID + "/access/robot/whoever", http.StatusBadRequest},
		"bad type on rm": {"/api/v1/tools/" + tool.ID + "/access/robot/whoever", http.StatusBadRequest},
	}
	for name, tc := range cases {
		method := http.MethodPut
		if name == "bad type on rm" {
			method = http.MethodDelete
		}
		resp, data := ts.do(t, owner, method, tc.path, nil, nil)
		if resp.StatusCode != tc.want {
			t.Errorf("%s: status = %d, want %d: %s", name, resp.StatusCode, tc.want, data)
		}
	}

	grants, err := ts.app.db.ListCredentialAccess(tool.ID)
	if err != nil {
		t.Fatalf("list access: %v", err)
	}
	if len(grants) != 0 {
		t.Errorf("rejected grants still landed: %+v", grants)
	}
}

func TestNonOwnerCannotApprove(t *testing.T) {
	ts := newTestServer(t)
	owner := ts.client(t)
	signup(t, ts, owner, "boss@example.com", "password123") // admin
	dev := ts.client(t)
	signup(t, ts, dev, "dev@example.com", "password123")
	tool := createTool(t, ts, dev, toolInput{Name: "DevBox"}) // dev owns it

	third := ts.client(t)
	signup(t, ts, third, "eve@example.com", "password123")
	resp, data := ts.do(t, third, http.MethodPost, "/api/v1/tools/"+tool.ID+"/access-requests", map[string]string{}, nil)
	var req requestView
	json.Unmarshal(data, &req)

	// Eve (not owner, not admin) cannot approve.
	resp, _ = ts.do(t, third, http.MethodPost, "/api/v1/access-requests/"+req.ID+"/approve", map[string]string{}, nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("non-owner approve = %d, want 403", resp.StatusCode)
	}
}

func TestCredentialRotate(t *testing.T) {
	ts := newTestServer(t)
	c := ts.client(t)
	signup(t, ts, c, "boss@example.com", "password123")
	tool := createTool(t, ts, c, toolInput{Name: "Box"})
	cred := createCred(t, ts, c, tool.ID, credentialInput{Type: "kv", Secret: map[string]string{"k": "old"}})

	resp, _ := ts.do(t, c, http.MethodPatch, "/api/v1/credentials/"+cred.ID,
		credentialInput{Type: "kv", Label: "rotated", Secret: map[string]string{"k": "new"}}, nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("rotate = %d", resp.StatusCode)
	}
	if _, secret := revealSecret(t, ts, c, cred.ID); secret["k"] != "new" {
		t.Errorf("rotated secret = %v, want k=new", secret)
	}
}

func TestAuditAdminOnly(t *testing.T) {
	ts := newTestServer(t)
	signup(t, ts, ts.client(t), "boss@example.com", "password123") // admin
	basic := ts.client(t)
	signup(t, ts, basic, "dev@example.com", "password123")
	resp, _ := ts.do(t, basic, http.MethodGet, "/api/v1/admin/audit/reveals", nil, nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("basic audit access = %d, want 403", resp.StatusCode)
	}
}

func TestOutsiderCannotSeeRestrictedCredential(t *testing.T) {
	ts := newTestServer(t)
	owner := ts.client(t)
	signup(t, ts, owner, "boss@example.com", "password123")
	tool := createTool(t, ts, owner, toolInput{Name: "Secret", Visibility: "restricted"})
	cred := createCred(t, ts, owner, tool.ID, credentialInput{Type: "kv", Secret: map[string]string{"k": "v"}})

	outsider := ts.client(t)
	signup(t, ts, outsider, "dev@example.com", "password123")
	// Reveal on a tool they can't even see → 404 (existence hidden), not 403.
	code, _ := revealSecret(t, ts, outsider, cred.ID)
	if code != http.StatusNotFound {
		t.Errorf("outsider reveal of restricted = %d, want 404", code)
	}
}
