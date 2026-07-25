// Package collect holds the agent's telemetry parsers. Each parser is pure: it
// turns raw command output or /proc content into contract types, so it can be
// tested against fixtures without touching the host.
package collect

import (
	"strings"

	"github.com/thehelvijs/Reeve/contracts"
)

// ParseSystemctl parses `systemctl list-units --type=service --all --no-legend
// --plain --no-pager` output into service states.
func ParseSystemctl(output string) []contracts.ServiceState {
	var out []contracts.ServiceState
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		unit := fields[0]
		if !strings.HasSuffix(unit, ".service") {
			continue
		}
		out = append(out, contracts.ServiceState{
			Unit:        unit,
			ActiveState: fields[2],
			SubState:    fields[3],
		})
	}
	return out
}
