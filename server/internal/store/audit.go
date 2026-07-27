package store

import (
	"database/sql"
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

// SetAuditMAC installs the keyed MAC that chains audit rows. Without one the
// tables still record events, they just cannot prove they were not edited, so a
// server started with no cipher (tests, tooling) keeps working.
func (db *DB) SetAuditMAC(mac func([]byte) string) {
	db.auditMAC = mac
}

// auditLink is the canonical bytes of one row, joined with a separator that
// cannot appear in any field, so no two different rows serialize alike.
func auditLink(prevHash string, fields ...string) []byte {
	return []byte(prevHash + "\x00" + strings.Join(fields, "\x00"))
}

// appendAudit writes one chained row. The previous hash is read and the new row
// written in a single transaction so two concurrent events cannot fork the chain
// by reading the same predecessor.
func (db *DB) appendAudit(table, insert string, fields ...string) error {
	tx, err := db.sql.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var prev string
	err = tx.QueryRow(`SELECT hash FROM ` + table + ` ORDER BY rowid DESC LIMIT 1`).Scan(&prev)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	hash := ""
	if db.auditMAC != nil {
		hash = db.auditMAC(auditLink(prev, fields...))
	}
	args := make([]any, 0, len(fields)+2)
	for _, f := range fields {
		args = append(args, f)
	}
	if _, err := tx.Exec(insert, append(args, prev, hash)...); err != nil {
		return err
	}
	return tx.Commit()
}

// RecordReveal appends a reveal event to the append-only reveal_audit table.
func (db *DB) RecordReveal(credentialID, hostID, userID, sourceIP string) error {
	return db.appendAudit("reveal_audit",
		`INSERT INTO reveal_audit(id, credential_id, host_id, user_id, source_ip, revealed_at, prev_hash, hash)
		 VALUES (?,?,?,?,?,?,?,?)`,
		NewID(), credentialID, hostID, userID, sourceIP, time.Now().UTC().Format(time.RFC3339Nano))
}

// RecordGrant appends a grant/revoke event to the append-only grant_audit table.
func (db *DB) RecordGrant(hostID, ptype, pid, action, actorID string) error {
	return db.appendAudit("grant_audit",
		`INSERT INTO grant_audit(id, host_id, principal_type, principal_id, action, actor_id, at, prev_hash, hash)
		 VALUES (?,?,?,?,?,?,?,?,?)`,
		NewID(), hostID, ptype, pid, action, actorID, time.Now().UTC().Format(time.RFC3339Nano))
}

// AuditChainResult reports what verifying one audit table found.
type AuditChainResult struct {
	Table     string `json:"table"`
	Rows      int    `json:"rows"`
	Unchained int    `json:"unchained"`
	OK        bool   `json:"ok"`
	BrokenAt  string `json:"broken_at,omitempty"`
	Detail    string `json:"detail,omitempty"`
}

// VerifyAuditChain recomputes both audit chains and reports the first row that
// does not match. A row whose hash is empty predates the chain and is counted,
// not failed: it cannot be proven either way.
func (db *DB) VerifyAuditChain() ([]AuditChainResult, error) {
	reveal, err := db.verifyChain("reveal_audit",
		`SELECT id, credential_id, host_id, user_id, source_ip, revealed_at, prev_hash, hash
		   FROM reveal_audit ORDER BY rowid`, 6)
	if err != nil {
		return nil, err
	}
	grant, err := db.verifyChain("grant_audit",
		`SELECT id, host_id, principal_type, principal_id, action, actor_id, at, prev_hash, hash
		   FROM grant_audit ORDER BY rowid`, 7)
	if err != nil {
		return nil, err
	}
	return []AuditChainResult{reveal, grant}, nil
}

func (db *DB) verifyChain(table, query string, nFields int) (AuditChainResult, error) {
	res := AuditChainResult{Table: table, OK: true}
	if db.auditMAC == nil {
		res.OK = false
		res.Detail = "no master key: the chain cannot be checked"
		return res, nil
	}
	rows, err := db.sql.Query(query)
	if err != nil {
		return res, err
	}
	defer rows.Close()

	cells := make([]string, nFields+2)
	scan := make([]any, len(cells))
	for i := range cells {
		scan[i] = &cells[i]
	}
	expectedPrev := ""
	for rows.Next() {
		if err := rows.Scan(scan...); err != nil {
			return res, err
		}
		res.Rows++
		fields, prev, hash := cells[:nFields], cells[nFields], cells[nFields+1]
		if hash == "" {
			res.Unchained++
			continue
		}
		if !res.OK {
			continue
		}
		if prev != expectedPrev {
			res.OK, res.BrokenAt = false, fields[0]
			res.Detail = "a row is missing or reordered ahead of this one"
			continue
		}
		if db.auditMAC(auditLink(prev, fields...)) != hash {
			res.OK, res.BrokenAt = false, fields[0]
			res.Detail = "this row was edited after it was written"
			continue
		}
		expectedPrev = hash
	}
	return res, rows.Err()
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
