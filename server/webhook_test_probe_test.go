package main

import (
	"encoding/json"
	"errors"
	"github.com/thehelvijs/Reeve/server/internal/store"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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

	saved, err := ts.app.db.CreateWebhook(store.Webhook{OwnerType: "global", URL: "http://ok.invalid/hook", Format: "webhook", Config: mustSeal(t, ts, "tok123"), MinSeverity: "info"})
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

// The probe has to put on the wire exactly what an alert would, or a green Test
// proves nothing. This drives the real route into a real listener and reads the
// bytes back, which is the one seam the shaper's own tests do not cross.
func TestWebhookProbePutsTheShapedBodyOnTheWire(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	type got struct {
		contentType string
		auth        string
		body        string
	}
	var seen got
	sink := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		seen = got{r.Header.Get("Content-Type"), r.Header.Get("Authorization"), string(b)}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer sink.Close()

	saved, err := ts.app.db.CreateWebhook(store.Webhook{OwnerType: "global", URL: sink.URL, Format: fmtDiscord, Config: mustSeal(t, ts, "tok123"), MinSeverity: "info"})
	if err != nil {
		t.Fatal(err)
	}
	resp, data := ts.do(t, admin, http.MethodPost, "/api/admin/webhooks/test", map[string]string{"id": saved.ID}, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("probe = %d: %s", resp.StatusCode, data)
	}
	var out struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	json.Unmarshal(data, &out)
	if !out.OK {
		t.Fatalf("probe verdict = %+v", out)
	}
	var body map[string]string
	if err := json.Unmarshal([]byte(seen.body), &body); err != nil {
		t.Fatalf("wire body is not JSON: %v (%s)", err, seen.body)
	}
	if !strings.HasPrefix(body["content"], "**[info]** Test delivery from Reeve. This channel works.") {
		t.Errorf("content = %q, want the probe message in discord's field", body["content"])
	}
	if len(body) != 1 {
		t.Errorf("discord got extra fields it rejects: %v", body)
	}
	if seen.contentType != "application/json" || seen.auth != "Bearer tok123" {
		t.Errorf("headers = %q / %q", seen.contentType, seen.auth)
	}
}
