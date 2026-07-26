package collect

import (
	"runtime"
	"testing"
)

func TestSampleHostMetricsDoesNotPanicAndReportsMemory(t *testing.T) {
	m := NewHostSampler().Sample()
	if runtime.GOOS == "linux" {
		if m.MemTotal == 0 {
			t.Errorf("expected mem_total > 0 on linux, got 0")
		}
		if m.CPUPct < 0 || m.CPUPct > 100 {
			t.Errorf("cpu_pct out of range: %v", m.CPUPct)
		}
	}
}

// CPU is measured across ticks, not inside a sleep of its own: the first sample
// has no interval behind it and must report 0 rather than a lifetime average,
// and the second must be a real percentage.
func TestHostSamplerMeasuresCPUAcrossCalls(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("reads /proc/stat")
	}
	s := NewHostSampler()
	if first := s.Sample(); first.CPUPct != 0 {
		t.Errorf("first sample cpu_pct = %v, want 0: there is no interval to measure yet", first.CPUPct)
	}
	second := s.Sample()
	if second.CPUPct < 0 || second.CPUPct > 100 {
		t.Errorf("cpu_pct out of range: %v", second.CPUPct)
	}
}
