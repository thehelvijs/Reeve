package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/thehelvijs/Reeve/server/internal/store"
)

var channelSeverities = map[string]bool{"info": true, "warning": true, "error": true}

// channelView is the API shape of a channel: config secrets are redacted and it
// carries the min_severity gate. Detected reports which receiver an "auto"
// channel resolves to, so the form can name it without repeating the sniffing.
type channelView struct {
	ID          string            `json:"id"`
	OwnerType   string            `json:"owner_type"`
	OwnerID     string            `json:"owner_id"`
	URL         string            `json:"url"`
	Enabled     bool              `json:"enabled"`
	Format      string            `json:"format"`
	Detected    string            `json:"detected"`
	Config      map[string]string `json:"config"`
	MinSeverity string            `json:"min_severity"`
	Events      []string          `json:"events"`
}

// handleChannelFormats hands the form the receiver vocabulary, the event types
// and the template placeholders, so none of the three can drift from the code
// that reads them.
func (a *app) handleChannelFormats(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"formats":   channelFormats,
		"events":    channelEvents,
		"variables": templateVariables,
	})
}

var knownEvents = func() map[string]bool {
	m := make(map[string]bool, len(channelEvents))
	for _, e := range channelEvents {
		m[e] = true
	}
	return m
}()

// cleanEvents turns the requested subscription into the stored comma-separated
// list. Selecting every type stores as empty, the same as selecting none, so a
// channel that wants everything keeps saying so when a new type is added.
func cleanEvents(in []string) (string, error) {
	seen := make(map[string]bool, len(in))
	out := make([]string, 0, len(in))
	for _, e := range in {
		e = strings.TrimSpace(e)
		if e == "" || seen[e] {
			continue
		}
		if !knownEvents[e] {
			return "", errors.New("unknown event " + e)
		}
		seen[e] = true
		out = append(out, e)
	}
	if len(out) == len(channelEvents) {
		return "", nil
	}
	return strings.Join(out, ","), nil
}

func splitEvents(stored string) []string {
	if stored == "" {
		return []string{}
	}
	return strings.Split(stored, ",")
}

func (a *app) channelToView(w store.Webhook) channelView {
	cfg, err := a.openConfig(w.Config)
	if err != nil {
		cfg = map[string]string{}
	}
	format := w.Format
	if !knownFormats[format] {
		format = fmtAuto
	}
	return channelView{
		ID: w.ID, OwnerType: w.OwnerType, OwnerID: w.OwnerID, URL: w.URL,
		Enabled: w.Enabled, Format: format, Detected: detectFormat(w.URL),
		Config: redactConfig(cfg), MinSeverity: w.MinSeverity, Events: splitEvents(w.Events),
	}
}

func (a *app) handleListWebhooks(w http.ResponseWriter, _ *http.Request) {
	hooks, err := a.db.ListWebhooks()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not list webhooks")
		return
	}
	out := make([]channelView, 0, len(hooks))
	for _, h := range hooks {
		out = append(out, a.channelToView(h))
	}
	writeJSON(w, http.StatusOK, out)
}

// validateChannelShape defaults an empty format to auto and rejects a shape the
// shaper would only fail on at delivery time: an unknown receiver, a custom
// format with no template, or a template that is not JSON once its placeholders
// are filled in. A rejected save is worth more than a dead delivery an hour later.
func validateChannelShape(format *string, cfg map[string]string) error {
	if *format == "" {
		*format = fmtAuto
	}
	if !knownFormats[*format] {
		return errors.New("unknown channel format")
	}
	tmpl := strings.TrimSpace(cfg["template"])
	if *format != fmtCustom {
		return nil
	}
	if tmpl == "" {
		return errors.New("a custom format needs a template")
	}
	probe := make(map[string]string, len(templateVariables))
	for _, v := range templateVariables {
		probe[v] = "x"
	}
	if !json.Valid([]byte(renderTemplate(tmpl, probe))) {
		return errors.New("the template is not valid JSON once its variables are filled in")
	}
	return nil
}

// handleUpdateWebhook edits a saved channel in place. A blank token leaves the
// stored one alone, because every read redacts it and the form never holds it.
func (a *app) handleUpdateWebhook(w http.ResponseWriter, r *http.Request) {
	hook, err := a.db.GetWebhook(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "webhook not found")
		return
	}
	cfg, err := a.openConfig(hook.Config)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "channel config unreadable")
		return
	}
	in := struct {
		URL         string            `json:"url"`
		Format      string            `json:"format"`
		Config      map[string]string `json:"config"`
		MinSeverity string            `json:"min_severity"`
		Events      []string          `json:"events"`
		Enabled     bool              `json:"enabled"`
	}{URL: hook.URL, Format: hook.Format, MinSeverity: hook.MinSeverity,
		Events: splitEvents(hook.Events), Enabled: hook.Enabled}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	events, err := cleanEvents(in.Events)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_events", err.Error())
		return
	}
	for k, v := range in.Config {
		if secretConfigKeys[k] && v == "" {
			continue
		}
		cfg[k] = v
	}
	if err := validateChannelShape(&in.Format, cfg); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_format", err.Error())
		return
	}
	if !strings.HasPrefix(in.URL, "http://") && !strings.HasPrefix(in.URL, "https://") {
		writeError(w, http.StatusBadRequest, "invalid_url", "url must be http(s)")
		return
	}
	if !channelSeverities[in.MinSeverity] {
		writeError(w, http.StatusBadRequest, "invalid_severity", "min_severity must be info, warning, or error")
		return
	}
	sealed, err := a.sealConfig(cfg)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not encrypt channel config")
		return
	}
	if err := a.db.UpdateWebhook(store.Webhook{
		ID: hook.ID, URL: in.URL, Format: in.Format, Config: sealed,
		MinSeverity: in.MinSeverity, Events: events, Enabled: in.Enabled,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not update webhook")
		return
	}
	updated, err := a.db.GetWebhook(hook.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not read the webhook back")
		return
	}
	writeJSON(w, http.StatusOK, a.channelToView(updated))
}

func (a *app) handleCreateWebhook(w http.ResponseWriter, r *http.Request) {
	var in struct {
		OwnerType   string            `json:"owner_type"`
		OwnerID     string            `json:"owner_id"`
		URL         string            `json:"url"`
		Format      string            `json:"format"`
		Config      map[string]string `json:"config"`
		MinSeverity string            `json:"min_severity"`
		Events      []string          `json:"events"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	events, err := cleanEvents(in.Events)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_events", err.Error())
		return
	}
	switch in.OwnerType {
	case "global":
		in.OwnerID = ""
	case "tool", "host", "group":
		if strings.TrimSpace(in.OwnerID) == "" {
			writeError(w, http.StatusBadRequest, "invalid_owner", "a "+in.OwnerType+" channel needs a "+in.OwnerType+" to watch")
			return
		}
	default:
		writeError(w, http.StatusBadRequest, "invalid_owner", "owner_type must be global, tool, host, or group")
		return
	}
	if err := validateChannelShape(&in.Format, in.Config); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_format", err.Error())
		return
	}
	if !strings.HasPrefix(in.URL, "http://") && !strings.HasPrefix(in.URL, "https://") {
		writeError(w, http.StatusBadRequest, "invalid_url", "url must be http(s)")
		return
	}
	if in.MinSeverity == "" {
		in.MinSeverity = "info"
	}
	if !channelSeverities[in.MinSeverity] {
		writeError(w, http.StatusBadRequest, "invalid_severity", "min_severity must be info, warning, or error")
		return
	}
	sealed, err := a.sealConfig(in.Config)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not encrypt channel config")
		return
	}
	wh, err := a.db.CreateWebhook(store.Webhook{
		OwnerType: in.OwnerType, OwnerID: in.OwnerID, URL: in.URL, Format: in.Format,
		Config: sealed, MinSeverity: in.MinSeverity, Events: events,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not create webhook")
		return
	}
	writeJSON(w, http.StatusCreated, a.channelToView(wh))
}

// handleTestWebhook delivers a probe so a wrong URL or a receiver that rejects
// the payload surfaces at setup time, not the first time an alert fires. A saved
// channel is named by id, because every read redacts its token; a channel the
// form is still holding carries its URL and config instead.
//
// A receiver that answers badly is not an API failure: the response is 200 with
// ok=false and the reason, which is what the page shows next to the row.
func (a *app) handleTestWebhook(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ID     string            `json:"id"`
		URL    string            `json:"url"`
		Format string            `json:"format"`
		Config map[string]string `json:"config"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	ch := notifyChannel{URL: strings.TrimSpace(in.URL), Format: in.Format, Config: in.Config, BaseURL: a.baseURL(r)}
	if in.ID != "" {
		hook, err := a.db.GetWebhook(in.ID)
		if err != nil {
			writeError(w, http.StatusNotFound, "not_found", "webhook not found")
			return
		}
		cfg, err := a.openConfig(hook.Config)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal", "channel config unreadable")
			return
		}
		ch = notifyChannel{URL: hook.URL, Format: hook.Format, Config: cfg, BaseURL: a.baseURL(r)}
	}
	if !strings.HasPrefix(ch.URL, "http://") && !strings.HasPrefix(ch.URL, "https://") {
		writeError(w, http.StatusBadRequest, "invalid_url", "url must be http(s)")
		return
	}
	send := a.send
	if send == nil {
		send = postWebhook
	}
	payload, _ := json.Marshal(map[string]any{
		"event":    "test",
		"severity": "info",
		"message":  "Test delivery from Reeve. This channel works.",
		"sent_at":  time.Now().UTC().Format(time.RFC3339),
	})
	if err := send(ch, string(payload)); err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (a *app) handleDeleteWebhook(w http.ResponseWriter, r *http.Request) {
	if err := a.db.DeleteWebhook(r.PathValue("id")); err != nil {
		writeError(w, http.StatusNotFound, "not_found", "webhook not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *app) handleListAlertEvents(w http.ResponseWriter, _ *http.Request) {
	events, err := a.db.ListAlertEvents(200)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not list alerts")
		return
	}
	if events == nil {
		events = []store.AlertEvent{}
	}
	writeJSON(w, http.StatusOK, events)
}

func (a *app) handleListDeliveries(w http.ResponseWriter, _ *http.Request) {
	deliveries, err := a.db.ListDeliveries(200)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not list deliveries")
		return
	}
	if deliveries == nil {
		deliveries = []store.Delivery{}
	}
	writeJSON(w, http.StatusOK, deliveries)
}
