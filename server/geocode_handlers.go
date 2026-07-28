package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Address search is proxied, not called from the page, for two reasons: the
// page keeps its `connect-src 'self'` policy, and an operator watching outbound
// traffic sees one host talking to the geocoder instead of every browser.
// Photon is the OSM typeahead geocoder; it needs no key and answers partial
// queries, which a plain Nominatim search does not.
var (
	geocodeClient = &http.Client{Timeout: 5 * time.Second}
	geocodeURL    = "https://photon.komoot.io/api"
)

// geocodeHit is one suggestion: a display label and the pin to drop.
type geocodeHit struct {
	Label string  `json:"label"`
	Lat   float64 `json:"lat"`
	Lon   float64 `json:"lon"`
}

type photonFeature struct {
	Geometry struct {
		Coordinates []float64 `json:"coordinates"`
	} `json:"geometry"`
	Properties struct {
		Name        string `json:"name"`
		Street      string `json:"street"`
		HouseNumber string `json:"housenumber"`
		City        string `json:"city"`
		District    string `json:"district"`
		State       string `json:"state"`
		Country     string `json:"country"`
	} `json:"properties"`
}

type photonResponse struct {
	Features []photonFeature `json:"features"`
}

// photonHits flattens a Photon answer into labelled pins, dropping features
// with no coordinate or no name to show.
func photonHits(resp photonResponse) []geocodeHit {
	hits := make([]geocodeHit, 0, len(resp.Features))
	for _, f := range resp.Features {
		if len(f.Geometry.Coordinates) < 2 {
			continue
		}
		label := photonLabel(f)
		if label == "" {
			continue
		}
		hits = append(hits, geocodeHit{Label: label, Lat: f.Geometry.Coordinates[1], Lon: f.Geometry.Coordinates[0]})
	}
	return hits
}

// photonLabel reads a feature as a postal address: the thing, then the places
// that contain it, skipping the ones the thing already names.
func photonLabel(f photonFeature) string {
	p := f.Properties
	head := p.Name
	if head == "" && p.Street != "" {
		head = p.Street
		if p.HouseNumber != "" {
			head = p.Street + " " + p.HouseNumber
		}
	}
	parts := []string{}
	for _, part := range []string{head, p.City, p.District, p.State, p.Country} {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		seen := false
		for _, kept := range parts {
			if strings.EqualFold(kept, part) {
				seen = true
			}
		}
		if !seen {
			parts = append(parts, part)
		}
	}
	return strings.Join(parts, ", ")
}

// handleGeocode answers the location picker's typeahead. A query too short to
// mean anything returns an empty list rather than an error, so the field stays
// quiet while the first characters are typed.
func (a *app) handleGeocode(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if len(q) < 3 {
		writeJSON(w, http.StatusOK, []geocodeHit{})
		return
	}
	params := url.Values{"q": {q}, "limit": {"8"}, "lang": {"en"}}
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, geocodeURL+"?"+params.Encode(), nil)
	if err != nil {
		writeError(w, http.StatusBadGateway, "geocode_failed", "address lookup is unavailable")
		return
	}
	req.Header.Set("User-Agent", "Reeve/"+version)
	resp, err := geocodeClient.Do(req)
	if err != nil {
		writeError(w, http.StatusBadGateway, "geocode_failed", "address lookup is unavailable")
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		writeError(w, http.StatusBadGateway, "geocode_failed", "address lookup is unavailable")
		return
	}
	var payload photonResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&payload); err != nil {
		writeError(w, http.StatusBadGateway, "geocode_failed", "address lookup returned junk")
		return
	}
	writeJSON(w, http.StatusOK, photonHits(payload))
}
