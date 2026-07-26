package store

import (
	"strings"
	"time"
)

// RevealAudit is one credential-reveal event (append-only).
type RevealAudit struct {
	ID           string    `json:"id"`
	CredentialID string    `json:"credential_id"`
	HostID       string    `json:"host_id"`
	UserID       string    `json:"user_id"`
	SourceIP     string    `json:"source_ip"`
	RevealedAt   time.Time `json:"revealed_at"`
}

// GrantAudit is one grant/revoke event (append-only).
type GrantAudit struct {
	ID            string    `json:"id"`
	HostID        string    `json:"host_id"`
	PrincipalType string    `json:"principal_type"`
	PrincipalID   string    `json:"principal_id"`
	Action        string    `json:"action"`
	ActorID       string    `json:"actor_id"`
	At            time.Time `json:"at"`
}

// AuditFilter narrows audit queries. Empty fields are ignored.
type AuditFilter struct {
	UserID string
	HostID string
	From   string // RFC3339
	To     string
}

// RecordReveal appends a reveal event to the append-only reveal_audit table.
func (db *DB) RecordReveal(credentialID, hostID, userID, sourceIP string) error {
	_, err := db.sql.Exec(
		`INSERT INTO reveal_audit(id, credential_id, host_id, user_id, source_ip, revealed_at)
		 VALUES (?,?,?,?,?,?)`,
		NewID(), credentialID, hostID, userID, sourceIP, time.Now().UTC().Format(time.RFC3339Nano))
	return err
}

// RecordGrant appends a grant/revoke event to the append-only grant_audit table.
func (db *DB) RecordGrant(hostID, ptype, pid, action, actorID string) error {
	_, err := db.sql.Exec(
		`INSERT INTO grant_audit(id, host_id, principal_type, principal_id, action, actor_id, at)
		 VALUES (?,?,?,?,?,?,?)`,
		NewID(), hostID, ptype, pid, action, actorID, time.Now().UTC().Format(time.RFC3339Nano))
	return err
}

// ListRevealAudit returns reveal events matching the filter, newest first.
func (db *DB) ListRevealAudit(f AuditFilter) ([]RevealAudit, error) {
	q := `SELECT id, credential_id, host_id, user_id, source_ip, revealed_at FROM reveal_audit`
	where, args := auditWhere(f, "user_id", "host_id", "revealed_at")
	q += where + ` ORDER BY revealed_at DESC LIMIT 500`
	rows, err := db.sql.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []RevealAudit
	for rows.Next() {
		var a RevealAudit
		var at string
		if err := rows.Scan(&a.ID, &a.CredentialID, &a.HostID, &a.UserID, &a.SourceIP, &at); err != nil {
			return nil, err
		}
		a.RevealedAt, _ = time.Parse(time.RFC3339Nano, at)
		out = append(out, a)
	}
	return out, rows.Err()
}

// ListGrantAudit returns grant/revoke events matching the filter, newest first.
func (db *DB) ListGrantAudit(f AuditFilter) ([]GrantAudit, error) {
	q := `SELECT id, host_id, principal_type, principal_id, action, actor_id, at FROM grant_audit`
	where, args := auditWhere(f, "actor_id", "host_id", "at")
	q += where + ` ORDER BY at DESC LIMIT 500`
	rows, err := db.sql.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []GrantAudit
	for rows.Next() {
		var a GrantAudit
		var at string
		if err := rows.Scan(&a.ID, &a.HostID, &a.PrincipalType, &a.PrincipalID, &a.Action, &a.ActorID, &at); err != nil {
			return nil, err
		}
		a.At, _ = time.Parse(time.RFC3339Nano, at)
		out = append(out, a)
	}
	return out, rows.Err()
}

func auditWhere(f AuditFilter, userCol, hostCol, timeCol string) (string, []any) {
	var clauses []string
	var args []any
	if f.UserID != "" {
		clauses = append(clauses, userCol+" = ?")
		args = append(args, f.UserID)
	}
	if f.HostID != "" {
		clauses = append(clauses, hostCol+" = ?")
		args = append(args, f.HostID)
	}
	if f.From != "" {
		clauses = append(clauses, timeCol+" >= ?")
		args = append(args, f.From)
	}
	if f.To != "" {
		clauses = append(clauses, timeCol+" <= ?")
		args = append(args, f.To)
	}
	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}
