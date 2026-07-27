package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func createCred(t *testing.T, ts *testServer, c *http.Client, hostID string, in credentialInput) credentialView {
	t.Helper()
	resp, data := ts.do(t, c, http.MethodPost, "/api/admin/hosts/"+hostID+"/credentials", in, nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create cred = %d: %s", resp.StatusCode, data)
	}
	var cv credentialView
	json.Unmarshal(data, &cv)
	return cv
}

func revealSecret(t *testing.T, ts *testServer, c *http.Client, cid string) (int, map[string]string) {
	t.Helper()
	resp, data := ts.do(t, c, http.MethodPost, "/api/credentials/"+cid+"/reveal", nil, nil)
	var out struct {
		Secret map[string]string `json:"secret"`
	}
	json.Unmarshal(data, &out)
	return resp.StatusCode, out.Secret
}

// newHost enrolls a host and returns its id, discarding the agent token.
func newHost(t *testing.T, ts *testServer, admin *http.Client, name string) string {
	t.Helper()
	id, _ := enrollHost(t, ts, admin, name)
	return id
}

func TestCredentialStoredEncrypted(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	host := newHost(t, ts, admin, "box")
	createCred(t, ts, admin, host, credentialInput{
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

func TestAdminCanRevealAndAudited(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	host := newHost(t, ts, admin, "box")
	cred := createCred(t, ts, admin, host, credentialInput{
		Type: "api_token", Label: "grafana", Secret: map[string]string{"token": "abc123"},
	})

	code, secret := revealSecret(t, ts, admin, cred.ID)
	if code != http.StatusOK || secret["token"] != "abc123" {
		t.Fatalf("reveal = %d secret=%v", code, secret)
	}
	// The audit row names the host the secret belongs to.
	var gotHost string
	err := ts.app.db.SQL().QueryRow(
		`SELECT host_id FROM reveal_audit WHERE credential_id = ?`, cred.ID).Scan(&gotHost)
	if err != nil {
		t.Fatalf("no reveal audit row: %v", err)
	}
	if gotHost != host {
		t.Errorf("audit host = %q, want %q", gotHost, host)
	}
}

// Any signed-in user may see that a host holds a credential; only a grant lets
// them read it. Hiding the existence would make "request access" unusable.
func TestNonAdminSeesMetadataButCannotReveal(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	host := newHost(t, ts, admin, "box")
	cred := createCred(t, ts, admin, host, credentialInput{
		Type: "api_token", Secret: map[string]string{"token": "secret"},
	})

	other := ts.client(t)
	signup(t, ts, other, "dev@example.com", "password123")

	_, data := ts.do(t, other, http.MethodGet, "/api/hosts/"+host+"/credentials", nil, nil)
	var metas []credentialView
	json.Unmarshal(data, &metas)
	if len(metas) != 1 || metas[0].CanReveal {
		t.Fatalf("metadata leak or wrong can_reveal: %+v", metas)
	}
	if code, _ := revealSecret(t, ts, other, cred.ID); code != http.StatusForbidden {
		t.Errorf("outsider reveal = %d, want 403", code)
	}
}

// A host has no creator, so managing its credentials is an admin right and
// nothing else. A grant buys reveal, never management.
func TestNonAdminCannotManageCredentials(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	host := newHost(t, ts, admin, "box")
	cred := createCred(t, ts, admin, host, credentialInput{Type: "kv", Secret: map[string]string{"k": "v"}})

	dev := ts.client(t)
	_, devUser := signup(t, ts, dev, "dev@example.com", "password123")
	ts.do(t, admin, http.MethodPut, "/api/admin/hosts/"+host+"/access/user/"+devUser.ID, nil, nil)
	if code, _ := revealSecret(t, ts, dev, cred.ID); code != http.StatusOK {
		t.Fatalf("granted reveal = %d, want 200", code)
	}

	cases := []struct {
		name, method, path string
		body               any
	}{
		{"create", http.MethodPost, "/api/admin/hosts/" + host + "/credentials", credentialInput{Type: "kv", Secret: map[string]string{"k": "v"}}},
		{"rotate", http.MethodPatch, "/api/admin/credentials/" + cred.ID, credentialInput{Type: "kv", Secret: map[string]string{"k": "v2"}}},
		{"delete", http.MethodDelete, "/api/admin/credentials/" + cred.ID, nil},
		{"grant", http.MethodPut, "/api/admin/hosts/" + host + "/access/user/" + devUser.ID, nil},
		{"list access", http.MethodGet, "/api/admin/hosts/" + host + "/access", nil},
	}
	for _, tc := range cases {
		resp, data := ts.do(t, dev, tc.method, tc.path, tc.body, nil)
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("%s as a grantee = %d, want 403: %s", tc.name, resp.StatusCode, data)
		}
	}
}

func TestRequestApproveRevealFlow(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	host := newHost(t, ts, admin, "box")
	cred := createCred(t, ts, admin, host, credentialInput{
		Type: "api_token", Secret: map[string]string{"token": "xyz"},
	})

	dev := ts.client(t)
	_, devUser := signup(t, ts, dev, "dev@example.com", "password123")

	resp, data := ts.do(t, dev, http.MethodPost, "/api/hosts/"+host+"/access-requests",
		map[string]string{"note": "need it"}, nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("request = %d: %s", resp.StatusCode, data)
	}
	var req requestView
	json.Unmarshal(data, &req)
	if req.HostName != "box" {
		t.Errorf("request host name = %q, want box: the requester has to see what they asked for", req.HostName)
	}

	// Duplicate open request blocked.
	resp, _ = ts.do(t, dev, http.MethodPost, "/api/hosts/"+host+"/access-requests", map[string]string{}, nil)
	if resp.StatusCode != http.StatusConflict {
		t.Errorf("duplicate request = %d, want 409", resp.StatusCode)
	}

	_, data = ts.do(t, admin, http.MethodGet, "/api/access-requests?box=inbox", nil, nil)
	var inbox []requestView
	json.Unmarshal(data, &inbox)
	if len(inbox) != 1 || inbox[0].RequesterID != devUser.ID {
		t.Fatalf("inbox mismatch: %+v", inbox)
	}

	if code, _ := revealSecret(t, ts, dev, cred.ID); code != http.StatusForbidden {
		t.Fatalf("pre-approval reveal = %d, want 403", code)
	}

	resp, _ = ts.do(t, admin, http.MethodPost, "/api/access-requests/"+req.ID+"/approve", map[string]string{}, nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("approve = %d", resp.StatusCode)
	}

	if code, secret := revealSecret(t, ts, dev, cred.ID); code != http.StatusOK || secret["token"] != "xyz" {
		t.Fatalf("post-approval reveal = %d secret=%v", code, secret)
	}

	if n := countRows(t, ts, `SELECT COUNT(*) FROM grant_audit WHERE host_id = ? AND action='grant'`, host); n != 1 {
		t.Errorf("grant audit rows = %d, want 1", n)
	}
}

func TestRevokeRemovesAccess(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	host := newHost(t, ts, admin, "box")
	cred := createCred(t, ts, admin, host, credentialInput{Type: "kv", Secret: map[string]string{"k": "v"}})

	dev := ts.client(t)
	_, devUser := signup(t, ts, dev, "dev@example.com", "password123")
	ts.do(t, admin, http.MethodPut, "/api/admin/hosts/"+host+"/access/user/"+devUser.ID, nil, nil)

	if code, _ := revealSecret(t, ts, dev, cred.ID); code != http.StatusOK {
		t.Fatalf("granted reveal = %d, want 200", code)
	}
	ts.do(t, admin, http.MethodDelete, "/api/admin/hosts/"+host+"/access/user/"+devUser.ID, nil, nil)
	if code, _ := revealSecret(t, ts, dev, cred.ID); code != http.StatusForbidden {
		t.Errorf("post-revoke reveal = %d, want 403", code)
	}
}

// A grant naming nobody can never match a caller, so it is a typo, not a
// grant: recording it would only pad the access list and the audit trail.
func TestGrantRejectsAnUnknownPrincipal(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	host := newHost(t, ts, admin, "box")
	base := "/api/admin/hosts/" + host + "/access"

	cases := map[string]struct {
		path string
		want int
	}{
		"unknown user":   {base + "/user/nobody", http.StatusNotFound},
		"unknown group":  {base + "/group/nogroup", http.StatusNotFound},
		"bad type":       {base + "/robot/whoever", http.StatusBadRequest},
		"bad type on rm": {base + "/robot/whoever", http.StatusBadRequest},
	}
	for name, tc := range cases {
		method := http.MethodPut
		if name == "bad type on rm" {
			method = http.MethodDelete
		}
		resp, data := ts.do(t, admin, method, tc.path, nil, nil)
		if resp.StatusCode != tc.want {
			t.Errorf("%s: status = %d, want %d: %s", name, resp.StatusCode, tc.want, data)
		}
	}

	grants, err := ts.app.db.ListCredentialAccess(host)
	if err != nil {
		t.Fatalf("list access: %v", err)
	}
	if len(grants) != 0 {
		t.Errorf("rejected grants still landed: %+v", grants)
	}
}

func TestNonAdminCannotApprove(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	host := newHost(t, ts, admin, "box")

	dev := ts.client(t)
	signup(t, ts, dev, "dev@example.com", "password123")
	_, data := ts.do(t, dev, http.MethodPost, "/api/hosts/"+host+"/access-requests", map[string]string{}, nil)
	var req requestView
	json.Unmarshal(data, &req)

	eve := ts.client(t)
	signup(t, ts, eve, "eve@example.com", "password123")
	resp, _ := ts.do(t, eve, http.MethodPost, "/api/access-requests/"+req.ID+"/approve", map[string]string{}, nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("non-admin approve = %d, want 403", resp.StatusCode)
	}
	// Not even the requester can wave their own request through.
	resp, _ = ts.do(t, dev, http.MethodPost, "/api/access-requests/"+req.ID+"/approve", map[string]string{}, nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("self-approve = %d, want 403", resp.StatusCode)
	}
}

func TestCredentialRotate(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	host := newHost(t, ts, admin, "box")
	cred := createCred(t, ts, admin, host, credentialInput{Type: "kv", Secret: map[string]string{"k": "old"}})

	resp, _ := ts.do(t, admin, http.MethodPatch, "/api/admin/credentials/"+cred.ID,
		credentialInput{Type: "kv", Label: "rotated", Secret: map[string]string{"k": "new"}}, nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("rotate = %d", resp.StatusCode)
	}
	if _, secret := revealSecret(t, ts, admin, cred.ID); secret["k"] != "new" {
		t.Errorf("rotated secret = %v, want k=new", secret)
	}
}

// A deleted credential must stop revealing, and the ciphertext must actually
// leave the database rather than just being hidden from the list.
func TestDeleteCredential(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	host := newHost(t, ts, admin, "box")
	cred := createCred(t, ts, admin, host, credentialInput{Type: "kv", Secret: map[string]string{"k": "gone"}})

	resp, data := ts.do(t, admin, http.MethodDelete, "/api/admin/credentials/"+cred.ID, nil, nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete = %d: %s", resp.StatusCode, data)
	}
	if n := countRows(t, ts, `SELECT COUNT(*) FROM credentials WHERE id = ?`, cred.ID); n != 0 {
		t.Error("the credential row survived delete")
	}
	if code, _ := revealSecret(t, ts, admin, cred.ID); code != http.StatusNotFound {
		t.Errorf("reveal after delete = %d, want 404", code)
	}
	_, data = ts.do(t, admin, http.MethodGet, "/api/hosts/"+host+"/credentials", nil, nil)
	var list []credentialView
	json.Unmarshal(data, &list)
	if len(list) != 0 {
		t.Errorf("credential still listed: %+v", list)
	}
	if resp, _ = ts.do(t, admin, http.MethodDelete, "/api/admin/credentials/"+cred.ID, nil, nil); resp.StatusCode != http.StatusNotFound {
		t.Errorf("second delete = %d, want 404", resp.StatusCode)
	}
}

// Deleting a host takes its secrets with it: a credential outliving the machine
// it opens is a secret nobody is watching any more.
func TestDeletingAHostDropsItsCredentials(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	host := newHost(t, ts, admin, "box")
	cred := createCred(t, ts, admin, host, credentialInput{Type: "kv", Secret: map[string]string{"k": "v"}})

	resp, data := ts.do(t, admin, http.MethodDelete, "/api/admin/hosts/"+host, nil, nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete host = %d: %s", resp.StatusCode, data)
	}
	if n := countRows(t, ts, `SELECT COUNT(*) FROM credentials WHERE id = ?`, cred.ID); n != 0 {
		t.Errorf("credential rows after host delete = %d, want 0", n)
	}
}

// Denying leaves the requester with no access and records no grant, and the
// request cannot then be approved on a second pass.
func TestDenyAccessRequest(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	host := newHost(t, ts, admin, "box")
	cred := createCred(t, ts, admin, host, credentialInput{Type: "kv", Secret: map[string]string{"k": "v"}})

	dev := ts.client(t)
	signup(t, ts, dev, "dev@example.com", "password123")
	_, data := ts.do(t, dev, http.MethodPost, "/api/hosts/"+host+"/access-requests",
		map[string]string{"note": "please"}, nil)
	var req requestView
	json.Unmarshal(data, &req)

	resp, data := ts.do(t, admin, http.MethodPost, "/api/access-requests/"+req.ID+"/deny", nil, nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("deny = %d: %s", resp.StatusCode, data)
	}
	if code, _ := revealSecret(t, ts, dev, cred.ID); code != http.StatusForbidden {
		t.Errorf("reveal after denial = %d, want 403", code)
	}
	if n := countRows(t, ts, `SELECT COUNT(*) FROM grant_audit WHERE host_id = ?`, host); n != 0 {
		t.Errorf("a denial recorded %d grant-audit rows, want 0", n)
	}

	_, data = ts.do(t, dev, http.MethodGet, "/api/access-requests", nil, nil)
	var mine []requestView
	json.Unmarshal(data, &mine)
	if len(mine) != 1 || mine[0].Status != "denied" {
		t.Errorf("requester's list = %s", data)
	}

	// Deciding twice is a conflict, so a denial cannot be quietly reversed.
	if resp, _ = ts.do(t, admin, http.MethodPost, "/api/access-requests/"+req.ID+"/approve", nil, nil); resp.StatusCode != http.StatusConflict {
		t.Errorf("approve after deny = %d, want 409", resp.StatusCode)
	}
	if resp, _ = ts.do(t, admin, http.MethodPost, "/api/access-requests/nope/deny", nil, nil); resp.StatusCode != http.StatusNotFound {
		t.Errorf("deny of an unknown request = %d, want 404", resp.StatusCode)
	}
}

// The access list answers "who can read this machine's secrets", so it has to
// show both principal kinds and drop a revoked one.
func TestListHostAccess(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	host := newHost(t, ts, admin, "box")
	base := "/api/admin/hosts/" + host + "/access"

	dev := ts.client(t)
	_, devUser := signup(t, ts, dev, "dev@example.com", "password123")

	_, data := ts.do(t, admin, http.MethodPost, "/api/admin/groups", map[string]string{"name": "ops"}, nil)
	var g groupView
	json.Unmarshal(data, &g)

	resp, data := ts.do(t, admin, http.MethodGet, base, nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list access = %d: %s", resp.StatusCode, data)
	}
	if got := strings.TrimSpace(string(data)); got != "[]" {
		t.Errorf("access on a fresh host = %s, want []", got)
	}

	ts.do(t, admin, http.MethodPut, base+"/user/"+devUser.ID, nil, nil)
	ts.do(t, admin, http.MethodPut, base+"/group/"+g.ID, nil, nil)
	_, data = ts.do(t, admin, http.MethodGet, base, nil, nil)
	var grants []accessGrantView
	json.Unmarshal(data, &grants)
	kinds := map[string]string{}
	for _, gr := range grants {
		kinds[gr.PrincipalType] = gr.PrincipalID
	}
	if kinds["user"] != devUser.ID || kinds["group"] != g.ID {
		t.Fatalf("access list = %+v", grants)
	}

	ts.do(t, admin, http.MethodDelete, base+"/group/"+g.ID, nil, nil)
	_, data = ts.do(t, admin, http.MethodGet, base, nil, nil)
	json.Unmarshal(data, &grants)
	if len(grants) != 1 || grants[0].PrincipalType != "user" {
		t.Errorf("access list after revoke = %+v", grants)
	}
}

// A group grant reaches its members, which is the point of granting to one.
func TestGroupGrantRevealsForMembers(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	host := newHost(t, ts, admin, "box")
	cred := createCred(t, ts, admin, host, credentialInput{Type: "kv", Secret: map[string]string{"k": "v"}})

	dev := ts.client(t)
	_, devUser := signup(t, ts, dev, "dev@example.com", "password123")

	_, data := ts.do(t, admin, http.MethodPost, "/api/admin/groups", map[string]string{"name": "ops"}, nil)
	var g groupView
	json.Unmarshal(data, &g)
	ts.do(t, admin, http.MethodPut, "/api/admin/groups/"+g.ID+"/members/"+devUser.ID, nil, nil)
	ts.do(t, admin, http.MethodPut, "/api/admin/hosts/"+host+"/access/group/"+g.ID, nil, nil)

	if code, secret := revealSecret(t, ts, dev, cred.ID); code != http.StatusOK || secret["k"] != "v" {
		t.Errorf("group member reveal = %d secret=%v, want 200", code, secret)
	}
}

// Credentials belong to admin-managed hosts, so a non-admin approver has an
// empty inbox rather than one built from what they created.
func TestNonAdminApproverInboxIsEmpty(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	host := newHost(t, ts, admin, "box")

	dev := ts.client(t)
	_, devUser := signup(t, ts, dev, "dev@example.com", "password123")
	resp, data := ts.do(t, dev, http.MethodPost, "/api/hosts/"+host+"/access-requests",
		map[string]string{"note": "please"}, nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("request = %d: %s", resp.StatusCode, data)
	}

	other := ts.client(t)
	signup(t, ts, other, "other@example.com", "password123")
	_, data = ts.do(t, other, http.MethodGet, "/api/access-requests?box=inbox", nil, nil)
	var inbox []requestView
	json.Unmarshal(data, &inbox)
	if len(inbox) != 0 {
		t.Errorf("non-admin inbox = %s, want empty", data)
	}

	_, data = ts.do(t, admin, http.MethodGet, "/api/access-requests?box=inbox", nil, nil)
	json.Unmarshal(data, &inbox)
	if len(inbox) != 1 || inbox[0].RequesterID != devUser.ID {
		t.Errorf("admin inbox = %s, want the one pending request", data)
	}
}

func TestAccessRequestOnAnUnknownHost(t *testing.T) {
	ts := newTestServer(t)
	adminClient(t, ts)
	dev := ts.client(t)
	signup(t, ts, dev, "dev@example.com", "password123")
	resp, _ := ts.do(t, dev, http.MethodPost, "/api/hosts/nope/access-requests", map[string]string{}, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("request on an unknown host = %d, want 404", resp.StatusCode)
	}
}

func TestAuditAdminOnly(t *testing.T) {
	ts := newTestServer(t)
	adminClient(t, ts)
	basic := ts.client(t)
	signup(t, ts, basic, "dev@example.com", "password123")
	resp, _ := ts.do(t, basic, http.MethodGet, "/api/admin/audit/reveals", nil, nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("basic audit access = %d, want 403", resp.StatusCode)
	}
}

// TestCiphertextCannotBeMovedBetweenHosts pins the AAD binding: a database
// writer who copies one host's sealed secret into another host's credential row
// gets an undecryptable row, not a working secret under the wrong identity and
// the wrong audit trail.
func TestCiphertextCannotBeMovedBetweenHosts(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	hostA := newHost(t, ts, admin, "box-a")
	hostB := newHost(t, ts, admin, "box-b")

	credA := createCred(t, ts, admin, hostA, credentialInput{
		Type: "ssh_password", Label: "a", Secret: map[string]string{"password": "secret-a"},
	})
	credB := createCred(t, ts, admin, hostB, credentialInput{
		Type: "ssh_password", Label: "b", Secret: map[string]string{"password": "secret-b"},
	})

	ct, nonce, err := ts.app.db.GetCredentialSecret(credA.ID)
	if err != nil {
		t.Fatalf("read host A secret: %v", err)
	}
	if _, err := ts.app.db.SQL().Exec(
		`UPDATE credentials SET ciphertext = ?, nonce = ? WHERE id = ?`, ct, nonce, credB.ID); err != nil {
		t.Fatalf("plant host A ciphertext on host B: %v", err)
	}

	status, secret := revealSecret(t, ts, admin, credB.ID)
	if status == http.StatusOK {
		t.Fatalf("host A's secret revealed through host B's credential: %v", secret)
	}
	if secret["password"] == "secret-a" {
		t.Fatal("host A's password leaked through host B's row")
	}

	// The untouched credential still opens, so the binding did not simply break
	// decryption for everyone.
	if status, secret := revealSecret(t, ts, admin, credA.ID); status != http.StatusOK || secret["password"] != "secret-a" {
		t.Fatalf("host A's own credential no longer reveals: %d %v", status, secret)
	}
}
