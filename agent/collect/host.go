package collect

import (
	"os"
	"os/exec"
	"sort"
	"syscall"

	"github.com/thehelvijs/Reeve/contracts"
)

// HostSampler reads host-level metrics, deriving CPU utilization from the
// /proc/stat delta between calls the way ProcSampler does per process. The push
// loop supplies the window, so nothing sleeps to create one.
//
// A window of its own would be a window the agent spends collecting, and the
// number that came back was mostly the agent measuring its own burst: an idle
// two-core box reported a median 13% and peaks near 50% while its load average
// sat at zero.
//
// Not safe for concurrent use: one sampler belongs to one serial tick loop.
type HostSampler struct {
	prev     CPUSample
	seen     bool
	gpuTried bool
	gpuPath  string
}

// NewHostSampler returns a sampler with no previous CPU reading.
func NewHostSampler() *HostSampler {
	return &HostSampler{}
}

// Sample reads one host-level metric snapshot from /proc and the root
// filesystem. Best-effort: a missing source yields a zero section rather than
// an error. The first call reports 0% CPU, having no interval to measure.
func (s *HostSampler) Sample() contracts.HostMetrics {
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
	m.CPUPct = s.cpuPercent()
	m.DiskUsed, m.DiskTotal = diskUsage("/")
	m.Disks = sampleDisks()
	s.sampleGPU(&m)
	return m
}

// sampleDisks measures every filesystem ParseMounts kept. A mount that cannot
// be statfs'd (a stale network share, a device that vanished mid-scan) or that
// reports no capacity is dropped rather than listed as a zero-byte disk.
func sampleDisks() []contracts.DiskUsage {
	content, err := os.ReadFile("/proc/mounts")
	if err != nil {
		return nil
	}
	mounts := ParseMounts(string(content))
	out := make([]contracts.DiskUsage, 0, len(mounts))
	for _, mnt := range mounts {
		used, total := diskUsage(mnt.Path)
		if total == 0 {
			continue
		}
		out = append(out, contracts.DiskUsage{
			Mount: mnt.Path, Device: mnt.Device, FSType: mnt.FSType, Used: used, Total: total,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Mount < out[j].Mount })
	return out
}

// sampleGPU fills the GPU fields via nvidia-smi when present; a missing binary
// or execution error leaves the fields at zero (no GPU). The binary is located
// once per sampler rather than searched out of PATH every tick, which a driver
// installed after the agent started does not see until the next restart.
func (s *HostSampler) sampleGPU(m *contracts.HostMetrics) {
	if !s.gpuTried {
		s.gpuTried = true
		s.gpuPath, _ = exec.LookPath("nvidia-smi")
	}
	if s.gpuPath == "" {
		return
	}
	out, err := exec.Command(s.gpuPath, "--query-gpu=utilization.gpu,memory.used,memory.total", "--format=csv,noheader,nounits").Output()
	if err != nil {
		return
	}
	util, memUsed, memTotal := ParseNvidiaSMI(string(out))
	m.GPUUtil = util
	m.GPUMemUsed = memUsed * (1 << 20)
	m.GPUMemTotal = memTotal * (1 << 20)
}

func (s *HostSampler) cpuPercent() float64 {
	content, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0
	}
	cur, ok := ParseCPUStat(string(content))
	if !ok {
		return 0
	}
	prev, seen := s.prev, s.seen
	s.prev, s.seen = cur, true
	if !seen {
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
