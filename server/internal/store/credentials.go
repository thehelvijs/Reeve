package store

import (
	"database/sql"
	"errors"
	"time"
)

// Credential is a secret attached to a tool. The plaintext never lives here;
// only the ciphertext + nonce do.
type Credential struct {
	ID        string
	ToolID    string
	Type      string
	Label     string
	CreatedBy string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// CreateCredential stores an encrypted credential. Encryption happens in the
// caller; the store only ever sees ciphertext.
func (db *DB) CreateCredential(toolID, ctype, label string, ciphertext, nonce []byte, createdBy string) (Credential, error) {
	c := Credential{
		ID: NewID(), ToolID: toolID, Type: ctype, Label: label, CreatedBy: createdBy,
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	_, err := db.sql.Exec(
		`INSERT INTO credentials(id, tool_id, type, label, ciphertext, nonce, created_by, created_at, updated_at)
		 VALUES (?,?,?,?,?,?,?,?,?)`,
		c.ID, c.ToolID, c.Type, c.Label, ciphertext, nonce, c.CreatedBy,
		c.CreatedAt.Format(time.RFC3339Nano), c.UpdatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return Credential{}, err
	}
	return c, nil
}

// ListCredentials returns credential metadata (no secret) for a tool.
func (db *DB) ListCredentials(toolID string) ([]Credential, error) {
	rows, err := db.sql.Query(
		`SELECT id, tool_id, type, label, created_by, created_at, updated_at
		 FROM credentials WHERE tool_id = ? ORDER BY created_at`, toolID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Credential
	for rows.Next() {
		c, err := scanCredentialMeta(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// GetCredential returns credential metadata by id.
func (db *DB) GetCredential(id string) (Credential, error) {
	c, err := scanCredentialMeta(db.sql.QueryRow(
		`SELECT id, tool_id, type, label, created_by, created_at, updated_at
		 FROM credentials WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Credential{}, ErrNotFound
	}
	return c, err
}

// GetCredentialSecret returns the ciphertext + nonce for decryption on reveal.
func (db *DB) GetCredentialSecret(id string) (ciphertext, nonce []byte, err error) {
	err = db.sql.QueryRow(`SELECT ciphertext, nonce FROM credentials WHERE id = ?`, id).
		Scan(&ciphertext, &nonce)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil, ErrNotFound
	}
	return ciphertext, nonce, err
}

// UpdateCredential rotates a credential's label and secret (re-encrypted by the
// caller).
func (db *DB) UpdateCredential(id, label string, ciphertext, nonce []byte) error {
	return db.exec1(
		`UPDATE credentials SET label = ?, ciphertext = ?, nonce = ?, updated_at = ? WHERE id = ?`,
		label, ciphertext, nonce, time.Now().UTC().Format(time.RFC3339Nano), id)
}

// DeleteCredential removes a credential.
func (db *DB) DeleteCredential(id string) error {
	return db.exec1(`DELETE FROM credentials WHERE id = ?`, id)
}

func scanCredentialMeta(row scanner) (Credential, error) {
	var c Credential
	var created, updated string
	if err := row.Scan(&c.ID, &c.ToolID, &c.Type, &c.Label, &c.CreatedBy, &created, &updated); err != nil {
		return Credential{}, err
	}
	c.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	c.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
	return c, nil
}

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
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return one == 1, nil
}
