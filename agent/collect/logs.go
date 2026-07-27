package collect

import (
	"regexp"
	"strings"
	"time"

	"github.com/thehelvijs/Reeve/contracts"
)

// DefaultErrorPattern matches common error markers in log lines. It is
// deliberately broad; per-tool toggles and thresholds on the server are the
// noise control.
var DefaultErrorPattern = regexp.MustCompile(`(?i)\b(error|fatal|panic|critical|segfault)\b`)

// ScanLogErrors returns a LogEvent for each line matching the error pattern.
// The timestamp is supplied by the caller so the parser stays pure/testable.
// Credentials are masked here for the same reason as in a command line: an
// error is exactly where a program echoes the invocation that failed.
func ScanLogErrors(source string, lines []string, at time.Time, pattern *regexp.Regexp) []contracts.LogEvent {
	if pattern == nil {
		pattern = DefaultErrorPattern
	}
	var out []contracts.LogEvent
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || !pattern.MatchString(line) {
			continue
		}
		out = append(out, contracts.LogEvent{
			Source:  source,
			Level:   "error",
			Message: redactSecrets(line),
			At:      at,
		})
	}
	return out
}
