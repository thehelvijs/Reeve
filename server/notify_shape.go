package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// A receiver rejects a body it does not recognise, and each popular one insists
// on its own field: Discord wants "content", Slack "text", Webex "markdown".
// Reeve's own JSON reaches none of them, so every channel names the receiver it
// posts to and the payload is rewritten on the way out.
//
// "auto" reads the receiver off the URL and is the default. "generic" sends
// Reeve's JSON untouched, which is what a sink of your own wants. "custom"
// renders the channel's template, so a receiver nobody here anticipated is still
// reachable without a code change.
const (
	fmtAuto       = "auto"
	fmtGeneric    = "generic"
	fmtCustom     = "custom"
	fmtDiscord    = "discord"
	fmtSlack      = "slack"
	fmtMattermost = "mattermost"
	fmtRocketChat = "rocketchat"
	fmtGoogleChat = "googlechat"
	fmtTeams      = "teams"
	fmtTeamsFlow  = "teamsflow"
	fmtWebex      = "webex"
	fmtNtfy       = "ntfy"
	fmtGotify     = "gotify"
	fmtTelegram   = "telegram"
	fmtPagerDuty  = "pagerduty"
)

// channelFormats is the vocabulary the API validates a channel's format against,
// in the order the UI offers them.
var channelFormats = []string{
	fmtAuto, fmtDiscord, fmtSlack, fmtMattermost, fmtRocketChat, fmtGoogleChat,
	fmtTeams, fmtTeamsFlow, fmtWebex, fmtNtfy, fmtGotify, fmtTelegram,
	fmtPagerDuty, fmtGeneric, fmtCustom,
}

var knownFormats = func() map[string]bool {
	m := make(map[string]bool, len(channelFormats))
	for _, f := range channelFormats {
		m[f] = true
	}
	return m
}()

// urlSignatures maps a receiver's own hosted URL shape to its format. Only
// vendor-hosted addresses are listed: Mattermost and Rocket.Chat run on the
// operator's own domain under a path a private sink could also use, so guessing
// from those would rewrite a payload somebody meant to send raw. Both are chosen
// by name instead.
var urlSignatures = []struct {
	match  string
	format string
}{
	{"discord.com/api/webhooks/", fmtDiscord},
	{"discordapp.com/api/webhooks/", fmtDiscord},
	{"hooks.slack.com/", fmtSlack},
	{"chat.googleapis.com/", fmtGoogleChat},
	{"webhook.office.com/", fmtTeams},
	{"outlook.office.com/webhook", fmtTeams},
	{"logic.azure.com/", fmtTeamsFlow},
	{"webexapis.com/v1/webhooks/incoming/", fmtWebex},
	{"ntfy.sh/", fmtNtfy},
	{"api.telegram.org/bot", fmtTelegram},
	{"events.pagerduty.com/v2/enqueue", fmtPagerDuty},
	{"api.opsgenie.com/v2/alerts", fmtGeneric},
}

// detectFormat names the receiver a URL belongs to. An address nobody
// recognises is a sink of the operator's own and gets Reeve's JSON.
func detectFormat(url string) string {
	lower := strings.ToLower(url)
	for _, sig := range urlSignatures {
		if strings.Contains(lower, sig.match) {
			return sig.format
		}
	}
	return fmtGeneric
}

// A line is emphasised and linked the way its receiver spells it. Slack and
// Google Chat share their own syntax, the chat clients that take markdown share
// that, and the push services render neither and get the bare URL.
const (
	styleText     = "text"
	styleMarkdown = "markdown"
	styleSlack    = "slack"
)

var lineStyles = map[string]string{
	fmtDiscord:    styleMarkdown,
	fmtMattermost: styleMarkdown,
	fmtRocketChat: styleMarkdown,
	fmtTeams:      styleMarkdown,
	fmtTeamsFlow:  styleMarkdown,
	fmtWebex:      styleMarkdown,
	fmtSlack:      styleSlack,
	fmtGoogleChat: styleSlack,
}

// shaped is the body and content type one receiver takes.
type shaped struct {
	body        string
	contentType string
}

// jsonBody is every receiver but ntfy, which takes the message as plain text.
func jsonBody(v any) (shaped, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return shaped{}, err
	}
	return shaped{body: string(b), contentType: "application/json"}, nil
}

// shapePayload rewrites one Reeve payload for one receiver. A payload it cannot
// read, or one carrying no message, is passed through whole rather than sent as
// an empty post, which is its own rejection at Discord.
func shapePayload(ch notifyChannel, payload string) (shaped, error) {
	// A channel stored before the vocabulary existed carries a word nobody
	// recognises. Reading the receiver off its URL is what it wanted all along.
	format := ch.Format
	if !knownFormats[format] || format == fmtAuto {
		format = detectFormat(ch.URL)
	}
	raw := shaped{body: payload, contentType: "application/json"}
	if format == fmtGeneric {
		return raw, nil
	}

	fields := payloadFields(payload)
	if format == fmtCustom {
		tmpl := ch.Config["template"]
		if strings.TrimSpace(tmpl) == "" {
			return raw, errors.New("channel format is custom but its template is empty")
		}
		return shaped{body: renderTemplate(tmpl, fields), contentType: "application/json"}, nil
	}
	line := messageLine(lineStyles[format], fields, payload, ch.BaseURL)

	switch format {
	case fmtDiscord:
		return jsonBody(map[string]string{"content": line})
	case fmtSlack, fmtMattermost, fmtRocketChat, fmtGoogleChat, fmtTeams:
		return jsonBody(map[string]string{"text": line})
	case fmtWebex:
		return jsonBody(map[string]string{"markdown": line})
	case fmtGotify:
		return jsonBody(map[string]string{"title": gotifyTitle(fields), "message": line})
	case fmtNtfy:
		return shaped{body: line, contentType: "text/plain"}, nil
	case fmtTeamsFlow:
		return jsonBody(adaptiveCard(line))
	case fmtTelegram:
		chatID := ch.Config["chat_id"]
		if chatID == "" {
			return shaped{}, errors.New("telegram needs a chat_id in the channel config")
		}
		return jsonBody(map[string]string{"chat_id": chatID, "text": line})
	case fmtPagerDuty:
		key := ch.Config["routing_key"]
		if key == "" {
			return shaped{}, errors.New("pagerduty needs a routing_key in the channel config")
		}
		return jsonBody(pagerDutyEvent(key, fields, line, alertLink(ch.BaseURL, fields)))
	}
	return raw, fmt.Errorf("unknown channel format %q", format)
}

// payloadFields flattens a Reeve payload into the variables a line or a template
// is built from. Non-string values render the way JSON would print them.
func payloadFields(payload string) map[string]string {
	var m map[string]any
	if err := json.Unmarshal([]byte(payload), &m); err != nil {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		if s, ok := v.(string); ok {
			out[k] = s
			continue
		}
		b, err := json.Marshal(v)
		if err != nil {
			continue
		}
		out[k] = string(b)
	}
	return out
}

// messageLine renders a payload for one receiver: severity and what it is about
// in that receiver's emphasis, the message the alert already wrote, then a link
// back to Reeve on its own line. A payload with no message keeps its whole JSON,
// so nothing is lost to a receiver that would have taken it.
func messageLine(style string, fields map[string]string, raw, base string) string {
	if fields["message"] == "" {
		return raw
	}
	line := fields["message"]
	if head := messageHead(fields); head != "" {
		line = emphasise(style, head) + " " + line
	}
	if link := alertLink(base, fields); link != "" {
		line += "\n" + linkText(style, link)
	}
	return line
}

// messageHead is the severity and the subject an alert is about, the part a
// chat client shows in bold. The colon belongs to the subject: an alert that
// names none reads as "[info] the message", not "[info]: the message".
func messageHead(fields map[string]string) string {
	var b strings.Builder
	if sev := fields["severity"]; sev != "" {
		b.WriteString("[" + sev + "]")
	}
	tool, host := fields["tool"], fields["host"]
	var subject string
	switch {
	case tool != "" && host != "":
		subject = tool + " on " + host + ":"
	case tool != "":
		subject = tool + ":"
	case host != "":
		subject = host + ":"
	}
	if subject != "" {
		if b.Len() > 0 {
			b.WriteString(" ")
		}
		b.WriteString(subject)
	}
	return b.String()
}

func emphasise(style, text string) string {
	switch style {
	case styleMarkdown:
		return "**" + text + "**"
	case styleSlack:
		return "*" + text + "*"
	}
	return text
}

func linkText(style, link string) string {
	switch style {
	case styleMarkdown:
		return "[Open in Reeve](" + link + ")"
	case styleSlack:
		return "<" + link + "|Open in Reeve>"
	}
	return link
}

// alertLink points at the page that shows what fired: the service when the
// alert names one, otherwise the host. It stays empty until REEVE_PUBLIC_URL is
// set, because a link to localhost helps nobody reading it in Discord.
func alertLink(base string, fields map[string]string) string {
	base = strings.TrimSuffix(base, "/")
	if base == "" {
		return ""
	}
	if id := fields["tool_id"]; id != "" {
		return base + "/services/" + id
	}
	if id := fields["host_id"]; id != "" {
		return base + "/hosts/" + id
	}
	return base
}

func gotifyTitle(fields map[string]string) string {
	if tool := fields["tool"]; tool != "" {
		return "Reeve: " + tool
	}
	return "Reeve"
}

// adaptiveCard is the shape a Power Automate flow's Teams webhook trigger takes;
// the connector URLs that accepted a bare {"text": …} were retired in 2025.
func adaptiveCard(line string) map[string]any {
	return map[string]any{
		"type": "message",
		"attachments": []map[string]any{{
			"contentType": "application/vnd.microsoft.card.adaptive",
			"content": map[string]any{
				"type":    "AdaptiveCard",
				"$schema": "http://adaptivecards.io/schemas/adaptive-card.json",
				"version": "1.4",
				"body":    []map[string]any{{"type": "TextBlock", "text": line, "wrap": true}},
			},
		}},
	}
}

// pagerDutySeverities maps Reeve's three levels onto the four the Events API
// accepts. PagerDuty rejects any other word outright.
var pagerDutySeverities = map[string]string{"info": "info", "warning": "warning", "error": "error"}

// pagerDutyEvent builds an Events API v2 payload. dedup_key on the alert subject
// is what lets a resolve close the incident the fire opened.
func pagerDutyEvent(routingKey string, fields map[string]string, line, link string) map[string]any {
	severity := pagerDutySeverities[fields["severity"]]
	if severity == "" {
		severity = "error"
	}
	action := "trigger"
	if fields["event"] == "resolved" {
		action = "resolve"
	}
	source := fields["host"]
	if source == "" {
		source = "reeve"
	}
	ev := map[string]any{
		"routing_key":  routingKey,
		"event_action": action,
		"payload": map[string]any{
			"summary":  line,
			"source":   source,
			"severity": severity,
		},
	}
	if key := fields["tool_id"] + fields["type"]; key != "" {
		ev["dedup_key"] = key
	}
	if link != "" {
		ev["links"] = []map[string]string{{"href": link, "text": "Open in Reeve"}}
	}
	return ev
}

// templateVar matches the {{name}} placeholders a custom template substitutes,
// the same spelling GitLab's custom webhook templates use.
var templateVar = regexp.MustCompile(`\{\{\s*([a-zA-Z0-9_]+)\s*\}\}`)

// renderTemplate substitutes the payload's fields into a template. Values are
// escaped for a JSON string, because that is where a placeholder sits in every
// body a receiver accepts; a message with a quote in it would otherwise produce
// JSON the receiver rejects. A name the payload does not carry is left standing,
// so a typo shows up in the delivery rather than silently emptying a field.
func renderTemplate(tmpl string, fields map[string]string) string {
	return templateVar.ReplaceAllStringFunc(tmpl, func(m string) string {
		name := templateVar.FindStringSubmatch(m)[1]
		v, ok := fields[name]
		if !ok {
			return m
		}
		b, err := json.Marshal(v)
		if err != nil {
			return m
		}
		return string(b[1 : len(b)-1])
	})
}

// templateVariables lists the placeholders a template may use, for the form that
// edits one. Every alert payload carries these; the test probe carries the first
// three and sent_at.
var templateVariables = []string{
	"event", "severity", "message", "type", "tool", "tool_id", "host", "host_id", "timestamp", "sent_at",
}
