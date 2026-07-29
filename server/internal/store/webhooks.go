package store

import (
	"strings"
	"time"
)

// Webhook is a notification channel. Config carries the transport's settings,
// stored as an opaque blob (encrypted at rest by the app); the store never
// decrypts it. MinSeverity gates how loud an alert must be, and Events which
// kinds it may be, before the channel receives it.
type Webhook struct {
	ID          string `json:"id"`
	OwnerType   string `json:"owner_type"`
	OwnerID     string `json:"owner_id"`
	URL         string `json:"url"`
	Enabled     bool   `json:"enabled"`
	Format      string `json:"format"`
	Config      string `json:"-"`
	MinSeverity string `json:"min_severity"`
	// Events is the comma-separated list of event types this channel wants.
	// Empty is every type, which is what a channel nobody has narrowed means.
	Events string `json:"-"`
}

// Channel is the channel-oriented alias for a Webhook row.
type Channel = Webhook

const webhookCols = `id, owner_type, owner_id, url, enabled, format, config, min_severity, events`

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
// config to '{}', and an empty min_severity to info. Scope, config and the event
// list arrive as the caller built them; the store persists them verbatim and
// never decrypts the config.
func (db *DB) CreateWebhook(w Webhook) (Webhook, error) {
	if w.Format == "" {
		w.Format = "generic"
	}
	if w.Config == "" {
		w.Config = "{}"
	}
	if w.MinSeverity == "" {
		w.MinSeverity = "info"
	}
	w.ID = NewID()
	w.Enabled = true
	_, err := db.sql.Exec(
		`INSERT INTO webhooks(id, owner_type, owner_id, url, enabled, format, config, min_severity, events, created_at)
		 VALUES (?,?,?,?,1,?,?,?,?,?)`,
		w.ID, w.OwnerType, w.OwnerID, w.URL, w.Format, w.Config, w.MinSeverity, w.Events,
		time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return Webhook{}, err
	}
	return w, nil
}

// UpdateWebhook replaces the editable fields of a channel: where it posts, the
// receiver shape, the gates and the sealed config that carries a token and a
// custom template. Scope is not one of them, because a channel that should
// watch something else is a different channel. config is already sealed.
func (db *DB) UpdateWebhook(w Webhook) error {
	return db.exec1(
		`UPDATE webhooks SET url=?, format=?, config=?, min_severity=?, events=?, enabled=? WHERE id = ?`,
		w.URL, w.Format, w.Config, w.MinSeverity, w.Events, boolToInt(w.Enabled), w.ID)
}

// ListWebhooks returns all channels.
func (db *DB) ListWebhooks() ([]Webhook, error) {
	return db.queryWebhooks(`SELECT ` + webhookCols + ` FROM webhooks ORDER BY owner_type, created_at`)
}

// GetWebhook returns one channel by id, or ErrNotFound.
func (db *DB) GetWebhook(id string) (Webhook, error) {
	hooks, err := db.queryWebhooks(`SELECT `+webhookCols+` FROM webhooks WHERE id = ?`, id)
	if err != nil {
		return Webhook{}, err
	}
	if len(hooks) == 0 {
		return Webhook{}, ErrNotFound
	}
	return hooks[0], nil
}

// DeleteWebhook removes a channel.
func (db *DB) DeleteWebhook(id string) error {
	return db.exec1(`DELETE FROM webhooks WHERE id = ?`, id)
}

// ChannelSubject is one event as the routing sees it: what it is about, what
// kind it is and how loud. Every field narrows which channels receive it.
type ChannelSubject struct {
	ToolID   string
	HostID   string
	Event    string
	Severity string
}

// channelWants reports whether a channel subscribed to these event types wants
// this one. An empty subscription is every type. An event with no type of its
// own (the setup probe) is never withheld, because it is testing the wire.
func channelWants(events, event string) bool {
	if events == "" || event == "" {
		return true
	}
	for _, e := range strings.Split(events, ",") {
		if e == event {
			return true
		}
	}
	return false
}

// ResolveChannels returns the enabled channels that cover one event: every
// global channel, the ones scoped to the tool it names or the host it happened
// on, and the group ones whose members can reach either. A channel is listed
// once, because a channel has one scope.
//
// Scope is the whole rule. A global channel receives everything, so an operator
// who wants one receiver for one host adds a host channel and leaves global for
// the instance as a whole, rather than discovering that the first narrow channel
// silently took traffic away from a broad one.
//
// Credential access is held against a host, not a tool, so a group is reached
// two ways: the groups a tool is visible to, and the groups that can get into
// the machine. A tool alert carries its host, so both branches see it.
func (db *DB) ResolveChannels(s ChannelSubject) ([]Webhook, error) {
	q := `
		SELECT ` + webhookCols + ` FROM webhooks
		WHERE enabled=1 AND (
			owner_type='global'
			OR (owner_type='tool' AND ? <> '' AND owner_id = ?)
			OR (owner_type='host' AND ? <> '' AND owner_id = ?)
			OR (owner_type='group' AND owner_id IN (
				SELECT principal_id FROM tool_visibility WHERE tool_id = ? AND principal_type='group'
				UNION
				SELECT principal_id FROM credential_access
				WHERE principal_type='group' AND host_id = ?
			))
		)
		ORDER BY owner_type, created_at`
	hooks, err := db.queryWebhooks(q, s.ToolID, s.ToolID, s.HostID, s.HostID, s.ToolID, s.HostID)
	if err != nil {
		return nil, err
	}
	out := make([]Webhook, 0, len(hooks))
	for _, h := range hooks {
		if channelAccepts(h.MinSeverity, s.Severity) && channelWants(h.Events, s.Event) {
			out = append(out, h)
		}
	}
	return out, nil
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
		if err := rows.Scan(&w.ID, &w.OwnerType, &w.OwnerID, &w.URL, &enabled, &w.Format, &w.Config, &w.MinSeverity, &w.Events); err != nil {
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
