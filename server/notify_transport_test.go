package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestPostWebhookSendsPayload(t *testing.T) {
	var gotBody, gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b := make([]byte, r.ContentLength)
		r.Body.Read(b)
		gotBody = string(b)
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	if err := postWebhook(notifyChannel{URL: srv.URL}, `{"x":1}`); err != nil {
		t.Fatalf("postWebhook: %v", err)
	}
	if gotBody != `{"x":1}` {
		t.Errorf("body = %q", gotBody)
	}
	if gotAuth != "" {
		t.Errorf("unexpected auth header %q", gotAuth)
	}
}

func TestPostWebhookBearerToken(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	ch := notifyChannel{URL: srv.URL, Config: map[string]string{"token": "tok123"}}
	if err := postWebhook(ch, "alert"); err != nil {
		t.Fatalf("postWebhook: %v", err)
	}
	if gotAuth != "Bearer tok123" {
		t.Errorf("auth = %q, want Bearer tok123", gotAuth)
	}
}

// Discord and Slack answer 400 to a body without their own text field, so the
// URL decides the shape. Every other receiver keeps getting Reeve's JSON.
func TestChatPayloadShapePerReceiver(t *testing.T) {
	const alert = `{"event":"fired","severity":"error","tool":"postgres","host":"db-1","message":"cpu 97%"}`
	const line = "[error] postgres on db-1: cpu 97%"
	for _, tc := range []struct{ name, url, want string }{
		{"discord", "https://discord.com/api/webhooks/1/abc", `{"content":"` + line + `"}`},
		{"discordapp", "https://discordapp.com/api/webhooks/1/abc", `{"content":"` + line + `"}`},
		{"slack", "https://hooks.slack.com/services/T/B/x", `{"text":"` + line + `"}`},
		{"google chat", "https://chat.googleapis.com/v1/spaces/x/messages?key=k", `{"text":"` + line + `"}`},
		{"generic sink", "https://sink.example.com/hook", alert},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := alert
			if field := chatPayloadField(tc.url); field != "" {
				got = asChatMessage(field, alert)
			}
			if got != tc.want {
				t.Errorf("body = %s, want %s", got, tc.want)
			}
		})
	}
}

// A payload with no message must not vanish into an empty chat post, which is
// exactly the 400 Discord answers with.
func TestChatMessageKeepsAPayloadWithNoMessage(t *testing.T) {
	if got := asChatMessage("content", `{"x":1}`); got != `{"content":"{\"x\":1}"}` {
		t.Errorf("messageless payload = %s, want the raw JSON in the field", got)
	}
	if got := asChatMessage("content", "not json"); got != "not json" {
		t.Errorf("unparseable payload = %s, want it through untouched", got)
	}
}

// "webhook returned 400" alone told nobody which field the receiver wanted.
func TestPostWebhookErrorCarriesResponseBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"message":"Cannot send an empty message","code":50006}`))
	}))
	defer srv.Close()
	err := postWebhook(notifyChannel{URL: srv.URL}, `{"x":1}`)
	if err == nil {
		t.Fatal("postWebhook on a 400 returned nil, want an error")
	}
	if !strings.Contains(err.Error(), "Cannot send an empty message") {
		t.Errorf("error = %q, want the receiver's reason", err)
	}
}

// TestDispatchMarksSentAndFailed pins the delivery bookkeeping: a transport
// error must record the failure and its message, and a success must not.
func TestDispatchMarksSentAndFailed(t *testing.T) {
	ts := newTestServer(t)
	var sent []string
	ts.app.send = func(ch notifyChannel, _ string) error {
		sent = append(sent, ch.URL)
		if ch.URL == "http://sink.invalid" {
			return errors.New("boom")
		}
		return nil
	}

	okCh, _ := ts.app.db.CreateWebhook("global", "", "http://ok.invalid", "webhook", "{}", "info")
	ts.app.db.EnqueueDelivery(okCh.ID, `{"x":1}`, time.Now().UTC())
	failCh, _ := ts.app.db.CreateWebhook("global", "", "http://sink.invalid", "generic", "{}", "info")
	ts.app.db.EnqueueDelivery(failCh.ID, `{"x":1}`, time.Now().UTC())

	n := ts.app.dispatchDue(time.Now().UTC())
	if n != 2 {
		t.Fatalf("attempted %d, want 2", n)
	}
	if len(sent) != 2 {
		t.Fatalf("transport calls = %v, want 2", sent)
	}
	if got := countRows(t, ts, `SELECT COUNT(*) FROM webhook_deliveries WHERE status='sent'`); got != 1 {
		t.Errorf("sent = %d, want 1", got)
	}
	if got := countRows(t, ts, `SELECT COUNT(*) FROM webhook_deliveries WHERE status='failed' AND last_error='boom'`); got != 1 {
		t.Errorf("failed with message = %d, want 1", got)
	}
}

// A channel with no URL cannot be delivered, and the attempt must be recorded as
// failed rather than silently dropped.
func TestDispatchFailsChannelWithNoURL(t *testing.T) {
	ts := newTestServer(t)
	ch, _ := ts.app.db.CreateWebhook("global", "", "", "webhook", "{}", "info")
	ts.app.db.EnqueueDelivery(ch.ID, `{"x":1}`, time.Now().UTC())

	if n := ts.app.dispatchDue(time.Now().UTC()); n != 1 {
		t.Fatalf("attempted %d, want 1", n)
	}
	if got := countRows(t, ts, `SELECT COUNT(*) FROM webhook_deliveries WHERE status='failed'`); got != 1 {
		t.Errorf("failed = %d, want 1", got)
	}
}

func TestDueDeliveriesReturnsKindConfig(t *testing.T) {
	ts := newTestServer(t)
	sealed, err := ts.app.sealConfig(map[string]string{"host": "mx", "password": "secret"})
	if err != nil {
		t.Fatalf("sealConfig: %v", err)
	}
	ch, _ := ts.app.db.CreateWebhook("global", "", "http://mx.invalid", "webhook", sealed, "info")
	ts.app.db.EnqueueDelivery(ch.ID, `{"x":1}`, time.Now().UTC())

	due, err := ts.app.db.DueDeliveries(time.Now().UTC(), 10)
	if err != nil {
		t.Fatalf("DueDeliveries: %v", err)
	}
	if len(due) != 1 {
		t.Fatalf("due = %d, want 1", len(due))
	}
	if due[0].Kind != "webhook" {
		t.Errorf("kind = %q, want webhook", due[0].Kind)
	}
	cfg, err := ts.app.openConfig(due[0].Config)
	if err != nil {
		t.Fatalf("openConfig: %v", err)
	}
	if cfg["host"] != "mx" || cfg["password"] != "secret" {
		t.Errorf("decrypted config = %v", cfg)
	}
}
