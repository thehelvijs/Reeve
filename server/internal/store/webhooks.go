package store

import (
	"time"
)

// Webhook is a notification channel. Config carries the transport's settings,
// stored as an opaque blob (encrypted at rest by the app); the store never
// decrypts it. MinSeverity gates which alerts the channel receives.
type Webhook struct {
	ID          string `json:"id"`
	OwnerType   string `json:"owner_type"`
	OwnerID     string `json:"owner_id"`
	URL         string `json:"url"`
	Enabled     bool   `json:"enabled"`
	Format      string `json:"format"`
	Config      string `json:"-"`
	MinSeverity string `json:"min_severity"`
}

// Channel is the channel-oriented alias for a Webhook row.
type Channel = Webhook

const webhookCols = `id, owner_type, owner_id, url, enabled, format, config, min_severity`

// severityRank orders alert severities so a channel's min_severity can gate
// delivery. Unknown values rank as info so nothing is silently dropped.
func severityRank(s string) int {
	switch s {
	case "warning":
		return 1
	case "error":
		return 2
	default:
		return 0
	}
}

// channelAccepts reports whether a channel with the given min_severity should
// receive an event of the given severity.
func channelAccepts(minSeverity, eventSeverity string) bool {
	return severityRank(eventSeverity) >= severityRank(minSeverity)
}

// CreateWebhook stores a channel. An empty format defaults to generic, an empty
// config to '{}', and an empty min_severity to info. config is already sealed by
// the caller; the store persists it verbatim.
func (db *DB) CreateWebhook(ownerType, ownerID, url, format, config, minSeverity string) (Webhook, error) {
	if format == "" {
		format = "generic"
	}
	if config == "" {
		config = "{}"
	}
	if minSeverity == "" {
		minSeverity = "info"
	}
	w := Webhook{ID: NewID(), OwnerType: ownerType, OwnerID: ownerID, URL: url, Enabled: true, Format: format, Config: config, MinSeverity: minSeverity}
	_, err := db.sql.Exec(
		`INSERT INTO webhooks(id, owner_type, owner_id, url, enabled, format, config, min_severity, created_at) VALUES (?,?,?,?,1,?,?,?,?)`,
		w.ID, w.OwnerType, w.OwnerID, w.URL, w.Format, w.Config, w.MinSeverity, time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return Webhook{}, err
	}
	return w, nil
}

// ListWebhooks returns all channels.
func (db *DB) ListWebhooks() ([]Webhook, error) {
	return db.queryWebhooks(`SELECT ` + webhookCols + ` FROM webhooks ORDER BY owner_type, created_at`)
}

// DeleteWebhook removes a channel.
func (db *DB) DeleteWebhook(id string) error {
	return db.exec1(`DELETE FROM webhooks WHERE id = ?`, id)
}

// GlobalWebhooks returns enabled global channels.
func (db *DB) GlobalWebhooks() ([]Webhook, error) {
	return db.queryWebhooks(`SELECT ` + webhookCols + ` FROM webhooks WHERE owner_type='global' AND enabled=1`)
}

// GlobalChannels returns enabled global channels that accept the given severity.
func (db *DB) GlobalChannels(severity string) ([]Webhook, error) {
	hooks, err := db.GlobalWebhooks()
	if err != nil {
		return nil, err
	}
	return filterBySeverity(hooks, severity), nil
}

// ResolveWebhooksForTool returns the channels to notify for a tool alert:
// tool-specific plus any group (visibility or access) channels; if none, the
// global channels are the fallback.
//
// Credential access is held against a host, not a tool, so the second branch
// reaches it through the tool's host: the groups that can get into the machine
// this tool runs on. A tool with no host matches nothing there.
func (db *DB) ResolveWebhooksForTool(toolID string) ([]Webhook, error) {
	q := `
		SELECT ` + webhookCols + ` FROM webhooks
		WHERE enabled=1 AND (
			(owner_type='tool' AND owner_id = ?)
			OR (owner_type='group' AND owner_id IN (
				SELECT principal_id FROM tool_visibility WHERE tool_id = ? AND principal_type='group'
				UNION
				SELECT principal_id FROM credential_access
				WHERE principal_type='group'
				  AND host_id = (SELECT host_id FROM tools WHERE id = ?)
			))
		)`
	hooks, err := db.queryWebhooks(q, toolID, toolID, toolID)
	if err != nil {
		return nil, err
	}
	if len(hooks) == 0 {
		return db.GlobalWebhooks()
	}
	return hooks, nil
}

// ResolveChannelsForTool is ResolveWebhooksForTool filtered by min_severity.
func (db *DB) ResolveChannelsForTool(toolID, severity string) ([]Webhook, error) {
	hooks, err := db.ResolveWebhooksForTool(toolID)
	if err != nil {
		return nil, err
	}
	return filterBySeverity(hooks, severity), nil
}

func filterBySeverity(hooks []Webhook, severity string) []Webhook {
	out := make([]Webhook, 0, len(hooks))
	for _, h := range hooks {
		if channelAccepts(h.MinSeverity, severity) {
			out = append(out, h)
		}
	}
	return out
}

func (db *DB) queryWebhooks(q string, args ...any) ([]Webhook, error) {
	rows, err := db.sql.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Webhook
	for rows.Next() {
		var w Webhook
		var enabled int
		if err := rows.Scan(&w.ID, &w.OwnerType, &w.OwnerID, &w.URL, &enabled, &w.Format, &w.Config, &w.MinSeverity); err != nil {
			return nil, err
		}
		w.Enabled = enabled == 1
		if w.Format == "" {
			w.Format = "generic"
		}
		if w.MinSeverity == "" {
			w.MinSeverity = "info"
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

// MaxDeliveryAttempts caps webhook retries before a delivery is marked dead.
const MaxDeliveryAttempts = 6

// Delivery is a queued webhook POST. Kind + Config are the channel's transport
// selector and opaque (still-sealed) config.
type Delivery struct {
	ID        string `json:"id"`
	WebhookID string `json:"webhook_id"`
	URL       string `json:"-"`
	Payload   string `json:"-"`
	Status    string `json:"status"`
	Attempts  int    `json:"attempts"`
	LastError string `json:"last_error"`
	Kind      string `json:"-"`
	Config    string `json:"-"`
}

// EnqueueDelivery queues a payload for a webhook, due immediately.
func (db *DB) EnqueueDelivery(webhookID, payload string, now time.Time) error {
	t := now.UTC().Format(time.RFC3339Nano)
	_, err := db.sql.Exec(
		`INSERT INTO webhook_deliveries(id, webhook_id, payload, status, next_attempt_at, created_at)
		 VALUES (?,?,?, 'pending', ?, ?)`,
		NewID(), webhookID, payload, t, t)
	return err
}

// DueDeliveries returns deliveries ready to attempt, resolving the target
// webhook URL + channel config.
func (db *DB) DueDeliveries(now time.Time, limit int) ([]Delivery, error) {
	rows, err := db.sql.Query(
		`SELECT d.id, COALESCE(d.webhook_id,''),
		        COALESCE(w.url,''), COALESCE(w.format,''), COALESCE(w.config,''),
		        d.payload, d.status, d.attempts, d.last_error
		 FROM webhook_deliveries d
		 JOIN webhooks w ON w.id = d.webhook_id
		 WHERE d.status IN ('pending','failed') AND d.next_attempt_at <= ? AND d.attempts < ?
		 ORDER BY d.next_attempt_at LIMIT ?`,
		now.UTC().Format(time.RFC3339Nano), MaxDeliveryAttempts, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Delivery
	for rows.Next() {
		var d Delivery
		if err := rows.Scan(&d.ID, &d.WebhookID,
			&d.URL, &d.Kind, &d.Config,
			&d.Payload, &d.Status, &d.Attempts, &d.LastError); err != nil {
			return nil, err
		}
		if d.Kind == "" {
			d.Kind = "generic"
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// MarkDeliverySent marks a delivery successful.
func (db *DB) MarkDeliverySent(id string) error {
	_, err := db.sql.Exec(
		`UPDATE webhook_deliveries SET status='sent', attempts=attempts+1, last_error='' WHERE id = ?`, id)
	return err
}

// MarkDeliveryFailed records a failed attempt and schedules the next one, or
// marks the delivery dead once attempts are exhausted.
func (db *DB) MarkDeliveryFailed(id, errMsg string, attempts int, now time.Time) error {
	status := "failed"
	backoff := time.Duration(1<<uint(attempts)) * time.Second
	if attempts+1 >= MaxDeliveryAttempts {
		status = "dead"
	}
	next := now.Add(backoff).UTC().Format(time.RFC3339Nano)
	_, err := db.sql.Exec(
		`UPDATE webhook_deliveries SET status=?, attempts=attempts+1, last_error=?, next_attempt_at=? WHERE id = ?`,
		status, errMsg, next, id)
	return err
}

// ListDeliveries returns recent deliveries for the delivery log.
func (db *DB) ListDeliveries(limit int) ([]Delivery, error) {
	rows, err := db.sql.Query(
		`SELECT id, COALESCE(webhook_id,''), status, attempts, last_error
		 FROM webhook_deliveries ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Delivery
	for rows.Next() {
		var d Delivery
		if err := rows.Scan(&d.ID, &d.WebhookID, &d.Status, &d.Attempts, &d.LastError); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}
