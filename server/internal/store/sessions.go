package store

import (
	"time"
)

// Session is a server-side login session keyed by an opaque cookie value.
type Session struct {
	ID        string
	UserID    string
	ExpiresAt time.Time
}

// CreateSession stores a session and returns its id (the cookie value).
func (db *DB) CreateSession(userID string, ttl time.Duration) (Session, error) {
	s := Session{
		ID:        NewID() + NewID(),
		UserID:    userID,
		ExpiresAt: time.Now().UTC().Add(ttl),
	}
	_, err := db.sql.Exec(
		`INSERT INTO sessions(id, user_id, expires_at, created_at) VALUES (?, ?, ?, ?)`,
		s.ID, s.UserID, s.ExpiresAt.Format(time.RFC3339Nano), time.Now().UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return Session{}, err
	}
	return s, nil
}

// GetSession returns a non-expired session, or ErrNotFound.
func (db *DB) GetSession(id string) (Session, error) {
	var s Session
	var exp string
	err := db.sql.QueryRow(
		`SELECT id, user_id, expires_at FROM sessions WHERE id = ?`, id,
	).Scan(&s.ID, &s.UserID, &exp)
	if err != nil {
		return Session{}, ErrNotFound
	}
	s.ExpiresAt, _ = time.Parse(time.RFC3339Nano, exp)
	if time.Now().UTC().After(s.ExpiresAt) {
		db.DeleteSession(id)
		return Session{}, ErrNotFound
	}
	return s, nil
}

// DeleteSession removes a session (logout).
func (db *DB) DeleteSession(id string) error {
	_, err := db.sql.Exec(`DELETE FROM sessions WHERE id = ?`, id)
	return err
}

// DeleteSessionsForUser removes every session belonging to a user, signing them
// out everywhere. Called whenever their password changes.
func (db *DB) DeleteSessionsForUser(userID string) error {
	_, err := db.sql.Exec(`DELETE FROM sessions WHERE user_id = ?`, userID)
	return err
}
