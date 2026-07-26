package store

import (
	"database/sql"
	"errors"
	"time"

	"github.com/thehelvijs/Reeve/contracts"
)

// Command statuses.
const (
	CommandPending = "pending"
	CommandSent    = "sent"
	CommandDone    = "done"
	CommandFailed  = "failed"
	CommandExpired = "expired"
)

// CommandStaleAfter is how long a command may sit undelivered or unanswered
// before it is expired. An agent that went offline mid-command must not run it
// on its return an hour later.
const CommandStaleAfter = 10 * time.Minute

// HostCommand is one queued remote action and its outcome.
type HostCommand struct {
	ID          string     `json:"id"`
	HostID      string     `json:"host_id"`
	Action      string     `json:"action"`
	Target      string     `json:"target"`
	Status      string     `json:"status"`
	Output      string     `json:"output"`
	RequestedBy string     `json:"requested_by"`
	RequestedAt time.Time  `json:"requested_at"`
	SentAt      *time.Time `json:"sent_at,omitempty"`
	FinishedAt  *time.Time `json:"finished_at,omitempty"`
}

// EnqueueCommand queues an action for a host. requestedBy is the admin who
// asked, kept verbatim for the life of the row.
func (db *DB) EnqueueCommand(hostID, action, target, requestedBy string, now time.Time) (HostCommand, error) {
	c := HostCommand{
		ID: NewID(), HostID: hostID, Action: action, Target: target,
		Status: CommandPending, RequestedBy: requestedBy, RequestedAt: now.UTC(),
	}
	_, err := db.sql.Exec(
		`INSERT INTO host_commands(id, host_id, action, target, status, requested_by, requested_at)
		 VALUES (?,?,?,?,?,?,?)`,
		c.ID, c.HostID, c.Action, c.Target, c.Status, c.RequestedBy,
		c.RequestedAt.Format(time.RFC3339Nano))
	if err != nil {
		return HostCommand{}, err
	}
	return c, nil
}

// ClaimPendingCommands hands a host its queued commands and marks them sent, so
// one push collects a command exactly once.
func (db *DB) ClaimPendingCommands(hostID string, limit int, now time.Time) ([]contracts.Command, error) {
	rows, err := db.sql.Query(
		`SELECT id, action, target FROM host_commands
		 WHERE host_id = ? AND status = ? ORDER BY requested_at LIMIT ?`,
		hostID, CommandPending, limit)
	if err != nil {
		return nil, err
	}
	var out []contracts.Command
	for rows.Next() {
		var c contracts.Command
		if err := rows.Scan(&c.ID, &c.Action, &c.Target); err != nil {
			rows.Close()
			return nil, err
		}
		out = append(out, c)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, c := range out {
		if _, err := db.sql.Exec(
			`UPDATE host_commands SET status = ?, sent_at = ? WHERE id = ?`,
			CommandSent, now.UTC().Format(time.RFC3339Nano), c.ID); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// ApplyCommandResult records an agent's report. The host id is part of the
// match, so a compromised agent cannot close another host's command.
func (db *DB) ApplyCommandResult(hostID string, r contracts.CommandResult) error {
	status := CommandFailed
	if r.OK {
		status = CommandDone
	}
	output := r.Output
	if len(output) > contracts.MaxCommandOutput {
		output = output[:contracts.MaxCommandOutput]
	}
	finished := r.FinishedAt
	if finished.IsZero() {
		finished = time.Now().UTC()
	}
	_, err := db.sql.Exec(
		`UPDATE host_commands SET status = ?, output = ?, finished_at = ?
		 WHERE id = ? AND host_id = ? AND status = ?`,
		status, output, finished.UTC().Format(time.RFC3339Nano), r.ID, hostID, CommandSent)
	return err
}

// ExpireStaleCommands closes out commands an agent never collected or never
// answered, and reports how many it closed.
func (db *DB) ExpireStaleCommands(now time.Time) (int64, error) {
	cutoff := now.UTC().Add(-CommandStaleAfter).Format(time.RFC3339Nano)
	res, err := db.sql.Exec(
		`UPDATE host_commands SET status = ?, finished_at = ?,
		        output = 'expired: the agent did not report within the window'
		 WHERE status IN (?, ?) AND requested_at < ?`,
		CommandExpired, now.UTC().Format(time.RFC3339Nano), CommandPending, CommandSent, cutoff)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// ListHostCommands returns a host's recent commands, newest first.
func (db *DB) ListHostCommands(hostID string, limit int) ([]HostCommand, error) {
	rows, err := db.sql.Query(
		`SELECT id, host_id, action, target, status, output, requested_by,
		        requested_at, sent_at, finished_at
		 FROM host_commands WHERE host_id = ? ORDER BY requested_at DESC LIMIT ?`,
		hostID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []HostCommand{}
	for rows.Next() {
		var c HostCommand
		var requested string
		var sent, finished sql.NullString
		if err := rows.Scan(&c.ID, &c.HostID, &c.Action, &c.Target, &c.Status, &c.Output,
			&c.RequestedBy, &requested, &sent, &finished); err != nil {
			return nil, err
		}
		c.RequestedAt, _ = time.Parse(time.RFC3339Nano, requested)
		c.SentAt = parseNullTime(sent)
		c.FinishedAt = parseNullTime(finished)
		out = append(out, c)
	}
	return out, rows.Err()
}

// GetCommand returns one command by id.
func (db *DB) GetCommand(id string) (HostCommand, error) {
	var c HostCommand
	var requested string
	var sent, finished sql.NullString
	err := db.sql.QueryRow(
		`SELECT id, host_id, action, target, status, output, requested_by,
		        requested_at, sent_at, finished_at FROM host_commands WHERE id = ?`, id).
		Scan(&c.ID, &c.HostID, &c.Action, &c.Target, &c.Status, &c.Output,
			&c.RequestedBy, &requested, &sent, &finished)
	if errors.Is(err, sql.ErrNoRows) {
		return HostCommand{}, ErrNotFound
	}
	if err != nil {
		return HostCommand{}, err
	}
	c.RequestedAt, _ = time.Parse(time.RFC3339Nano, requested)
	c.SentAt = parseNullTime(sent)
	c.FinishedAt = parseNullTime(finished)
	return c, nil
}

// HostReportsService reports whether the host last pushed this systemd unit, so
// a command cannot name a unit the machine has never mentioned.
func (db *DB) HostReportsService(hostID, unit string) bool {
	var one int
	err := db.sql.QueryRow(
		`SELECT 1 FROM service_status WHERE host_id = ? AND unit = ?`, hostID, unit).Scan(&one)
	return err == nil
}

// HostReportsContainer is HostReportsService for a Docker container name.
func (db *DB) HostReportsContainer(hostID, name string) bool {
	var one int
	err := db.sql.QueryRow(
		`SELECT 1 FROM container_status WHERE host_id = ? AND name = ?`, hostID, name).Scan(&one)
	return err == nil
}

func parseNullTime(v sql.NullString) *time.Time {
	if !v.Valid || v.String == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339Nano, v.String)
	if err != nil {
		return nil
	}
	return &t
}
