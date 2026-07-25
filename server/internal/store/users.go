package store

import (
	"database/sql"
	"errors"
	"time"
)

// Role values for a user.
const (
	RoleAdmin = "admin"
	RoleBasic = "basic"
)

// ErrNotFound is returned when a lookup matches no row.
var ErrNotFound = errors.New("not found")

// User is an account row.
type User struct {
	ID           string
	Email        string
	PasswordHash string
	Role         string
	Active       bool
	CreatedAt    time.Time
	DisplayName  string
	AvatarPath   string
}

// CreateUser inserts a user and returns it. The caller supplies the hash.
func (db *DB) CreateUser(email, passwordHash, role string) (User, error) {
	u := User{
		ID:           NewID(),
		Email:        email,
		PasswordHash: passwordHash,
		Role:         role,
		Active:       true,
		CreatedAt:    time.Now().UTC(),
	}
	_, err := db.sql.Exec(
		`INSERT INTO users(id, email, password_hash, role, active, created_at)
		 VALUES (?, ?, ?, ?, 1, ?)`,
		u.ID, u.Email, u.PasswordHash, u.Role, u.CreatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return User{}, err
	}
	return u, nil
}

// CountUsers returns the total number of users.
func (db *DB) CountUsers() (int, error) {
	var n int
	err := db.sql.QueryRow("SELECT COUNT(*) FROM users").Scan(&n)
	return n, err
}

// GetUserByEmail looks up a user by email.
func (db *DB) GetUserByEmail(email string) (User, error) {
	return db.scanUser(db.sql.QueryRow(
		`SELECT id, email, password_hash, role, active, created_at, display_name, avatar_path FROM users WHERE email = ?`, email))
}

// GetUserByID looks up a user by id.
func (db *DB) GetUserByID(id string) (User, error) {
	return db.scanUser(db.sql.QueryRow(
		`SELECT id, email, password_hash, role, active, created_at, display_name, avatar_path FROM users WHERE id = ?`, id))
}

// ListUsers returns all users ordered by creation time.
func (db *DB) ListUsers() ([]User, error) {
	rows, err := db.sql.Query(
		`SELECT id, email, password_hash, role, active, created_at, display_name, avatar_path FROM users ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []User
	for rows.Next() {
		u, err := scanUserRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// SetUserRole changes a user's role.
func (db *DB) SetUserRole(id, role string) error {
	return db.exec1(`UPDATE users SET role = ? WHERE id = ?`, role, id)
}

// SetUserActive enables or disables a user's login.
func (db *DB) SetUserActive(id string, active bool) error {
	v := 0
	if active {
		v = 1
	}
	return db.exec1(`UPDATE users SET active = ? WHERE id = ?`, v, id)
}

// SetUserDisplayName sets a user's display name.
func (db *DB) SetUserDisplayName(id, name string) error {
	return db.exec1(`UPDATE users SET display_name = ? WHERE id = ?`, name, id)
}

// SetUserPassword replaces a user's password hash.
func (db *DB) SetUserPassword(id, hash string) error {
	return db.exec1(`UPDATE users SET password_hash = ? WHERE id = ?`, hash, id)
}

// SetUserAvatarPath sets (or clears, when empty) a user's stored avatar path.
func (db *DB) SetUserAvatarPath(id, path string) error {
	return db.exec1(`UPDATE users SET avatar_path = ? WHERE id = ?`, path, id)
}

// CountAdmins returns the number of active-or-not admin accounts.
func (db *DB) CountAdmins() (int, error) {
	var n int
	err := db.sql.QueryRow(`SELECT COUNT(*) FROM users WHERE role = ?`, RoleAdmin).Scan(&n)
	return n, err
}

// CountActiveAdmins returns the number of admins who can still sign in. It is
// the number that matters for lockout: a deactivated admin cannot fix anything.
func (db *DB) CountActiveAdmins() (int, error) {
	var n int
	err := db.sql.QueryRow(`SELECT COUNT(*) FROM users WHERE role = ? AND active = 1`, RoleAdmin).Scan(&n)
	return n, err
}

// OldestAdminExcluding returns the earliest-created admin other than excludeID.
func (db *DB) OldestAdminExcluding(excludeID string) (User, error) {
	return db.scanUser(db.sql.QueryRow(
		`SELECT id, email, password_hash, role, active, created_at, display_name, avatar_path
		 FROM users WHERE role = ? AND id != ? ORDER BY created_at LIMIT 1`, RoleAdmin, excludeID))
}

// DeleteUser removes a user, reassigning the tools and credentials they own to
// reassignTo first so the RESTRICT foreign keys hold. Sessions, tokens, group
// memberships, and access requests cascade. Runs in one transaction.
func (db *DB) DeleteUser(id, reassignTo string) error {
	tx, err := db.sql.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`UPDATE tools SET creator_id = ? WHERE creator_id = ?`, reassignTo, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE credentials SET created_by = ? WHERE created_by = ?`, reassignTo, id); err != nil {
		return err
	}
	res, err := tx.Exec(`DELETE FROM users WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return tx.Commit()
}

type scanner interface {
	Scan(dest ...any) error
}

func (db *DB) scanUser(row scanner) (User, error) {
	u, err := scanUserRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrNotFound
	}
	return u, err
}

func scanUserRow(row scanner) (User, error) {
	var u User
	var created string
	var active int
	if err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &active, &created, &u.DisplayName, &u.AvatarPath); err != nil {
		return User{}, err
	}
	u.Active = active == 1
	u.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	return u, nil
}

// exec1 runs a statement and requires exactly one row to be affected.
func (db *DB) exec1(query string, args ...any) error {
	res, err := db.sql.Exec(query, args...)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

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
