// Package store owns the SQLite database: connection, schema, and
// the typed queries the rest of the server uses.
package store

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	_ "modernc.org/sqlite"
)

// DB wraps the SQL handle. It is safe for concurrent use.
type DB struct {
	sql *sql.DB
}

// Open opens the SQLite DB at path (WAL, FKs, busy timeout) and applies the schema.
func Open(path string) (*DB, error) {
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)&_pragma=busy_timeout(5000)", path)
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	// modernc's driver is not safe for concurrent writers on one connection;
	// a single write connection with WAL keeps writes serialized and correct.
	sqlDB.SetMaxOpenConns(1)
	db := &DB{sql: sqlDB}
	if err := db.applySchema(); err != nil {
		sqlDB.Close()
		return nil, err
	}
	return db, nil
}

// ApplyStagedRestore replaces the database at dbPath with dbPath+".restore" if
// that staged file exists, removing stale -wal/-shm sidecars first. It is a
// no-op when no staged file is present. Call it at startup before Open.
func ApplyStagedRestore(dbPath string) error {
	staged := dbPath + ".restore"
	if _, err := os.Stat(staged); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, p := range []string{dbPath, dbPath + "-wal", dbPath + "-shm"} {
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return os.Rename(staged, dbPath)
}

// Close closes the underlying database.
func (db *DB) Close() error {
	return db.sql.Close()
}

// SQL exposes the raw handle for packages that build their own queries.
func (db *DB) SQL() *sql.DB {
	return db.sql
}

// inTx runs fn inside a transaction, rolling back on error.
func (db *DB) inTx(fn func(*sql.Tx) error) error {
	tx, err := db.sql.Begin()
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

// BackupTo writes a consistent copy of the database to destPath (which must not
// exist) via VACUUM INTO. Credentials in the copy stay AES-256-GCM ciphertext;
// the master key is never part of the database.
func (db *DB) BackupTo(destPath string) error {
	_, err := db.sql.Exec("VACUUM INTO ?", destPath)
	return err
}

// NewID returns a random 128-bit identifier as 32 hex characters.
func NewID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	return hex.EncodeToString(b)
}

// parseNullableTime parses an optional RFC3339Nano timestamp, returning nil for
// a nil pointer or an unparseable value.
func parseNullableTime(s *string) *time.Time {
	if s == nil {
		return nil
	}
	t, err := time.Parse(time.RFC3339Nano, *s)
	if err != nil {
		return nil
	}
	return &t
}

// GetSetting returns a setting value and whether it was present.
func (db *DB) GetSetting(key string) (string, bool) {
	var v string
	err := db.sql.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&v)
	if err != nil {
		return "", false
	}
	return v, true
}

// SetSetting upserts a setting value.
func (db *DB) SetSetting(key, value string) error {
	_, err := db.sql.Exec(
		`INSERT INTO settings(key, value) VALUES (?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
	return err
}

// GetBoolSetting returns a boolean setting, defaulting when unset.
func (db *DB) GetBoolSetting(key string, def bool) bool {
	v, ok := db.GetSetting(key)
	if !ok {
		return def
	}
	return v == "true"
}

// GetIntSetting returns an integer setting, defaulting when unset or unparseable.
func (db *DB) GetIntSetting(key string, def int) int {
	v, ok := db.GetSetting(key)
	if !ok {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

// sqliteMagic is the 16-byte header every SQLite file starts with.
const sqliteMagic = "SQLite format 3\x00"

// StagedRestorePath is where a validated restore waits for the next startup.
func StagedRestorePath(dbPath string) string {
	return dbPath + ".restore"
}

// ValidateBackup reports whether path is a SQLite database this server can
// adopt: right magic, openable, and already carrying the tables a Reeve
// database must have. Staging an unvalidated file would brick the next startup.
//
// It opens the candidate query-only rather than through Open, which would apply
// the schema and so create the very tables this is checking for.
func ValidateBackup(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	header := make([]byte, len(sqliteMagic))
	n, err := f.Read(header)
	f.Close()
	if err != nil || n < len(sqliteMagic) || string(header) != sqliteMagic {
		return errors.New("not a SQLite database file")
	}

	dsn := fmt.Sprintf("file:%s?_pragma=query_only(true)&_pragma=busy_timeout(5000)", path)
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return fmt.Errorf("cannot open as a Reeve database: %w", err)
	}
	defer sqlDB.Close()
	for _, table := range []string{"users", "tools", "credentials", "hosts"} {
		var name string
		err := sqlDB.QueryRow(
			`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&name)
		if err != nil {
			return fmt.Errorf("database is missing the %s table", table)
		}
	}
	return nil
}
