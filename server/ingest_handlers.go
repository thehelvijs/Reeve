package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/thehelvijs/Reeve/contracts"
	"github.com/thehelvijs/Reeve/server/internal/auth"
)

// handleIngest accepts an agent's telemetry push. It authenticates by the host's
// enrollment token, validates the payload, and never panics on bad input.
func (a *app) handleIngest(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if rec := recover(); rec != nil {
			a.ingestRejected.Add(1)
			log.Printf("ingest: recovered from panic: %v", rec)
			writeError(w, http.StatusBadRequest, "bad_push", "malformed push")
		}
	}()

	hdr := r.Header.Get("Authorization")
	if !strings.HasPrefix(hdr, "Bearer ") {
		a.ingestRejected.Add(1)
		writeError(w, http.StatusUnauthorized, "unauthenticated", "agent token required")
		return
	}
	token := strings.TrimPrefix(hdr, "Bearer ")
	host, err := a.db.GetHostByTokenHash(auth.HashToken(token))
	if err != nil {
		a.ingestRejected.Add(1)
		writeError(w, http.StatusUnauthorized, "unauthenticated", "unknown agent token")
		return
	}

	var push contracts.Push
	dec := json.NewDecoder(io.LimitReader(r.Body, 8<<20))
	if err := dec.Decode(&push); err != nil {
		a.ingestRejected.Add(1)
		writeError(w, http.StatusBadRequest, "bad_push", "invalid push body")
		return
	}
	if push.ProtocolVersion != contracts.PushProtocolVersion {
		a.ingestRejected.Add(1)
		writeError(w, http.StatusBadRequest, "bad_protocol", "unsupported push protocol version")
		return
	}

	if err := a.storePush(host.ID, push); err != nil {
		log.Printf("ingest: store failed for host %s: %v", host.ID, err)
		writeError(w, http.StatusInternalServerError, "internal", "could not store telemetry")
		return
	}
	a.db.TouchHost(host.ID, push.AgentVersion)
	if a.cfg.LogRequests {
		log.Printf("ingest: host=%s agent=%s services=%d containers=%d cron=%d logs=%d cpu=%.0f%% mem=%d/%d",
			host.Name, push.AgentVersion, len(push.Services), len(push.Containers), len(push.CronJobs),
			len(push.LogEvents), push.Metrics.CPUPct, push.Metrics.MemUsed, push.Metrics.MemTotal)
	}
	w.WriteHeader(http.StatusNoContent)
}

// storePush persists the inventory, status, metrics, and log parts of a push.
func (a *app) storePush(hostID string, push contracts.Push) error {
	if err := a.db.ReplaceServiceStatus(hostID, push.Services); err != nil {
		return err
	}
	if err := a.db.ReplaceContainerStatus(hostID, push.Containers); err != nil {
		return err
	}
	if err := a.db.ReplaceCronJobs(hostID, push.CronJobs); err != nil {
		return err
	}
	now := time.Now().UTC()
	if err := a.db.InsertHostMetric(hostID, push.Metrics, now); err != nil {
		return err
	}
	if err := a.db.InsertContainerStats(hostID, push.ContainerStats, now); err != nil {
		return err
	}
	return a.db.InsertLogEvents(hostID, push.LogEvents)
}
