package collect

import (
	"runtime"
	"testing"
)

func TestSampleHostMetricsDoesNotPanicAndReportsMemory(t *testing.T) {
	m := SampleHostMetrics()
	if runtime.GOOS == "linux" {
		if m.MemTotal == 0 {
			t.Errorf("expected mem_total > 0 on linux, got 0")
		}
		if m.CPUPct < 0 || m.CPUPct > 100 {
			t.Errorf("cpu_pct out of range: %v", m.CPUPct)
		}
	}
}
