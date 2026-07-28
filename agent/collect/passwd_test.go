package collect

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCachedFileReparsesOnlyOnChange(t *testing.T) {
	path := filepath.Join(t.TempDir(), "passwd")
	if err := os.WriteFile(path, []byte("root:x:0:0::/root:/bin/bash\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	parses := 0
	load := cachedFile(path, func(content string) map[string]string {
		parses++
		return ParsePasswd(content)
	})

	if first, second := load(), load(); first["0"] != "root" || second["0"] != "root" {
		t.Fatalf("cached passwd = %v / %v, want uid 0 -> root", first, second)
	}
	if parses != 1 {
		t.Fatalf("parses = %d, want 1: an unchanged file must reuse the parse", parses)
	}

	if err := os.WriteFile(path, []byte("root:x:0:0::/root:/bin/bash\nhal:x:500:500::/:/bin/false\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// mtime granularity can hide a fast rewrite, so force it visibly newer.
	if fi, err := os.Stat(path); err == nil {
		os.Chtimes(path, fi.ModTime().Add(time.Hour), fi.ModTime().Add(time.Hour))
	}
	if third := load(); third["500"] != "hal" {
		t.Fatalf("cached passwd after rewrite = %v, want uid 500 -> hal", third)
	}
	if parses != 2 {
		t.Fatalf("parses = %d, want 2: a changed file must be reparsed", parses)
	}
}

// A file that disappears must keep serving the last good parse, not an empty map
// that would blank every username on the next push.
func TestCachedFileKeepsTheLastGoodValue(t *testing.T) {
	path := filepath.Join(t.TempDir(), "passwd")
	if err := os.WriteFile(path, []byte("root:x:0:0::/root:/bin/bash\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	load := cachedFile(path, ParsePasswd)
	if got := load(); got["0"] != "root" {
		t.Fatalf("cached passwd = %v, want uid 0 -> root", got)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if got := load(); got["0"] != "root" {
		t.Errorf("after the file vanished = %v, want the last good parse", got)
	}
}

// ParsePasswd cuts at colons rather than splitting; a line short of three fields
// must be skipped, not indexed into.
func TestParsePasswdSkipsShortLines(t *testing.T) {
	content := "root:x:0:0::/root:/bin/bash\nbroken\nalso:broken\n\nhal:x:500:500::/:/bin/false"
	got := ParsePasswd(content)
	want := map[string]string{"0": "root", "500": "hal"}
	if len(got) != len(want) {
		t.Fatalf("ParsePasswd = %v, want %v", got, want)
	}
	for uid, name := range want {
		if got[uid] != name {
			t.Errorf("uid %s = %q, want %q", uid, got[uid], name)
		}
	}
}
