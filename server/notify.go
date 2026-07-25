package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// notifyChannel is a transport's view of a channel: its URL and decrypted
// config. The store never sees this; the app decrypts before dispatch.
type notifyChannel struct {
	URL    string
	Config map[string]string
}

// Notifier sends one already-formatted payload over a channel's transport.
type Notifier interface {
	Send(ch notifyChannel, payload string, now time.Time) error
}

// secretConfigKeys are redacted from every API read of a channel's config.
var secretConfigKeys = map[string]bool{"token": true}

// newNotifiers builds the transport registry keyed by channel kind. Both kinds
// share the generic HTTP notifier.
func newNotifiers() map[string]Notifier {
	httpN := &httpNotifier{client: &http.Client{Timeout: 10 * time.Second}}
	return map[string]Notifier{
		"generic": httpN,
		"webhook": httpN,
	}
}

// httpNotifier POSTs the JSON payload to the channel URL, adding a bearer token
// when config carries one.
type httpNotifier struct {
	client *http.Client
}

func (n *httpNotifier) Send(ch notifyChannel, payload string, _ time.Time) error {
	if ch.URL == "" {
		return errors.New("channel has no url")
	}
	req, err := http.NewRequest(http.MethodPost, ch.URL, bytes.NewReader([]byte(payload)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if token := ch.Config["token"]; token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := n.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned %d", resp.StatusCode)
	}
	return nil
}

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
	ct, nonce, err := a.cipher.Seal(plain)
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
	plain, err := a.cipher.Open(ct, nonce)
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
