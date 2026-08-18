package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// settingHeartbeatURL is a push URL for an external dead-man's switch
// (healthchecks.io and friends). Empty means no heartbeat.
const settingHeartbeatURL = "heartbeat.url"

// heartbeatEvery is fixed: the alerting window is the receiver's setting, not
// this server's, and one ping a minute is what lets it be a short one.
const heartbeatEvery = time.Minute

// runHeartbeatLoop pings the configured URL while one is set. There is nothing
// to escalate when a ping fails — the silence is the alert, and it fires from
// the other end.
func (a *app) runHeartbeatLoop(ctx context.Context) {
	every(ctx, heartbeatEvery, func() {
		url, _ := a.db.GetSetting(settingHeartbeatURL)
		if url == "" {
			return
		}
		if err := pingHeartbeat(url); err != nil {
			log.Printf("heartbeat: %v", err)
		}
	})
}

// pingHeartbeat GETs the URL, treating any non-2xx as a failure to report.
func pingHeartbeat(url string) error {
	resp, err := webhookClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("heartbeat url returned %d", resp.StatusCode)
	}
	return nil
}

// handleTestHeartbeat pings now, so a wrong URL surfaces while the admin is
// looking at the field rather than as an alert that never arrives.
func (a *app) handleTestHeartbeat(w http.ResponseWriter, _ *http.Request) {
	url, _ := a.db.GetSetting(settingHeartbeatURL)
	if url == "" {
		writeError(w, http.StatusBadRequest, "no_heartbeat_url", "save a heartbeat URL first")
		return
	}
	if err := pingHeartbeat(url); err != nil {
		writeError(w, http.StatusBadGateway, "heartbeat_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"pinged": url})
}
