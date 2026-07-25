package store

import (
	"time"
)

// PasswordReset is a single-use, expiring token that authorizes a password
// change for one account. Only the token's hash is stored.
type PasswordReset struct {
	ID        string
	UserID    string
	ExpiresAt time.Time
	UsedAt    *time.Time
}

// CreatePasswordReset records a reset token hash for a user.
func (db *DB) CreatePasswordReset(userID, tokenHash string, expiresAt, now time.Time) (PasswordReset, error) {
	pr := PasswordReset{ID: NewID(), UserID: userID, ExpiresAt: expiresAt}
	_, err := db.sql.Exec(
		`INSERT INTO password_resets(id, user_id, token_hash, expires_at, created_at) VALUES (?, ?, ?, ?, ?)`,
		pr.ID, userID, tokenHash, expiresAt.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano),
	)
	if err != nil {
		return PasswordReset{}, err
	}
	return pr, nil
}

// GetPasswordResetByHash returns an unused, unexpired reset for the token hash,
// or ErrNotFound. A spent or stale token is indistinguishable from a wrong one.
func (db *DB) GetPasswordResetByHash(tokenHash string, now time.Time) (PasswordReset, error) {
	var pr PasswordReset
	var exp string
	var used *string
	err := db.sql.QueryRow(
		`SELECT id, user_id, expires_at, used_at FROM password_resets WHERE token_hash = ?`, tokenHash,
	).Scan(&pr.ID, &pr.UserID, &exp, &used)
	if err != nil {
		return PasswordReset{}, ErrNotFound
	}
	pr.ExpiresAt, _ = time.Parse(time.RFC3339Nano, exp)
	pr.UsedAt = parseNullableTime(used)
	if pr.UsedAt != nil || now.After(pr.ExpiresAt) {
		return PasswordReset{}, ErrNotFound
	}
	return pr, nil
}

// UsePasswordReset marks a reset spent and invalidates that user's other open
// resets, so one request cannot be replayed or shadowed by an older link.
func (db *DB) UsePasswordReset(id, userID string, now time.Time) error {
	ts := now.Format(time.RFC3339Nano)
	if err := db.exec1(`UPDATE password_resets SET used_at = ? WHERE id = ? AND used_at IS NULL`, ts, id); err != nil {
		return err
	}
	_, err := db.sql.Exec(
		`UPDATE password_resets SET used_at = ? WHERE user_id = ? AND used_at IS NULL`, ts, userID)
	return err
}

// PrunePasswordResets deletes resets that expired before cutoff.
func (db *DB) PrunePasswordResets(cutoff time.Time) error {
	_, err := db.sql.Exec(`DELETE FROM password_resets WHERE expires_at < ?`, cutoff.Format(time.RFC3339Nano))
	return err
}
