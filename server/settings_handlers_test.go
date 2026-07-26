package main

import (
	"encoding/json"
	"net/http"
	"testing"
)

func getSettings(t *testing.T, ts *testServer, c *http.Client) (*http.Response, settingsView) {
	t.Helper()
	resp, data := ts.do(t, c, http.MethodGet, "/api/v1/admin/settings", nil, nil)
	var v settingsView
	json.Unmarshal(data, &v)
	return resp, v
}

func putSettings(t *testing.T, ts *testServer, c *http.Client, body any) (*http.Response, settingsView) {
	t.Helper()
	resp, data := ts.do(t, c, http.MethodPut, "/api/v1/admin/settings", body, nil)
	var v settingsView
	json.Unmarshal(data, &v)
	return resp, v
}

func TestSettingsDefaults(t *testing.T) {
	ts := newTestServer(t)
	c := ts.client(t)
	signup(t, ts, c, "boss@example.com", "password123")

	resp, v := getSettings(t, ts, c)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if !v.SignupEnabled {
		t.Error("signup_enabled defaults to false, want true")
	}
	if v.Retention.RawSecs != 48*3600 || v.Retention.OneHourSecs != 30*24*3600 {
		t.Errorf("retention defaults = %+v", v.Retention)
	}
}

// The signup toggle is the fix for an instance being open to anyone who can
// reach it, so the write path has to actually close signup.
func TestSettingsDisableSignupBlocksNewAccounts(t *testing.T) {
	ts := newTestServer(t)
	c := ts.client(t)
	signup(t, ts, c, "boss@example.com", "password123")

	resp, v := putSettings(t, ts, c, map[string]any{"signup_enabled": false})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("put status = %d, want 200", resp.StatusCode)
	}
	if v.SignupEnabled {
		t.Error("response still reports signup_enabled true")
	}

	resp, _ = signup(t, ts, ts.client(t), "dev@example.com", "password123")
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("signup after disabling status = %d, want 403", resp.StatusCode)
	}

	_, data := ts.do(t, nil, http.MethodGet, "/api/v1/auth/status", nil, nil)
	var status map[string]bool
	json.Unmarshal(data, &status)
	if status["signup_enabled"] {
		t.Error("public auth status still advertises open signup")
	}

	// And re-opening it works.
	putSettings(t, ts, c, map[string]any{"signup_enabled": true})
	resp, _ = signup(t, ts, ts.client(t), "dev@example.com", "password123")
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("signup after re-enabling status = %d, want 201", resp.StatusCode)
	}
}

func TestSettingsRetentionRoundTrip(t *testing.T) {
	ts := newTestServer(t)
	c := ts.client(t)
	signup(t, ts, c, "boss@example.com", "password123")

	body := map[string]any{"retention": map[string]int{
		"raw_secs": 3600, "fivemin_secs": 7200, "onehour_secs": 86400,
	}}
	resp, v := putSettings(t, ts, c, body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("put status = %d, want 200", resp.StatusCode)
	}
	if v.Retention.RawSecs != 3600 || v.Retention.FiveMinSecs != 7200 || v.Retention.OneHourSecs != 86400 {
		t.Fatalf("retention after put = %+v", v.Retention)
	}
	// The rollup job must see the same numbers, not just the API.
	if got := int(ts.app.db.EffectiveRetention().Raw.Seconds()); got != 3600 {
		t.Errorf("EffectiveRetention raw = %d, want 3600", got)
	}
}

func TestSettingsRetentionRejectsBadWindows(t *testing.T) {
	ts := newTestServer(t)
	c := ts.client(t)
	signup(t, ts, c, "boss@example.com", "password123")

	cases := map[string]map[string]int{
		"below the floor": {"raw_secs": 5, "fivemin_secs": 7200, "onehour_secs": 86400},
		"out of order":    {"raw_secs": 86400, "fivemin_secs": 7200, "onehour_secs": 3600},
		"zero":            {"raw_secs": 0, "fivemin_secs": 0, "onehour_secs": 0},
	}
	for name, ret := range cases {
		resp, _ := putSettings(t, ts, c, map[string]any{"retention": ret})
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("%s status = %d, want 400", name, resp.StatusCode)
		}
	}
	if got := int(ts.app.db.EffectiveRetention().Raw.Seconds()); got != 48*3600 {
		t.Errorf("retention changed despite rejection: raw = %d", got)
	}
}

// A section-scoped save must not reset the sections the form did not send.
func TestSettingsPartialUpdateKeepsOtherSections(t *testing.T) {
	ts := newTestServer(t)
	c := ts.client(t)
	signup(t, ts, c, "boss@example.com", "password123")

	putSettings(t, ts, c, map[string]any{"signup_enabled": false})
	_, v := putSettings(t, ts, c, map[string]any{"retention": map[string]int{
		"raw_secs": 3600, "fivemin_secs": 7200, "onehour_secs": 86400,
	}})
	if v.SignupEnabled {
		t.Error("saving retention re-opened signup")
	}

	putSettings(t, ts, c, map[string]any{
		"agent_update": map[string]any{"enabled": false, "concurrency": 7, "stall_secs": 300},
	})
	_, v2 := putSettings(t, ts, c, map[string]any{"retention": map[string]int{
		"raw_secs": 7200, "fivemin_secs": 14400, "onehour_secs": 172800,
	}})
	if v2.AgentUpdate.Concurrency != 7 || v2.AgentUpdate.StallSecs != 300 || v2.AgentUpdate.Enabled {
		t.Errorf("saving retention clobbered agent_update: %+v", v2.AgentUpdate)
	}

	_, v3 := putSettings(t, ts, c, map[string]any{
		"agent_update": map[string]any{"enabled": true, "concurrency": 4, "stall_secs": 120},
	})
	if v3.Retention.RawSecs != 7200 {
		t.Errorf("saving agent_update clobbered retention: %+v", v3.Retention)
	}
}

func TestSettingsAdminOnly(t *testing.T) {
	ts := newTestServer(t)
	signup(t, ts, ts.client(t), "boss@example.com", "password123")
	basic := ts.client(t)
	signup(t, ts, basic, "dev@example.com", "password123")

	if resp, _ := getSettings(t, ts, basic); resp.StatusCode != http.StatusForbidden {
		t.Errorf("basic GET status = %d, want 403", resp.StatusCode)
	}
	if resp, _ := putSettings(t, ts, basic, map[string]any{"signup_enabled": true}); resp.StatusCode != http.StatusForbidden {
		t.Errorf("basic PUT status = %d, want 403", resp.StatusCode)
	}
	if resp, _ := getSettings(t, ts, nil); resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("anonymous GET status = %d, want 401", resp.StatusCode)
	}
}

// Secret settings (SMTP password, OAuth client secret) must never rest in
// plaintext, and must survive a round trip through the master cipher.
func TestSealedSettingRoundTrip(t *testing.T) {
	ts := newTestServer(t)
	if err := ts.app.setSealedSetting("test.secret", "hunter2"); err != nil {
		t.Fatalf("setSealedSetting: %v", err)
	}
	raw, _ := ts.app.db.GetSetting("test.secret")
	if raw == "hunter2" {
		t.Fatal("secret stored in plaintext")
	}
	got, ok := ts.app.sealedSetting("test.secret")
	if !ok || got != "hunter2" {
		t.Fatalf("sealedSetting = %q, %v; want \"hunter2\", true", got, ok)
	}
	if _, ok := ts.app.sealedSetting("test.missing"); ok {
		t.Error("sealedSetting reported a value for an unset key")
	}
	ts.app.db.SetSetting("test.garbage", "not-sealed")
	if _, ok := ts.app.sealedSetting("test.garbage"); ok {
		t.Error("sealedSetting accepted a value that was never sealed")
	}
}
