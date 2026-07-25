package store

import "time"

// AccessGrant is a principal with standing credential access to a tool.
type AccessGrant struct {
	PrincipalType string
	PrincipalID   string
	GrantedBy     string
	GrantedAt     time.Time
}

// GrantCredentialAccess gives a user or group standing access to a tool's
// credentials (idempotent).
func (db *DB) GrantCredentialAccess(toolID, ptype, pid, grantedBy string) error {
	_, err := db.sql.Exec(
		`INSERT INTO credential_access(tool_id, principal_type, principal_id, granted_by, granted_at)
		 VALUES (?,?,?,?,?)
		 ON CONFLICT(tool_id, principal_type, principal_id) DO NOTHING`,
		toolID, ptype, pid, grantedBy, time.Now().UTC().Format(time.RFC3339Nano))
	return err
}

// RevokeCredentialAccess removes a standing grant.
func (db *DB) RevokeCredentialAccess(toolID, ptype, pid string) error {
	_, err := db.sql.Exec(
		`DELETE FROM credential_access WHERE tool_id = ? AND principal_type = ? AND principal_id = ?`,
		toolID, ptype, pid)
	return err
}

// ListCredentialAccess returns the grants for a tool.
func (db *DB) ListCredentialAccess(toolID string) ([]AccessGrant, error) {
	rows, err := db.sql.Query(
		`SELECT principal_type, principal_id, granted_by, granted_at
		 FROM credential_access WHERE tool_id = ?`, toolID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AccessGrant
	for rows.Next() {
		var g AccessGrant
		var at string
		if err := rows.Scan(&g.PrincipalType, &g.PrincipalID, &g.GrantedBy, &at); err != nil {
			return nil, err
		}
		g.GrantedAt, _ = time.Parse(time.RFC3339Nano, at)
		out = append(out, g)
	}
	return out, rows.Err()
}

// HasCredentialAccess reports whether the user may reveal a tool's credentials:
// directly granted, or a member of a granted group.
func (db *DB) HasCredentialAccess(userID, toolID string) (bool, error) {
	var one int
	err := db.sql.QueryRow(
		`SELECT 1 FROM credential_access
		 WHERE tool_id = ? AND (
			(principal_type='user' AND principal_id = ?)
			OR (principal_type='group' AND principal_id IN (
				SELECT group_id FROM group_members WHERE user_id = ?))
		 ) LIMIT 1`,
		toolID, userID, userID).Scan(&one)
	if err != nil {
		return false, nil
	}
	return one == 1, nil
}
