package collect

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/thehelvijs/Reeve/contracts"
)

// procCount is what a busy host actually runs. The collectors are sized by this,
// so a benchmark below it measures nothing the fleet will ever hit.
const procCount = 300

// benchProcTree writes a synthetic /proc holding n processes, each with the
// stat, status and cmdline files ProcSampler reads. One name carries spaces and
// parentheses, the shape that breaks a naive stat parser.
func benchProcTree(tb testing.TB, n int) string {
	tb.Helper()
	root := tb.TempDir()
	for i := 0; i < n; i++ {
		pid := i + 1
		dir := filepath.Join(root, fmt.Sprint(pid))
		if err := os.Mkdir(dir, 0o755); err != nil {
			tb.Fatal(err)
		}
		comm := fmt.Sprintf("proc%d", i)
		if i%17 == 0 {
			comm = "Web Content (isolated)"
		}
		stat := fmt.Sprintf("%d (%s) S 1 %d %d 0 -1 4194304 900 0 3 0 %d %d 0 0 20 0 4 0 1234",
			pid, comm, pid, pid, i*13, i*7)
		status := fmt.Sprintf("Name:\t%s\nState:\tS (sleeping)\nTgid:\t%d\nUid:\t%d\t%d\t%d\t%d\nVmRSS:\t%d kB\n",
			comm, pid, 1000+i%4, 1000+i%4, 1000+i%4, 1000+i%4, 4096+i*128)
		cmdline := fmt.Sprintf("/usr/bin/%s\x00--config\x00/etc/%s.conf\x00--token\x00hunter2\x00", comm, comm)
		for name, content := range map[string]string{"stat": stat, "status": status, "cmdline": cmdline} {
			if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
				tb.Fatal(err)
			}
		}
	}
	return root
}

// BenchmarkProcSamplerSample is the per-tick cost that matters: one full walk of
// /proc, including the /etc/passwd read every sample used to repeat.
func BenchmarkProcSamplerSample(b *testing.B) {
	root := benchProcTree(b, procCount)
	s := NewProcSampler(root)
	now := time.Unix(1_700_000_000, 0)
	s.Sample(now) // prime the jiffy map so the loop measures steady state

	b.ReportAllocs()
	for b.Loop() {
		now = now.Add(15 * time.Second)
		s.Sample(now)
	}
}

func BenchmarkHostSamplerSample(b *testing.B) {
	s := NewHostSampler()
	s.Sample()

	b.ReportAllocs()
	for b.Loop() {
		s.Sample()
	}
}

func benchProcs(n int) []contracts.ProcessSample {
	procs := make([]contracts.ProcessSample, n)
	for i := range procs {
		procs[i] = contracts.ProcessSample{
			PID:     i + 1,
			User:    fmt.Sprintf("user%d", i%4),
			Command: "/usr/bin/thing --config /etc/thing.conf",
			CPUPct:  float64((i*37)%1000) / 10,
			MemRSS:  uint64((i*7919)%4000) << 20,
		}
	}
	return procs
}

func BenchmarkTopProcs(b *testing.B) {
	procs := benchProcs(procCount)
	b.ReportAllocs()
	for b.Loop() {
		TopProcs(procs, 25)
	}
}

func BenchmarkParsePasswd(b *testing.B) {
	var sb strings.Builder
	for i := 0; i < 45; i++ {
		fmt.Fprintf(&sb, "user%d:x:%d:%d:User %d:/home/user%d:/bin/bash\n", i, 1000+i, 1000+i, i, i)
	}
	sb.WriteString("nobody:x:65534:65534:nobody:/nonexistent:/usr/sbin/nologin\n")
	content := sb.String()

	b.ReportAllocs()
	for b.Loop() {
		ParsePasswd(content)
	}
}

func BenchmarkParseMounts(b *testing.B) {
	var sb strings.Builder
	sb.WriteString("/dev/sda1 / ext4 rw,relatime 0 0\nproc /proc proc rw,nosuid,nodev,noexec 0 0\n")
	for i := 0; i < 40; i++ {
		fmt.Fprintf(&sb, "/dev/mapper/vg-lv%d /mnt/data%d xfs rw,relatime 0 0\n", i, i)
		fmt.Fprintf(&sb, "tmpfs /run/user/%d tmpfs rw,nosuid,nodev 0 0\n", 1000+i)
	}
	content := sb.String()

	b.ReportAllocs()
	for b.Loop() {
		ParseMounts(content)
	}
}

func BenchmarkParseCrontab(b *testing.B) {
	var sb strings.Builder
	sb.WriteString("# m h dom mon dow command\nSHELL=/bin/sh\n\n")
	for i := 0; i < 30; i++ {
		fmt.Fprintf(&sb, "%d 3 * * * root /usr/local/bin/job%d --quiet\n", i%60, i)
	}
	content := sb.String()

	b.ReportAllocs()
	for b.Loop() {
		ParseCrontab(content, true)
	}
}
