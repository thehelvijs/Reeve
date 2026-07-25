// Package mail sends outgoing email through an operator-configured SMTP relay.
// It owns message construction and the SMTP conversation; callers hand it a
// config and a message and never touch net/smtp themselves.
package mail

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"
)

// TLS modes an operator can pick for the relay.
const (
	TLSStartTLS = "starttls"
	TLSImplicit = "tls"
	TLSNone     = "none"
)

// dialTimeout bounds the whole SMTP conversation so a wedged relay cannot hold
// a request handler open.
const dialTimeout = 20 * time.Second

// Config describes the relay. Password is plaintext in memory only; at rest it
// is sealed with the master key by the caller.
type Config struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	TLS      string
}

// Message is one outgoing email.
type Message struct {
	To      string
	Subject string
	Body    string
}

// Validate reports whether the config is complete and internally consistent.
func (c Config) Validate() error {
	if strings.TrimSpace(c.Host) == "" {
		return errors.New("smtp host is required")
	}
	if c.Port <= 0 || c.Port > 65535 {
		return errors.New("smtp port must be between 1 and 65535")
	}
	if strings.TrimSpace(c.From) == "" {
		return errors.New("from address is required")
	}
	if !strings.Contains(c.From, "@") {
		return errors.New("from address must be an email address")
	}
	switch c.TLS {
	case TLSStartTLS, TLSImplicit, TLSNone:
	default:
		return fmt.Errorf("tls mode must be %q, %q, or %q", TLSStartTLS, TLSImplicit, TLSNone)
	}
	if hasCRLF(c.From) || hasCRLF(c.Username) {
		return errors.New("from and username must not contain line breaks")
	}
	return nil
}

// hasCRLF reports whether s carries a bare CR or LF, which would let a caller
// inject extra SMTP or header lines.
func hasCRLF(s string) bool {
	return strings.ContainsAny(s, "\r\n")
}

// BuildMessage renders m as an RFC 5322 message with CRLF line endings.
func BuildMessage(from string, m Message, now time.Time) ([]byte, error) {
	if hasCRLF(m.To) || hasCRLF(m.Subject) {
		return nil, errors.New("recipient and subject must not contain line breaks")
	}
	if !strings.Contains(m.To, "@") {
		return nil, errors.New("recipient must be an email address")
	}
	var b strings.Builder
	writeHeader := func(k, v string) {
		b.WriteString(k)
		b.WriteString(": ")
		b.WriteString(v)
		b.WriteString("\r\n")
	}
	writeHeader("From", from)
	writeHeader("To", m.To)
	writeHeader("Subject", m.Subject)
	writeHeader("Date", now.Format(time.RFC1123Z))
	writeHeader("MIME-Version", "1.0")
	writeHeader("Content-Type", "text/plain; charset=utf-8")
	b.WriteString("\r\n")
	// Bare LFs in the body would end the DATA stanza early on a strict relay.
	b.WriteString(strings.ReplaceAll(strings.ReplaceAll(m.Body, "\r\n", "\n"), "\n", "\r\n"))
	return []byte(b.String()), nil
}

// Send delivers one message through the configured relay.
func Send(c Config, m Message, now time.Time) error {
	if err := c.Validate(); err != nil {
		return err
	}
	body, err := BuildMessage(c.From, m, now)
	if err != nil {
		return err
	}
	addr := net.JoinHostPort(c.Host, fmt.Sprint(c.Port))

	var conn net.Conn
	if c.TLS == TLSImplicit {
		conn, err = tls.DialWithDialer(&net.Dialer{Timeout: dialTimeout}, "tcp", addr, &tls.Config{ServerName: c.Host})
	} else {
		conn, err = net.DialTimeout("tcp", addr, dialTimeout)
	}
	if err != nil {
		return fmt.Errorf("connect to %s: %w", addr, err)
	}
	defer conn.Close()
	conn.SetDeadline(now.Add(dialTimeout))

	client, err := smtp.NewClient(conn, c.Host)
	if err != nil {
		return fmt.Errorf("smtp handshake: %w", err)
	}
	defer client.Close()

	if c.TLS == TLSStartTLS {
		if ok, _ := client.Extension("STARTTLS"); !ok {
			return errors.New("relay does not offer STARTTLS; pick another TLS mode")
		}
		if err := client.StartTLS(&tls.Config{ServerName: c.Host}); err != nil {
			return fmt.Errorf("starttls: %w", err)
		}
	}
	if c.Username != "" {
		if err := client.Auth(smtp.PlainAuth("", c.Username, c.Password, c.Host)); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}
	if err := client.Mail(c.From); err != nil {
		return fmt.Errorf("mail from: %w", err)
	}
	if err := client.Rcpt(m.To); err != nil {
		return fmt.Errorf("rcpt to: %w", err)
	}
	wc, err := client.Data()
	if err != nil {
		return fmt.Errorf("data: %w", err)
	}
	if _, err := wc.Write(body); err != nil {
		return fmt.Errorf("write body: %w", err)
	}
	if err := wc.Close(); err != nil {
		return fmt.Errorf("finish body: %w", err)
	}
	return client.Quit()
}
