package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The updater whitelists the tags it will run, so a channel added here without
// the compose file learning about it would read as switched in the UI and be
// silently ignored on the next poll.
func TestEveryChannelTagIsOneTheUpdaterAccepts(t *testing.T) {
	script, err := os.ReadFile(filepath.Join("..", "deploy", "docker-compose.yml"))
	if err != nil {
		t.Fatalf("read compose file: %v", err)
	}
	for channel, tag := range channelTags {
		if !strings.Contains(string(script), tag+"|") && !strings.Contains(string(script), "|"+tag+")") {
			t.Errorf("channel %q resolves to tag %q, which the updater does not accept", channel, tag)
		}
	}
}

func TestServerUpdateChannelDefaultsToRelease(t *testing.T) {
	ts := newTestServer(t)
	if got := ts.app.serverUpdateChannel(); got != channelRelease {
		t.Fatalf("channel = %q, want %q", got, channelRelease)
	}
}

// A stored value the running build no longer knows must not park the instance
// on a tag nothing publishes: a downgrade would otherwise read it back.
func TestUnknownStoredChannelReadsAsRelease(t *testing.T) {
	ts := newTestServer(t)
	if err := ts.app.db.SetSetting(settingServerUpdateChannel, "experimental"); err != nil {
		t.Fatalf("set setting: %v", err)
	}
	if got := ts.app.serverUpdateChannel(); got != channelRelease {
		t.Fatalf("channel = %q, want %q", got, channelRelease)
	}
}

func TestWriteChannelFilePublishesTheTag(t *testing.T) {
	ts := newTestServer(t)
	if err := ts.app.writeChannelFile(channelDevelop); err != nil {
		t.Fatalf("write channel file: %v", err)
	}
	body, err := os.ReadFile(ts.app.channelFilePath())
	if err != nil {
		t.Fatalf("read channel file: %v", err)
	}
	if strings.TrimSpace(string(body)) != "develop" {
		t.Fatalf("published %q, want develop", strings.TrimSpace(string(body)))
	}
	if err := ts.app.writeChannelFile("nonsense"); err == nil {
		t.Fatal("writeChannelFile accepted a channel with no tag")
	}
}

// The tag the updater acts on has to survive the switch that set it; a stored
// channel with no file behind it leaves the instance following the old one.
func TestSyncChannelFileRepublishesTheStoredChannel(t *testing.T) {
	ts := newTestServer(t)
	if err := ts.app.db.SetSetting(settingServerUpdateChannel, channelMain); err != nil {
		t.Fatalf("set setting: %v", err)
	}
	if err := os.Remove(ts.app.channelFilePath()); err != nil && !os.IsNotExist(err) {
		t.Fatalf("remove channel file: %v", err)
	}
	ts.app.syncChannelFile()
	body, err := os.ReadFile(ts.app.channelFilePath())
	if err != nil {
		t.Fatalf("read channel file: %v", err)
	}
	if strings.TrimSpace(string(body)) != "main" {
		t.Fatalf("published %q, want main", strings.TrimSpace(string(body)))
	}
}

// "Last updated" has to mean a new build arrived, not that the process
// restarted, or a reboot would read as an update and hide how stale an instance
// really is.
func TestRecordServerVersionStampsOnlyRealChanges(t *testing.T) {
	ts := newTestServer(t)
	ts.app.cfg.Version = "1.0.0"

	ts.app.recordServerVersion()
	if at, ok := ts.app.db.GetSetting(settingServerUpdateAt); ok {
		t.Fatalf("first start recorded an update at %q", at)
	}
	if v := ts.app.serverUpdateView(); v.Version != "1.0.0" || v.UpdatedAt != "" {
		t.Fatalf("view after first start = %+v", v)
	}

	ts.app.recordServerVersion()
	if _, ok := ts.app.db.GetSetting(settingServerUpdateAt); ok {
		t.Fatal("a restart on the same build recorded an update")
	}

	ts.app.cfg.Version = "1.1.0"
	ts.app.recordServerVersion()
	view := ts.app.serverUpdateView()
	if view.Version != "1.1.0" {
		t.Errorf("Version = %q, want 1.1.0", view.Version)
	}
	if view.UpdatedAt == "" {
		t.Error("a new build did not stamp an update time")
	}
}
