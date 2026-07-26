package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/thehelvijs/Reeve/contracts"
)

// pusher sends telemetry to the server, buffering to disk when it is
// unreachable and replaying the buffer on the next successful connection.
type pusher struct {
	serverURL string
	token     string
	bufferDir string
	client    *http.Client
}

// defaultBufferDir is where unsent pushes wait. Not /tmp: this process runs as
// root, and a predictable path there is one an unprivileged local user can
// pre-create as a symlink, redirecting root's writes and choosing what the
// replay later sends to the server.
const defaultBufferDir = "/var/lib/reeve-agent/buffer"

func newPusher(cfg config) *pusher {
	dir := os.Getenv("REEVE_BUFFER_DIR")
	if dir == "" {
		dir = defaultBufferDir
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		log.Printf("push buffer: %v; pushes will not survive an outage", err)
	}
	return &pusher{
		serverURL: cfg.ServerURL,
		token:     cfg.Token,
		bufferDir: dir,
		client:    &http.Client{Timeout: 15 * time.Second},
	}
}

// maxAckBytes bounds the ack body so a misbehaving server can't make the agent
// buffer an unbounded reply.
const maxAckBytes = 4096

// statusError is a non-2xx ingest reply. It carries the code so a caller can
// tell a body the server will never accept from one worth retrying.
type statusError struct {
	code int
}

func (e *statusError) Error() string {
	return fmt.Sprintf("ingest returned %d", e.code)
}

// permanentReject reports whether err means the server will refuse this exact
// body however often it is replayed, so keeping it only blocks the queue behind
// it. 408 and 429 ask for a retry, and 401/403 say the caller isn't
// authenticated right now, not that the body is bad, so none of the four are
// permanent.
func permanentReject(err error) bool {
	var se *statusError
	if !errors.As(err, &se) {
		return false
	}
	switch se.code {
	case http.StatusRequestTimeout, http.StatusTooManyRequests,
		http.StatusUnauthorized, http.StatusForbidden:
		return false
	}
	return se.code >= 400 && se.code < 500
}

// send posts a push, buffering it to disk on failure. A nil ack means the
// server answered but said nothing the agent can act on.
func (p *pusher) send(push contracts.Push) (*contracts.PushAck, error) {
	body, err := json.Marshal(push)
	if err != nil {
		return nil, err
	}
	ack, err := p.post(body)
	if err != nil {
		p.buffer(body)
		return nil, err
	}
	return ack, nil
}

// flushBuffer replays buffered pushes oldest-first, stopping at the first
// retryable failure so ordering and backoff are preserved. A body the server
// permanently rejects is dropped instead, or it would wedge the queue behind it
// forever and silently disable offline buffering on this host.
func (p *pusher) flushBuffer() {
	entries, err := os.ReadDir(p.bufferDir)
	if err != nil {
		return
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	for _, name := range names {
		path := filepath.Join(p.bufferDir, name)
		body, err := os.ReadFile(path)
		if err != nil {
			os.Remove(path)
			continue
		}
		if _, err := p.post(body); err != nil {
			if !permanentReject(err) {
				return
			}
			log.Printf("dropping buffered push the server will never accept: %v", err)
		}
		os.Remove(path)
	}
}

func (p *pusher) post(body []byte) (*contracts.PushAck, error) {
	req, err := http.NewRequest(http.MethodPost, p.serverURL+"/api/ingest", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.token)
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, readErr := io.ReadAll(io.LimitReader(resp.Body, maxAckBytes))
	io.Copy(io.Discard, resp.Body) // drain past the cap so the connection can be reused
	if resp.StatusCode >= 300 {
		return nil, &statusError{code: resp.StatusCode}
	}
	if readErr != nil {
		return nil, nil
	}
	var ack contracts.PushAck
	if err := json.Unmarshal(respBody, &ack); err != nil {
		return nil, nil
	}
	return &ack, nil
}

// buffer writes a push to disk, keyed by a monotonic-ish name so replay order
// matches send order.
func (p *pusher) buffer(body []byte) {
	name := fmt.Sprintf("%d-%s.json", time.Now().UTC().UnixNano(), randSuffix())
	os.WriteFile(filepath.Join(p.bufferDir, name), body, 0o600)
}

func randSuffix() string {
	b := make([]byte, 4)
	rand.Read(b)
	return hex.EncodeToString(b)
}
