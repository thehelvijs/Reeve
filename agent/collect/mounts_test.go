package collect

import "testing"

// A desktop mounts dozens of snaps, each a read-only squashfs on a loop device.
// Reporting them as disks buries the ones that can actually fill up, which is
// the only reason to read the list.
const realMounts = `sysfs /sys sysfs rw,nosuid 0 0
proc /proc proc rw,nosuid 0 0
udev /dev devtmpfs rw,nosuid,size=7687784k 0 0
tmpfs /run tmpfs rw,nosuid,size=1546272k 0 0
/dev/sda3 / ext4 rw,relatime,errors=remount-ro 0 0
cgroup2 /sys/fs/cgroup cgroup2 rw,nosuid 0 0
/dev/loop0 /snap/core18/2979 squashfs ro,nodev 0 0
/dev/loop1 /snap/bare/5 squashfs ro,nodev 0 0
/dev/sda2 /boot/efi vfat rw,relatime 0 0
/dev/sdb1 /mnt/media ext4 rw,relatime 0 0
192.168.1.5:/export /mnt/nas nfs4 rw,relatime 0 0
portal /run/user/1000/doc fuse.portal rw,nosuid 0 0
`

func TestParseMountsKeepsRealFilesystems(t *testing.T) {
	got := ParseMounts(realMounts)
	want := []Mount{
		{Device: "/dev/sda3", Path: "/", FSType: "ext4"},
		{Device: "/dev/sda2", Path: "/boot/efi", FSType: "vfat"},
		{Device: "/dev/sdb1", Path: "/mnt/media", FSType: "ext4"},
		{Device: "192.168.1.5:/export", Path: "/mnt/nas", FSType: "nfs4"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d mounts, want %d: %+v", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("mount %d = %+v, want %+v", i, got[i], w)
		}
	}
}

// A bind mount reports the same device twice under different paths, and both
// statfs to the same numbers. A container host has many.
func TestParseMountsDedupesByDevice(t *testing.T) {
	got := ParseMounts("/dev/sda3 / ext4 rw 0 0\n/dev/sda3 /var/lib/docker/overlay2 ext4 rw 0 0\n")
	if len(got) != 1 || got[0].Path != "/" {
		t.Errorf("got %+v, want only the first path for the device", got)
	}
}

// /proc/mounts is whitespace-separated, so anything in a path that would break
// that is octal-escaped.
func TestParseMountsDecodesOctalEscapes(t *testing.T) {
	got := ParseMounts(`/dev/sdc1 /mnt/My\040Drive ext4 rw 0 0` + "\n")
	if len(got) != 1 || got[0].Path != "/mnt/My Drive" {
		t.Errorf("got %+v, want the space decoded", got)
	}
}

func TestParseMountsIgnoresGarbage(t *testing.T) {
	for _, in := range []string{"", "\n\n", "nonsense", "/dev/sda1 /only-two-fields"} {
		if got := ParseMounts(in); len(got) != 0 {
			t.Errorf("ParseMounts(%q) = %+v, want none", in, got)
		}
	}
}

// An escape that is not three octal digits is left alone rather than eating the
// following characters.
func TestUnescapeMountLeavesMalformedEscapes(t *testing.T) {
	for in, want := range map[string]string{
		`/mnt/a\040b`: "/mnt/a b",
		`/mnt/a\09b`:  `/mnt/a\09b`,
		`/mnt/plain`:  "/mnt/plain",
		`/mnt/trail\`: `/mnt/trail\`,
	} {
		if got := unescapeMount(in); got != want {
			t.Errorf("unescapeMount(%q) = %q, want %q", in, got, want)
		}
	}
}
