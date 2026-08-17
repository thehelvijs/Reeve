package main

import (
	"encoding/base64"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/thehelvijs/Reeve/server/internal/store"
)

// Retention bounds keep an operator from configuring a window that either
// prunes on every tick or never prunes at all.
const (
	minRetentionSecs = 600
	maxRetentionSecs = 5 * 365 * 24 * 3600
)

// sealedPrefix marks a settings value as master-key ciphertext, stored as
// "sealed:" + base64(nonce) + "." + base64(ciphertext).
const sealedPrefix = "sealed:"

// settingAAD binds a sealed setting to its key, so one secret setting cannot be
// swapped for another.
func settingAAD(key string) []byte {
	return []byte("reeve/setting/" + key)
}

// setSealedSetting encrypts a secret setting before it touches disk.
func (a *app) setSealedSetting(key, plain string) error {
	ct, nonce, err := a.cipher.Seal([]byte(plain), settingAAD(key))
	if err != nil {
		return err
	}
	enc := sealedPrefix + base64.StdEncoding.EncodeToString(nonce) + "." + base64.StdEncoding.EncodeToString(ct)
	return a.db.SetSetting(key, enc)
}

// sealedSetting decrypts a secret setting, reporting whether one is stored.
func (a *app) sealedSetting(key string) (string, bool) {
	v, ok := a.db.GetSetting(key)
	if !ok || v == "" {
		return "", false
	}
	body, ok := strings.CutPrefix(v, sealedPrefix)
	if !ok {
		return "", false
	}
	nonceB64, ctB64, ok := strings.Cut(body, ".")
	if !ok {
		return "", false
	}
	nonce, err := base64.StdEncoding.DecodeString(nonceB64)
	if err != nil {
		return "", false
	}
	ct, err := base64.StdEncoding.DecodeString(ctB64)
	if err != nil {
		return "", false
	}
	plain, err := a.cipher.Open(ct, nonce, settingAAD(key))
	if err != nil {
		return "", false
	}
	return string(plain), true
}

type retentionView struct {
	RawSecs     int `json:"raw_secs"`
	FiveMinSecs int `json:"fivemin_secs"`
	OneHourSecs int `json:"onehour_secs"`
}

type agentUpdateView struct {
	Enabled     bool `json:"enabled"`
	Concurrency int  `json:"concurrency"`
	StallSecs   int  `json:"stall_secs"`
}

type serverUpdateView struct {
	Channel string `json:"channel"`
	// Version is the build running now; UpdatedAt is when it replaced a different
	// one, absent on an instance that has never been updated.
	Version   string `json:"version"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

type settingsView struct {
	SignupEnabled bool             `json:"signup_enabled"`
	Retention     retentionView    `json:"retention"`
	SMTP          smtpView         `json:"smtp"`
	Google        googleAuthView   `json:"google"`
	GitLab        gitlabView       `json:"gitlab"`
	GitHub        githubView       `json:"github"`
	AgentUpdate   agentUpdateView  `json:"agent_update"`
	ServerUpdate  serverUpdateView `json:"server_update"`
}

// handleGetSettings returns every instance-wide setting an admin can change.
func (a *app) handleGetSettings(w http.ResponseWriter, _ *http.Request) {
	ret := a.db.EffectiveRetention()
	au := a.agentUpdateConfig()
	writeJSON(w, http.StatusOK, settingsView{
		SignupEnabled: a.db.GetBoolSetting(settingSignupEnabled, true),
		Retention: retentionView{
			RawSecs:     int(ret.Raw.Seconds()),
			FiveMinSecs: int(ret.FiveMin.Seconds()),
			OneHourSecs: int(ret.OneHour.Seconds()),
		},
		SMTP:        a.smtpView(),
		Google:      a.googleAuthView(),
		GitLab:      a.gitlabView(),
		GitHub:      a.githubView(),
		AgentUpdate:  agentUpdateView{Enabled: au.Enabled, Concurrency: au.Concurrency, StallSecs: au.StallSecs},
		ServerUpdate: a.serverUpdateView(),
	})
}

// handlePutSettings applies a partial settings update; absent fields are left
// as they are so one page section cannot clobber another.
func (a *app) handlePutSettings(w http.ResponseWriter, r *http.Request) {
	var in struct {
		SignupEnabled *bool `json:"signup_enabled"`
		Retention     *struct {
			RawSecs     int `json:"raw_secs"`
			FiveMinSecs int `json:"fivemin_secs"`
			OneHourSecs int `json:"onehour_secs"`
		} `json:"retention"`
		SMTP        *smtpInput   `json:"smtp"`
		Google      *googleInput `json:"google"`
		GitLab      *gitlabInput `json:"gitlab"`
		GitHub      *githubInput `json:"github"`
		AgentUpdate *struct {
			Enabled     bool `json:"enabled"`
			Concurrency int  `json:"concurrency"`
			StallSecs   int  `json:"stall_secs"`
		} `json:"agent_update"`
		ServerUpdate *struct {
			Channel string `json:"channel"`
		} `json:"server_update"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	if in.SignupEnabled != nil {
		if err := a.db.SetSetting(settingSignupEnabled, strconv.FormatBool(*in.SignupEnabled)); err != nil {
			writeError(w, http.StatusInternalServerError, "internal", "could not save signup setting")
			return
		}
	}
	if in.Retention != nil {
		ret := store.Retention{
			Raw:     time.Duration(in.Retention.RawSecs) * time.Second,
			FiveMin: time.Duration(in.Retention.FiveMinSecs) * time.Second,
			OneHour: time.Duration(in.Retention.OneHourSecs) * time.Second,
		}
		if err := validateRetention(ret); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_retention", err.Error())
			return
		}
		for key, secs := range map[string]int{
			"retention.raw_secs":     in.Retention.RawSecs,
			"retention.fivemin_secs": in.Retention.FiveMinSecs,
			"retention.onehour_secs": in.Retention.OneHourSecs,
		} {
			if err := a.db.SetSetting(key, strconv.Itoa(secs)); err != nil {
				writeError(w, http.StatusInternalServerError, "internal", "could not save retention")
				return
			}
		}
	}
	if in.SMTP != nil {
		if err := a.saveSMTPSettings(*in.SMTP); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_smtp", err.Error())
			return
		}
	}
	if in.Google != nil {
		if err := a.saveGoogleSettings(*in.Google); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_google", err.Error())
			return
		}
	}
	if in.GitLab != nil {
		if err := a.saveGitLabSettings(*in.GitLab); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_gitlab", err.Error())
			return
		}
	}
	if in.GitHub != nil {
		if err := a.saveGitHubSettings(*in.GitHub); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_github", err.Error())
			return
		}
	}
	if in.AgentUpdate != nil {
		if err := validateAgentUpdate(in.AgentUpdate.Concurrency, in.AgentUpdate.StallSecs); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_agent_update", err.Error())
			return
		}
		writes := map[string]string{
			settingAgentUpdateEnabled:     strconv.FormatBool(in.AgentUpdate.Enabled),
			settingAgentUpdateConcurrency: strconv.Itoa(in.AgentUpdate.Concurrency),
			settingAgentUpdateStallSecs:   strconv.Itoa(in.AgentUpdate.StallSecs),
		}
		for key, value := range writes {
			if err := a.db.SetSetting(key, value); err != nil {
				writeError(w, http.StatusInternalServerError, "internal", "could not save agent update settings")
				return
			}
		}
	}
	if in.ServerUpdate != nil {
		if channelTags[in.ServerUpdate.Channel] == "" {
			writeError(w, http.StatusBadRequest, "invalid_update_channel",
				"channel must be one of release, develop or main")
			return
		}
		// The file is what the updater acts on, so it is written first: a stored
		// channel the updater never sees would show as switched while the
		// instance kept following the old one.
		if err := a.writeChannelFile(in.ServerUpdate.Channel); err != nil {
			writeError(w, http.StatusInternalServerError, "internal", "could not publish the update channel")
			return
		}
		if err := a.db.SetSetting(settingServerUpdateChannel, in.ServerUpdate.Channel); err != nil {
			writeError(w, http.StatusInternalServerError, "internal", "could not save the update channel")
			return
		}
	}
	a.handleGetSettings(w, r)
}

// validateRetention keeps the tiers ordered and inside sane bounds; an
// out-of-order set would prune a coarser tier before the finer one.
func validateRetention(ret store.Retention) error {
	tiers := []struct {
		name string
		d    time.Duration
	}{{"raw", ret.Raw}, {"5m", ret.FiveMin}, {"1h", ret.OneHour}}
	for _, t := range tiers {
		secs := int(t.d.Seconds())
		if secs < minRetentionSecs || secs > maxRetentionSecs {
			return errors.New(t.name + " retention must be between 600 and 157680000 seconds")
		}
	}
	if !(ret.Raw <= ret.FiveMin && ret.FiveMin <= ret.OneHour) {
		return errors.New("retention must not shrink as resolution coarsens: raw <= 5m <= 1h")
	}
	return nil
}

// validateAgentUpdate keeps a rollout from being configured into a stampede or
// a stall window so short that every host looks wedged.
func validateAgentUpdate(concurrency, stallSecs int) error {
	if concurrency < minAgentUpdateConcurrency || concurrency > maxAgentUpdateConcurrency {
		return errors.New("concurrency must be between 1 and 100")
	}
	if stallSecs < minAgentUpdateStallSecs || stallSecs > maxAgentUpdateStallSecs {
		return errors.New("stall timeout must be between 1 and 86400 seconds")
	}
	return nil
}
