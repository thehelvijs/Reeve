package main

import (
	"strings"
	"testing"
)

func TestAgentInstallCommandShape(t *testing.T) {
	a := &app{cfg: config{PublicURL: "http://10.0.0.2:8080"}}
	cmd := a.agentInstallCommand("tok-123")
	for _, want := range []string{
		"curl -fsSL http://10.0.0.2:8080/install.sh",
		"REEVE_SERVER_URL=http://10.0.0.2:8080",
		"REEVE_AGENT_TOKEN=tok-123",
		"sudo",
		"bash",
	} {
		if !strings.Contains(cmd, want) {
			t.Errorf("install command missing %q: %s", want, cmd)
		}
	}
}
