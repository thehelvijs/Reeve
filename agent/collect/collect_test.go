package collect

import (
	"testing"
	"time"
)

func TestParseSystemctl(t *testing.T) {
	out := `nginx.service        loaded active   running A high performance web server
redis-server.service loaded active   running Advanced key-value store
ssh.service          loaded failed   failed  OpenBSD Secure Shell server
proc-sys.mount       loaded active   mounted Kernel mount`

	svcs := ParseSystemctl(out)
	if len(svcs) != 3 {
		t.Fatalf("got %d services, want 3 (.service only): %+v", len(svcs), svcs)
	}
	if svcs[2].Unit != "ssh.service" || svcs[2].ActiveState != "failed" || svcs[2].SubState != "failed" {
		t.Errorf("ssh parse wrong: %+v", svcs[2])
	}
}

func TestParseDockerPS(t *testing.T) {
	out := `{"ID":"abc123","Names":"web","Image":"nginx:latest","State":"running","Status":"Up 2 hours (healthy)"}
{"ID":"def456","Names":"db","Image":"postgres:16","State":"exited","Status":"Exited (0) 5 minutes ago"}
` + "\n"
	cs := ParseDockerPS(out)
	if len(cs) != 2 {
		t.Fatalf("got %d containers, want 2", len(cs))
	}
	if cs[0].Health != "healthy" {
		t.Errorf("web health = %q, want healthy", cs[0].Health)
	}
	if cs[1].State != "exited" || cs[1].Health != "" {
		t.Errorf("db parse wrong: %+v", cs[1])
	}
}

func TestParseCrontabUserForm(t *testing.T) {
	content := `# a comment
SHELL=/bin/bash
0 3 * * * /usr/local/bin/backup.sh
*/5 * * * * curl -s http://localhost/health`
	jobs := ParseCrontab(content, false)
	if len(jobs) != 2 {
		t.Fatalf("got %d jobs, want 2: %+v", len(jobs), jobs)
	}
	if jobs[0].Schedule != "0 3 * * *" || jobs[0].Name != "/usr/local/bin/backup.sh" {
		t.Errorf("job0 wrong: %+v", jobs[0])
	}
}

func TestParseCrontabSystemForm(t *testing.T) {
	content := `0 6 * * * root /usr/bin/apt update`
	jobs := ParseCrontab(content, true)
	if len(jobs) != 1 || jobs[0].Name != "/usr/bin/apt update" || jobs[0].Schedule != "0 6 * * *" {
		t.Errorf("system cron parse wrong: %+v", jobs)
	}
}

func TestParseMemInfo(t *testing.T) {
	content := `MemTotal:       16384000 kB
MemFree:         1000000 kB
MemAvailable:    8192000 kB
Buffers:          500000 kB`
	used, total := ParseMemInfo(content)
	if total != 16384000*1024 {
		t.Errorf("total = %d", total)
	}
	if used != (16384000-8192000)*1024 {
		t.Errorf("used = %d", used)
	}
}

func TestParseCPUStatAndPercent(t *testing.T) {
	prevC := "cpu  100 0 100 800 0 0 0 0 0 0\n"
	curC := "cpu  300 0 200 900 0 0 0 0 0 0\n"
	prev, ok1 := ParseCPUStat(prevC)
	cur, ok2 := ParseCPUStat(curC)
	if !ok1 || !ok2 {
		t.Fatal("cpu parse failed")
	}
	// prev total=1000 idle=800; cur total=1400 idle=900.
	// delta total=400, delta idle=100 -> 75% busy.
	if pct := CPUPercent(prev, cur); pct < 74.9 || pct > 75.1 {
		t.Errorf("cpu pct = %f, want ~75", pct)
	}
}

func TestParseUptimeAndNetDev(t *testing.T) {
	if ParseUptime("12345.67 9999.00\n") != 12345 {
		t.Error("uptime parse wrong")
	}
	net := `Inter-|   Receive                                                |  Transmit
 face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets
    lo:  1000       10    0    0    0     0          0         0   1000       10
  eth0:  5000       50    0    0    0     0          0         0   3000       30`
	rx, tx := ParseNetDev(net)
	if rx != 5000 || tx != 3000 {
		t.Errorf("netdev rx=%d tx=%d, want 5000/3000 (lo skipped)", rx, tx)
	}
}

func TestParseLoadAvg(t *testing.T) {
	l1, l5, l15 := ParseLoadAvg("0.52 0.41 0.38 1/234 5678\n")
	if l1 != 0.52 || l5 != 0.41 || l15 != 0.38 {
		t.Fatalf("got %v %v %v", l1, l5, l15)
	}
	if a, b, c := ParseLoadAvg(""); a != 0 || b != 0 || c != 0 {
		t.Fatalf("empty should be zeros, got %v %v %v", a, b, c)
	}
}

func TestParseNvidiaSMI(t *testing.T) {
	util, memUsed, memTotal := ParseNvidiaSMI("12, 2048, 8192\n")
	if util != 12 || memUsed != 2048 || memTotal != 8192 {
		t.Fatalf("got %v %v %v, want 12 2048 8192", util, memUsed, memTotal)
	}
	if u, m, mt := ParseNvidiaSMI(""); u != 0 || m != 0 || mt != 0 {
		t.Fatalf("empty should be zeros, got %v %v %v", u, m, mt)
	}
}

func TestScanLogErrors(t *testing.T) {
	lines := []string{
		"INFO all good",
		"2026-01-01 ERROR connection refused",
		"a warning only",
		"FATAL out of memory",
	}
	events := ScanLogErrors("app", lines, time.Unix(0, 0).UTC(), nil)
	if len(events) != 2 {
		t.Fatalf("got %d error events, want 2: %+v", len(events), events)
	}
	if events[0].Source != "app" || events[0].Level != "error" {
		t.Errorf("event0 wrong: %+v", events[0])
	}
}

func TestParseDockerStats(t *testing.T) {
	out := `{"ID":"abc123","CPUPerc":"12.34%","MemUsage":"256MiB / 2GiB"}
{"ID":"def456","CPUPerc":"0.00%","MemUsage":"10.5MiB / 512MiB"}`
	s := ParseDockerStats(out)
	if len(s) != 2 {
		t.Fatalf("got %d samples, want 2", len(s))
	}
	if s[0].ContainerID != "abc123" || s[0].CPUPct < 12.33 || s[0].CPUPct > 12.35 {
		t.Errorf("sample0 cpu wrong: %+v", s[0])
	}
	if s[0].MemUsed != 256*(1<<20) || s[0].MemLimit != 2*(1<<30) {
		t.Errorf("sample0 mem wrong: used=%d limit=%d", s[0].MemUsed, s[0].MemLimit)
	}
}
