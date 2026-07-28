package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"
)

// The probe answers 200 with a verdict rather than an HTTP error, because a
// receiver that refuses the payload is the thing being reported, not a fault in
// this API. A saved channel is named by id so the server puts the real token on
// the wire; an unsaved one carries the URL the form holds.
func TestWebhookProbeReportsBothOutcomes(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	var seen []notifyChannel
	ts.app.send = func(ch notifyChannel, payload string) error {
		seen = append(seen, ch)
		if ch.URL == "http://bad.invalid/hook" {
			return errors.New("webhook returned 500")
		}
		var body map[string]any
		if err := json.Unmarshal([]byte(payload), &body); err != nil {
			t.Errorf("probe payload is not JSON: %v", err)
		}
		if body["event"] != "test" {
			t.Errorf("probe payload = %v, want event test", body)
		}
		return nil
	}

	saved, err := ts.app.db.CreateWebhook("global", "", "http://ok.invalid/hook", "webhook", mustSeal(t, ts, "tok123"), "info")
	if err != nil {
		t.Fatal(err)
	}

	resp, data := ts.do(t, admin, http.MethodPost, "/api/admin/webhooks/test", map[string]string{"id": saved.ID}, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("probe a saved channel = %d: %s", resp.StatusCode, data)
	}
	var out struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	json.Unmarshal(data, &out)
	if !out.OK {
		t.Errorf("saved channel probe = %+v, want ok", out)
	}
	if len(seen) != 1 || seen[0].Config["token"] != "tok123" {
		t.Errorf("probe sent %+v, want the stored token", seen)
	}

	// An unsaved channel: the URL comes from the request.
	_, data = ts.do(t, admin, http.MethodPost, "/api/admin/webhooks/test",
		map[string]any{"url": "http://bad.invalid/hook"}, nil)
	json.Unmarshal(data, &out)
	if out.OK || out.Error == "" {
		t.Errorf("failing probe = %+v, want ok=false with a reason", out)
	}
}

func TestWebhookProbeRejectsABadTarget(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	ts.app.send = func(notifyChannel, string) error { return nil }

	for _, tc := range []struct {
		name string
		body map[string]any
		want int
	}{
		{"no scheme", map[string]any{"url": "sink.example.com/hook"}, http.StatusBadRequest},
		{"empty", map[string]any{}, http.StatusBadRequest},
		{"unknown id", map[string]any{"id": "nope"}, http.StatusNotFound},
	} {
		resp, data := ts.do(t, admin, http.MethodPost, "/api/admin/webhooks/test", tc.body, nil)
		if resp.StatusCode != tc.want {
			t.Errorf("%s = %d, want %d: %s", tc.name, resp.StatusCode, tc.want, data)
		}
	}
}

func TestWebhookProbeIsAdminOnly(t *testing.T) {
	ts := newTestServer(t)
	adminClient(t, ts)
	basic := ts.client(t)
	signup(t, ts, basic, "basic@example.com", "password123")
	resp, _ := ts.do(t, basic, http.MethodPost, "/api/admin/webhooks/test",
		map[string]any{"url": "http://sink.invalid/hook"}, nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("basic probe = %d, want 403", resp.StatusCode)
	}
}

func mustSeal(t *testing.T, ts *testServer, token string) string {
	t.Helper()
	sealed, err := ts.app.sealConfig(map[string]string{"token": token})
	if err != nil {
		t.Fatal(err)
	}
	return sealed
}
