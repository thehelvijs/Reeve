package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// bearer builds an Authorization header map for an API token.
func bearer(token string) map[string]string {
	return map[string]string{"Authorization": "Bearer " + token}
}

func TestAPIErrorEnvelope(t *testing.T) {
	ts := newTestServer(t)
	// Unauthenticated request returns the consistent error envelope.
	resp, data := ts.do(t, nil, http.MethodGet, "/api/v1/tools", nil, nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
	var e struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	json.Unmarshal(data, &e)
	if e.Code == "" || e.Message == "" {
		t.Errorf("error envelope missing code/message: %s", data)
	}
}

func TestPublicEndpointsNoAuthExcludeRestricted(t *testing.T) {
	ts := newTestServer(t)
	owner := ts.client(t)
	signup(t, ts, owner, "boss@example.com", "password123")
	createTool(t, ts, owner, toolInput{Name: "PublicGrafana", Address: "10.0.0.5", Port: 3000})
	createTool(t, ts, owner, toolInput{Name: "SecretVault", Visibility: "restricted"})

	// No auth header, no session — the portal endpoint must still respond.
	resp, data := ts.do(t, nil, http.MethodGet, "/api/v1/public/tools", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("public tools = %d: %s", resp.StatusCode, data)
	}
	if !strings.Contains(string(data), "PublicGrafana") {
		t.Errorf("public tool missing: %s", data)
	}
	if strings.Contains(string(data), "SecretVault") {
		t.Errorf("restricted tool leaked to anonymous caller: %s", data)
	}

	var tools []map[string]any
	json.Unmarshal(data, &tools)
	if len(tools) != 1 {
		t.Fatalf("expected 1 public tool, got %d", len(tools))
	}
	if _, ok := tools[0]["status"]; !ok {
		t.Error("public tool DTO missing status field")
	}
	if edit, _ := tools[0]["can_edit"].(bool); edit {
		t.Error("anonymous caller must not have can_edit=true")
	}
	for _, field := range []string{"creator_id", "source_ref", "log_alert_enabled"} {
		if strings.Contains(string(data), field) {
			t.Errorf("public tool response leaked %q: %s", field, data)
		}
	}
}

func TestPublicHostsNoAuth(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	hostID, token := enrollHost(t, ts, admin, "host-a")
	ts.do(t, nil, http.MethodPost, "/api/v1/ingest", samplePush(), bearer(token))
	createTool(t, ts, admin, toolInput{Name: "PublicOnHost", HostID: hostID})

	resp, data := ts.do(t, nil, http.MethodGet, "/api/v1/public/hosts", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("public hosts = %d: %s", resp.StatusCode, data)
	}
	if !strings.Contains(string(data), "host-a") {
		t.Fatalf("expected host-a in public hosts: %s", data)
	}
	for _, field := range []string{"agent_version", "physical_location", "\"os\""} {
		if strings.Contains(string(data), field) {
			t.Errorf("public host response leaked %q: %s", field, data)
		}
	}
}
