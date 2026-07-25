package store

import "time"

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
