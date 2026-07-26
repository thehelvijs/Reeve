package collect

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/thehelvijs/Reeve/contracts"
)

func TestParseProcStatHandlesSpacesAndParensInName(t *testing.T) {
	// utime=100 stime=50 for a name that contains both a space and parens.
	line := "42 (Web Content (tab)) S 1 42 42 0 -1 4194304 900 0 0 0 100 50 0 0 20 0 30 0 900"
	comm, jiffies, ok := ParseProcStat(line)
	if !ok {
		t.Fatal("a name with spaces and parentheses must still parse")
	}
	if comm != "Web Content (tab)" {
		t.Errorf("comm = %q, want %q", comm, "Web Content (tab)")
	}
	if jiffies != 150 {
		t.Errorf("jiffies = %d, want 150 (utime+stime)", jiffies)
	}
}

func TestParseProcStatRejectsTruncated(t *testing.T) {
	for _, in := range []string{"", "42 (bash", "42 (bash) S 1 2 3"} {
		if _, _, ok := ParseProcStat(in); ok {
			t.Errorf("ParseProcStat(%q) reported ok on a line it cannot read", in)
		}
	}
}

func TestParseProcStatus(t *testing.T) {
	uid, rss := ParseProcStatus("Name:\tbash\nUid:\t1000\t1000\t1000\t1000\nVmRSS:\t  2048 kB\n")
	if uid != "1000" {
		t.Errorf("uid = %q, want 1000", uid)
	}
	if rss != 2048*1024 {
		t.Errorf("rss = %d, want %d bytes", rss, 2048*1024)
	}

	// A kernel thread has no VmRSS line at all.
	uid, rss = ParseProcStatus("Name:\tkthreadd\nUid:\t0\t0\t0\t0\n")
	if uid != "0" || rss != 0 {
		t.Errorf("kernel thread = uid %q rss %d, want uid 0 rss 0", uid, rss)
	}
}

func TestParsePasswd(t *testing.T) {
	names := ParsePasswd("root:x:0:0:root:/root:/bin/bash\nbroken-line\nplex:x:997:997::/var/lib/plex:/bin/false\n")
	if names["0"] != "root" {
		t.Errorf("uid 0 = %q, want root", names["0"])
	}
	if names["997"] != "plex" {
		t.Errorf("uid 997 = %q, want plex", names["997"])
	}
	if len(names) != 2 {
		t.Errorf("parsed %d entries, want 2 (the malformed line is skipped)", len(names))
	}
}

func TestParseCmdlineJoinsNulSeparatedArgs(t *testing.T) {
	if got := ParseCmdline("/usr/bin/postgres\x00-D\x00/var/lib/pg\x00"); got != "/usr/bin/postgres -D /var/lib/pg" {
		t.Errorf("cmdline = %q", got)
	}
	if got := ParseCmdline(""); got != "" {
		t.Errorf("empty cmdline = %q, want empty so the caller falls back to comm", got)
	}
	long := make([]byte, maxCommandLen+50)
	for i := range long {
		long[i] = 'x'
	}
	if got := ParseCmdline(string(long)); len(got) != maxCommandLen {
		t.Errorf("truncated length = %d, want %d", len(got), maxCommandLen)
	}
}

func TestTopProcsUnionsBothDimensions(t *testing.T) {
	procs := []contracts.ProcessSample{
		{PID: 1, CPUPct: 90, MemRSS: 1},
		{PID: 2, CPUPct: 80, MemRSS: 2},
		{PID: 3, CPUPct: 0, MemRSS: 9000},
		{PID: 4, CPUPct: 0, MemRSS: 8000},
	}
	original := make([]contracts.ProcessSample, len(procs))
	copy(original, procs)

	top := TopProcs(procs, 2)
	if len(top) != 4 {
		t.Fatalf("union size = %d, want 4 (two per dimension, no overlap)", len(top))
	}
	seen := map[int]bool{}
	for _, p := range top {
		if seen[p.PID] {
			t.Errorf("pid %d appears twice; the union must dedupe", p.PID)
		}
		seen[p.PID] = true
	}
	for _, pid := range []int{1, 2, 3, 4} {
		if !seen[pid] {
			t.Errorf("pid %d missing from the union", pid)
		}
	}
	for i := range original {
		if procs[i] != original[i] {
			t.Fatalf("TopProcs mutated its input at %d", i)
		}
	}
}

func TestTopProcsDedupesWhenOneProcessLeadsBoth(t *testing.T) {
	procs := []contracts.ProcessSample{
		{PID: 1, CPUPct: 90, MemRSS: 9000},
		{PID: 2, CPUPct: 10, MemRSS: 10},
	}
	if top := TopProcs(procs, 1); len(top) != 1 || top[0].PID != 1 {
		t.Errorf("top = %+v, want just pid 1", top)
	}
	if top := TopProcs(nil, 5); top != nil {
		t.Errorf("empty input = %+v, want nil", top)
	}
	if top := TopProcs(procs, 0); top != nil {
		t.Errorf("n=0 = %+v, want nil", top)
	}
}

// writeProc lays out the three files the sampler reads for one pid.
func writeProc(t *testing.T, root, pid, comm string, jiffies int, rssKB int, cmdline string) {
	t.Helper()
	dir := filepath.Join(root, pid)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	stat := "" + pid + " (" + comm + ") S 1 1 1 0 -1 0 0 0 0 0 " +
		itoa(jiffies) + " 0 0 0 20 0 1 0 0"
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("stat", stat)
	write("status", "Uid:\t0\t0\t0\t0\nVmRSS:\t"+itoa(rssKB)+" kB\n")
	write("cmdline", cmdline)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

func TestProcSamplerFirstTickReportsZeroCPU(t *testing.T) {
	root := t.TempDir()
	writeProc(t, root, "1", "init", 500, 1024, "/sbin/init\x00")

	s := NewProcSampler(root)
	got := s.Sample(time.Unix(1000, 0))
	if len(got) != 1 {
		t.Fatalf("sampled %d processes, want 1", len(got))
	}
	if got[0].CPUPct != 0 {
		t.Errorf("first sighting cpu = %v, want 0: there is no previous sample to subtract", got[0].CPUPct)
	}
	if got[0].MemRSS != 1024*1024 {
		t.Errorf("rss = %d, want %d", got[0].MemRSS, 1024*1024)
	}
	if got[0].Command != "/sbin/init" {
		t.Errorf("command = %q", got[0].Command)
	}
}

func TestProcSamplerComputesCPUFromTickDelta(t *testing.T) {
	root := t.TempDir()
	writeProc(t, root, "1", "busy", 0, 1024, "busy\x00")

	s := NewProcSampler(root)
	s.Sample(time.Unix(1000, 0))

	// 1500 jiffies over 10s at 100 Hz is 15 core-seconds in 10s: 150%.
	writeProc(t, root, "1", "busy", 1500, 1024, "busy\x00")
	got := s.Sample(time.Unix(1010, 0))
	if len(got) != 1 {
		t.Fatalf("sampled %d, want 1", len(got))
	}
	if got[0].CPUPct < 149.9 || got[0].CPUPct > 150.1 {
		t.Errorf("cpu = %v, want ~150 (percent of one core, so >100 is two cores' worth)", got[0].CPUPct)
	}
}

func TestProcSamplerIgnoresAReusedPidGoingBackwards(t *testing.T) {
	root := t.TempDir()
	writeProc(t, root, "7", "old", 9000, 10, "old\x00")
	s := NewProcSampler(root)
	s.Sample(time.Unix(1000, 0))

	// Same pid, fewer jiffies: the old process exited and the number was reused.
	writeProc(t, root, "7", "new", 5, 10, "new\x00")
	got := s.Sample(time.Unix(1010, 0))
	if got[0].CPUPct != 0 {
		t.Errorf("cpu = %v, want 0 rather than a fabricated value from a negative delta", got[0].CPUPct)
	}
}

func TestProcSamplerSkipsNonPidEntries(t *testing.T) {
	root := t.TempDir()
	writeProc(t, root, "1", "init", 10, 10, "init\x00")
	if err := os.MkdirAll(filepath.Join(root, "self"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "uptime"), []byte("123 456"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := NewProcSampler(root).Sample(time.Unix(1000, 0)); len(got) != 1 {
		t.Errorf("sampled %d, want 1: /proc holds non-pid entries too", len(got))
	}
}
