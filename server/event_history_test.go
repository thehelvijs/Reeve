package main

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/thehelvijs/Reeve/server/internal/store"
)

func storeEvent(hostID string) store.AlertEvent {
	return store.AlertEvent{SubjectKey: "k", HostID: hostID, Type: "down", Severity: "error", Message: "m"}
}

func nowUTC() time.Time {
	return time.Now().UTC()
}

func TestHostEventsEndpoint(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.client(t)
	signup(t, ts, admin, "boss@example.com", "password123")
	host, err := ts.app.db.CreateHost("h1", "linux", "", "hh1", 60)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ts.app.db.CreateAlertEvent(storeEvent(host.ID), nowUTC()); err != nil {
		t.Fatal(err)
	}
	resp, data := ts.do(t, admin, http.MethodGet, "/api/hosts/"+host.ID+"/events", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("host events = %d: %s", resp.StatusCode, data)
	}
	var evs []map[string]any
	json.Unmarshal(data, &evs)
	if len(evs) != 1 {
		t.Fatalf("expected 1 event, got %d", len(evs))
	}
}

func TestToolEventsHiddenForOutsider(t *testing.T) {
	ts := newTestServer(t)
	owner := ts.client(t)
	signup(t, ts, owner, "boss@example.com", "password123")
	tool := createTool(t, ts, owner, toolInput{Name: "Secret", Visibility: "restricted"})

	outsider := ts.client(t)
	signup(t, ts, outsider, "dev@example.com", "password123")
	resp, data := ts.do(t, outsider, http.MethodGet, "/api/tools/"+tool.ID+"/events", nil, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("outsider tool events = %d, want 404: %s", resp.StatusCode, data)
	}

	resp2, data2 := ts.do(t, owner, http.MethodGet, "/api/tools/"+tool.ID+"/events", nil, nil)
	if resp2.StatusCode != http.StatusOK {
		t.Errorf("owner tool events = %d, want 200: %s", resp2.StatusCode, data2)
	}
}
