package collect

import (
	"encoding/json"
	"strconv"
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
	Labels string `json:"Labels"`
}

// displayNameLabels are the labels a deployer leaves behind saying what it just
// deployed, best first. Coolify spells it differently across versions, so all
// three are tried; a key a future version adds is one line here.
var displayNameLabels = []string{
	"coolify.name",
	"coolify.resourceName",
	"coolify.serviceName",
}

// containerDisplayName reads the friendliest name a container's labels carry.
//
// Docker flattens labels into one comma-separated k=v string, and a value may
// itself contain a comma, so this is best-effort by construction: a segment that
// splits wrong simply fails to match a key we asked for.
func containerDisplayName(labels string) string {
	if labels == "" {
		return ""
	}
	found := map[string]string{}
	for _, pair := range strings.Split(labels, ",") {
		key, value, ok := strings.Cut(pair, "=")
		if !ok {
			continue
		}
		found[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	for _, key := range displayNameLabels {
		if v := found[key]; v != "" {
			return v
		}
	}
	return ""
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
			ID:          d.ID,
			Name:        d.Names,
			Image:       d.Image,
			DisplayName: containerDisplayName(d.Labels),
			State:       d.State,
			Health:      healthFromStatus(d.Status),
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

type dockerStatsLine struct {
	ID       string `json:"ID"`
	CPUPerc  string `json:"CPUPerc"`
	MemUsage string `json:"MemUsage"`
}

// ParseDockerStats parses newline-delimited JSON from `docker stats --no-stream
// --format '{{json .}}'` into per-container samples.
func ParseDockerStats(output string) []contracts.ContainerSample {
	var out []contracts.ContainerSample
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var d dockerStatsLine
		if err := json.Unmarshal([]byte(line), &d); err != nil {
			continue
		}
		used, limit := parseMemUsage(d.MemUsage)
		out = append(out, contracts.ContainerSample{
			ContainerID: d.ID,
			CPUPct:      parsePercent(d.CPUPerc),
			MemUsed:     used,
			MemLimit:    limit,
		})
	}
	return out
}

func parsePercent(s string) float64 {
	v, err := strconv.ParseFloat(strings.TrimSuffix(strings.TrimSpace(s), "%"), 64)
	if err != nil {
		return 0
	}
	return v
}

// parseMemUsage parses "10MiB / 100MiB" into used and limit bytes.
func parseMemUsage(s string) (used, limit uint64) {
	parts := strings.Split(s, "/")
	if len(parts) != 2 {
		return 0, 0
	}
	return parseBytes(parts[0]), parseBytes(parts[1])
}

func parseBytes(s string) uint64 {
	s = strings.TrimSpace(s)
	units := []struct {
		suffix string
		mult   float64
	}{
		{"GiB", 1 << 30}, {"MiB", 1 << 20}, {"KiB", 1 << 10},
		{"GB", 1e9}, {"MB", 1e6}, {"kB", 1e3}, {"B", 1},
	}
	for _, u := range units {
		if strings.HasSuffix(s, u.suffix) {
			num := strings.TrimSpace(strings.TrimSuffix(s, u.suffix))
			v, err := strconv.ParseFloat(num, 64)
			if err != nil {
				return 0
			}
			return uint64(v * u.mult)
		}
	}
	return 0
}
