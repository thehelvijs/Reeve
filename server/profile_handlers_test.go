package main

import (
	"bytes"
	"encoding/json"
	"image"
	"image/png"
	"mime/multipart"
	"net/http"
	"testing"
)

func pngBytes(t *testing.T) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return b.Bytes()
}

func uploadAvatar(t *testing.T, ts *testServer, c *http.Client, data []byte) (*http.Response, []byte) {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("avatar", "a.png")
	fw.Write(data)
	mw.Close()
	req, _ := http.NewRequest(http.MethodPost, ts.srv.URL+"/api/v1/me/avatar", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	resp, err := c.Do(req)
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	body := readBody(t, resp)
	return resp, body
}

func readBody(t *testing.T, resp *http.Response) []byte {
	t.Helper()
	defer resp.Body.Close()
	var buf bytes.Buffer
	buf.ReadFrom(resp.Body)
	return buf.Bytes()
}

func TestUpdateDisplayName(t *testing.T) {
	ts := newTestServer(t)
	c := ts.client(t)
	signup(t, ts, c, "boss@example.com", "password123")

	resp, data := ts.do(t, c, http.MethodPatch, "/api/v1/me", map[string]string{"display_name": "  Boss Lady  "}, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("patch me = %d: %s", resp.StatusCode, data)
	}
	var v userView
	json.Unmarshal(data, &v)
	if v.DisplayName != "Boss Lady" {
		t.Errorf("display name = %q, want trimmed 'Boss Lady'", v.DisplayName)
	}

	long := make([]byte, maxDisplayNameLen+1)
	for i := range long {
		long[i] = 'x'
	}
	resp, _ = ts.do(t, c, http.MethodPatch, "/api/v1/me", map[string]string{"display_name": string(long)}, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("over-long name status = %d, want 400", resp.StatusCode)
	}
}

func TestChangePassword(t *testing.T) {
	ts := newTestServer(t)
	c := ts.client(t)
	signup(t, ts, c, "boss@example.com", "password123")

	// Wrong current password is rejected.
	resp, _ := ts.do(t, c, http.MethodPost, "/api/v1/me/password",
		map[string]string{"current_password": "nope", "new_password": "newpassword1"}, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("wrong current = %d, want 400", resp.StatusCode)
	}
	// Weak new password is rejected.
	resp, _ = ts.do(t, c, http.MethodPost, "/api/v1/me/password",
		map[string]string{"current_password": "password123", "new_password": "short"}, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("weak new = %d, want 400", resp.StatusCode)
	}
	// Valid change succeeds; new password logs in, old does not.
	resp, _ = ts.do(t, c, http.MethodPost, "/api/v1/me/password",
		map[string]string{"current_password": "password123", "new_password": "newpassword1"}, nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("change = %d, want 204", resp.StatusCode)
	}
	fresh := ts.client(t)
	resp, _ = ts.do(t, fresh, http.MethodPost, "/api/v1/auth/login",
		map[string]string{"email": "boss@example.com", "password": "password123"}, nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("old password login = %d, want 401", resp.StatusCode)
	}
	resp, _ = ts.do(t, fresh, http.MethodPost, "/api/v1/auth/login",
		map[string]string{"email": "boss@example.com", "password": "newpassword1"}, nil)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("new password login = %d, want 200", resp.StatusCode)
	}
}

func TestAvatarUploadServeDelete(t *testing.T) {
	ts := newTestServer(t)
	c := ts.client(t)
	_, su := signup(t, ts, c, "boss@example.com", "password123")

	resp, data := uploadAvatar(t, ts, c, pngBytes(t))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("upload = %d: %s", resp.StatusCode, data)
	}
	var v userView
	json.Unmarshal(data, &v)
	want := "/api/v1/users/" + su.ID + "/avatar"
	if v.AvatarURL != want {
		t.Fatalf("avatar_url = %q, want %q", v.AvatarURL, want)
	}

	resp, img := ts.do(t, c, http.MethodGet, want, nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get avatar = %d", resp.StatusCode)
	}
	if !bytes.Equal(img[:8], []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}) {
		t.Errorf("served bytes are not the PNG we stored")
	}

	resp, _ = ts.do(t, c, http.MethodDelete, "/api/v1/me/avatar", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("delete avatar = %d", resp.StatusCode)
	}
	resp, _ = ts.do(t, c, http.MethodGet, want, nil, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("avatar after delete = %d, want 404", resp.StatusCode)
	}
}

func TestAvatarRejectsNonImageAndOversize(t *testing.T) {
	ts := newTestServer(t)
	c := ts.client(t)
	signup(t, ts, c, "boss@example.com", "password123")

	resp, _ := uploadAvatar(t, ts, c, []byte("this is definitely not an image"))
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("text upload = %d, want 400", resp.StatusCode)
	}
	resp, _ = uploadAvatar(t, ts, c, make([]byte, maxImageBytes+10))
	if resp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Errorf("oversize upload = %d, want 413", resp.StatusCode)
	}
}

func TestSelfDeleteReassignsToolsAndEndsSession(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.client(t)
	signup(t, ts, admin, "admin@example.com", "password123")
	basic := ts.client(t)
	signup(t, ts, basic, "basic@example.com", "password123")

	createTool(t, ts, basic, toolInput{Name: "BasicGrafana", Address: "10.0.0.5", Port: 3000})

	resp, _ := ts.do(t, basic, http.MethodDelete, "/api/v1/me", nil, nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("self-delete = %d, want 204", resp.StatusCode)
	}
	// Session is gone.
	resp, _ = ts.do(t, basic, http.MethodGet, "/api/v1/me", nil, nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("deleted user /me = %d, want 401", resp.StatusCode)
	}
	// The tool survived, reassigned to the admin.
	resp, data := ts.do(t, admin, http.MethodGet, "/api/v1/tools", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list tools = %d", resp.StatusCode)
	}
	if !bytes.Contains(data, []byte("BasicGrafana")) {
		t.Errorf("tool was not preserved after owner self-delete: %s", data)
	}
}

func TestLastAdminCannotSelfDelete(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.client(t)
	signup(t, ts, admin, "admin@example.com", "password123")

	resp, data := ts.do(t, admin, http.MethodDelete, "/api/v1/me", nil, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("last admin self-delete = %d, want 400: %s", resp.StatusCode, data)
	}
	if !bytes.Contains(data, []byte("last_admin")) {
		t.Errorf("expected last_admin code: %s", data)
	}
}

func TestAdminDeleteUser(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.client(t)
	_, adminView := signup(t, ts, admin, "admin@example.com", "password123")
	basic := ts.client(t)
	_, basicView := signup(t, ts, basic, "basic@example.com", "password123")

	// Admin cannot delete self via the admin endpoint.
	resp, data := ts.do(t, admin, http.MethodDelete, "/api/v1/admin/users/"+adminView.ID, nil, nil)
	if resp.StatusCode != http.StatusBadRequest || !bytes.Contains(data, []byte("self_delete")) {
		t.Fatalf("admin self-delete = %d: %s", resp.StatusCode, data)
	}
	// Admin deletes the basic user.
	resp, _ = ts.do(t, admin, http.MethodDelete, "/api/v1/admin/users/"+basicView.ID, nil, nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("admin delete basic = %d, want 204", resp.StatusCode)
	}
	resp, data = ts.do(t, admin, http.MethodGet, "/api/v1/admin/users", nil, nil)
	if bytes.Contains(data, []byte("basic@example.com")) {
		t.Errorf("deleted user still listed: %s", data)
	}
}
