package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
)

func TestUpdateHostLocation(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.client(t)
	signup(t, ts, admin, "admin@example.com", "password123")

	h, err := ts.app.db.CreateHost("riga-box", "linux", "", "tok", 60)
	if err != nil {
		t.Fatalf("create host: %v", err)
	}

	body := map[string]any{"physical_location": "Riga, Latvia", "latitude": 56.946, "longitude": 24.106}
	resp, data := ts.do(t, admin, http.MethodPatch, "/api/v1/admin/hosts/"+h.ID, body, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("patch host = %d: %s", resp.StatusCode, data)
	}
	var v struct {
		PhysicalLocation string   `json:"physical_location"`
		Latitude         *float64 `json:"latitude"`
		Longitude        *float64 `json:"longitude"`
	}
	json.Unmarshal(data, &v)
	if v.PhysicalLocation != "Riga, Latvia" || v.Latitude == nil || *v.Latitude != 56.946 || v.Longitude == nil {
		t.Fatalf("location not persisted: %s", data)
	}

	// Listing reflects the stored coordinates.
	_, list := ts.do(t, admin, http.MethodGet, "/api/v1/hosts", nil, nil)
	if !bytes.Contains(list, []byte("56.946")) {
		t.Errorf("host list missing latitude: %s", list)
	}
}
