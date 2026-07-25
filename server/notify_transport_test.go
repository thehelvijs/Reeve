package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func testClient() *http.Client {
	return &http.Client{Timeout: 5 * time.Second}
}

func TestHTTPNotifierGeneric(t *testing.T) {
	var gotBody, gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b := make([]byte, r.ContentLength)
		r.Body.Read(b)
		gotBody = string(b)
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	n := &httpNotifier{client: testClient()}
	if err := n.Send(notifyChannel{URL: srv.URL}, `{"x":1}`, time.Now()); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if gotBody != `{"x":1}` {
		t.Errorf("body = %q", gotBody)
	}
	if gotAuth != "" {
		t.Errorf("unexpected auth header %q", gotAuth)
	}
}

func TestHTTPNotifierBearerToken(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	n := &httpNotifier{client: testClient()}
	ch := notifyChannel{URL: srv.URL, Config: map[string]string{"token": "tok123"}}
	if err := n.Send(ch, "alert", time.Now()); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if gotAuth != "Bearer tok123" {
		t.Errorf("auth = %q, want Bearer tok123", gotAuth)
	}
}

// recordNotifier records which kind handled a delivery and can force an error.
type recordNotifier struct {
	kind string
	log  *[]string
	fail bool
}

func (r *recordNotifier) Send(_ notifyChannel, _ string, _ time.Time) error {
	*r.log = append(*r.log, r.kind)
	if r.fail {
		return errTestSend
	}
	return nil
}

var errTestSend = &sendErr{}

type sendErr struct{}

func (*sendErr) Error() string { return "boom" }

func TestDispatchPicksNotifier(t *testing.T) {
	ts := newTestServer(t)
	var log []string
	ts.app.notifiers = map[string]Notifier{
		"webhook": &recordNotifier{kind: "webhook", log: &log},
		"generic": &recordNotifier{kind: "generic", log: &log, fail: true},
	}

	okCh, _ := ts.app.db.CreateWebhook("global", "", "http://ok.invalid", "webhook", "{}", "info")
	ts.app.db.EnqueueDelivery(okCh.ID, `{"x":1}`, time.Now().UTC())
	failCh, _ := ts.app.db.CreateWebhook("global", "", "http://sink.invalid", "generic", "{}", "info")
	ts.app.db.EnqueueDelivery(failCh.ID, `{"x":1}`, time.Now().UTC())

	n := ts.app.dispatchDue(time.Now().UTC())
	if n != 2 {
		t.Fatalf("attempted %d, want 2", n)
	}
	if len(log) != 2 {
		t.Fatalf("notifier calls = %v, want 2", log)
	}
	if got := countRows(t, ts, `SELECT COUNT(*) FROM webhook_deliveries WHERE status='sent'`); got != 1 {
		t.Errorf("sent = %d, want 1", got)
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
