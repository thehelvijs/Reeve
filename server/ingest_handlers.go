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
	if section := push.TooLarge(); section != "" {
		a.ingestRejected.Add(1)
		log.Printf("ingest: host %s sent an oversized %s section", host.ID, section)
		writeError(w, http.StatusRequestEntityTooLarge, "push_too_large", "the "+section+" section exceeds the per-push limit")
		return
	}

	// Naming happens before storing, so every reader downstream — inventory,
	// charts, the catalog — sees one name for a container rather than each
	// deciding for itself.
	nameCoolifyContainers(push.Containers, a.coolifyResourceNames(r.Context()))

	if err := a.storePush(host.ID, push); err != nil {
		log.Printf("ingest: store failed for host %s: %v", host.ID, err)
		writeError(w, http.StatusInternalServerError, "internal", "could not store telemetry")
		return
	}
	// Results first, then hand out new work: a command answered in this push
	// must be closed before the same tick could collect anything else.
	for _, res := range push.CommandResults {
		if err := a.db.ApplyCommandResult(host.ID, res); err != nil {
			log.Printf("ingest: applying command result for host %s: %v", host.ID, err)
		}
	}
	ack := contracts.PushAck{
		CheckNow: a.decideCheckNow(host, push.AgentChecksum, push.AutoUpdateVetoed, time.Now().UTC()),
	}
	if push.ControlEnabled {
		cmds, err := a.db.ClaimPendingCommands(host.ID, contracts.MaxAckCommands, time.Now().UTC())
		if err != nil {
			log.Printf("ingest: claiming commands for host %s: %v", host.ID, err)
		}
		ack.Commands = cmds
	}
	if a.cfg.LogRequests {
		log.Printf("ingest: host=%s agent=%s check_now=%v services=%d containers=%d cron=%d logs=%d cpu=%.0f%% mem=%d/%d",
			host.Name, push.AgentVersion, ack.CheckNow, len(push.Services), len(push.Containers), len(push.CronJobs),
			len(push.LogEvents), push.Metrics.CPUPct, push.Metrics.MemUsed, push.Metrics.MemTotal)
	}
	writeJSON(w, http.StatusOK, ack)
}

// storePush persists a push and the host's heartbeat in one transaction.
func (a *app) storePush(hostID string, push contracts.Push) error {
	return a.db.ApplyPush(hostID, push, time.Now().UTC())
}
