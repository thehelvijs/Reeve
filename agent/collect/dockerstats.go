package collect

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/thehelvijs/Reeve/contracts"
)

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
