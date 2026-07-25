package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
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

func newPusher(cfg config) *pusher {
	dir := os.Getenv("REEVE_BUFFER_DIR")
	if dir == "" {
		dir = filepath.Join(os.TempDir(), "reeve-agent-buffer")
	}
	os.MkdirAll(dir, 0o700)
	return &pusher{
		serverURL: cfg.ServerURL,
		token:     cfg.Token,
		bufferDir: dir,
		client:    &http.Client{Timeout: 15 * time.Second},
	}
}

// send posts a push, buffering it to disk on failure.
func (p *pusher) send(push contracts.Push) error {
	body, err := json.Marshal(push)
	if err != nil {
		return err
	}
	if err := p.post(body); err != nil {
		p.buffer(body)
		return err
	}
	return nil
}

// flushBuffer replays buffered pushes oldest-first, stopping at the first
// failure so ordering and backoff are preserved.
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
		if err := p.post(body); err != nil {
			return
		}
		os.Remove(path)
	}
}

func (p *pusher) post(body []byte) error {
	req, err := http.NewRequest(http.MethodPost, p.serverURL+"/api/v1/ingest", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.token)
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("ingest returned %d", resp.StatusCode)
	}
	return nil
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
