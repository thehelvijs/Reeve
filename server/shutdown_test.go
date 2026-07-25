package main

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestServeDrainsInFlightRequest asserts a request already being handled when
// the shutdown signal lands still completes, instead of being severed.
func TestServeDrainsInFlightRequest(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := ln.Addr().String()
	ln.Close()

	started := make(chan struct{})
	mux := http.NewServeMux()
	mux.HandleFunc("/slow", func(w http.ResponseWriter, _ *http.Request) {
		close(started)
		time.Sleep(200 * time.Millisecond)
		io.WriteString(w, "finished")
	})
	srv := &http.Server{Addr: addr, Handler: mux}

	ctx, cancel := context.WithCancel(context.Background())
	serveDone := make(chan error, 1)
	go func() { serveDone <- serve(ctx, srv) }()

	waitForListener(t, addr)
	respDone := make(chan string, 1)
	go func() {
		resp, err := http.Get("http://" + addr + "/slow")
		if err != nil {
			respDone <- "error: " + err.Error()
			return
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		respDone <- string(body)
	}()

	<-started
	cancel()

	select {
	case body := <-respDone:
		if body != "finished" {
			t.Errorf("in-flight response body = %q, want %q", body, "finished")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("in-flight request never completed")
	}
	select {
	case err := <-serveDone:
		if err != nil {
			t.Errorf("serve returned %v, want nil", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("serve did not return after shutdown")
	}
}

// TestServeReturnsListenError surfaces a bind failure instead of hanging.
func TestServeReturnsListenError(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	srv := &http.Server{Addr: ln.Addr().String(), Handler: http.NewServeMux()}
	err = serve(context.Background(), srv)
	if err == nil {
		t.Fatal("serve on an occupied port returned nil, want an error")
	}
}

// TestEveryStopsOnCancel guards the background loops against outliving the
// process context and writing to a closed database.
func TestEveryStopsOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		every(ctx, time.Millisecond, func() {})
		close(done)
	}()
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("every did not return after cancel")
	}
}

func waitForListener(t *testing.T, addr string) {
	t.Helper()
	for i := 0; i < 200; i++ {
		c, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if err == nil {
			c.Close()
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("server never came up on %s", addr)
}

// The container healthcheck shells out to this binary, so the probe must agree
// with a live /healthz and fail when nothing is listening.
func TestProbeHealth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/healthz" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	addr := strings.TrimPrefix(srv.URL, "http://")
	if err := probeHealth(addr); err != nil {
		t.Errorf("probeHealth on a live server = %v, want nil", err)
	}
	_, port, _ := net.SplitHostPort(addr)
	if err := probeHealth("0.0.0.0:" + port); err != nil {
		t.Errorf("probeHealth on 0.0.0.0 = %v, want nil (dialed on loopback)", err)
	}

	srv.Close()
	if err := probeHealth(addr); err == nil {
		t.Error("probeHealth against a closed port = nil, want an error")
	}
	if err := probeHealth("not-an-address"); err == nil {
		t.Error("probeHealth with a malformed address = nil, want an error")
	}
}

// A non-200 from /healthz means unhealthy, not healthy-with-a-warning.
func TestProbeHealthFailsOnNon200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()
	if err := probeHealth(strings.TrimPrefix(srv.URL, "http://")); err == nil {
		t.Error("probeHealth on a 503 = nil, want an error")
	}
}
