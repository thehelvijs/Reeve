package main

import (
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/thehelvijs/Reeve/server/internal/store"
)

// The notify tests drive channels through the store directly; these cover the
// admin HTTP surface, which is how a channel is actually created and removed.
func TestWebhookCRUDOverHTTP(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.client(t)
	signup(t, ts, admin, "boss@example.com", "password123")

	resp, data := ts.do(t, admin, http.MethodPost, "/api/admin/webhooks", map[string]any{
		"owner_type": "global", "url": "https://sink.invalid/hook",
		"config": map[string]string{"token": "s3cret"}, "min_severity": "warning",
	}, nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create = %d: %s", resp.StatusCode, data)
	}
	var created channelView
	json.Unmarshal(data, &created)
	if created.ID == "" || created.MinSeverity != "warning" {
		t.Fatalf("created channel = %+v", created)
	}
	if created.Config["token"] == "s3cret" {
		t.Error("the config secret came back unredacted")
	}

	_, data = ts.do(t, admin, http.MethodGet, "/api/admin/webhooks", nil, nil)
	var list []channelView
	json.Unmarshal(data, &list)
	if len(list) != 1 || list[0].ID != created.ID {
		t.Fatalf("list = %+v", list)
	}

	if resp, _ = ts.do(t, admin, http.MethodDelete, "/api/admin/webhooks/"+created.ID, nil, nil); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete = %d", resp.StatusCode)
	}
	_, data = ts.do(t, admin, http.MethodGet, "/api/admin/webhooks", nil, nil)
	json.Unmarshal(data, &list)
	if len(list) != 0 {
		t.Errorf("channel survived delete: %+v", list)
	}
	if resp, _ = ts.do(t, admin, http.MethodDelete, "/api/admin/webhooks/"+created.ID, nil, nil); resp.StatusCode != http.StatusNotFound {
		t.Errorf("second delete = %d, want 404", resp.StatusCode)
	}
}

// A channel's receiver and its template are edited after the fact: an operator
// who learns a Discord hook needed a different shape must not have to delete and
// recreate it, and losing the stored token to a blank field is the same loss.
func TestWebhookUpdateEditsFormatAndKeepsTheToken(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.client(t)
	signup(t, ts, admin, "boss@example.com", "password123")

	_, data := ts.do(t, admin, http.MethodPost, "/api/admin/webhooks", map[string]any{
		"owner_type": "global", "url": "https://sink.invalid/hook", "format": "auto",
		"config": map[string]string{"token": "s3cret"}, "min_severity": "info",
	}, nil)
	var created channelView
	json.Unmarshal(data, &created)

	resp, data := ts.do(t, admin, http.MethodPatch, "/api/admin/webhooks/"+created.ID, map[string]any{
		"url": "https://chat.example.com/hooks/abc", "format": "mattermost",
		"min_severity": "warning", "enabled": true,
		"config": map[string]string{"token": ""},
	}, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("patch = %d: %s", resp.StatusCode, data)
	}
	var updated channelView
	json.Unmarshal(data, &updated)
	if updated.Format != "mattermost" || updated.MinSeverity != "warning" {
		t.Fatalf("updated = %+v", updated)
	}

	hook, err := ts.app.db.GetWebhook(created.ID)
	if err != nil {
		t.Fatalf("GetWebhook: %v", err)
	}
	cfg, err := ts.app.openConfig(hook.Config)
	if err != nil {
		t.Fatalf("openConfig: %v", err)
	}
	if cfg["token"] != "s3cret" {
		t.Errorf("token = %q, want the stored one kept by a blank field", cfg["token"])
	}

	// A custom format with no template is refused here, not at the first delivery.
	resp, _ = ts.do(t, admin, http.MethodPatch, "/api/admin/webhooks/"+created.ID, map[string]any{
		"url": hook.URL, "format": "custom", "min_severity": "info", "enabled": true,
	}, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("custom with no template = %d, want 400", resp.StatusCode)
	}
}

// The form's receiver list and its template variables come from the server, so
// the two cannot drift from the shaper that reads them.
func TestChannelFormatsEndpoint(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.client(t)
	signup(t, ts, admin, "boss@example.com", "password123")
	_, data := ts.do(t, admin, http.MethodGet, "/api/admin/webhook-formats", nil, nil)
	var out struct {
		Formats   []string `json:"formats"`
		Variables []string `json:"variables"`
	}
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal: %v (%s)", err, data)
	}
	if len(out.Formats) != len(channelFormats) || out.Formats[0] != fmtAuto {
		t.Errorf("formats = %v", out.Formats)
	}
	for _, want := range []string{"severity", "message", "tool", "host"} {
		if !slices.Contains(out.Variables, want) {
			t.Errorf("variables missing %q: %v", want, out.Variables)
		}
	}
}

func TestWebhookRoutesAreAdminOnly(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.client(t)
	signup(t, ts, admin, "boss@example.com", "password123")
	basic := ts.client(t)
	signup(t, ts, basic, "dev@example.com", "password123")

	hook, err := ts.app.db.CreateWebhook("global", "", "https://sink.invalid", "generic", "{}", "info")
	if err != nil {
		t.Fatalf("create channel: %v", err)
	}
	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/api/admin/webhooks"},
		{http.MethodPost, "/api/admin/webhooks"},
		{http.MethodPatch, "/api/admin/webhooks/" + hook.ID},
		{http.MethodDelete, "/api/admin/webhooks/" + hook.ID},
		{http.MethodGet, "/api/admin/webhook-formats"},
		{http.MethodGet, "/api/admin/alerts"},
		{http.MethodGet, "/api/admin/deliveries"},
	} {
		if resp, _ := ts.do(t, basic, tc.method, tc.path, map[string]any{}, nil); resp.StatusCode != http.StatusForbidden {
			t.Errorf("basic user on %s %s = %d, want 403", tc.method, tc.path, resp.StatusCode)
		}
		if resp, _ := ts.do(t, nil, tc.method, tc.path, map[string]any{}, nil); resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("anonymous on %s %s = %d, want 401", tc.method, tc.path, resp.StatusCode)
		}
	}
}

// The alert and delivery lists are the operator's evidence that notification
// actually happened; both answer an empty array rather than null when there is
// nothing yet, since the UI maps over them.
func TestAlertEventAndDeliveryLists(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.client(t)
	signup(t, ts, admin, "boss@example.com", "password123")

	for _, path := range []string{"/api/admin/alerts", "/api/admin/deliveries"} {
		resp, data := ts.do(t, admin, http.MethodGet, path, nil, nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s = %d: %s", path, resp.StatusCode, data)
		}
		if got := strings.TrimSpace(string(data)); got != "[]" {
			t.Errorf("%s on a fresh instance = %s, want []", path, got)
		}
	}

	hook, err := ts.app.db.CreateWebhook("global", "", "https://sink.invalid", "generic", "{}", "info")
	if err != nil {
		t.Fatalf("create channel: %v", err)
	}
	now := time.Now().UTC()
	if _, err := ts.app.db.CreateAlertEvent(store.AlertEvent{
		SubjectKey: "tool:x", Type: "down", Severity: "error", Message: "Box is down",
	}, now); err != nil {
		t.Fatalf("create alert event: %v", err)
	}
	if err := ts.app.db.EnqueueDelivery(hook.ID, `{"event":"down"}`, now); err != nil {
		t.Fatalf("enqueue delivery: %v", err)
	}

	_, data := ts.do(t, admin, http.MethodGet, "/api/admin/alerts", nil, nil)
	var events []struct {
		ID      string `json:"id"`
		Type    string `json:"type"`
		Message string `json:"message"`
	}
	json.Unmarshal(data, &events)
	if len(events) != 1 || events[0].Type != "down" || events[0].Message != "Box is down" {
		t.Errorf("alert events = %s", data)
	}

	_, data = ts.do(t, admin, http.MethodGet, "/api/admin/deliveries", nil, nil)
	var deliveries []struct {
		WebhookID string `json:"webhook_id"`
		Status    string `json:"status"`
	}
	json.Unmarshal(data, &deliveries)
	if len(deliveries) != 1 || deliveries[0].WebhookID != hook.ID {
		t.Errorf("deliveries = %s", data)
	}
}
