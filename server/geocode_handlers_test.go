package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

const photonRiga = `{"features":[
  {"geometry":{"type":"Point","coordinates":[24.1052,56.9496]},
   "properties":{"name":"Riga","city":"Riga","state":"Riga","country":"Latvia"}},
  {"geometry":{"type":"Point","coordinates":[24.1136,56.9520]},
   "properties":{"street":"Brivibas iela","housenumber":"32","city":"Riga","country":"Latvia"}},
  {"geometry":{"type":"Point","coordinates":[]},
   "properties":{"name":"No coordinate"}}
]}`

func TestGeocodeProxiesAndLabels(t *testing.T) {
	var gotQuery, gotAgent string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query().Get("q")
		gotAgent = r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(photonRiga))
	}))
	defer upstream.Close()
	restore := geocodeURL
	geocodeURL = upstream.URL
	defer func() { geocodeURL = restore }()

	ts := newTestServer(t)
	admin := adminClient(t, ts)

	resp, body := ts.do(t, admin, http.MethodGet, "/api/admin/geocode?q=Brivibas+iela", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("geocode = %d: %s", resp.StatusCode, body)
	}
	var hits []geocodeHit
	if err := json.Unmarshal(body, &hits); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if gotQuery != "Brivibas iela" {
		t.Errorf("upstream q = %q", gotQuery)
	}
	if gotAgent == "" {
		t.Error("upstream got no User-Agent; Photon rejects anonymous callers")
	}
	// The coordinate-less feature is dropped, and lon/lat are not swapped.
	if len(hits) != 2 {
		t.Fatalf("hits = %d, want 2: %+v", len(hits), hits)
	}
	if hits[0].Label != "Riga, Latvia" {
		t.Errorf("city label = %q", hits[0].Label)
	}
	if hits[0].Lat != 56.9496 || hits[0].Lon != 24.1052 {
		t.Errorf("city pin = %v,%v", hits[0].Lat, hits[0].Lon)
	}
	if hits[1].Label != "Brivibas iela 32, Riga, Latvia" {
		t.Errorf("address label = %q", hits[1].Label)
	}
}

func TestGeocodeShortQuerySkipsUpstream(t *testing.T) {
	called := false
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.Write([]byte(`{"features":[]}`))
	}))
	defer upstream.Close()
	restore := geocodeURL
	geocodeURL = upstream.URL
	defer func() { geocodeURL = restore }()

	ts := newTestServer(t)
	admin := adminClient(t, ts)

	resp, body := ts.do(t, admin, http.MethodGet, "/api/admin/geocode?q=ri", nil, nil)
	if resp.StatusCode != http.StatusOK || string(body) != "[]\n" {
		t.Fatalf("short query = %d: %s", resp.StatusCode, body)
	}
	if called {
		t.Error("a two-character query reached the geocoder")
	}
}

func TestGeocodeUpstreamFailureIsBadGateway(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer upstream.Close()
	restore := geocodeURL
	geocodeURL = upstream.URL
	defer func() { geocodeURL = restore }()

	ts := newTestServer(t)
	admin := adminClient(t, ts)

	if resp, body := ts.do(t, admin, http.MethodGet, "/api/admin/geocode?q=riga", nil, nil); resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("throttled upstream = %d: %s", resp.StatusCode, body)
	}
}

func TestGeocodeRequiresAdmin(t *testing.T) {
	ts := newTestServer(t)
	adminClient(t, ts) // the first account is the admin; the next one is not
	basic := ts.client(t)
	signup(t, ts, basic, "basic@example.com", "password123")

	if resp, _ := ts.do(t, basic, http.MethodGet, "/api/admin/geocode?q=riga", nil, nil); resp.StatusCode != http.StatusForbidden {
		t.Errorf("basic account geocode = %d, want 403", resp.StatusCode)
	}
	if resp, _ := ts.do(t, nil, http.MethodGet, "/api/admin/geocode?q=riga", nil, nil); resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("anonymous geocode = %d, want 401", resp.StatusCode)
	}
}
