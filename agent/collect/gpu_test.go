package collect

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/thehelvijs/Reeve/contracts"
)

func TestSampleGPUReadsAReportingCard(t *testing.T) {
	s := NewHostSampler()
	s.gpuTried, s.gpuPath = true, fakeNvidiaSMI(t, "42, 1024, 8192")

	var m contracts.HostMetrics
	s.sampleGPU(&m)
	if m.GPUUtil != 42 || m.GPUMemUsed != 1024<<20 || m.GPUMemTotal != 8192<<20 {
		t.Errorf("metrics = %+v, want util 42, 1GiB of 8GiB", m)
	}
	if s.gpuPath == "" {
		t.Error("a working probe must survive the sample")
	}
}

// An idle card reports zero utilization and zero used memory. That must still
// read as a GPU, not as a host without one.
func TestSampleGPUKeepsAnIdleCard(t *testing.T) {
	s := NewHostSampler()
	s.gpuTried, s.gpuPath = true, fakeNvidiaSMI(t, "0, 0, 8192")

	var m contracts.HostMetrics
	s.sampleGPU(&m)
	if m.GPUMemTotal != 8192<<20 {
		t.Errorf("GPUMemTotal = %d, want 8GiB: an idle card is still a card", m.GPUMemTotal)
	}
	if s.gpuPath == "" {
		t.Error("an idle card must not retire the probe")
	}
}

// One unreadable answer must not retire the probe for the life of the process.
// The agent runs for weeks, and nvidia-smi says nothing useful while a driver
// reloads or a card resets.
func TestSampleGPURecoversFromOneBadReading(t *testing.T) {
	dir := t.TempDir()
	probe := filepath.Join(dir, "nvidia-smi")
	script := "#!/bin/sh\nn=$(cat " + dir + "/n 2>/dev/null || echo 0)\n" +
		"echo $((n+1)) > " + dir + "/n\n" +
		"if [ \"$n\" = 0 ]; then echo 'no devices were found'; else echo '42, 1024, 8192'; fi\n"
	if err := os.WriteFile(probe, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := exec.Command(probe).Output(); err != nil {
		t.Skipf("cannot exec a test probe here: %v", err)
	}
	os.Remove(filepath.Join(dir, "n"))

	s := NewHostSampler()
	s.gpuTried, s.gpuPath = true, probe

	var bad, good contracts.HostMetrics
	s.sampleGPU(&bad)
	if bad.GPUMemTotal != 0 {
		t.Errorf("unparseable reading filled metrics: %+v", bad)
	}
	s.sampleGPU(&good)
	if good.GPUMemTotal != 8192<<20 {
		t.Errorf("GPUMemTotal = %d, want 8GiB: one bad reading must not be permanent", good.GPUMemTotal)
	}
}

// A probe that fails leaves the fields at zero and reports no GPU.
func TestSampleGPUToleratesAFailingProbe(t *testing.T) {
	probe := filepath.Join(t.TempDir(), "nvidia-smi")
	if err := os.WriteFile(probe, []byte("#!/bin/sh\nexit 9\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	s := NewHostSampler()
	s.gpuTried, s.gpuPath = true, probe

	var m contracts.HostMetrics
	s.sampleGPU(&m)
	if m.GPUUtil != 0 || m.GPUMemUsed != 0 || m.GPUMemTotal != 0 {
		t.Errorf("failing probe filled metrics: %+v", m)
	}
}

// A sampler that finds no nvidia-smi must not run anything.
func TestSampleGPUWithoutABinary(t *testing.T) {
	s := NewHostSampler()
	s.gpuTried = true

	var m contracts.HostMetrics
	s.sampleGPU(&m)
	if m.GPUMemTotal != 0 {
		t.Errorf("metrics = %+v, want all zero", m)
	}
}

// fakeNvidiaSMI writes an executable that answers like nvidia-smi's CSV mode.
// It skips when the filesystem refuses exec, which some sandboxes do.
func fakeNvidiaSMI(t *testing.T, csvLine string) string {
	t.Helper()
	probe := filepath.Join(t.TempDir(), "nvidia-smi")
	if err := os.WriteFile(probe, []byte("#!/bin/sh\necho '"+csvLine+"'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := exec.Command(probe).Output(); err != nil {
		t.Skipf("cannot exec a test probe here: %v", err)
	}
	return probe
}
