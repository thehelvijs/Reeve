package store

import (
	_ "embed"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

//go:embed schema.sql
var schemaSQL string

// applySchema creates anything the schema declares that the database does not
// already have, then refuses to run against a database that has drifted from
// it. Reeve is pre-release with no installed base, so the schema is one file
// edited in place rather than a chain of migrations; every statement in it is
// idempotent, which is what makes re-running it on every open safe.
func (db *DB) applySchema() error {
	if _, err := db.sql.Exec(schemaSQL); err != nil {
		return fmt.Errorf("apply schema: %w", err)
	}
	return db.checkSchemaDrift()
}

// checkSchemaDrift fails startup when a table the schema declares is missing
// columns the running build expects.
//
// `CREATE TABLE IF NOT EXISTS` creates a table that is absent and does nothing
// at all to one that exists, so adding a column to schema.sql silently leaves
// every older database without it. Without this check the server starts happily
// and then fails somewhere far away — a renamed column once turned every agent
// push into an unexplained 401. With no migrations by design, the answer is to
// recreate the database, and the operator needs to be told that in those words
// rather than left to debug it.
func (db *DB) checkSchemaDrift() error {
	var drift []string
	for table, want := range declaredColumns(schemaSQL) {
		have, err := db.tableColumns(table)
		if err != nil {
			return err
		}
		if len(have) == 0 {
			continue // a view, or a table this build no longer creates
		}
		for _, col := range want {
			if !have[col] {
				drift = append(drift, table+"."+col)
			}
		}
	}
	if len(drift) == 0 {
		return nil
	}
	sort.Strings(drift)
	return fmt.Errorf(
		"this database predates the current build and is missing %s. "+
			"Reeve has no migrations by design, so a schema change means a fresh database: "+
			"back the file up and delete it, then restart", strings.Join(drift, ", "))
}

func (db *DB) tableColumns(table string) (map[string]bool, error) {
	rows, err := db.sql.Query(`SELECT name FROM pragma_table_info(?)`, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		out[name] = true
	}
	return out, rows.Err()
}

var createTableRe = regexp.MustCompile(`(?is)CREATE TABLE IF NOT EXISTS\s+(\w+)\s*\((.*?)\n\);`)

var columnTypes = map[string]bool{"TEXT": true, "INTEGER": true, "REAL": true, "BLOB": true, "NUMERIC": true}

var identifierRe = regexp.MustCompile(`^\w+$`)

// declaredColumns reads the column names each CREATE TABLE in the schema
// declares. A column definition is an identifier followed by a SQL type;
// anything else in the body is a constraint, or the wrapped continuation of one
// (a multi-line CHECK list looks a lot like a column if you only read the first
// token), and is skipped.
func declaredColumns(schema string) map[string][]string {
	out := map[string][]string{}
	for _, m := range createTableRe.FindAllStringSubmatch(schema, -1) {
		table, body := m[1], m[2]
		var cols []string
		for _, line := range strings.Split(body, "\n") {
			fields := strings.Fields(strings.TrimSpace(line))
			if len(fields) < 2 || strings.HasPrefix(fields[0], "--") {
				continue
			}
			name := fields[0]
			if !identifierRe.MatchString(name) || !columnTypes[strings.ToUpper(fields[1])] {
				continue
			}
			cols = append(cols, name)
		}
		out[table] = cols
	}
	return out
}
