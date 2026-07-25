package store

import (
	"errors"
	"fmt"
	"os"
)

// sqliteMagic is the 16-byte header every SQLite file starts with.
const sqliteMagic = "SQLite format 3\x00"

// StagedRestorePath is where a validated restore waits for the next startup.
func StagedRestorePath(dbPath string) string {
	return dbPath + ".restore"
}

// ValidateBackup reports whether path is a SQLite database this server can
// adopt: right magic, openable, and carrying the tables a Reeve database
// must have. Staging an unvalidated file would brick the next startup.
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

	db, err := Open(path)
	if err != nil {
		return fmt.Errorf("cannot open as a Reeve database: %w", err)
	}
	defer db.Close()
	for _, table := range []string{"users", "tools", "credentials", "schema_migrations"} {
		var name string
		err := db.sql.QueryRow(
			`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&name)
		if err != nil {
			return fmt.Errorf("database is missing the %s table", table)
		}
	}
	return nil
}
