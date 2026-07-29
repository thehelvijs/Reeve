package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/thehelvijs/Reeve/server/internal/auth"
	"github.com/thehelvijs/Reeve/server/internal/rbac"
	"github.com/thehelvijs/Reeve/server/internal/store"
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

	okCh, _ := ts.app.db.CreateWebhook(store.Webhook{OwnerType: "global", URL: "http://ok.invalid", Format: "webhook", Config: "{}", MinSeverity: "info"})
	ts.app.db.EnqueueDelivery(okCh.ID, `{"x":1}`, time.Now().UTC())
	failCh, _ := ts.app.db.CreateWebhook(store.Webhook{OwnerType: "global", URL: "http://sink.invalid", Format: "generic", Config: "{}", MinSeverity: "info"})
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
	ch, _ := ts.app.db.CreateWebhook(store.Webhook{OwnerType: "global", Format: "webhook", Config: "{}", MinSeverity: "info"})
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
	ch, _ := ts.app.db.CreateWebhook(store.Webhook{OwnerType: "global", URL: "http://mx.invalid", Format: "webhook", Config: sealed, MinSeverity: "info"})
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

// An alert fires from a ticker, so there is no request to read the host from,
// and REEVE_PUBLIC_URL is empty by default. Without a fallback every LAN
// install would send links-free alerts, so the last address an admin reached
// the server on is what the dispatcher uses.
func TestDispatchLinksToTheAddressAdminsUse(t *testing.T) {
	ts := newTestServer(t)
	var seen notifyChannel
	ts.app.send = func(ch notifyChannel, _ string) error {
		seen = ch
		return nil
	}
	hook, _ := ts.app.db.CreateWebhook(store.Webhook{
		OwnerType: "global", URL: "http://sink.invalid", Format: fmtDiscord})
	now := time.Now().UTC()

	// Nothing seen yet: no link is better than one nobody can follow.
	ts.app.db.EnqueueDelivery(hook.ID, `{"severity":"error","message":"boom"}`, now)
	ts.app.dispatchDue(now)
	if seen.BaseURL != "" {
		t.Errorf("base url = %q before any admin request, want empty", seen.BaseURL)
	}

	// A loopback origin is the admin on the box itself; it must not be kept.
	ts.app.rememberOrigin(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})).
		ServeHTTP(httptest.NewRecorder(), adminRequest(t, "http://localhost:8080/api/hosts"))
	if got := ts.app.publicBase(); got != "" {
		t.Errorf("base url = %q after a localhost request, want empty", got)
	}

	ts.app.rememberOrigin(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})).
		ServeHTTP(httptest.NewRecorder(), adminRequest(t, "http://192.168.1.50:8080/api/hosts"))
	ts.app.db.EnqueueDelivery(hook.ID, `{"severity":"error","message":"boom","host_id":"h1"}`, now)
	ts.app.dispatchDue(now)
	if seen.BaseURL != "http://192.168.1.50:8080" {
		t.Errorf("base url = %q, want the LAN address the admin used", seen.BaseURL)
	}

	// A configured public URL always wins over whatever a browser reported.
	ts.app.cfg.PublicURL = "https://reeve.example.com/"
	if got := ts.app.publicBase(); got != "https://reeve.example.com" {
		t.Errorf("base url = %q, want the configured public url", got)
	}
}

// adminRequest is a request already carrying an admin principal, which is the
// only kind rememberOrigin trusts.
func adminRequest(t *testing.T, url string) *http.Request {
	t.Helper()
	r := httptest.NewRequest(http.MethodGet, url, nil)
	return r.WithContext(rbac.WithPrincipal(r.Context(),
		auth.Principal{UserID: "u1", Email: "boss@example.com", Role: store.RoleAdmin}))
}
