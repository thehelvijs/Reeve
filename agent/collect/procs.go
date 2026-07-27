package collect

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/thehelvijs/Reeve/contracts"
)

// userHZ is the unit /proc/[pid]/stat reports CPU time in. It is a constant of
// the proc ABI on Linux, not the kernel's configured HZ.
const userHZ = 100

// maxCommandLen bounds one process's reported command line.
const maxCommandLen = 200

// ParseProcStat returns a process's name and total CPU jiffies (utime+stime)
// from /proc/[pid]/stat.
//
// The name sits in parentheses and may itself contain spaces and parentheses
// ("(Web Content)"), so the numeric fields are located from the LAST ')' in the
// line. Splitting the whole line on whitespace misreads every field for such a
// process, which is the classic bug in this parser.
func ParseProcStat(content string) (comm string, jiffies uint64, ok bool) {
	open := strings.IndexByte(content, '(')
	close := strings.LastIndexByte(content, ')')
	if open < 0 || close < open {
		return "", 0, false
	}
	comm = content[open+1 : close]
	// After the name, field 1 is state; utime and stime are fields 12 and 13.
	rest := strings.Fields(content[close+1:])
	if len(rest) < 13 {
		return "", 0, false
	}
	utime, err := strconv.ParseUint(rest[11], 10, 64)
	if err != nil {
		return "", 0, false
	}
	stime, err := strconv.ParseUint(rest[12], 10, 64)
	if err != nil {
		return "", 0, false
	}
	return comm, utime + stime, true
}

// ParseProcStatus returns the real uid and resident set size in bytes from
// /proc/[pid]/status. A kernel thread reports no VmRSS and yields 0.
func ParseProcStatus(content string) (uid string, rssBytes uint64) {
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "Uid:") {
			fields := strings.Fields(line)
			if len(fields) > 1 {
				uid = fields[1]
			}
			continue
		}
		if strings.HasPrefix(line, "VmRSS:") {
			fields := strings.Fields(line)
			if len(fields) > 1 {
				kb, err := strconv.ParseUint(fields[1], 10, 64)
				if err == nil {
					rssBytes = kb * 1024
				}
			}
		}
	}
	return uid, rssBytes
}

// ParsePasswd maps uid to username from /etc/passwd content.
func ParsePasswd(content string) map[string]string {
	names := map[string]string{}
	for _, line := range strings.Split(content, "\n") {
		fields := strings.Split(line, ":")
		if len(fields) < 3 {
			continue
		}
		names[fields[2]] = fields[0]
	}
	return names
}

// ParseCmdline returns the command a process was started with, NUL-separated in
// /proc/[pid]/cmdline. Empty for a kernel thread, whose caller falls back to the
// name from stat. Credentials in argv are masked here, before the command
// reaches a push payload or the on-disk buffer.
func ParseCmdline(content string) string {
	cleaned := strings.ReplaceAll(strings.TrimRight(content, "\x00"), "\x00", " ")
	return truncate(redactSecrets(strings.TrimSpace(cleaned)), maxCommandLen)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// TopProcs returns the union of the n heaviest processes by CPU and the n
// heaviest by memory, CPU-descending. It does not mutate procs.
func TopProcs(procs []contracts.ProcessSample, n int) []contracts.ProcessSample {
	if n <= 0 || len(procs) == 0 {
		return nil
	}
	byCPU := make([]contracts.ProcessSample, len(procs))
	copy(byCPU, procs)
	sort.Slice(byCPU, func(i, j int) bool { return byCPU[i].CPUPct > byCPU[j].CPUPct })

	byMem := make([]contracts.ProcessSample, len(procs))
	copy(byMem, procs)
	sort.Slice(byMem, func(i, j int) bool { return byMem[i].MemRSS > byMem[j].MemRSS })

	seen := map[int]bool{}
	out := make([]contracts.ProcessSample, 0, 2*n)
	for _, list := range [][]contracts.ProcessSample{byCPU, byMem} {
		for i := 0; i < n && i < len(list); i++ {
			if seen[list[i].PID] {
				continue
			}
			seen[list[i].PID] = true
			out = append(out, list[i])
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CPUPct > out[j].CPUPct })
	return out
}

// ProcSampler reads per-process usage from a /proc tree, deriving CPU percent
// from the jiffy delta between calls. The push loop provides the window, so no
// sampling sleep is needed; a process first seen on this call reports 0%.
//
// Not safe for concurrent use: one sampler belongs to one serial tick loop.
type ProcSampler struct {
	root   string
	prev   map[int]uint64
	prevAt time.Time
}

// NewProcSampler returns a sampler reading the /proc tree rooted at root.
func NewProcSampler(root string) *ProcSampler {
	return &ProcSampler{root: root}
}

// Sample returns every process the tree reports, with CPU percent measured
// against the previous call. Best-effort: a process that exits mid-scan is
// skipped rather than failing the sample.
func (s *ProcSampler) Sample(now time.Time) []contracts.ProcessSample {
	entries, err := os.ReadDir(s.root)
	if err != nil {
		return nil
	}
	users := map[string]string{}
	if content, err := os.ReadFile("/etc/passwd"); err == nil {
		users = ParsePasswd(string(content))
	}

	elapsed := now.Sub(s.prevAt).Seconds()
	cur := make(map[int]uint64, len(entries))
	out := make([]contracts.ProcessSample, 0, len(entries))

	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		statRaw, err := os.ReadFile(filepath.Join(s.root, e.Name(), "stat"))
		if err != nil {
			continue
		}
		comm, jiffies, ok := ParseProcStat(string(statRaw))
		if !ok {
			continue
		}
		cur[pid] = jiffies

		statusRaw, _ := os.ReadFile(filepath.Join(s.root, e.Name(), "status"))
		uid, rss := ParseProcStatus(string(statusRaw))
		user := users[uid]
		if user == "" {
			user = uid
		}
		cmdlineRaw, _ := os.ReadFile(filepath.Join(s.root, e.Name(), "cmdline"))
		command := ParseCmdline(string(cmdlineRaw))
		if command == "" {
			command = truncate(comm, maxCommandLen)
		}

		out = append(out, contracts.ProcessSample{
			PID:     pid,
			User:    user,
			Command: command,
			CPUPct:  s.cpuPercent(pid, jiffies, elapsed),
			MemRSS:  rss,
		})
	}

	s.prev = cur
	s.prevAt = now
	return out
}

// cpuPercent converts a jiffy delta into percent of one core. A process with no
// previous sample, or a counter that went backwards because the pid was reused,
// reports 0 rather than a fabricated spike.
func (s *ProcSampler) cpuPercent(pid int, jiffies uint64, elapsed float64) float64 {
	if elapsed <= 0 {
		return 0
	}
	before, seen := s.prev[pid]
	if !seen || jiffies < before {
		return 0
	}
	return float64(jiffies-before) / userHZ / elapsed * 100
}
