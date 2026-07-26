package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/thehelvijs/Reeve/contracts"
)

func TestSendReturnsTheAck(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"check_now":true}`)
	}))
	defer srv.Close()

	p := newPusher(config{ServerURL: srv.URL, Token: "t"})
	p.bufferDir = t.TempDir()

	ack, err := p.send(contracts.Push{ProtocolVersion: contracts.PushProtocolVersion})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if ack == nil || !ack.CheckNow {
		t.Fatalf("ack = %+v, want check_now true", ack)
	}
}

func TestMalformedAckIsNotContact(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `<html>not json</html>`)
	}))
	defer srv.Close()

	p := newPusher(config{ServerURL: srv.URL, Token: "t"})
	p.bufferDir = t.TempDir()

	ack, err := p.send(contracts.Push{ProtocolVersion: contracts.PushProtocolVersion})
	if err != nil {
		t.Fatalf("send returned an error for an accepted push: %v", err)
	}
	if ack != nil {
		t.Error("a malformed body was accepted as an ack")
	}
}

func TestNon2xxBuffersThePush(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	p := newPusher(config{ServerURL: srv.URL, Token: "t"})
	p.bufferDir = t.TempDir()

	_, err := p.send(contracts.Push{ProtocolVersion: contracts.PushProtocolVersion})
	if err == nil {
		t.Fatal("send returned no error for a 400 response")
	}
	entries, readErr := os.ReadDir(p.bufferDir)
	if readErr != nil {
		t.Fatalf("ReadDir: %v", readErr)
	}
	if len(entries) != 1 {
		t.Fatalf("buffered files = %d, want 1", len(entries))
	}
}

// seedBuffer writes bodies into the buffer under names that replay in the given
// order.
func seedBuffer(t *testing.T, p *pusher, bodies ...string) {
	t.Helper()
	for i, body := range bodies {
		name := filepath.Join(p.bufferDir, fmt.Sprintf("%03d-seed.json", i))
		if err := os.WriteFile(name, []byte(body), 0o600); err != nil {
			t.Fatalf("seed buffer: %v", err)
		}
	}
}

// A body the server permanently refuses must not hold the queue: everything
// buffered behind it would never be delivered, silently killing offline
// buffering on that host.
func TestFlushBufferDropsAPermanentlyRejectedBody(t *testing.T) {
	var got []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		got = append(got, string(body))
		if string(body) == `{"doomed":true}` {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		io.WriteString(w, `{"check_now":false}`)
	}))
	defer srv.Close()

	p := newPusher(config{ServerURL: srv.URL, Token: "t"})
	p.bufferDir = t.TempDir()
	seedBuffer(t, p, `{"doomed":true}`, `{"good":true}`)

	p.flushBuffer()

	entries, err := os.ReadDir(p.bufferDir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("buffered files after flush = %d, want 0", len(entries))
	}
	want := []string{`{"doomed":true}`, `{"good":true}`}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("posted bodies = %v, want %v", got, want)
	}
}

func TestFlushBufferKeepsEverythingOnServerError(t *testing.T) {
	posts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		posts++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	p := newPusher(config{ServerURL: srv.URL, Token: "t"})
	p.bufferDir = t.TempDir()
	seedBuffer(t, p, `{"first":true}`, `{"second":true}`)

	p.flushBuffer()

	entries, err := os.ReadDir(p.bufferDir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("buffered files after a 5xx flush = %d, want 2", len(entries))
	}
	if posts != 1 {
		t.Errorf("posts = %d, want 1: the flush must stop at the first retryable failure", posts)
	}
}

// A 401 says the caller isn't authenticated right now, not that the body is
// bad. During an auth outage (rotated enrollment token, server restored from
// a backup predating enrollment) the fix must not drop telemetry that would
// have delivered the moment the token is fixed.
func TestFlushBufferKeepsEverythingOn401(t *testing.T) {
	posts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		posts++
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	p := newPusher(config{ServerURL: srv.URL, Token: "t"})
	p.bufferDir = t.TempDir()
	seedBuffer(t, p, `{"first":true}`, `{"second":true}`)

	p.flushBuffer()

	entries, err := os.ReadDir(p.bufferDir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("buffered files after a 401 flush = %d, want 2", len(entries))
	}
	if posts != 1 {
		t.Errorf("posts = %d, want 1: the flush must stop at the first retryable failure", posts)
	}
}

func TestPermanentRejectSpansOnlyNonRetryable4xx(t *testing.T) {
	cases := []struct {
		err  error
		want bool
	}{
		{&statusError{code: http.StatusBadRequest}, true},
		{&statusError{code: http.StatusUnauthorized}, false},
		{&statusError{code: http.StatusForbidden}, false},
		{&statusError{code: http.StatusRequestTimeout}, false},
		{&statusError{code: http.StatusTooManyRequests}, false},
		{&statusError{code: http.StatusInternalServerError}, false},
		{&statusError{code: http.StatusBadGateway}, false},
		{errors.New("dial tcp: connection refused"), false},
	}
	for _, c := range cases {
		if got := permanentReject(c.err); got != c.want {
			t.Errorf("permanentReject(%v) = %v, want %v", c.err, got, c.want)
		}
	}
}
