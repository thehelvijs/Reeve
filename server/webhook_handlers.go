package main

import (
	"net/http"
	"strings"

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
	writeJSON(w, http.StatusOK, events)
}

func (a *app) handleListDeliveries(w http.ResponseWriter, _ *http.Request) {
	deliveries, err := a.db.ListDeliveries(200)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not list deliveries")
		return
	}
	writeJSON(w, http.StatusOK, deliveries)
}
