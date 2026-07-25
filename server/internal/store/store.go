// Package store owns the SQLite database: connection, schema migrations, and
// the typed queries the rest of the server uses.
package store

import (
	"database/sql"
	"fmt"
	"os"

	_ "modernc.org/sqlite"
)

// DB wraps the SQL handle. It is safe for concurrent use.
type DB struct {
	sql *sql.DB
}

// Open opens the SQLite DB at path (WAL, FKs, busy timeout) and migrates it.
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
	if err := db.migrate(); err != nil {
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
