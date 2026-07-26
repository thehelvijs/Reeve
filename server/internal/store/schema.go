package store

import (
	_ "embed"
	"fmt"
)

//go:embed schema.sql
var schemaSQL string

// applySchema creates anything the schema declares that the database does not
// already have. Reeve is pre-release with no installed base, so the schema is
// one file edited in place rather than a chain of migrations; every statement
// in it is idempotent, which is what makes re-running it on every open safe.
func (db *DB) applySchema() error {
	if _, err := db.sql.Exec(schemaSQL); err != nil {
		return fmt.Errorf("apply schema: %w", err)
	}
	return nil
}
