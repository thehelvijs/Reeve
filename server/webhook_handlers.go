package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/thehelvijs/Reeve/server/internal/store"
)

// channelKinds is the single source of the allowed channel-kind vocabulary,
// shared by create-validation and the notifier registry.
var channelKinds = map[string]bool{
	"generic": true, "webhook": true,
}

var channelSeverities = map[string]bool{"info": true, "warning": true, "error": true}

// channelView is the API shape of a channel: config secrets are redacted and it
// carries the min_severity gate.
type channelView struct {
	ID          string            `json:"id"`
	OwnerType   string            `json:"owner_type"`
	OwnerID     string            `json:"owner_id"`
	URL         string            `json:"url"`
	Enabled     bool              `json:"enabled"`
	Format      string            `json:"format"`
	Config      map[string]string `json:"config"`
	MinSeverity string            `json:"min_severity"`
}

func (a *app) channelToView(w store.Webhook) channelView {
	cfg, err := a.openConfig(w.Config)
	if err != nil {
		cfg = map[string]string{}
	}
	return channelView{
		ID: w.ID, OwnerType: w.OwnerType, OwnerID: w.OwnerID, URL: w.URL,
		Enabled: w.Enabled, Format: w.Format, Config: redactConfig(cfg), MinSeverity: w.MinSeverity,
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

func (a *app) handleCreateWebhook(w http.ResponseWriter, r *http.Request) {
	var in struct {
		OwnerType   string            `json:"owner_type"`
		OwnerID     string            `json:"owner_id"`
		URL         string            `json:"url"`
		Format      string            `json:"format"`
		Config      map[string]string `json:"config"`
		MinSeverity string            `json:"min_severity"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	switch in.OwnerType {
	case "tool", "group", "global":
	default:
		writeError(w, http.StatusBadRequest, "invalid_owner", "owner_type must be tool, group, or global")
		return
	}
	if in.Format == "" {
		in.Format = "generic"
	}
	if !channelKinds[in.Format] {
		writeError(w, http.StatusBadRequest, "invalid_format", "unknown channel kind")
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
	wh, err := a.db.CreateWebhook(in.OwnerType, in.OwnerID, in.URL, in.Format, sealed, in.MinSeverity)
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
		Config map[string]string `json:"config"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	ch := notifyChannel{URL: strings.TrimSpace(in.URL), Config: in.Config}
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
		ch = notifyChannel{URL: hook.URL, Config: cfg}
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
