package collect

import (
	"strconv"
	"strings"
)

// CPUSample is the aggregate CPU time counters from /proc/stat.
type CPUSample struct {
	Idle  uint64
	Total uint64
}

// ParseCPUStat reads the aggregate "cpu" line from /proc/stat.
func ParseCPUStat(content string) (CPUSample, bool) {
	for _, line := range strings.Split(content, "\n") {
		if !strings.HasPrefix(line, "cpu ") {
			continue
		}
		fields := strings.Fields(line)[1:]
		var total uint64
		var idle uint64
		for i, f := range fields {
			v, err := strconv.ParseUint(f, 10, 64)
			if err != nil {
				continue
			}
			total += v
			if i == 3 || i == 4 { // idle + iowait
				idle += v
			}
		}
		return CPUSample{Idle: idle, Total: total}, true
	}
	return CPUSample{}, false
}

// CPUPercent computes CPU utilization between two /proc/stat samples.
func CPUPercent(prev, cur CPUSample) float64 {
	dTotal := float64(cur.Total - prev.Total)
	dIdle := float64(cur.Idle - prev.Idle)
	if dTotal <= 0 {
		return 0
	}
	return (dTotal - dIdle) / dTotal * 100
}

// ParseMemInfo returns used and total bytes from /proc/meminfo.
func ParseMemInfo(content string) (used, total uint64) {
	vals := map[string]uint64{}
	for _, line := range strings.Split(content, "\n") {
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		key := strings.TrimSuffix(parts[0], ":")
		kb, err := strconv.ParseUint(parts[1], 10, 64)
		if err != nil {
			continue
		}
		vals[key] = kb * 1024
	}
	total = vals["MemTotal"]
	avail, ok := vals["MemAvailable"]
	if !ok {
		avail = vals["MemFree"]
	}
	if total >= avail {
		used = total - avail
	}
	return used, total
}

// ParseUptime returns uptime seconds from /proc/uptime.
func ParseUptime(content string) uint64 {
	fields := strings.Fields(content)
	if len(fields) == 0 {
		return 0
	}
	secs, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0
	}
	return uint64(secs)
}

// ParseLoadAvg reads the first three fields of /proc/loadavg (1/5/15-minute).
func ParseLoadAvg(content string) (float64, float64, float64) {
	fields := strings.Fields(content)
	if len(fields) < 3 {
		return 0, 0, 0
	}
	l1, _ := strconv.ParseFloat(fields[0], 64)
	l5, _ := strconv.ParseFloat(fields[1], 64)
	l15, _ := strconv.ParseFloat(fields[2], 64)
	return l1, l5, l15
}

// ParseNvidiaSMI parses the first CSV line of
// `nvidia-smi --query-gpu=utilization.gpu,memory.used,memory.total --format=csv,noheader,nounits`
// (percent, MiB, MiB). Returns zeros if the line has fewer than 3 fields.
func ParseNvidiaSMI(content string) (util float64, memUsed, memTotal uint64) {
	line := strings.SplitN(content, "\n", 2)[0]
	fields := strings.Split(line, ",")
	if len(fields) < 3 {
		return 0, 0, 0
	}
	util, _ = strconv.ParseFloat(strings.TrimSpace(fields[0]), 64)
	memUsed, _ = strconv.ParseUint(strings.TrimSpace(fields[1]), 10, 64)
	memTotal, _ = strconv.ParseUint(strings.TrimSpace(fields[2]), 10, 64)
	return util, memUsed, memTotal
}

// diskSectorSize is the unit /proc/diskstats counts in. It is 512 bytes by
// definition of the proc ABI, not the device's physical sector size.
const diskSectorSize = 512

// virtualDiskPrefixes are devices whose counters are not physical disk traffic.
// loop, ram and zram are memory or image backed. A dm or md device is stacked on
// real disks that report the same traffic again, so counting both doubles it.
var virtualDiskPrefixes = []string{"loop", "ram", "zram", "dm-", "md", "sr", "fd"}

// ParseDiskStats sums bytes read and written across whole physical disks in
// /proc/diskstats.
//
// Partitions are skipped. The kernel counts traffic on both a partition and its
// parent disk, so summing every line reports nearly double: on a single-disk
// host sda reads 43750082 sectors while its sda3 reads 43730482 of them.
//
// The counters are cumulative since boot, like the ones ParseNetDev returns, and
// reset when the host reboots.
func ParseDiskStats(content string) (read, write uint64) {
	type counters struct {
		read, write uint64
	}
	disks := map[string]counters{}
	var names []string
	for line := range strings.Lines(content) {
		fields := strings.Fields(line)
		if len(fields) < 10 {
			continue
		}
		name := fields[2]
		if virtualDisk(name) {
			continue
		}
		sectorsRead, err := strconv.ParseUint(fields[5], 10, 64)
		if err != nil {
			continue
		}
		sectorsWritten, err := strconv.ParseUint(fields[9], 10, 64)
		if err != nil {
			continue
		}
		if _, dup := disks[name]; !dup {
			names = append(names, name)
		}
		disks[name] = counters{read: sectorsRead, write: sectorsWritten}
	}
	for _, name := range names {
		parent := partitionParent(name)
		if _, isPartition := disks[parent]; isPartition {
			continue
		}
		read += disks[name].read * diskSectorSize
		write += disks[name].write * diskSectorSize
	}
	return read, write
}

func virtualDisk(name string) bool {
	for _, prefix := range virtualDiskPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

// partitionParent returns the whole-disk name a partition belongs to, or "" when
// the name is not shaped like one. The caller decides: only a parent that also
// appears in /proc/diskstats makes this a partition, which is what keeps a disk
// whose own name ends in a digit (nvme0n1) from looking like one.
func partitionParent(name string) string {
	trimmed := strings.TrimRight(name, "0123456789")
	if trimmed == name || trimmed == "" {
		return ""
	}
	// nvme0n1p1 and mmcblk0p1 separate disk from partition with a 'p'.
	return strings.TrimSuffix(trimmed, "p")
}

// ParseNetDev sums received and transmitted bytes across real interfaces in
// /proc/net/dev (skipping loopback).
func ParseNetDev(content string) (rx, tx uint64) {
	for _, line := range strings.Split(content, "\n") {
		colon := strings.Index(line, ":")
		if colon < 0 {
			continue
		}
		iface := strings.TrimSpace(line[:colon])
		if iface == "lo" || iface == "" {
			continue
		}
		fields := strings.Fields(line[colon+1:])
		if len(fields) < 9 {
			continue
		}
		r, _ := strconv.ParseUint(fields[0], 10, 64)
		t, _ := strconv.ParseUint(fields[8], 10, 64)
		rx += r
		tx += t
	}
	return rx, tx
}
