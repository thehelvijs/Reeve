package collect

import (
	"strings"
)

// pseudoFS are kernel and virtual filesystems that have no capacity worth
// reporting: their "size" is either zero or a slice of RAM.
var pseudoFS = map[string]bool{
	"proc": true, "sysfs": true, "devtmpfs": true, "devpts": true, "tmpfs": true,
	"cgroup": true, "cgroup2": true, "pstore": true, "efivarfs": true, "bpf": true,
	"autofs": true, "hugetlbfs": true, "mqueue": true, "debugfs": true, "tracefs": true,
	"fusectl": true, "configfs": true, "securityfs": true, "ramfs": true,
	"binfmt_misc": true, "rpc_pipefs": true, "nsfs": true, "selinuxfs": true,
	"fuse.portal": true, "fuse.gvfsd-fuse": true, "overlay": true,
	// A snap is a read-only squashfs image, and a desktop has dozens mounted.
	// Reporting each as a full disk buries the ones that can actually fill up.
	"squashfs": true,
}

// Mount is one entry from /proc/mounts, before its capacity is measured.
type Mount struct {
	Device string
	Path   string
	FSType string
}

// ParseMounts returns the filesystems worth reporting capacity for: real
// devices and network shares, with kernel and virtual mounts dropped.
//
// A bind mount and its source share a device and a size, so the first path for
// a given device wins and the rest are dropped — otherwise a container host
// reports the same filesystem a dozen times under different names.
//
// The escaping in /proc/mounts is octal (\040 for a space), which matters for
// any mount point with a space in it.
func ParseMounts(content string) []Mount {
	seen := map[string]bool{}
	var out []Mount
	for line := range strings.Lines(content) {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		device, path, fsType := unescapeMount(fields[0]), unescapeMount(fields[1]), fields[2]
		if pseudoFS[fsType] || strings.HasPrefix(device, "/dev/loop") {
			continue
		}
		// A real filesystem is backed by something: a device node, or a remote
		// share. Anything else here is a virtual mount this list does not know.
		if !strings.HasPrefix(device, "/dev/") && !strings.Contains(fsType, "nfs") && !strings.Contains(fsType, "cifs") {
			continue
		}
		if seen[device] {
			continue
		}
		seen[device] = true
		out = append(out, Mount{Device: device, Path: path, FSType: fsType})
	}
	return out
}

// unescapeMount decodes the octal escapes /proc/mounts uses for characters that
// would otherwise break its whitespace-separated format.
func unescapeMount(s string) string {
	if !strings.Contains(s, `\`) {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] != '\\' || i+3 >= len(s) {
			b.WriteByte(s[i])
			continue
		}
		var v byte
		ok := true
		for _, c := range []byte(s[i+1 : i+4]) {
			if c < '0' || c > '7' {
				ok = false
				break
			}
			v = v*8 + (c - '0')
		}
		if !ok {
			b.WriteByte(s[i])
			continue
		}
		b.WriteByte(v)
		i += 3
	}
	return b.String()
}
