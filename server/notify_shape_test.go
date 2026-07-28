package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const testAlert = `{"event":"fired","type":"cpu","severity":"error","tool":"postgres","tool_id":"t1","host":"db-1","message":"cpu 97%"}`

const testLine = "[error] postgres on db-1: cpu 97%"

// Every popular receiver 400s on a body without its own field, so the URL alone
// has to produce the right shape with nothing configured.
func TestAutoFormatShapesPerReceiver(t *testing.T) {
	for _, tc := range []struct{ name, url, want string }{
		{"discord", "https://discord.com/api/webhooks/1/abc", `{"content":"` + testLine + `"}`},
		{"discordapp", "https://discordapp.com/api/webhooks/1/abc", `{"content":"` + testLine + `"}`},
		{"slack", "https://hooks.slack.com/services/T/B/x", `{"text":"` + testLine + `"}`},
		{"google chat", "https://chat.googleapis.com/v1/spaces/s/messages?key=k", `{"text":"` + testLine + `"}`},
		{"teams connector", "https://acme.webhook.office.com/webhookb2/abc", `{"text":"` + testLine + `"}`},
		{"webex", "https://webexapis.com/v1/webhooks/incoming/abc", `{"markdown":"` + testLine + `"}`},
		{"gotify", "https://push.example.com/message?token=k", testAlert},
		{"private sink", "https://sink.example.com/hook", testAlert},
		{"mattermost is not guessed", "https://chat.example.com/hooks/abc", testAlert},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, err := shapePayload(notifyChannel{URL: tc.url, Format: fmtAuto}, testAlert)
			if err != nil {
				t.Fatalf("shapePayload: %v", err)
			}
			if out.body != tc.want {
				t.Errorf("body = %s, want %s", out.body, tc.want)
			}
			if out.contentType != "application/json" {
				t.Errorf("content type = %q", out.contentType)
			}
		})
	}
}

// A receiver on the operator's own domain, or one whose shape the sniff would get
// wrong, is chosen by name. ntfy is the only one that takes plain text.
func TestNamedFormatOverridesTheURL(t *testing.T) {
	for _, tc := range []struct{ format, url, wantBody, wantType string }{
		{fmtMattermost, "https://chat.example.com/hooks/abc", `{"text":"` + testLine + `"}`, "application/json"},
		{fmtRocketChat, "https://chat.example.com/hooks/a/b", `{"text":"` + testLine + `"}`, "application/json"},
		{fmtGeneric, "https://discord.com/api/webhooks/1/abc", testAlert, "application/json"},
		{fmtNtfy, "https://ntfy.example.com/reeve", testLine, "text/plain"},
		{fmtGotify, "https://push.example.com/message?token=k", `{"message":"` + testLine + `","title":"Reeve: postgres"}`, "application/json"},
	} {
		t.Run(tc.format, func(t *testing.T) {
			out, err := shapePayload(notifyChannel{URL: tc.url, Format: tc.format}, testAlert)
			if err != nil {
				t.Fatalf("shapePayload: %v", err)
			}
			if out.body != tc.wantBody {
				t.Errorf("body = %s, want %s", out.body, tc.wantBody)
			}
			if out.contentType != tc.wantType {
				t.Errorf("content type = %q, want %q", out.contentType, tc.wantType)
			}
		})
	}
}

// A channel saved before the vocabulary existed carries a word nobody knows. It
// must fall back to reading the URL, not fail every delivery.
func TestLegacyFormatFallsBackToTheURL(t *testing.T) {
	out, err := shapePayload(notifyChannel{URL: "https://discord.com/api/webhooks/1/a", Format: "webhook"}, testAlert)
	if err != nil {
		t.Fatalf("shapePayload: %v", err)
	}
	if out.body != `{"content":"`+testLine+`"}` {
		t.Errorf("body = %s, want the discord shape", out.body)
	}
}

// Teams retired the connector that took {"text": …}; a Power Automate flow URL
// needs the adaptive-card wrapper instead.
func TestTeamsFlowSendsAnAdaptiveCard(t *testing.T) {
	out, err := shapePayload(notifyChannel{URL: "https://prod-1.westus.logic.azure.com/workflows/x"}, testAlert)
	if err != nil {
		t.Fatalf("shapePayload: %v", err)
	}
	var got struct {
		Type        string `json:"type"`
		Attachments []struct {
			ContentType string `json:"contentType"`
			Content     struct {
				Body []struct{ Text string } `json:"body"`
			} `json:"content"`
		} `json:"attachments"`
	}
	if err := json.Unmarshal([]byte(out.body), &got); err != nil {
		t.Fatalf("unmarshal: %v (%s)", err, out.body)
	}
	if got.Type != "message" || len(got.Attachments) != 1 {
		t.Fatalf("wrapper = %s", out.body)
	}
	if got.Attachments[0].ContentType != "application/vnd.microsoft.card.adaptive" {
		t.Errorf("content type = %q", got.Attachments[0].ContentType)
	}
	if got.Attachments[0].Content.Body[0].Text != testLine {
		t.Errorf("card text = %q, want %q", got.Attachments[0].Content.Body[0].Text, testLine)
	}
}

// PagerDuty rejects any severity outside its own four words, and a resolve has to
// arrive as event_action=resolve or the incident it opened never closes.
func TestPagerDutyEventShape(t *testing.T) {
	ch := notifyChannel{URL: "https://events.pagerduty.com/v2/enqueue", Config: map[string]string{"routing_key": "R1"}}
	out, err := shapePayload(ch, `{"event":"resolved","type":"cpu","severity":"warning","tool_id":"t1","host":"db-1","message":"back under"}`)
	if err != nil {
		t.Fatalf("shapePayload: %v", err)
	}
	var got struct {
		RoutingKey  string `json:"routing_key"`
		EventAction string `json:"event_action"`
		DedupKey    string `json:"dedup_key"`
		Payload     struct {
			Summary  string `json:"summary"`
			Source   string `json:"source"`
			Severity string `json:"severity"`
		} `json:"payload"`
	}
	if err := json.Unmarshal([]byte(out.body), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.RoutingKey != "R1" || got.EventAction != "resolve" {
		t.Errorf("routing/action = %q/%q", got.RoutingKey, got.EventAction)
	}
	if got.DedupKey != "t1cpu" {
		t.Errorf("dedup_key = %q, want the subject so a resolve closes the fire", got.DedupKey)
	}
	if got.Payload.Severity != "warning" || got.Payload.Source != "db-1" {
		t.Errorf("payload = %+v", got.Payload)
	}
}

// A receiver whose body needs a secret cannot be shaped from the URL alone. That
// has to be said at setup, not swallowed into an unexplained rejection.
func TestFormatsThatNeedConfigSayWhatIsMissing(t *testing.T) {
	for _, tc := range []struct{ url, want string }{
		{"https://api.telegram.org/bot123/sendMessage", "chat_id"},
		{"https://events.pagerduty.com/v2/enqueue", "routing_key"},
	} {
		_, err := shapePayload(notifyChannel{URL: tc.url}, testAlert)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: error = %v, want it to name %s", tc.url, err, tc.want)
		}
	}
	out, err := shapePayload(notifyChannel{URL: "https://api.telegram.org/bot123/sendMessage",
		Config: map[string]string{"chat_id": "-100"}}, testAlert)
	if err != nil {
		t.Fatalf("shapePayload with a chat_id: %v", err)
	}
	if out.body != `{"chat_id":"-100","text":"`+testLine+`"}` {
		t.Errorf("body = %s", out.body)
	}
}

// A custom template is the escape hatch for a receiver nobody anticipated. A
// value carrying a quote must not break the JSON it is substituted into.
func TestCustomTemplateSubstitutesAndEscapes(t *testing.T) {
	ch := notifyChannel{
		URL:    "https://sink.example.com/hook",
		Format: fmtCustom,
		Config: map[string]string{"template": `{"who":"{{host}}","why":"{{message}}","at":"{{ timestamp }}","nope":"{{bogus}}"}`},
	}
	out, err := shapePayload(ch, `{"host":"db-1","message":"disk is \"full\"","timestamp":"2026-07-29T00:00:00Z"}`)
	if err != nil {
		t.Fatalf("shapePayload: %v", err)
	}
	if !json.Valid([]byte(out.body)) {
		t.Fatalf("template produced invalid JSON: %s", out.body)
	}
	var got map[string]string
	json.Unmarshal([]byte(out.body), &got)
	if got["why"] != `disk is "full"` {
		t.Errorf("why = %q, want the quotes intact", got["why"])
	}
	if got["who"] != "db-1" || got["at"] != "2026-07-29T00:00:00Z" {
		t.Errorf("substitution = %v", got)
	}
	if got["nope"] != "{{bogus}}" {
		t.Errorf("nope = %q, want an unknown name left standing so the typo shows", got["nope"])
	}
}

// A custom format with no template would post nothing at all. The save is what
// has to refuse it, so the operator hears about it while the form is open.
func TestValidateChannelShape(t *testing.T) {
	for _, tc := range []struct {
		name, format string
		cfg          map[string]string
		wantErr      string
	}{
		{"empty defaults to auto", "", nil, ""},
		{"unknown receiver", "pigeon", nil, "unknown channel format"},
		{"custom with no template", fmtCustom, nil, "needs a template"},
		{"custom with broken json", fmtCustom, map[string]string{"template": `{"a":}`}, "not valid JSON"},
		{"custom that holds up", fmtCustom, map[string]string{"template": `{"a":"{{message}}"}`}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			format := tc.format
			err := validateChannelShape(&format, tc.cfg)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("err = %v, want nil", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("err = %v, want it to mention %q", err, tc.wantErr)
			}
		})
	}
	format := ""
	validateChannelShape(&format, nil)
	if format != fmtAuto {
		t.Errorf("empty format defaulted to %q, want auto", format)
	}
}

// A payload with no message must not become an empty post, which is its own 400
// at Discord.
func TestMessagelessPayloadKeepsItsJSON(t *testing.T) {
	out, err := shapePayload(notifyChannel{URL: "https://discord.com/api/webhooks/1/a"}, `{"x":1}`)
	if err != nil {
		t.Fatalf("shapePayload: %v", err)
	}
	if out.body != `{"content":"{\"x\":1}"}` {
		t.Errorf("body = %s, want the raw JSON in the field", out.body)
	}
}

// Opsgenie authenticates the same token under its own scheme and 401s on Bearer.
func TestOpsgenieUsesItsOwnAuthScheme(t *testing.T) {
	if got := authScheme("https://api.opsgenie.com/v2/alerts"); got != "GenieKey" {
		t.Errorf("opsgenie scheme = %q, want GenieKey", got)
	}
	if got := authScheme("https://sink.example.com/hook"); got != "Bearer" {
		t.Errorf("default scheme = %q, want Bearer", got)
	}
}

// "webhook returned 400" alone told nobody which field the receiver wanted.
func TestPostWebhookErrorCarriesResponseBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"message":"Cannot send an empty message","code":50006}`))
	}))
	defer srv.Close()
	err := postWebhook(notifyChannel{URL: srv.URL}, `{"x":1}`)
	if err == nil {
		t.Fatal("postWebhook on a 400 returned nil, want an error")
	}
	if !strings.Contains(err.Error(), "Cannot send an empty message") {
		t.Errorf("error = %q, want the receiver's reason", err)
	}
}
