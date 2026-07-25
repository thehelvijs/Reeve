package main

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func captureLog(t *testing.T) (*bytes.Buffer, func()) {
	t.Helper()
	buf := &bytes.Buffer{}
	prevOut := log.Writer()
	prevFlags := log.Flags()
	log.SetOutput(buf)
	log.SetFlags(0)
	return buf, func() {
		log.SetOutput(prevOut)
		log.SetFlags(prevFlags)
	}
}

func TestLogRequestsLogsApiNotStatic(t *testing.T) {
	buf, restore := captureLog(t)
	defer restore()
	a := &app{cfg: config{LogRequests: true}}
	h := a.logRequests(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/tools", nil))
	out := buf.String()
	if !strings.Contains(out, "GET /api/v1/tools") || !strings.Contains(out, "418") {
		t.Errorf("expected api log line with method/path/status, got %q", out)
	}

	buf.Reset()
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/assets/app.js", nil))
	if buf.String() != "" {
		t.Errorf("static path should not be logged, got %q", buf.String())
	}
}

func TestLogRequestsToggleOff(t *testing.T) {
	buf, restore := captureLog(t)
	defer restore()
	a := &app{cfg: config{LogRequests: false}}
	h := a.logRequests(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/tools", nil))
	if buf.String() != "" {
		t.Errorf("logging disabled but produced %q", buf.String())
	}
}

func TestClientIPForwardedFor(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/tools", nil)
	r.Header.Set("X-Forwarded-For", "203.0.113.9, 10.0.0.1")
	if got := clientIP(r); got != "203.0.113.9" {
		t.Errorf("clientIP = %q, want 203.0.113.9", got)
	}
	r2 := httptest.NewRequest(http.MethodGet, "/api/v1/tools", nil)
	r2.RemoteAddr = "192.168.1.5:54321"
	if got := clientIP(r2); got != "192.168.1.5" {
		t.Errorf("clientIP = %q, want 192.168.1.5", got)
	}
}
