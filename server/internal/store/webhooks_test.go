package store

import "testing"

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

func TestGlobalChannelsFilterBySeverity(t *testing.T) {
	db := openTemp(t)
	db.CreateWebhook("global", "", "http://info.invalid", "generic", "", "info")
	db.CreateWebhook("global", "", "http://warn.invalid", "generic", "", "warning")
	db.CreateWebhook("global", "", "http://err.invalid", "generic", "", "error")

	warn, err := db.GlobalChannels("warning")
	if err != nil {
		t.Fatalf("GlobalChannels: %v", err)
	}
	if len(warn) != 2 {
		t.Fatalf("warning resolves %d channels, want 2 (info+warning)", len(warn))
	}
	all, _ := db.GlobalChannels("error")
	if len(all) != 3 {
		t.Fatalf("error resolves %d channels, want 3", len(all))
	}
	info, _ := db.GlobalChannels("info")
	if len(info) != 1 {
		t.Fatalf("info resolves %d channels, want 1", len(info))
	}
}
