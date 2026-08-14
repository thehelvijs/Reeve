package main

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The agent that ships with the deploy reads this file and pushes as the server's
// own host. If the file and the row ever disagreed, the machine Reeve runs on
// would be the one host nobody can monitor.
func TestSelfAgentTokenEnrollsTheServerHost(t *testing.T) {
	ts := newTestServer(t)
	if err := ts.app.db.EnsureServerHost("linux"); err != nil {
		t.Fatal(err)
	}
	if err := ts.app.syncSelfAgentToken(); err != nil {
		t.Fatalf("syncSelfAgentToken: %v", err)
	}

	raw, err := os.ReadFile(ts.app.selfAgentTokenPath())
	if err != nil {
		t.Fatalf("token file: %v", err)
	}
	token := strings.TrimSpace(string(raw))
	if token == "" {
		t.Fatal("token file is empty")
	}
	if base := filepath.Base(ts.app.selfAgentTokenPath()); base != selfAgentTokenFile {
		t.Errorf("token file name = %q, want %q", base, selfAgentTokenFile)
	}

	resp, data := ts.do(t, nil, http.MethodPost, "/api/ingest", samplePush(),
		map[string]string{"Authorization": "Bearer " + token})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("push as the server host = %d: %s", resp.StatusCode, data)
	}

	// A restart must not invalidate the token the running agent holds.
	if err := ts.app.syncSelfAgentToken(); err != nil {
		t.Fatalf("second syncSelfAgentToken: %v", err)
	}
	again, _ := ts.do(t, nil, http.MethodPost, "/api/ingest", samplePush(),
		map[string]string{"Authorization": "Bearer " + token})
	if again.StatusCode != http.StatusOK {
		t.Fatalf("push after a restart = %d, want 200", again.StatusCode)
	}
}
