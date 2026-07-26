package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
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

// A deleted credential must stop revealing, and the ciphertext must actually
// leave the database rather than just being hidden from the list.
func TestDeleteCredential(t *testing.T) {
	ts := newTestServer(t)
	owner := ts.client(t)
	signup(t, ts, owner, "boss@example.com", "password123")
	tool := createTool(t, ts, owner, toolInput{Name: "Box"})
	cred := createCred(t, ts, owner, tool.ID, credentialInput{Type: "kv", Secret: map[string]string{"k": "gone"}})

	// A user with a standing grant may reveal but may not delete: reveal and
	// manage are separate rights.
	dev := ts.client(t)
	_, devUser := signup(t, ts, dev, "dev@example.com", "password123")
	ts.do(t, owner, http.MethodPut, "/api/v1/tools/"+tool.ID+"/access/user/"+devUser.ID, nil, nil)
	if code, _ := revealSecret(t, ts, dev, cred.ID); code != http.StatusOK {
		t.Fatalf("granted reveal = %d, want 200", code)
	}
	if resp, _ := ts.do(t, dev, http.MethodDelete, "/api/v1/credentials/"+cred.ID, nil, nil); resp.StatusCode != http.StatusForbidden {
		t.Errorf("grantee delete = %d, want 403", resp.StatusCode)
	}

	resp, data := ts.do(t, owner, http.MethodDelete, "/api/v1/credentials/"+cred.ID, nil, nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete = %d: %s", resp.StatusCode, data)
	}
	if n := countRows(t, ts, `SELECT COUNT(*) FROM credentials WHERE id = ?`, cred.ID); n != 0 {
		t.Error("the credential row survived delete")
	}
	if code, _ := revealSecret(t, ts, owner, cred.ID); code != http.StatusNotFound {
		t.Errorf("reveal after delete = %d, want 404", code)
	}
	_, data = ts.do(t, owner, http.MethodGet, "/api/v1/tools/"+tool.ID+"/credentials", nil, nil)
	var list []credentialView
	json.Unmarshal(data, &list)
	if len(list) != 0 {
		t.Errorf("credential still listed: %+v", list)
	}
	if resp, _ = ts.do(t, owner, http.MethodDelete, "/api/v1/credentials/"+cred.ID, nil, nil); resp.StatusCode != http.StatusNotFound {
		t.Errorf("second delete = %d, want 404", resp.StatusCode)
	}
}

// Denying leaves the requester with no access and records no grant, and the
// request cannot then be approved on a second pass.
func TestDenyAccessRequest(t *testing.T) {
	ts := newTestServer(t)
	owner := ts.client(t)
	signup(t, ts, owner, "boss@example.com", "password123")
	tool := createTool(t, ts, owner, toolInput{Name: "Box"})
	cred := createCred(t, ts, owner, tool.ID, credentialInput{Type: "kv", Secret: map[string]string{"k": "v"}})

	dev := ts.client(t)
	signup(t, ts, dev, "dev@example.com", "password123")
	_, data := ts.do(t, dev, http.MethodPost, "/api/v1/tools/"+tool.ID+"/access-requests",
		map[string]string{"note": "please"}, nil)
	var req requestView
	json.Unmarshal(data, &req)

	resp, data := ts.do(t, owner, http.MethodPost, "/api/v1/access-requests/"+req.ID+"/deny", nil, nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("deny = %d: %s", resp.StatusCode, data)
	}
	if code, _ := revealSecret(t, ts, dev, cred.ID); code != http.StatusForbidden {
		t.Errorf("reveal after denial = %d, want 403", code)
	}
	if n := countRows(t, ts, `SELECT COUNT(*) FROM grant_audit WHERE tool_id = ?`, tool.ID); n != 0 {
		t.Errorf("a denial recorded %d grant-audit rows, want 0", n)
	}

	// The requester sees the decision on their own list.
	_, data = ts.do(t, dev, http.MethodGet, "/api/v1/access-requests", nil, nil)
	var mine []requestView
	json.Unmarshal(data, &mine)
	if len(mine) != 1 || mine[0].Status != "denied" {
		t.Errorf("requester's list = %s", data)
	}

	// Deciding twice is a conflict, so a denial cannot be quietly reversed.
	if resp, _ = ts.do(t, owner, http.MethodPost, "/api/v1/access-requests/"+req.ID+"/approve", nil, nil); resp.StatusCode != http.StatusConflict {
		t.Errorf("approve after deny = %d, want 409", resp.StatusCode)
	}
	if resp, _ = ts.do(t, owner, http.MethodPost, "/api/v1/access-requests/nope/deny", nil, nil); resp.StatusCode != http.StatusNotFound {
		t.Errorf("deny of an unknown request = %d, want 404", resp.StatusCode)
	}
}

// The access list is what the owner reads to answer "who can see this secret",
// so it has to show both principal kinds and drop a revoked one.
func TestListToolAccess(t *testing.T) {
	ts := newTestServer(t)
	owner := ts.client(t)
	signup(t, ts, owner, "boss@example.com", "password123")
	tool := createTool(t, ts, owner, toolInput{Name: "Box"})
	dev := ts.client(t)
	_, devUser := signup(t, ts, dev, "dev@example.com", "password123")

	_, data := ts.do(t, owner, http.MethodPost, "/api/v1/admin/groups", map[string]string{"name": "ops"}, nil)
	var g groupView
	json.Unmarshal(data, &g)

	resp, data := ts.do(t, owner, http.MethodGet, "/api/v1/tools/"+tool.ID+"/access", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list access = %d: %s", resp.StatusCode, data)
	}
	if got := strings.TrimSpace(string(data)); got != "[]" {
		t.Errorf("access on a fresh tool = %s, want []", got)
	}

	ts.do(t, owner, http.MethodPut, "/api/v1/tools/"+tool.ID+"/access/user/"+devUser.ID, nil, nil)
	ts.do(t, owner, http.MethodPut, "/api/v1/tools/"+tool.ID+"/access/group/"+g.ID, nil, nil)
	_, data = ts.do(t, owner, http.MethodGet, "/api/v1/tools/"+tool.ID+"/access", nil, nil)
	var grants []accessGrantView
	json.Unmarshal(data, &grants)
	kinds := map[string]string{}
	for _, gr := range grants {
		kinds[gr.PrincipalType] = gr.PrincipalID
	}
	if kinds["user"] != devUser.ID || kinds["group"] != g.ID {
		t.Fatalf("access list = %+v", grants)
	}

	ts.do(t, owner, http.MethodDelete, "/api/v1/tools/"+tool.ID+"/access/group/"+g.ID, nil, nil)
	_, data = ts.do(t, owner, http.MethodGet, "/api/v1/tools/"+tool.ID+"/access", nil, nil)
	json.Unmarshal(data, &grants)
	if len(grants) != 1 || grants[0].PrincipalType != "user" {
		t.Errorf("access list after revoke = %+v", grants)
	}

	// Only someone who may edit the tool may read who has access to it.
	if resp, _ = ts.do(t, dev, http.MethodGet, "/api/v1/tools/"+tool.ID+"/access", nil, nil); resp.StatusCode != http.StatusForbidden {
		t.Errorf("grantee reading the access list = %d, want 403", resp.StatusCode)
	}
}

// A basic user who created a tool is its approver. Their inbox is built from
// the tools they created rather than handed to them like an admin's, so it must
// show requests on their own tools and nothing else.
func TestNonAdminApproverInbox(t *testing.T) {
	ts := newTestServer(t)
	signup(t, ts, ts.client(t), "boss@example.com", "password123") // first user = admin

	owner := ts.client(t)
	signup(t, ts, owner, "owner@example.com", "password123")
	mine := createTool(t, ts, owner, toolInput{Name: "Mine"})

	other := ts.client(t)
	signup(t, ts, other, "other@example.com", "password123")
	theirs := createTool(t, ts, other, toolInput{Name: "Theirs"})

	dev := ts.client(t)
	_, devUser := signup(t, ts, dev, "dev@example.com", "password123")
	for _, id := range []string{mine.ID, theirs.ID} {
		resp, data := ts.do(t, dev, http.MethodPost, "/api/v1/tools/"+id+"/access-requests",
			map[string]string{"note": "please"}, nil)
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("request on %s = %d: %s", id, resp.StatusCode, data)
		}
	}

	resp, data := ts.do(t, owner, http.MethodGet, "/api/v1/access-requests?box=inbox", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("inbox = %d: %s", resp.StatusCode, data)
	}
	var inbox []requestView
	json.Unmarshal(data, &inbox)
	if len(inbox) != 1 || inbox[0].ToolID != mine.ID || inbox[0].RequesterID != devUser.ID {
		t.Fatalf("owner's inbox = %s, want only the request on their own tool", data)
	}

	// A user who created nothing has an empty inbox, not everyone's requests.
	_, data = ts.do(t, dev, http.MethodGet, "/api/v1/access-requests?box=inbox", nil, nil)
	json.Unmarshal(data, &inbox)
	if len(inbox) != 0 {
		t.Errorf("non-creator inbox = %s, want empty", data)
	}

	// The admin sees both, since they may decide anything.
	adminC := ts.client(t)
	loginWith(t, ts, adminC, "boss@example.com", "password123")
	_, data = ts.do(t, adminC, http.MethodGet, "/api/v1/access-requests?box=inbox", nil, nil)
	json.Unmarshal(data, &inbox)
	if len(inbox) != 2 {
		t.Errorf("admin inbox = %s, want both requests", data)
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
