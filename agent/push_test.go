package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
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
