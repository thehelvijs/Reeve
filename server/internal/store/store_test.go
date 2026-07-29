package store

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/thehelvijs/Reeve/contracts"
)

func openTemp(t *testing.T) *DB {
	t.Helper()
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestOpenAppliesSchema(t *testing.T) {
	db := openTemp(t)

	tables := []string{"users", "sessions", "settings", "hosts", "credentials"}
	for _, tbl := range tables {
		var name string
		err := db.SQL().QueryRow(
			"SELECT name FROM sqlite_master WHERE type='table' AND name=?", tbl,
		).Scan(&name)
		if err != nil {
			t.Errorf("table %q missing after Open: %v", tbl, err)
		}
	}
}

// The schema is re-executed on every Open, so a second one must neither fail on
// an existing object nor duplicate a seeded row.
func TestSchemaIsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	db1, err := Open(path)
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}
	first := thresholdCount(t, db1)
	db1.Close()

	db2, err := Open(path)
	if err != nil {
		t.Fatalf("second Open: %v", err)
	}
	defer db2.Close()

	if second := thresholdCount(t, db2); second != first {
		t.Errorf("seeded thresholds changed on re-open: %d -> %d", first, second)
	}
	if first == 0 {
		t.Error("the schema seeded no default thresholds")
	}
}

func thresholdCount(t *testing.T, db *DB) int {
	t.Helper()
	var n int
	if err := db.SQL().QueryRow("SELECT COUNT(*) FROM alert_thresholds").Scan(&n); err != nil {
		t.Fatalf("count thresholds: %v", err)
	}
	return n
}

// Validation must read the candidate, never open it through Open: applying the
// schema would create the tables it is checking for and pass anything.
func TestValidateBackupRejectsAForeignDatabaseWithoutTouchingIt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "foreign.db")
	sqlDB, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatalf("open foreign: %v", err)
	}
	if _, err := sqlDB.Exec(`CREATE TABLE unrelated (id TEXT)`); err != nil {
		t.Fatalf("seed foreign: %v", err)
	}
	sqlDB.Close()

	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateBackup(path); err == nil {
		t.Fatal("a database with none of Reeve's tables was accepted")
	}
	after, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if after.Size() != before.Size() {
		t.Errorf("validation wrote to the candidate: %d -> %d bytes", before.Size(), after.Size())
	}
}

func TestForeignKeysEnforced(t *testing.T) {
	db := openTemp(t)
	_, err := db.SQL().Exec(
		"INSERT INTO sessions(token_hash, user_id, expires_at, created_at) VALUES ('s1','nouser','x','y')",
	)
	if err == nil {
		t.Fatal("expected foreign key violation inserting session for missing user")
	}
}

func TestListPublicToolsExcludesRestricted(t *testing.T) {
	db := openTemp(t)
	user, err := db.CreateUser("user-1@example.com", "hash", RoleBasic)
	if err != nil {
		t.Fatal(err)
	}
	creator := user.ID
	if _, err := db.CreateTool(Tool{Name: "PublicGrafana", CreatorID: creator, Visibility: VisibilityPublic}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.CreateTool(Tool{Name: "SecretVault", CreatorID: creator, Visibility: VisibilityRestricted}); err != nil {
		t.Fatal(err)
	}

	tools, err := db.ListPublicTools(ToolFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(tools) != 1 {
		t.Fatalf("expected 1 public tool, got %d", len(tools))
	}
	if tools[0].Name != "PublicGrafana" {
		t.Fatalf("expected PublicGrafana, got %q", tools[0].Name)
	}
}

func TestListPublicHostsOnlyReferencedByPublicTools(t *testing.T) {
	db := openTemp(t)
	user, err := db.CreateUser("user-2@example.com", "hash", RoleBasic)
	if err != nil {
		t.Fatal(err)
	}
	pubHost, err := db.CreateHost("pub-host", "linux", "", "hash-pub", 60)
	if err != nil {
		t.Fatal(err)
	}
	secretHost, err := db.CreateHost("secret-host", "linux", "", "hash-secret", 60)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.CreateTool(Tool{Name: "PubTool", CreatorID: user.ID, Visibility: VisibilityPublic, HostID: pubHost.ID}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.CreateTool(Tool{Name: "SecretTool", CreatorID: user.ID, Visibility: VisibilityRestricted, HostID: secretHost.ID}); err != nil {
		t.Fatal(err)
	}

	hosts, err := db.ListPublicHosts()
	if err != nil {
		t.Fatal(err)
	}
	if len(hosts) != 1 || hosts[0].Name != "pub-host" {
		t.Fatalf("expected only pub-host, got %+v", hosts)
	}
}

func TestCountToolsAndHosts(t *testing.T) {
	db := openTemp(t)
	u, err := db.CreateUser("boss@example.com", "hash", "admin")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.CreateTool(Tool{Name: "A", CreatorID: u.ID, Visibility: VisibilityPublic}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.CreateTool(Tool{Name: "B", CreatorID: u.ID, Visibility: VisibilityPublic}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.CreateHost("h1", "linux", "", "hash1", 60); err != nil {
		t.Fatal(err)
	}

	tools, err := db.CountTools()
	if err != nil || tools != 2 {
		t.Fatalf("CountTools = %d, %v; want 2", tools, err)
	}
	hosts, err := db.CountHosts()
	if err != nil || hosts != 1 {
		t.Fatalf("CountHosts = %d, %v; want 1", hosts, err)
	}
}

func TestThresholdSetOverrideAndFallback(t *testing.T) {
	db := openTemp(t)
	loadSet := func() ThresholdSet {
		t.Helper()
		set, err := db.LoadThresholds()
		if err != nil {
			t.Fatalf("LoadThresholds: %v", err)
		}
		return set
	}

	// the schema seeds global cpu=90 and load disabled at 4
	set := loadSet()
	if got, ok := set.Effective("host-x", "cpu"); !ok || got.Value != 90 || !got.Enabled {
		t.Fatalf("global cpu default = %+v ok=%v, want 90 enabled", got, ok)
	}
	if got, ok := set.Effective("host-x", "load"); !ok || got.Enabled {
		t.Errorf("global load = %+v ok=%v, want present and disabled", got, ok)
	}
	if got, ok := set.Effective("host-x", "nosuch"); ok {
		t.Errorf("unknown metric = %+v ok=%v, want absent", got, ok)
	}

	if err := db.SetThreshold("host-x", "cpu", true, 55); err != nil {
		t.Fatal(err)
	}
	set = loadSet()
	got, ok := set.Effective("host-x", "cpu")
	if !ok || got.Value != 55 || !got.Enabled {
		t.Fatalf("override cpu = %+v ok=%v, want 55 enabled", got, ok)
	}
	// The override belongs to that host only.
	if other, ok := set.Effective("host-y", "cpu"); !ok || other.Value != 90 {
		t.Errorf("other host cpu = %+v ok=%v, want the global 90", other, ok)
	}
}

func TestLatestHostMetric(t *testing.T) {
	db := openTemp(t)
	h, err := db.CreateHost("h", "linux", "", "hh", 60)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.InsertHostMetric(h.ID, contracts.HostMetrics{CPUPct: 42, MemUsed: 5, MemTotal: 10}, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	m, ok := db.LatestHostMetric(h.ID)
	if !ok || m.CPUPct != 42 || m.MemTotal != 10 {
		t.Fatalf("latest = %+v ok=%v", m, ok)
	}
	if _, ok := db.LatestHostMetric("nope"); ok {
		t.Errorf("expected no metric for unknown host")
	}
}

func TestHostMetricLoadRoundTrip(t *testing.T) {
	db := openTemp(t)
	h, _ := db.CreateHost("h", "linux", "", "hh", 60)
	if err := db.InsertHostMetric(h.ID, contracts.HostMetrics{Load1: 1.5, Load5: 1.2, Load15: 0.9, MemTotal: 1}, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	m, ok := db.LatestHostMetric(h.ID)
	if !ok || m.Load1 != 1.5 || m.Load15 != 0.9 {
		t.Fatalf("latest load = %+v ok=%v", m, ok)
	}
}

func TestHostMetricGPURoundTrip(t *testing.T) {
	db := openTemp(t)
	h, _ := db.CreateHost("h", "linux", "", "hh", 60)
	if err := db.InsertHostMetric(h.ID, contracts.HostMetrics{GPUUtil: 50, GPUMemUsed: 1000, GPUMemTotal: 4000}, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	m, ok := db.LatestHostMetric(h.ID)
	if !ok || m.GPUUtil != 50 || m.GPUMemUsed != 1000 || m.GPUMemTotal != 4000 {
		t.Fatalf("latest gpu = %+v ok=%v", m, ok)
	}
}

func TestListAlertEventsForHostAndTool(t *testing.T) {
	db := openTemp(t)
	now := time.Now().UTC()
	if _, err := db.CreateAlertEvent(AlertEvent{SubjectKey: "k1", HostID: "h1", Type: "down", Severity: "error", Message: "m1"}, now); err != nil {
		t.Fatal(err)
	}
	if _, err := db.CreateAlertEvent(AlertEvent{SubjectKey: "k2", HostID: "h1", ToolID: "t1", Type: "log_error", Severity: "error", Message: "m2"}, now); err != nil {
		t.Fatal(err)
	}
	if _, err := db.CreateAlertEvent(AlertEvent{SubjectKey: "k3", HostID: "h2", Type: "down", Severity: "error", Message: "m3"}, now); err != nil {
		t.Fatal(err)
	}
	h1, err := db.ListAlertEventsForHost("h1", 50)
	if err != nil || len(h1) != 2 {
		t.Fatalf("host h1 events = %d, %v; want 2", len(h1), err)
	}
	t1, err := db.ListAlertEventsForTool("t1", 50)
	if err != nil || len(t1) != 1 {
		t.Fatalf("tool t1 events = %d, %v; want 1", len(t1), err)
	}
}

func TestDowntimeSecs(t *testing.T) {
	db := openTemp(t)
	now := time.Now().UTC()
	since := now.Add(-time.Hour)

	inWindowStart := now.Add(-30 * time.Minute)
	inWindowEnd := inWindowStart.Add(100 * time.Second)
	e1, err := db.CreateAlertEvent(AlertEvent{SubjectKey: "k1", ToolID: "t1", Type: "down", Severity: "error", Message: "m"}, inWindowStart)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.ResolveAlertEvent(e1.ID, inWindowEnd); err != nil {
		t.Fatal(err)
	}

	outsideStart := now.Add(-3 * time.Hour)
	outsideEnd := outsideStart.Add(50 * time.Second)
	e2, err := db.CreateAlertEvent(AlertEvent{SubjectKey: "k2", ToolID: "t1", Type: "down", Severity: "error", Message: "m"}, outsideStart)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.ResolveAlertEvent(e2.ID, outsideEnd); err != nil {
		t.Fatal(err)
	}

	unresolvedStart := now.Add(-10 * time.Second)
	if _, err := db.CreateAlertEvent(AlertEvent{SubjectKey: "k3", ToolID: "t1", Type: "down", Severity: "error", Message: "m"}, unresolvedStart); err != nil {
		t.Fatal(err)
	}

	secs, err := db.DowntimeSecs("tool_id", "t1", []string{"down"}, since, now)
	if err != nil {
		t.Fatal(err)
	}
	want := 100.0 + 10.0
	if secs < want-1 || secs > want+1 {
		t.Errorf("DowntimeSecs = %v, want ~%v", secs, want)
	}
}

func TestEffectiveRetentionDefaults(t *testing.T) {
	db := openTemp(t)
	got := db.EffectiveRetention()
	if got != DefaultRetention {
		t.Errorf("EffectiveRetention on fresh DB = %+v, want %+v", got, DefaultRetention)
	}
}

func TestEffectiveRetentionRoundTrip(t *testing.T) {
	db := openTemp(t)
	if err := db.SetSetting("retention.raw_secs", "3600"); err != nil {
		t.Fatal(err)
	}
	if err := db.SetSetting("retention.fivemin_secs", "7200"); err != nil {
		t.Fatal(err)
	}
	if err := db.SetSetting("retention.onehour_secs", "10800"); err != nil {
		t.Fatal(err)
	}
	got := db.EffectiveRetention()
	if got.Raw != time.Hour {
		t.Errorf("Raw = %v, want %v", got.Raw, time.Hour)
	}
	if got.FiveMin != 2*time.Hour {
		t.Errorf("FiveMin = %v, want %v", got.FiveMin, 2*time.Hour)
	}
	if got.OneHour != 3*time.Hour {
		t.Errorf("OneHour = %v, want %v", got.OneHour, 3*time.Hour)
	}
}

func TestEffectiveRetentionUnparseableFallsBack(t *testing.T) {
	db := openTemp(t)
	if err := db.SetSetting("retention.raw_secs", "abc"); err != nil {
		t.Fatal(err)
	}
	if err := db.SetSetting("retention.fivemin_secs", "7200"); err != nil {
		t.Fatal(err)
	}
	got := db.EffectiveRetention()
	if got.Raw != DefaultRetention.Raw {
		t.Errorf("Raw = %v, want default %v", got.Raw, DefaultRetention.Raw)
	}
	if got.FiveMin != 2*time.Hour {
		t.Errorf("FiveMin = %v, want %v", got.FiveMin, 2*time.Hour)
	}
	if got.OneHour != DefaultRetention.OneHour {
		t.Errorf("OneHour = %v, want default %v", got.OneHour, DefaultRetention.OneHour)
	}
}

func TestBackupTo(t *testing.T) {
	db := openTemp(t)
	if _, err := db.CreateHost("host-a", "linux", "rack-1", "hash", 60); err != nil {
		t.Fatalf("CreateHost: %v", err)
	}
	dest := filepath.Join(t.TempDir(), "copy.db")
	if err := db.BackupTo(dest); err != nil {
		t.Fatalf("BackupTo: %v", err)
	}

	copyDB, err := Open(dest)
	if err != nil {
		t.Fatalf("open copy: %v", err)
	}
	defer copyDB.Close()
	n, err := copyDB.CountHosts()
	if err != nil {
		t.Fatalf("CountHosts: %v", err)
	}
	if n != 1 {
		t.Errorf("copy CountHosts = %d, want 1", n)
	}
	var result string
	if err := copyDB.SQL().QueryRow("PRAGMA integrity_check").Scan(&result); err != nil {
		t.Fatalf("integrity_check: %v", err)
	}
	if result != "ok" {
		t.Errorf("integrity_check = %q, want ok", result)
	}
}

func TestBackupToRefusesExistingDest(t *testing.T) {
	db := openTemp(t)
	dir := t.TempDir()
	dest := filepath.Join(dir, "copy.db")
	if err := db.BackupTo(dest); err != nil {
		t.Fatalf("first BackupTo: %v", err)
	}
	if err := db.BackupTo(dest); err == nil {
		t.Error("BackupTo to an existing path should fail")
	}
}

func TestApplyStagedRestore(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "reeve.db")

	live, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open live: %v", err)
	}
	if _, err := live.CreateHost("old-host", "linux", "", "hash", 60); err != nil {
		t.Fatalf("CreateHost: %v", err)
	}
	live.Close()

	// Build a staged restore with a different host set.
	source, err := Open(filepath.Join(dir, "source.db"))
	if err != nil {
		t.Fatalf("open source: %v", err)
	}
	if _, err := source.CreateHost("restored-host", "linux", "", "hash", 60); err != nil {
		t.Fatalf("CreateHost source: %v", err)
	}
	if err := source.BackupTo(dbPath + ".restore"); err != nil {
		t.Fatalf("BackupTo staged: %v", err)
	}
	source.Close()

	if err := ApplyStagedRestore(dbPath); err != nil {
		t.Fatalf("ApplyStagedRestore: %v", err)
	}
	if _, err := os.Stat(dbPath + ".restore"); !os.IsNotExist(err) {
		t.Errorf(".restore file should be gone, stat err = %v", err)
	}

	restored, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open restored: %v", err)
	}
	defer restored.Close()
	var name string
	if err := restored.SQL().QueryRow("SELECT name FROM hosts").Scan(&name); err != nil {
		t.Fatalf("read restored host: %v", err)
	}
	if name != "restored-host" {
		t.Errorf("restored host = %q, want restored-host", name)
	}
}

func TestApplyStagedRestoreNoOp(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "reeve.db")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if _, err := db.CreateHost("keep", "linux", "", "hash", 60); err != nil {
		t.Fatalf("CreateHost: %v", err)
	}
	db.Close()

	if err := ApplyStagedRestore(dbPath); err != nil {
		t.Fatalf("ApplyStagedRestore (no staged) = %v, want nil", err)
	}
	db, err = Open(dbPath)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer db.Close()
	n, err := db.CountHosts()
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("original DB altered: CountHosts = %d, want 1", n)
	}
}

func TestDueDeliveriesResolvesWebhook(t *testing.T) {
	db := openTemp(t)
	now := time.Now().UTC()

	wh, err := db.CreateWebhook(Webhook{OwnerType: "global", URL: "https://sink.invalid/hook"})
	if err != nil {
		t.Fatalf("CreateWebhook: %v", err)
	}
	if err := db.EnqueueDelivery(wh.ID, `{"a":1}`, now); err != nil {
		t.Fatalf("EnqueueDelivery: %v", err)
	}

	due, err := db.DueDeliveries(now.Add(time.Second), 50)
	if err != nil {
		t.Fatalf("DueDeliveries: %v", err)
	}
	if len(due) != 1 {
		t.Fatalf("expected 1 due delivery, got %d", len(due))
	}
	if due[0].URL != "https://sink.invalid/hook" {
		t.Errorf("webhook URL not resolved: %+v", due[0])
	}
}

func TestDeliveryBackoff(t *testing.T) {
	db := openTemp(t)
	now := time.Now().UTC()
	wh, _ := db.CreateWebhook(Webhook{OwnerType: "global", URL: "https://sink.invalid/hook"})
	db.EnqueueDelivery(wh.ID, `{"a":1}`, now)

	due, _ := db.DueDeliveries(now.Add(time.Second), 10)
	if len(due) != 1 {
		t.Fatalf("expected 1 due, got %d", len(due))
	}
	if err := db.MarkDeliveryFailed(due[0].ID, "boom", due[0].Attempts, now); err != nil {
		t.Fatalf("MarkDeliveryFailed: %v", err)
	}
	// Immediately after failure the backoff should hold it back.
	again, _ := db.DueDeliveries(now, 10)
	if len(again) != 0 {
		t.Fatalf("expected backoff to defer the delivery, got %d due", len(again))
	}
	// After the backoff window it is due again.
	later, _ := db.DueDeliveries(now.Add(10*time.Second), 10)
	if len(later) != 1 {
		t.Fatalf("expected delivery due after backoff, got %d", len(later))
	}
}
