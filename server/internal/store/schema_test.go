package store

import (
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
)

func TestDeclaredColumnsReadsTheSchema(t *testing.T) {
	cols := declaredColumns(schemaSQL)
	hosts, ok := cols["hosts"]
	if !ok {
		t.Fatal("hosts table not found in the schema")
	}
	want := map[string]bool{"id": true, "name": true, "control_enabled": true, "offline_after_secs": true}
	for _, c := range hosts {
		delete(want, c)
	}
	if len(want) != 0 {
		t.Errorf("columns missed by the parser: %v", want)
	}
	// Constraint lines are not columns.
	for _, c := range cols["credential_access"] {
		if strings.EqualFold(c, "PRIMARY") {
			t.Error("a PRIMARY KEY line was read as a column")
		}
	}
}

// A database created by an older build is missing a column the current one
// selects. That must stop the server with an explanation, not surface later as
// an unexplained failure on an unrelated request.
func TestOpenRefusesADatabaseMissingAColumn(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old.db")
	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	db.Close()

	// Simulate the older shape by dropping a column the build expects.
	raw, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := raw.Exec(`ALTER TABLE hosts DROP COLUMN control_enabled`); err != nil {
		t.Fatalf("drop column: %v", err)
	}
	raw.Close()

	_, err = Open(path)
	if err == nil {
		t.Fatal("Open accepted a database missing a column the build selects")
	}
	if !strings.Contains(err.Error(), "hosts.control_enabled") {
		t.Errorf("error does not name the missing column: %v", err)
	}
	if !strings.Contains(err.Error(), "fresh database") {
		t.Errorf("error does not say what to do about it: %v", err)
	}
}

func TestOpenAcceptsACurrentDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fresh.db")
	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	db.Close()
	// Re-opening a database this build created must never trip the drift check.
	db2, err := Open(path)
	if err != nil {
		t.Fatalf("reopening a current database failed: %v", err)
	}
	db2.Close()
}
