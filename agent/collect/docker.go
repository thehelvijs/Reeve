package collect

import (
	"encoding/json"
	"strings"

	"github.com/thehelvijs/Reeve/contracts"
)

// dockerPSLine mirrors `docker ps -a --format '{{json .}}'` fields we use.
type dockerPSLine struct {
	ID     string `json:"ID"`
	Names  string `json:"Names"`
	Image  string `json:"Image"`
	State  string `json:"State"`
	Status string `json:"Status"`
}

// ParseDockerPS parses newline-delimited JSON from `docker ps -a --format
// '{{json .}}'` into container states, deriving health from the status text.
func ParseDockerPS(output string) []contracts.ContainerState {
	var out []contracts.ContainerState
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var d dockerPSLine
		if err := json.Unmarshal([]byte(line), &d); err != nil {
			continue
		}
		out = append(out, contracts.ContainerState{
			ID:     d.ID,
			Name:   d.Names,
			Image:  d.Image,
			State:  d.State,
			Health: healthFromStatus(d.Status),
		})
	}
	return out
}

func healthFromStatus(status string) string {
	switch {
	case strings.Contains(status, "(healthy)"):
		return "healthy"
	case strings.Contains(status, "(unhealthy)"):
		return "unhealthy"
	case strings.Contains(status, "(health: starting)"):
		return "starting"
	default:
		return ""
	}
}
