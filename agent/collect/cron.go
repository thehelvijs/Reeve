package collect

import (
	"strings"

	"github.com/thehelvijs/Reeve/contracts"
)

// ParseCrontab parses crontab content into cron jobs. It handles both the
// 5-field user form (`m h dom mon dow command`) and the 6-field system form
// (`m h dom mon dow user command`, used in /etc/crontab and /etc/cron.d).
// Comments, blank lines, and environment assignments are skipped. Last-run time
// is not derivable from a crontab and is left unset.
func ParseCrontab(content string, systemForm bool) []contracts.CronState {
	var out []contracts.CronState
	for _, raw := range strings.Split(content, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if isEnvAssignment(line) {
			continue
		}
		fields := strings.Fields(line)
		cmdStart := 5
		if systemForm {
			cmdStart = 6
		}
		if len(fields) <= cmdStart {
			continue
		}
		schedule := strings.Join(fields[:5], " ")
		command := strings.Join(fields[cmdStart:], " ")
		out = append(out, contracts.CronState{
			Name:     command,
			Schedule: schedule,
		})
	}
	return out
}

func isEnvAssignment(line string) bool {
	eq := strings.Index(line, "=")
	if eq <= 0 {
		return false
	}
	key := strings.TrimSpace(line[:eq])
	return !strings.ContainsAny(key, " \t")
}
