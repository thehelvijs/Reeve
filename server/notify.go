package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// notifyChannel is a transport's view of a channel: its URL, receiver format and
// decrypted config. The store never sees this; the app decrypts before dispatch.
// BaseURL is where a reader of the message can reach this Reeve, and is empty
// when the instance has not been told its own public address.
type notifyChannel struct {
	URL     string
	Format  string
	Config  map[string]string
	BaseURL string
}

// secretConfigKeys are redacted from every API read of a channel's config. A
// template is a setting, not a secret, so it reads back for the form to edit.
var secretConfigKeys = map[string]bool{"token": true, "routing_key": true}

// webhookClient bounds every outbound webhook, so a receiver that never answers
// cannot hold a dispatch tick open.
var webhookClient = &http.Client{Timeout: 10 * time.Second}

// postWebhook shapes the payload for the channel's receiver and POSTs it,
// adding a bearer token when config carries one. Every channel delivers here.
func postWebhook(ch notifyChannel, payload string) error {
	if ch.URL == "" {
		return errors.New("channel has no url")
	}
	out, err := shapePayload(ch, payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, ch.URL, strings.NewReader(out.body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", out.contentType)
	if token := ch.Config["token"]; token != "" {
		req.Header.Set("Authorization", authScheme(ch.URL)+" "+token)
	}
	resp, err := webhookClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	if resp.StatusCode >= 300 {
		reason := strings.TrimSpace(string(body))
		if reason == "" {
			return fmt.Errorf("webhook returned %d", resp.StatusCode)
		}
		return fmt.Errorf("webhook returned %d: %s", resp.StatusCode, reason)
	}
	return nil
}

// authScheme is Bearer everywhere but Opsgenie, which authenticates the same
// token under its own scheme name and 401s on anything else.
func authScheme(url string) string {
	if strings.Contains(strings.ToLower(url), "api.opsgenie.com/") {
		return "GenieKey"
	}
	return "Bearer"
}

// webhookConfigAAD keeps a channel config from being opened as a credential or a
// setting. The channel id is not known when the config is sealed, so this binds
// the kind of secret, not the row.
var webhookConfigAAD = []byte("reeve/webhook-config")

// sealConfig encrypts a channel config map for storage. An empty config stores
// as the plaintext sentinel '{}' (no secrets, matches the column default).
func (a *app) sealConfig(cfg map[string]string) (string, error) {
	if len(cfg) == 0 {
		return "{}", nil
	}
	plain, err := json.Marshal(cfg)
	if err != nil {
		return "", err
	}
	ct, nonce, err := a.cipher.Seal(plain, webhookConfigAAD)
	if err != nil {
		return "", err
	}
	return "enc:" + base64.StdEncoding.EncodeToString(nonce) + ":" + base64.StdEncoding.EncodeToString(ct), nil
}

// openConfig decrypts a stored channel config blob. The '{}' / empty sentinels
// read back as an empty map.
func (a *app) openConfig(blob string) (map[string]string, error) {
	if blob == "" || blob == "{}" {
		return map[string]string{}, nil
	}
	if !strings.HasPrefix(blob, "enc:") {
		return nil, errors.New("channel config unreadable")
	}
	parts := strings.SplitN(strings.TrimPrefix(blob, "enc:"), ":", 2)
	if len(parts) != 2 {
		return nil, errors.New("channel config unreadable")
	}
	nonce, err := base64.StdEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, errors.New("channel config unreadable")
	}
	ct, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, errors.New("channel config unreadable")
	}
	plain, err := a.cipher.Open(ct, nonce, webhookConfigAAD)
	if err != nil {
		return nil, errors.New("channel config unreadable")
	}
	var m map[string]string
	if err := json.Unmarshal(plain, &m); err != nil {
		return nil, errors.New("channel config unreadable")
	}
	return m, nil
}

// redactConfig blanks secret values for API reads, keeping non-secret fields.
func redactConfig(cfg map[string]string) map[string]string {
	out := make(map[string]string, len(cfg))
	for k, v := range cfg {
		if secretConfigKeys[k] && v != "" {
			out[k] = "••••"
		} else {
			out[k] = v
		}
	}
	return out
}

// dispatchDue attempts every delivery that is due, marking each sent or failed
// (with exponential backoff). Returns how many were attempted. `now` injected
// for tests.
func (a *app) dispatchDue(now time.Time) int {
	send := a.send
	if send == nil {
		send = postWebhook
	}
	deliveries, err := a.db.DueDeliveries(now, 50)
	if err != nil {
		return 0
	}
	for _, d := range deliveries {
		cfg, cfgErr := a.openConfig(d.Config)
		if cfgErr != nil {
			a.db.MarkDeliveryFailed(d.ID, "channel config unreadable", d.Attempts, now)
			continue
		}
		if err := send(notifyChannel{URL: d.URL, Format: d.Kind, Config: cfg, BaseURL: a.cfg.PublicURL}, d.Payload); err != nil {
			a.db.MarkDeliveryFailed(d.ID, err.Error(), d.Attempts, now)
			continue
		}
		a.db.MarkDeliverySent(d.ID)
	}
	return len(deliveries)
}

// notifyEvent enqueues an informational payload (e.g. an access request) to the
// channels that cover the tool or host it is about. Informational events route
// at info severity. Best-effort; delivery is handled by the dispatcher.
func (a *app) notifyEvent(toolID, hostID string, payload map[string]any, now time.Time) {
	hooks, err := a.db.ResolveChannels(toolID, hostID, "info")
	if err != nil {
		return
	}
	body, _ := json.Marshal(payload)
	for _, h := range hooks {
		a.db.EnqueueDelivery(h.ID, string(body), now)
	}
}
