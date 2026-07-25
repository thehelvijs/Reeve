package collect

import (
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/thehelvijs/Reeve/contracts"
)

// SampleHostMetrics reads a single host-level metric sample from /proc and the
// root filesystem. Best-effort: a missing source yields a zero section rather
// than an error. Blocks ~200ms to compute CPU utilization over a window.
func SampleHostMetrics() contracts.HostMetrics {
	var m contracts.HostMetrics
	if c, err := os.ReadFile("/proc/meminfo"); err == nil {
		m.MemUsed, m.MemTotal = ParseMemInfo(string(c))
	}
	if c, err := os.ReadFile("/proc/uptime"); err == nil {
		m.UptimeSecs = ParseUptime(string(c))
	}
	if c, err := os.ReadFile("/proc/net/dev"); err == nil {
		m.NetRx, m.NetTx = ParseNetDev(string(c))
	}
	if c, err := os.ReadFile("/proc/loadavg"); err == nil {
		m.Load1, m.Load5, m.Load15 = ParseLoadAvg(string(c))
	}
	m.CPUPct = sampleCPU()
	m.DiskUsed, m.DiskTotal = diskUsage("/")
	sampleGPU(&m)
	return m
}

// sampleGPU fills the GPU fields via nvidia-smi when present; a missing
// binary or execution error leaves the fields at zero (no GPU).
func sampleGPU(m *contracts.HostMetrics) {
	path, err := exec.LookPath("nvidia-smi")
	if err != nil {
		return
	}
	out, err := exec.Command(path, "--query-gpu=utilization.gpu,memory.used,memory.total", "--format=csv,noheader,nounits").Output()
	if err != nil {
		return
	}
	util, memUsed, memTotal := ParseNvidiaSMI(string(out))
	m.GPUUtil = util
	m.GPUMemUsed = memUsed * (1 << 20)
	m.GPUMemTotal = memTotal * (1 << 20)
}

func sampleCPU() float64 {
	c1, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0
	}
	prev, ok := ParseCPUStat(string(c1))
	if !ok {
		return 0
	}
	time.Sleep(200 * time.Millisecond)
	c2, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0
	}
	cur, ok := ParseCPUStat(string(c2))
	if !ok {
		return 0
	}
	return CPUPercent(prev, cur)
}

func diskUsage(path string) (used, total uint64) {
	var fs syscall.Statfs_t
	if err := syscall.Statfs(path, &fs); err != nil {
		return 0, 0
	}
	total = fs.Blocks * uint64(fs.Bsize)
	free := fs.Bavail * uint64(fs.Bsize)
	if total >= free {
		used = total - free
	}
	return used, total
}
