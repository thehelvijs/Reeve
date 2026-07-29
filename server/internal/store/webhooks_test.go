package store

import (
	"sort"
	"strings"
	"testing"
)

func TestChannelColumnsRoundTrip(t *testing.T) {
	db := openTemp(t)
	ch, err := db.CreateWebhook("global", "", "http://sink.invalid", "email", "enc:opaqueblob", "warning")
	if err != nil {
		t.Fatalf("CreateWebhook: %v", err)
	}
	if ch.Config != "enc:opaqueblob" || ch.MinSeverity != "warning" {
		t.Fatalf("returned channel = %+v", ch)
	}
	hooks, err := db.ListWebhooks()
	if err != nil {
		t.Fatalf("ListWebhooks: %v", err)
	}
	if len(hooks) != 1 {
		t.Fatalf("hooks = %d, want 1", len(hooks))
	}
	if hooks[0].Config != "enc:opaqueblob" || hooks[0].MinSeverity != "warning" || hooks[0].Format != "email" {
		t.Fatalf("read back = %+v", hooks[0])
	}
}

func TestChannelDefaults(t *testing.T) {
	db := openTemp(t)
	if _, err := db.CreateWebhook("global", "", "http://sink.invalid", "", "", ""); err != nil {
		t.Fatalf("CreateWebhook: %v", err)
	}
	hooks, _ := db.ListWebhooks()
	if hooks[0].Format != "generic" || hooks[0].Config != "{}" || hooks[0].MinSeverity != "info" {
		t.Fatalf("defaults = %+v", hooks[0])
	}
}

func TestChannelAcceptsSeverity(t *testing.T) {
	cases := []struct {
		min, event string
		want       bool
	}{
		{"info", "info", true},
		{"info", "warning", true},
		{"info", "error", true},
		{"warning", "info", false},
		{"warning", "warning", true},
		{"warning", "error", true},
		{"error", "info", false},
		{"error", "warning", false},
		{"error", "error", true},
	}
	for _, c := range cases {
		if got := channelAccepts(c.min, c.event); got != c.want {
			t.Errorf("channelAccepts(%q,%q) = %v, want %v", c.min, c.event, got, c.want)
		}
	}
}

func TestResolveChannelsFiltersBySeverity(t *testing.T) {
	db := openTemp(t)
	db.CreateWebhook("global", "", "http://info.invalid", "generic", "", "info")
	db.CreateWebhook("global", "", "http://warn.invalid", "generic", "", "warning")
	db.CreateWebhook("global", "", "http://err.invalid", "generic", "", "error")

	warn, err := db.ResolveChannels("", "", "warning")
	if err != nil {
		t.Fatalf("ResolveChannels: %v", err)
	}
	if len(warn) != 2 {
		t.Fatalf("warning resolves %d channels, want 2 (info+warning)", len(warn))
	}
	all, _ := db.ResolveChannels("", "", "error")
	if len(all) != 3 {
		t.Fatalf("error resolves %d channels, want 3", len(all))
	}
	info, _ := db.ResolveChannels("", "", "info")
	if len(info) != 1 {
		t.Fatalf("info resolves %d channels, want 1", len(info))
	}
}

// Scope is the whole rule: a global channel receives everything, and a narrower
// channel adds to it rather than taking traffic away from it.
func TestResolveChannelsByScope(t *testing.T) {
	db := openTemp(t)
	host, _ := db.CreateHost("db-1", "linux", "", "hash", 120)
	other, _ := db.CreateHost("web-1", "linux", "", "hash2", 120)
	owner, _ := db.CreateUser("owner@example.com", "hash", "admin")
	tool, err := db.CreateTool(Tool{
		Name: "postgres", Slug: "postgres", HostID: host.ID, SourceType: "systemd", CreatorID: owner.ID})
	if err != nil {
		t.Fatalf("CreateTool: %v", err)
	}
	group, _ := db.CreateGroup("dba")
	if err := db.GrantCredentialAccess(host.ID, "group", group.ID, "admin"); err != nil {
		t.Fatalf("GrantCredentialAccess: %v", err)
	}

	db.CreateWebhook("global", "", "http://global.invalid", "generic", "", "info")
	db.CreateWebhook("tool", tool.ID, "http://tool.invalid", "generic", "", "info")
	db.CreateWebhook("host", host.ID, "http://host.invalid", "generic", "", "info")
	db.CreateWebhook("host", other.ID, "http://otherhost.invalid", "generic", "", "info")
	db.CreateWebhook("group", group.ID, "http://group.invalid", "generic", "", "info")

	for _, tc := range []struct {
		name           string
		toolID, hostID string
		want           []string
	}{
		{"a tool alert reaches its tool, its host, the group and global", tool.ID, host.ID,
			[]string{"http://global.invalid", "http://group.invalid", "http://host.invalid", "http://tool.invalid"}},
		{"a host alert reaches its host, the group and global", "", host.ID,
			[]string{"http://global.invalid", "http://group.invalid", "http://host.invalid"}},
		{"another host reaches only its own and global", "", other.ID,
			[]string{"http://global.invalid", "http://otherhost.invalid"}},
		{"an event about nothing in particular reaches global only", "", "",
			[]string{"http://global.invalid"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			hooks, err := db.ResolveChannels(tc.toolID, tc.hostID, "error")
			if err != nil {
				t.Fatalf("ResolveChannels: %v", err)
			}
			got := make([]string, 0, len(hooks))
			for _, h := range hooks {
				got = append(got, h.URL)
			}
			sort.Strings(got)
			if strings.Join(got, " ") != strings.Join(tc.want, " ") {
				t.Errorf("resolved %v, want %v", got, tc.want)
			}
		})
	}
}

// A disabled channel is off, whatever its scope says.
func TestResolveChannelsSkipsDisabled(t *testing.T) {
	db := openTemp(t)
	ch, _ := db.CreateWebhook("global", "", "http://off.invalid", "generic", "", "info")
	if err := db.UpdateWebhook(ch.ID, ch.URL, ch.Format, ch.Config, ch.MinSeverity, false); err != nil {
		t.Fatalf("UpdateWebhook: %v", err)
	}
	hooks, _ := db.ResolveChannels("", "", "error")
	if len(hooks) != 0 {
		t.Errorf("resolved %d channels, want none", len(hooks))
	}
}
