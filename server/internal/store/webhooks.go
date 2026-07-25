package store

import "time"

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
func (db *DB) ResolveWebhooksForTool(toolID string) ([]Webhook, error) {
	q := `
		SELECT ` + webhookCols + ` FROM webhooks
		WHERE enabled=1 AND (
			(owner_type='tool' AND owner_id = ?)
			OR (owner_type='group' AND owner_id IN (
				SELECT principal_id FROM tool_visibility WHERE tool_id = ? AND principal_type='group'
				UNION
				SELECT principal_id FROM credential_access WHERE tool_id = ? AND principal_type='group'
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
