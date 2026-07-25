package main

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/thehelvijs/Reeve/server/internal/mail"
	"github.com/thehelvijs/Reeve/server/internal/rbac"
)

// Settings keys for the outgoing mail relay. The password is the only sealed
// one; the rest are operational config an admin reads back in the UI.
const (
	settingMailEnabled  = "mail.enabled"
	settingMailHost     = "mail.host"
	settingMailPort     = "mail.port"
	settingMailUsername = "mail.username"
	settingMailPassword = "mail.password"
	settingMailFrom     = "mail.from"
	settingMailTLS      = "mail.tls"
)

type smtpView struct {
	Enabled     bool   `json:"enabled"`
	Host        string `json:"host"`
	Port        int    `json:"port"`
	Username    string `json:"username"`
	From        string `json:"from"`
	TLS         string `json:"tls"`
	PasswordSet bool   `json:"password_set"`
}

// smtpInput is the write shape. Password is write-only: an empty string leaves
// the stored one alone so the form never has to round-trip a secret.
type smtpInput struct {
	Enabled  bool   `json:"enabled"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	From     string `json:"from"`
	TLS      string `json:"tls"`
}

func (a *app) smtpView() smtpView {
	_, hasPassword := a.sealedSetting(settingMailPassword)
	port, _ := strconv.Atoi(a.settingOr(settingMailPort, "587"))
	return smtpView{
		Enabled:     a.db.GetBoolSetting(settingMailEnabled, false),
		Host:        a.settingOr(settingMailHost, ""),
		Port:        port,
		Username:    a.settingOr(settingMailUsername, ""),
		From:        a.settingOr(settingMailFrom, ""),
		TLS:         a.settingOr(settingMailTLS, mail.TLSStartTLS),
		PasswordSet: hasPassword,
	}
}

// settingOr reads a plain setting with a fallback.
func (a *app) settingOr(key, def string) string {
	if v, ok := a.db.GetSetting(key); ok && v != "" {
		return v
	}
	return def
}

// saveSMTPSettings validates and persists the relay config. An enabled relay
// must be complete, so a half-filled form cannot silently break password
// resets later.
func (a *app) saveSMTPSettings(in smtpInput) error {
	in.Host = strings.TrimSpace(in.Host)
	in.From = strings.TrimSpace(in.From)
	in.Username = strings.TrimSpace(in.Username)
	if in.TLS == "" {
		in.TLS = mail.TLSStartTLS
	}
	if in.Enabled {
		cfg := mail.Config{Host: in.Host, Port: in.Port, Username: in.Username, From: in.From, TLS: in.TLS}
		if err := cfg.Validate(); err != nil {
			return err
		}
	}
	writes := map[string]string{
		settingMailEnabled:  strconv.FormatBool(in.Enabled),
		settingMailHost:     in.Host,
		settingMailPort:     strconv.Itoa(in.Port),
		settingMailUsername: in.Username,
		settingMailFrom:     in.From,
		settingMailTLS:      in.TLS,
	}
	for k, v := range writes {
		if err := a.db.SetSetting(k, v); err != nil {
			return err
		}
	}
	if in.Password != "" {
		return a.setSealedSetting(settingMailPassword, in.Password)
	}
	return nil
}

// mailConfig assembles the relay config, reporting false when mail is off or
// incomplete. Every mail caller goes through this, so a broken relay degrades
// to "no email" instead of a runtime error.
func (a *app) mailConfig() (mail.Config, bool) {
	if !a.db.GetBoolSetting(settingMailEnabled, false) {
		return mail.Config{}, false
	}
	v := a.smtpView()
	password, _ := a.sealedSetting(settingMailPassword)
	cfg := mail.Config{Host: v.Host, Port: v.Port, Username: v.Username, Password: password, From: v.From, TLS: v.TLS}
	if cfg.Validate() != nil {
		return mail.Config{}, false
	}
	return cfg, true
}

// sendMail delivers one message if a relay is configured.
func (a *app) sendMail(m mail.Message) error {
	cfg, ok := a.mailConfig()
	if !ok {
		return errors.New("no mail relay is configured")
	}
	return mail.Send(cfg, m, time.Now().UTC())
}

// handleTestEmail sends a probe to the calling admin so a misconfigured relay
// surfaces at setup time rather than the first time someone is locked out.
func (a *app) handleTestEmail(w http.ResponseWriter, r *http.Request) {
	p, _ := rbac.FromContext(r.Context())
	u, err := a.db.GetUserByID(p.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not load account")
		return
	}
	msg := mail.Message{
		To:      u.Email,
		Subject: "Reeve test email",
		Body:    "This is a test message from Reeve. Your mail relay works.\n",
	}
	if err := a.sendMail(msg); err != nil {
		writeError(w, http.StatusBadGateway, "mail_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"sent_to": u.Email})
}
