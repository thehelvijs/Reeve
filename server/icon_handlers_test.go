package main

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"testing"
)

func uploadIcon(t *testing.T, ts *testServer, c *http.Client, path string, data []byte) (*http.Response, []byte) {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("icon", "i.png")
	fw.Write(data)
	mw.Close()
	req, _ := http.NewRequest(http.MethodPost, ts.srv.URL+path, &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	resp, err := c.Do(req)
	if err != nil {
		t.Fatalf("upload icon: %v", err)
	}
	return resp, readBody(t, resp)
}

func pngHeader(b []byte) bool {
	return len(b) >= 8 && bytes.Equal(b[:8], []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a})
}

func TestToolIconVisibilityAndLifecycle(t *testing.T) {
	ts := newTestServer(t)
	owner := ts.client(t)
	signup(t, ts, owner, "owner@example.com", "password123")

	pub := createTool(t, ts, owner, toolInput{Name: "Pub", Address: "10.0.0.1", Port: 80})
	restricted := createTool(t, ts, owner, toolInput{Name: "Sec", Visibility: "restricted"})

	// Owner uploads icons for both.
	if resp, body := uploadIcon(t, ts, owner, "/api/v1/tools/"+pub.ID+"/icon", pngBytes(t)); resp.StatusCode != http.StatusOK {
		t.Fatalf("upload pub icon = %d: %s", resp.StatusCode, body)
	}
	if resp, _ := uploadIcon(t, ts, owner, "/api/v1/tools/"+restricted.ID+"/icon", pngBytes(t)); resp.StatusCode != http.StatusOK {
		t.Fatalf("upload restricted icon = %d", resp.StatusCode)
	}

	// Public tool icon serves to an anonymous caller, bytes intact.
	resp, body := ts.do(t, nil, http.MethodGet, "/api/v1/tools/"+pub.ID+"/icon", nil, nil)
	if resp.StatusCode != http.StatusOK || !pngHeader(body) {
		t.Fatalf("anon public icon = %d, png=%v", resp.StatusCode, pngHeader(body))
	}
	// Restricted tool icon is hidden from anonymous callers (404, no existence leak).
	if resp, _ := ts.do(t, nil, http.MethodGet, "/api/v1/tools/"+restricted.ID+"/icon", nil, nil); resp.StatusCode != http.StatusNotFound {
		t.Errorf("anon restricted icon = %d, want 404", resp.StatusCode)
	}
	// The owner may fetch the restricted icon.
	if resp, _ := ts.do(t, owner, http.MethodGet, "/api/v1/tools/"+restricted.ID+"/icon", nil, nil); resp.StatusCode != http.StatusOK {
		t.Errorf("owner restricted icon = %d, want 200", resp.StatusCode)
	}

	// A non-owner basic user cannot upload an icon.
	other := ts.client(t)
	signup(t, ts, other, "other@example.com", "password123")
	if resp, _ := uploadIcon(t, ts, other, "/api/v1/tools/"+pub.ID+"/icon", pngBytes(t)); resp.StatusCode != http.StatusForbidden {
		t.Errorf("non-owner upload = %d, want 403", resp.StatusCode)
	}

	// Delete clears it.
	if resp, _ := ts.do(t, owner, http.MethodDelete, "/api/v1/tools/"+pub.ID+"/icon", nil, nil); resp.StatusCode != http.StatusOK {
		t.Fatalf("delete icon = %d", resp.StatusCode)
	}
	if resp, _ := ts.do(t, nil, http.MethodGet, "/api/v1/tools/"+pub.ID+"/icon", nil, nil); resp.StatusCode != http.StatusNotFound {
		t.Errorf("icon after delete = %d, want 404", resp.StatusCode)
	}
}

func TestHostIconVisibility(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.client(t)
	signup(t, ts, admin, "admin@example.com", "password123")

	h, err := ts.app.db.CreateHost("h1", "linux", "", "tok-hash", 60)
	if err != nil {
		t.Fatalf("create host: %v", err)
	}

	if resp, body := uploadIcon(t, ts, admin, "/api/v1/admin/hosts/"+h.ID+"/icon", pngBytes(t)); resp.StatusCode != http.StatusOK {
		t.Fatalf("admin upload host icon = %d: %s", resp.StatusCode, body)
	}
	// Authenticated caller sees it.
	if resp, _ := ts.do(t, admin, http.MethodGet, "/api/v1/hosts/"+h.ID+"/icon", nil, nil); resp.StatusCode != http.StatusOK {
		t.Errorf("authed host icon = %d, want 200", resp.StatusCode)
	}
	// Anonymous caller does not, until the host backs a public tool.
	if resp, _ := ts.do(t, nil, http.MethodGet, "/api/v1/hosts/"+h.ID+"/icon", nil, nil); resp.StatusCode != http.StatusNotFound {
		t.Errorf("anon host icon (no public tool) = %d, want 404", resp.StatusCode)
	}
	createTool(t, ts, admin, toolInput{Name: "OnHost", HostID: h.ID})
	if resp, _ := ts.do(t, nil, http.MethodGet, "/api/v1/hosts/"+h.ID+"/icon", nil, nil); resp.StatusCode != http.StatusOK {
		t.Errorf("anon host icon (public tool) = %d, want 200", resp.StatusCode)
	}
}
