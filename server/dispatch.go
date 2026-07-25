package main

import (
	"encoding/json"
	"time"
)

// dispatchDue attempts every delivery that is due, routing each to the notifier
// for its channel kind and marking it sent or failed (with exponential
// backoff). Returns how many were attempted. `now` injected for tests.
func (a *app) dispatchDue(now time.Time) int {
	if a.notifiers == nil {
		a.notifiers = newNotifiers()
	}
	deliveries, err := a.db.DueDeliveries(now, 50)
	if err != nil {
		return 0
	}
	for _, d := range deliveries {
		cfg, cfgErr := a.openConfig(d.Config)
		if cfgErr != nil {
			a.db.MarkDeliveryFailed(d.ID, "channel config unreadable", d.Attempts, now)
			continue
		}
		n := a.notifiers[d.Kind]
		if n == nil {
			if d.URL == "" {
				a.db.MarkDeliveryFailed(d.ID, "unknown channel kind", d.Attempts, now)
				continue
			}
			n = a.notifiers["generic"]
		}
		err := n.Send(notifyChannel{URL: d.URL, Config: cfg}, d.Payload, now)
		if err == nil {
			a.db.MarkDeliverySent(d.ID)
			continue
		}
		a.db.MarkDeliveryFailed(d.ID, err.Error(), d.Attempts, now)
	}
	return len(deliveries)
}

// notifyToolEvent enqueues an informational payload (e.g. access request) to a
// tool's resolved channels. Informational events route at info severity.
// Best-effort; delivery is handled by the dispatcher.
func (a *app) notifyToolEvent(toolID string, payload map[string]any, now time.Time) {
	hooks, err := a.db.ResolveChannelsForTool(toolID, "info")
	if err != nil {
		return
	}
	body, _ := json.Marshal(payload)
	for _, h := range hooks {
		a.db.EnqueueDelivery(h.ID, string(body), now)
	}
}
