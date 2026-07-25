package store

import (
	"database/sql"
	"strings"
	"time"
)

// AlertState is the per-subject state machine row for transition detection and
// debounce.
type AlertState struct {
	SubjectKey   string
	State        string // ok | pending | firing
	PendingSince *time.Time
	OpenEventID  string
}

// AlertEvent is a fired alert (open until resolved_at is set).
type AlertEvent struct {
	ID         string     `json:"id"`
	SubjectKey string     `json:"subject_key"`
	ToolID     string     `json:"tool_id,omitempty"`
	HostID     string     `json:"host_id,omitempty"`
	Type       string     `json:"type"`
	Severity   string     `json:"severity"`
	Message    string     `json:"message"`
	FiredAt    time.Time  `json:"fired_at"`
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`
}

// GetAlertState returns the state for a subject, defaulting to ok when absent.
func (db *DB) GetAlertState(key string) AlertState {
	var s AlertState
	var pending, openID *string
	err := db.sql.QueryRow(
		`SELECT subject_key, state, pending_since, open_event_id FROM alert_state WHERE subject_key = ?`, key).
		Scan(&s.SubjectKey, &s.State, &pending, &openID)
	if err == sql.ErrNoRows {
		return AlertState{SubjectKey: key, State: "ok"}
	}
	s.PendingSince = parseNullableTime(pending)
	if openID != nil {
		s.OpenEventID = *openID
	}
	return s
}

// PutAlertState upserts a subject's state.
func (db *DB) PutAlertState(s AlertState) error {
	var pending *string
	if s.PendingSince != nil {
		v := s.PendingSince.UTC().Format(time.RFC3339Nano)
		pending = &v
	}
	var openID *string
	if s.OpenEventID != "" {
		openID = &s.OpenEventID
	}
	_, err := db.sql.Exec(
		`INSERT INTO alert_state(subject_key, state, pending_since, open_event_id) VALUES (?,?,?,?)
		 ON CONFLICT(subject_key) DO UPDATE SET state=excluded.state,
			pending_since=excluded.pending_since, open_event_id=excluded.open_event_id`,
		s.SubjectKey, s.State, pending, openID)
	return err
}

// CreateAlertEvent inserts a fired alert and returns it.
func (db *DB) CreateAlertEvent(e AlertEvent, now time.Time) (AlertEvent, error) {
	e.ID = NewID()
	e.FiredAt = now
	_, err := db.sql.Exec(
		`INSERT INTO alert_events(id, subject_key, tool_id, host_id, type, severity, message, fired_at)
		 VALUES (?,?,?,?,?,?,?,?)`,
		e.ID, e.SubjectKey, nullable(e.ToolID), nullable(e.HostID), e.Type, e.Severity, e.Message,
		e.FiredAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return AlertEvent{}, err
	}
	return e, nil
}

// ResolveAlertEvent stamps an event resolved.
func (db *DB) ResolveAlertEvent(id string, now time.Time) error {
	return db.exec1(`UPDATE alert_events SET resolved_at = ? WHERE id = ? AND resolved_at IS NULL`,
		now.UTC().Format(time.RFC3339Nano), id)
}

// ListAlertEvents returns recent alert events, newest first.
func (db *DB) ListAlertEvents(limit int) ([]AlertEvent, error) {
	rows, err := db.sql.Query(
		`SELECT id, subject_key, tool_id, host_id, type, severity, message, fired_at, resolved_at
		 FROM alert_events ORDER BY fired_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAlertEvents(rows)
}

// ListAlertEventsForHost returns a host's recent events, newest first.
func (db *DB) ListAlertEventsForHost(hostID string, limit int) ([]AlertEvent, error) {
	return db.queryAlertEvents(`WHERE host_id = ?`, limit, hostID)
}

// ListAlertEventsForTool returns a tool's recent events, newest first.
func (db *DB) ListAlertEventsForTool(toolID string, limit int) ([]AlertEvent, error) {
	return db.queryAlertEvents(`WHERE tool_id = ?`, limit, toolID)
}

func (db *DB) queryAlertEvents(where string, limit int, arg string) ([]AlertEvent, error) {
	rows, err := db.sql.Query(
		`SELECT id, subject_key, tool_id, host_id, type, severity, message, fired_at, resolved_at
		 FROM alert_events `+where+` ORDER BY fired_at DESC LIMIT ?`, arg, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAlertEvents(rows)
}

// scanAlertEvents reads and closes-free iterates the remaining rows of an
// alert_events query into events; the caller owns closing rows.
func scanAlertEvents(rows *sql.Rows) ([]AlertEvent, error) {
	var out []AlertEvent
	for rows.Next() {
		var e AlertEvent
		var toolID, hostID, resolved *string
		var fired string
		if err := rows.Scan(&e.ID, &e.SubjectKey, &toolID, &hostID, &e.Type, &e.Severity, &e.Message, &fired, &resolved); err != nil {
			return nil, err
		}
		if toolID != nil {
			e.ToolID = *toolID
		}
		if hostID != nil {
			e.HostID = *hostID
		}
		e.FiredAt, _ = time.Parse(time.RFC3339Nano, fired)
		e.ResolvedAt = parseNullableTime(resolved)
		out = append(out, e)
	}
	return out, rows.Err()
}

// DowntimeSecs sums how long the subject identified by filterCol/id was in one
// of the given bad states within [since, now], clamping each event to the
// window. filterCol is "tool_id" or "host_id".
func (db *DB) DowntimeSecs(filterCol, id string, types []string, since, now time.Time) (float64, error) {
	placeholders := strings.Repeat("?,", len(types))
	placeholders = strings.TrimSuffix(placeholders, ",")
	args := make([]any, 0, len(types)+3)
	args = append(args, id)
	for _, ty := range types {
		args = append(args, ty)
	}
	args = append(args, since.UTC().Format(time.RFC3339Nano))

	query := `SELECT fired_at, resolved_at FROM alert_events WHERE ` + filterCol + ` = ? AND type IN (` +
		placeholders + `) AND (resolved_at IS NULL OR resolved_at >= ?)`
	rows, err := db.sql.Query(query, args...)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var total float64
	for rows.Next() {
		var firedStr string
		var resolvedStr *string
		if err := rows.Scan(&firedStr, &resolvedStr); err != nil {
			return 0, err
		}
		fired, _ := time.Parse(time.RFC3339Nano, firedStr)
		end := now
		if resolvedAt := parseNullableTime(resolvedStr); resolvedAt != nil {
			end = *resolvedAt
		}
		start := fired
		if start.Before(since) {
			start = since
		}
		if end.After(now) {
			end = now
		}
		overlap := end.Sub(start).Seconds()
		if overlap > 0 {
			total += overlap
		}
	}
	return total, rows.Err()
}

// CountLogEventsForTool counts a tool's error log events since a cutoff. Log
// lines are matched to the tool by its source_ref appearing in the event's
// source or message.
func (db *DB) CountLogEventsForTool(hostID, sourceRef string, since time.Time) (int, error) {
	var n int
	like := "%" + sourceRef + "%"
	err := db.sql.QueryRow(
		`SELECT COUNT(*) FROM log_events WHERE host_id = ? AND at >= ? AND (source LIKE ? OR message LIKE ?)`,
		hostID, since.UTC().Format(time.RFC3339Nano), like, like).Scan(&n)
	return n, err
}
