package store

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

// testMAC stands in for the master-key MAC. It is keyed, so a test that tampers
// with a row cannot recompute the chain any more than an attacker could.
func testMAC(data []byte) string {
	m := hmac.New(sha256.New, []byte("test-audit-key"))
	m.Write(data)
	return hex.EncodeToString(m.Sum(nil))
}

func chainedDB(t *testing.T) *DB {
	t.Helper()
	db := openTemp(t)
	db.SetAuditMAC(testMAC)
	return db
}

func TestAuditChainVerifiesWhenUntouched(t *testing.T) {
	db := chainedDB(t)
	for i := 0; i < 3; i++ {
		if err := db.RecordReveal("cred", "host", "user", "10.0.0.1"); err != nil {
			t.Fatalf("record: %v", err)
		}
	}
	if err := db.RecordGrant("host", "user", "u1", "grant", "admin"); err != nil {
		t.Fatalf("grant: %v", err)
	}
	for _, r := range mustVerify(t, db) {
		if !r.OK || r.Unchained != 0 {
			t.Errorf("%s: ok=%v unchained=%d detail=%q", r.Table, r.OK, r.Unchained, r.Detail)
		}
	}
}

func TestAuditChainCatchesAnEditedRow(t *testing.T) {
	db := chainedDB(t)
	db.RecordReveal("cred", "host", "honest-user", "10.0.0.1")
	db.RecordReveal("cred", "host", "other-user", "10.0.0.2")

	// Rewrite who did it, the edit an admin covering their tracks would make.
	if _, err := db.SQL().Exec(
		`UPDATE reveal_audit SET user_id = 'someone-else' WHERE user_id = 'honest-user'`); err != nil {
		t.Fatal(err)
	}
	res := mustVerify(t, db)[0]
	if res.OK {
		t.Fatal("an edited reveal row still verified")
	}
	if res.BrokenAt == "" {
		t.Error("no row was named as broken")
	}
}

func TestAuditChainCatchesADeletedRow(t *testing.T) {
	db := chainedDB(t)
	db.RecordReveal("cred", "host", "user-1", "10.0.0.1")
	db.RecordReveal("cred", "host", "user-2", "10.0.0.2")
	db.RecordReveal("cred", "host", "user-3", "10.0.0.3")

	if _, err := db.SQL().Exec(`DELETE FROM reveal_audit WHERE user_id = 'user-2'`); err != nil {
		t.Fatal(err)
	}
	if res := mustVerify(t, db)[0]; res.OK {
		t.Fatal("the chain survived a deleted row: deletion is the whole point")
	}
}

func TestAuditRowsWrittenBeforeTheChainReadAsUnchained(t *testing.T) {
	db := openTemp(t)
	db.RecordReveal("cred", "host", "legacy", "10.0.0.1")
	db.SetAuditMAC(testMAC)
	db.RecordReveal("cred", "host", "modern", "10.0.0.2")

	res := mustVerify(t, db)[0]
	if !res.OK {
		t.Errorf("a legacy row should not read as tampered: %q", res.Detail)
	}
	if res.Unchained != 1 || res.Rows != 2 {
		t.Errorf("rows=%d unchained=%d, want 2 and 1", res.Rows, res.Unchained)
	}
}

func TestAuditChainWithoutAKeyReportsItCannotCheck(t *testing.T) {
	db := openTemp(t)
	db.RecordReveal("cred", "host", "user", "10.0.0.1")
	if res := mustVerify(t, db)[0]; res.OK {
		t.Error("verification claimed success with no key to check against")
	}
}

func mustVerify(t *testing.T, db *DB) []AuditChainResult {
	t.Helper()
	res, err := db.VerifyAuditChain()
	if err != nil {
		t.Fatalf("VerifyAuditChain: %v", err)
	}
	return res
}
