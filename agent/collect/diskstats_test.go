package collect

import (
	"os"
	"testing"
)

// realDiskStats is /proc/diskstats from a single-SATA-disk laptop, trimmed to
// one loop device. sda carries 43750082 read sectors and sda3 reports 43730482
// of the same traffic, which is the double count this parser has to avoid.
const realDiskStats = `   7       0 loop0 922 0 4622 31 0 0 0 0 0 34 31 0 0 0 0 0 0
   8       0 sda 1212919 356300 43750082 674427 1604455 3169446 155433386 3217002 0 2087065 5148539 323394 0 260796816 1036831 231314 220277
   8       1 sda1 65 0 520 7 0 0 0 0 0 7 7 0 0 0 0 0 0
   8       2 sda2 178 23 14656 53 2 0 2 5 0 62 79 3 0 1035944 20 0 0
   8       3 sda3 1212565 356277 43730482 674347 1604453 3169446 155433384 3216997 0 2219670 4928155 323391 0 259760872 1036810 0 0
`

func TestParseDiskStatsCountsTheWholeDiskOnce(t *testing.T) {
	read, write := ParseDiskStats(realDiskStats)
	wantRead := uint64(43750082) * diskSectorSize
	wantWrite := uint64(155433386) * diskSectorSize
	if read != wantRead {
		t.Errorf("read = %d, want %d (sda alone, not its partitions)", read, wantRead)
	}
	if write != wantWrite {
		t.Errorf("write = %d, want %d (sda alone, not its partitions)", write, wantWrite)
	}
}

// An NVMe disk's own name ends in a digit. It must not be mistaken for a
// partition of a "nvme0n" that does not exist.
func TestParseDiskStatsKeepsNVMeWholeDisks(t *testing.T) {
	content := ` 259       0 nvme0n1 100 0 200 5 50 0 400 6 0 10 11 0 0 0 0 0 0
 259       1 nvme0n1p1 10 0 20 1 5 0 40 1 0 2 2 0 0 0 0 0 0
 259       2 nvme0n1p2 90 0 180 4 45 0 360 5 0 8 9 0 0 0 0 0 0
`
	read, write := ParseDiskStats(content)
	if read != 200*diskSectorSize || write != 400*diskSectorSize {
		t.Errorf("read/write = %d/%d, want %d/%d from nvme0n1 only",
			read, write, 200*diskSectorSize, 400*diskSectorSize)
	}
}

// A stacked device reports the same traffic as the disks underneath it.
func TestParseDiskStatsSkipsVirtualDevices(t *testing.T) {
	content := `   7       0 loop0 900 0 5000 30 0 0 0 0 0 30 30 0 0 0 0 0 0
 252       0 dm-0 500 0 3000 20 400 0 2000 25 0 40 45 0 0 0 0 0 0
   9       0 md0 500 0 3000 20 400 0 2000 25 0 40 45 0 0 0 0 0 0
 251       0 zram0 100 0 700 5 0 0 0 0 0 5 5 0 0 0 0 0 0
   8       0 sdb 10 0 64 1 20 0 128 2 0 3 3 0 0 0 0 0 0
`
	read, write := ParseDiskStats(content)
	if read != 64*diskSectorSize || write != 128*diskSectorSize {
		t.Errorf("read/write = %d/%d, want %d/%d from sdb only",
			read, write, 64*diskSectorSize, 128*diskSectorSize)
	}
}

// Two whole disks both count.
func TestParseDiskStatsSumsSeparateDisks(t *testing.T) {
	content := `   8       0 sda 1 0 100 1 1 0 200 1 0 1 1 0 0 0 0 0 0
   8      16 sdb 1 0 300 1 1 0 400 1 0 1 1 0 0 0 0 0 0
`
	read, write := ParseDiskStats(content)
	if read != 400*diskSectorSize || write != 600*diskSectorSize {
		t.Errorf("read/write = %d/%d, want %d/%d", read, write, 400*diskSectorSize, 600*diskSectorSize)
	}
}

func TestParseDiskStatsToleratesJunk(t *testing.T) {
	read, write := ParseDiskStats("\n\nnot a diskstats line\n   8   0 sda x 0 y 1 1 0 200 1 0 1 1\n")
	if read != 0 || write != 0 {
		t.Errorf("read/write = %d/%d, want 0/0 for unparseable counters", read, write)
	}
}

// The sampler has to actually wire the parser to the payload. This is the gap the
// charts hit: every field below was plumbed to the UI while nothing set it.
func TestHostSamplerReportsDiskIO(t *testing.T) {
	if _, err := os.Stat("/proc/diskstats"); err != nil {
		t.Skipf("no /proc/diskstats here: %v", err)
	}
	m := NewHostSampler().Sample()
	if m.DiskRead == 0 && m.DiskWrite == 0 {
		t.Error("DiskRead and DiskWrite are both zero on a booted host")
	}
}
