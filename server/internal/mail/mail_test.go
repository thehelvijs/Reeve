package mail

import (
	"bufio"
	"net"
	"strings"
	"testing"
	"time"
)

var testNow = time.Date(2026, 7, 25, 10, 30, 0, 0, time.UTC)

func validConfig() Config {
	return Config{Host: "127.0.0.1", Port: 25, From: "reeve@example.com", TLS: TLSNone}
}

func TestConfigValidate(t *testing.T) {
	cases := map[string]struct {
		mutate func(*Config)
		wantOK bool
	}{
		"valid":             {func(*Config) {}, true},
		"no host":           {func(c *Config) { c.Host = "" }, false},
		"port zero":         {func(c *Config) { c.Port = 0 }, false},
		"port too high":     {func(c *Config) { c.Port = 70000 }, false},
		"no from":           {func(c *Config) { c.From = "" }, false},
		"from not an email": {func(c *Config) { c.From = "reeve" }, false},
		"bad tls mode":      {func(c *Config) { c.TLS = "ssl" }, false},
		"newline in from":   {func(c *Config) { c.From = "a@b.com\r\nBcc: c@d.com" }, false},
	}
	for name, tc := range cases {
		c := validConfig()
		tc.mutate(&c)
		err := c.Validate()
		if tc.wantOK && err != nil {
			t.Errorf("%s: Validate = %v, want nil", name, err)
		}
		if !tc.wantOK && err == nil {
			t.Errorf("%s: Validate = nil, want an error", name)
		}
	}
}

func TestBuildMessageHeadersAndCRLF(t *testing.T) {
	body, err := BuildMessage("reeve@example.com", Message{
		To:      "dev@example.com",
		Subject: "Reset your password",
		Body:    "line one\nline two",
	}, testNow)
	if err != nil {
		t.Fatalf("BuildMessage: %v", err)
	}
	got := string(body)
	for _, want := range []string{
		"From: reeve@example.com\r\n",
		"To: dev@example.com\r\n",
		"Subject: Reset your password\r\n",
		"Content-Type: text/plain; charset=utf-8\r\n",
		"\r\n\r\nline one\r\nline two",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("message missing %q\n---\n%s", want, got)
		}
	}
	if strings.Contains(strings.ReplaceAll(got, "\r\n", ""), "\n") {
		t.Error("message contains a bare LF")
	}
}

// A subject or recipient carrying CRLF would let a caller add headers such as
// Bcc, so message construction has to refuse it.
func TestBuildMessageRejectsHeaderInjection(t *testing.T) {
	cases := map[string]Message{
		"subject":  {To: "dev@example.com", Subject: "hi\r\nBcc: leak@evil.com"},
		"to":       {To: "dev@example.com\r\nBcc: leak@evil.com", Subject: "hi"},
		"not mail": {To: "nonsense", Subject: "hi"},
	}
	for name, m := range cases {
		if _, err := BuildMessage("reeve@example.com", m, testNow); err == nil {
			t.Errorf("%s: BuildMessage = nil error, want a rejection", name)
		}
	}
}

// stubSMTP speaks just enough SMTP to accept one message, and records it.
type stubSMTP struct {
	addr     string
	received chan string
}

func newStubSMTP(t *testing.T) *stubSMTP {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	s := &stubSMTP{addr: ln.Addr().String(), received: make(chan string, 1)}
	t.Cleanup(func() { ln.Close() })
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		r := bufio.NewReader(conn)
		w := bufio.NewWriter(conn)
		reply := func(line string) {
			w.WriteString(line + "\r\n")
			w.Flush()
		}
		reply("220 stub ESMTP")
		var data strings.Builder
		inData := false
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				return
			}
			line = strings.TrimRight(line, "\r\n")
			if inData {
				if line == "." {
					inData = false
					s.received <- data.String()
					reply("250 queued")
					continue
				}
				data.WriteString(line + "\n")
				continue
			}
			switch {
			case strings.HasPrefix(line, "EHLO"), strings.HasPrefix(line, "HELO"):
				reply("250-stub")
				reply("250 SIZE 10240000")
			case strings.HasPrefix(line, "MAIL FROM"), strings.HasPrefix(line, "RCPT TO"):
				reply("250 ok")
			case line == "DATA":
				inData = true
				reply("354 send it")
			case line == "QUIT":
				reply("221 bye")
				return
			default:
				reply("250 ok")
			}
		}
	}()
	return s
}

func TestSendDeliversToRelay(t *testing.T) {
	stub := newStubSMTP(t)
	host, port, _ := net.SplitHostPort(stub.addr)
	c := Config{Host: host, Port: atoi(t, port), From: "reeve@example.com", TLS: TLSNone}

	if err := Send(c, Message{To: "dev@example.com", Subject: "Ping", Body: "hello"}, time.Now()); err != nil {
		t.Fatalf("Send: %v", err)
	}
	select {
	case got := <-stub.received:
		for _, want := range []string{"To: dev@example.com", "Subject: Ping", "hello"} {
			if !strings.Contains(got, want) {
				t.Errorf("relay never saw %q\n---\n%s", want, got)
			}
		}
	case <-time.After(5 * time.Second):
		t.Fatal("relay received nothing")
	}
}

func TestSendFailsOnUnreachableRelay(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := ln.Addr().String()
	ln.Close()
	host, port, _ := net.SplitHostPort(addr)
	c := Config{Host: host, Port: atoi(t, port), From: "reeve@example.com", TLS: TLSNone}
	if err := Send(c, Message{To: "dev@example.com", Subject: "Ping", Body: "hi"}, time.Now()); err == nil {
		t.Fatal("Send to a closed port returned nil, want an error")
	}
}

func TestSendValidatesBeforeDialing(t *testing.T) {
	if err := Send(Config{}, Message{To: "dev@example.com"}, testNow); err == nil {
		t.Fatal("Send with an empty config returned nil, want a validation error")
	}
}

func atoi(t *testing.T, s string) int {
	t.Helper()
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			t.Fatalf("port %q is not numeric", s)
		}
		n = n*10 + int(r-'0')
	}
	return n
}
