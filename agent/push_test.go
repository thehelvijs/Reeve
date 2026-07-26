package main

import (
	"io"
	"net/http"
	"net/http/httptest"
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

func TestGatherReportsTheLocalVeto(t *testing.T) {
	push := gather("1.0.0", config{AutoUpdate: false})
	if !push.AutoUpdateVetoed {
		t.Error("REEVE_AUTO_UPDATE=false was not reported to the server")
	}
	push = gather("1.0.0", config{AutoUpdate: true})
	if push.AutoUpdateVetoed {
		t.Error("a host with auto-update on reported a veto")
	}
}
