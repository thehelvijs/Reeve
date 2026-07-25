package main

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func gzipRequest(t *testing.T, h http.Handler, path string) *http.Response {
	t.Helper()
	r := httptest.NewRequest(http.MethodGet, path, nil)
	r.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	return rec.Result()
}

func TestGzipCompressesJSON(t *testing.T) {
	body := strings.Repeat(`{"metric":"cpu","value":42},`, 200)
	h := gzipResponses(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, body)
	}))

	resp := gzipRequest(t, h, "/api/v1/hosts/x/metrics")
	if resp.Header.Get("Content-Encoding") != "gzip" {
		t.Fatalf("Content-Encoding = %q, want gzip", resp.Header.Get("Content-Encoding"))
	}
	if resp.Header.Get("Vary") != "Accept-Encoding" {
		t.Errorf("Vary = %q, want Accept-Encoding", resp.Header.Get("Vary"))
	}
	raw, _ := io.ReadAll(resp.Body)
	if len(raw) >= len(body) {
		t.Errorf("compressed size %d not smaller than %d", len(raw), len(body))
	}
	zr, err := gzip.NewReader(strings.NewReader(string(raw)))
	if err != nil {
		t.Fatalf("gzip reader: %v", err)
	}
	got, err := io.ReadAll(zr)
	if err != nil {
		t.Fatalf("gzip read: %v", err)
	}
	if string(got) != body {
		t.Error("decompressed body does not match what the handler wrote")
	}
}

func TestGzipSkipsWhatItShould(t *testing.T) {
	long := strings.Repeat("x", 4096)
	cases := []struct {
		name        string
		contentType string
		status      int
		body        string
		length      string
	}{
		{"binary", "application/octet-stream", http.StatusOK, long, ""},
		{"font", "font/woff2", http.StatusOK, long, ""},
		{"png", "image/png", http.StatusOK, long, ""},
		{"small json", "application/json", http.StatusOK, "{}", "2"},
		{"no content", "application/json", http.StatusNoContent, "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := gzipResponses(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", tc.contentType)
				if tc.length != "" {
					w.Header().Set("Content-Length", tc.length)
				}
				w.WriteHeader(tc.status)
				io.WriteString(w, tc.body)
			}))
			resp := gzipRequest(t, h, "/dl/agent-linux-amd64")
			if resp.Header.Get("Content-Encoding") != "" {
				t.Errorf("Content-Encoding = %q, want none", resp.Header.Get("Content-Encoding"))
			}
			raw, _ := io.ReadAll(resp.Body)
			if string(raw) != tc.body {
				t.Errorf("body was rewritten: got %d bytes, want %d", len(raw), len(tc.body))
			}
		})
	}
}

func TestGzipLeavesClientsThatDoNotAskAlone(t *testing.T) {
	body := strings.Repeat("a", 4096)
	h := gzipResponses(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, body)
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/tools", nil))
	if enc := rec.Result().Header.Get("Content-Encoding"); enc != "" {
		t.Errorf("Content-Encoding = %q, want none", enc)
	}
	if rec.Body.String() != body {
		t.Error("body was rewritten for a client that did not accept gzip")
	}
}

func TestAssetCacheControl(t *testing.T) {
	cases := map[string]string{
		"assets/index-BGaRmXPx.js": "public, max-age=31536000, immutable",
		"index.html":               "no-cache",
		"manifest.webmanifest":     "no-cache",
	}
	for path, want := range cases {
		if got := assetCacheControl(path); got != want {
			t.Errorf("assetCacheControl(%q) = %q, want %q", path, got, want)
		}
	}
}

// A static asset request must not cost a session and user lookup: with no store
// wired at all, resolvePrincipal has to pass it straight through.
func TestResolvePrincipalSkipsStaticRequests(t *testing.T) {
	a := &app{}
	h := a.resolvePrincipal(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	r := httptest.NewRequest(http.MethodGet, "/assets/index-BGaRmXPx.js", nil)
	r.AddCookie(&http.Cookie{Name: sessionCookie, Value: "whatever"})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}
