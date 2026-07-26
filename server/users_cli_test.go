package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/thehelvijs/Reeve/server/internal/auth"
	"github.com/thehelvijs/Reeve/server/internal/store"
)

// cliEnv is a temp database plus a helper that runs `users …` against it.
type cliEnv struct {
	t      *testing.T
	dbPath string
}

func newCLIEnv(t *testing.T) *cliEnv {
	t.Helper()
	return &cliEnv{t: t, dbPath: filepath.Join(t.TempDir(), "cli.db")}
}

// run executes one users action, returning its output and error.
func (e *cliEnv) run(args ...string) (string, error) {
	e.t.Helper()
	var out bytes.Buffer
	err := runUsersCLI(&out, e.dbPath, args)
	return out.String(), err
}

// mustRun fails the test if the action errors.
func (e *cliEnv) mustRun(args ...string) string {
	e.t.Helper()
	out, err := e.run(args...)
	if err != nil {
		e.t.Fatalf("users %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return out
}

// db opens the database for assertions.
func (e *cliEnv) db() *store.DB {
	e.t.Helper()
	db, err := store.Open(e.dbPath)
	if err != nil {
		e.t.Fatalf("open db: %v", err)
	}
	e.t.Cleanup(func() { db.Close() })
	return db
}

func (e *cliEnv) user(email string) store.User {
	e.t.Helper()
	u, err := e.db().GetUserByEmail(email)
	if err != nil {
		e.t.Fatalf("user %s missing: %v", email, err)
	}
	return u
}

func TestUsersCLICreateAndList(t *testing.T) {
	e := newCLIEnv(t)

	if out := e.mustRun("list"); !strings.Contains(out, "no accounts yet") {
		t.Errorf("list on an empty database = %q", out)
	}
	e.mustRun("create", "Boss@Example.com", "-password", "password123", "-role", "admin", "-name", "The Boss")
	e.mustRun("create", "dev@example.com", "-password", "password123")

	boss := e.user("boss@example.com")
	if boss.Role != store.RoleAdmin {
		t.Errorf("role = %q, want admin", boss.Role)
	}
	if boss.DisplayName != "The Boss" {
		t.Errorf("display name = %q", boss.DisplayName)
	}
	if !auth.VerifyPassword("password123", boss.PasswordHash) {
		t.Error("password does not verify")
	}
	if dev := e.user("dev@example.com"); dev.Role != store.RoleBasic {
		t.Errorf("default role = %q, want basic", dev.Role)
	}

	out := e.mustRun("list")
	for _, want := range []string{"EMAIL", "boss@example.com", "admin", "active", "The Boss", "dev@example.com"} {
		if !strings.Contains(out, want) {
			t.Errorf("list output missing %q:\n%s", want, out)
		}
	}
}

func TestUsersCLICreateRejects(t *testing.T) {
	e := newCLIEnv(t)
	e.mustRun("create", "boss@example.com", "-password", "password123", "-role", "admin")

	cases := map[string][]string{
		"no email":       {"create"},
		"not an email":   {"create", "nonsense", "-password", "password123"},
		"no password":    {"create", "new@example.com"},
		"bad role":       {"create", "new@example.com", "-password", "password123", "-role", "owner"},
		"duplicate":      {"create", "boss@example.com", "-password", "password123"},
		"unknown action": {"frobnicate"},
	}
	for name, args := range cases {
		if _, err := e.run(args...); err == nil {
			t.Errorf("%s: expected an error, got nil", name)
		}
	}
	if _, err := e.run(); err == nil {
		t.Error("no action: expected an error, got nil")
	}
}

func TestUsersCLISetPasswordSignsSessionsOut(t *testing.T) {
	e := newCLIEnv(t)
	e.mustRun("create", "boss@example.com", "-password", "old-password", "-role", "admin")
	u := e.user("boss@example.com")
	sess, err := e.db().CreateSession(u.ID, time.Hour)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	e.mustRun("set-password", "BOSS@example.com", "-password", "new-password")

	after := e.user("boss@example.com")
	if !auth.VerifyPassword("new-password", after.PasswordHash) {
		t.Error("new password does not verify")
	}
	if auth.VerifyPassword("old-password", after.PasswordHash) {
		t.Error("old password still verifies")
	}
	if _, err := e.db().GetSession(sess.ID); err == nil {
		t.Error("session survived the password change")
	}
	if _, err := e.run("set-password", "boss@example.com"); err == nil {
		t.Error("missing -password: expected an error")
	}
	if _, err := e.run("set-password", "nobody@example.com", "-password", "password123"); err == nil {
		t.Error("unknown email: expected an error")
	}
}

func TestUsersCLISetRole(t *testing.T) {
	e := newCLIEnv(t)
	e.mustRun("create", "boss@example.com", "-password", "password123", "-role", "admin")
	e.mustRun("create", "dev@example.com", "-password", "password123")

	e.mustRun("set-role", "dev@example.com", "admin")
	if e.user("dev@example.com").Role != store.RoleAdmin {
		t.Error("promotion did not take")
	}
	if out := e.mustRun("set-role", "dev@example.com", "admin"); !strings.Contains(out, "already") {
		t.Errorf("re-promoting reported %q", out)
	}
	e.mustRun("set-role", "dev@example.com", "basic")
	if e.user("dev@example.com").Role != store.RoleBasic {
		t.Error("demotion did not take")
	}
	for name, args := range map[string][]string{
		"no role":      {"set-role", "dev@example.com"},
		"invalid role": {"set-role", "dev@example.com", "owner"},
		"unknown user": {"set-role", "nobody@example.com", "admin"},
	} {
		if _, err := e.run(args...); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

// The whole point of this CLI is fixing lockouts, so it must not create one.
func TestUsersCLIRefusesToRemoveTheLastActiveAdmin(t *testing.T) {
	e := newCLIEnv(t)
	e.mustRun("create", "boss@example.com", "-password", "password123", "-role", "admin")
	e.mustRun("create", "dev@example.com", "-password", "password123")

	if _, err := e.run("set-role", "boss@example.com", "basic"); err == nil {
		t.Error("demoting the only admin was allowed")
	}
	if _, err := e.run("disable", "boss@example.com"); err == nil {
		t.Error("disabling the only admin was allowed")
	}
	if _, err := e.run("delete", "boss@example.com", "-yes"); err == nil {
		t.Error("deleting the only admin was allowed")
	}
	if e.user("boss@example.com").Role != store.RoleAdmin || !e.user("boss@example.com").Active {
		t.Fatal("the only admin was changed despite the refusals")
	}

	// With a second admin, the same actions are allowed.
	e.mustRun("set-role", "dev@example.com", "admin")
	e.mustRun("disable", "boss@example.com")
	if e.user("boss@example.com").Active {
		t.Error("disable did not take once another admin existed")
	}
	// And now dev is the last active admin, so disabling it is refused.
	if _, err := e.run("disable", "dev@example.com"); err == nil {
		t.Error("disabling the last active admin was allowed")
	}
}

func TestUsersCLIEnableDisable(t *testing.T) {
	e := newCLIEnv(t)
	e.mustRun("create", "boss@example.com", "-password", "password123", "-role", "admin")
	e.mustRun("create", "dev@example.com", "-password", "password123")

	e.mustRun("disable", "dev@example.com")
	if e.user("dev@example.com").Active {
		t.Error("disable did not take")
	}
	if out := e.mustRun("disable", "dev@example.com"); !strings.Contains(out, "already disabled") {
		t.Errorf("second disable reported %q", out)
	}
	e.mustRun("enable", "dev@example.com")
	if !e.user("dev@example.com").Active {
		t.Error("enable did not take")
	}
	if out := e.mustRun("enable", "dev@example.com"); !strings.Contains(out, "already enabled") {
		t.Errorf("second enable reported %q", out)
	}
}

func TestUsersCLISetName(t *testing.T) {
	e := newCLIEnv(t)
	e.mustRun("create", "dev@example.com", "-password", "password123")

	e.mustRun("set-name", "dev@example.com", "Dev", "Example")
	if got := e.user("dev@example.com").DisplayName; got != "Dev Example" {
		t.Errorf("display name = %q, want %q", got, "Dev Example")
	}
	e.mustRun("set-name", "dev@example.com", "")
	if got := e.user("dev@example.com").DisplayName; got != "" {
		t.Errorf("display name = %q, want it cleared", got)
	}
	if _, err := e.run("set-name", "dev@example.com"); err == nil {
		t.Error("missing name: expected an error")
	}
	if _, err := e.run("set-name", "dev@example.com", strings.Repeat("x", maxDisplayNameLen+1)); err == nil {
		t.Error("over-long name: expected an error")
	}
}

func TestUsersCLIDeleteReassignsOwnership(t *testing.T) {
	e := newCLIEnv(t)
	e.mustRun("create", "boss@example.com", "-password", "password123", "-role", "admin")
	e.mustRun("create", "dev@example.com", "-password", "password123")
	db := e.db()
	dev := e.user("dev@example.com")
	boss := e.user("boss@example.com")
	tool, err := db.CreateTool(store.Tool{Name: "Grafana", Address: "10.0.0.5", Port: 3000, CreatorID: dev.ID})
	if err != nil {
		t.Fatalf("create tool: %v", err)
	}

	// Without -yes nothing happens.
	if _, err := e.run("delete", "dev@example.com"); err == nil {
		t.Error("delete without -yes was allowed")
	}
	if _, err := db.GetUserByEmail("dev@example.com"); err != nil {
		t.Fatal("the account was deleted despite the missing -yes")
	}

	out := e.mustRun("delete", "dev@example.com", "-yes")
	if !strings.Contains(out, "boss@example.com") {
		t.Errorf("output does not name the heir: %q", out)
	}
	if _, err := db.GetUserByEmail("dev@example.com"); err == nil {
		t.Error("account still exists after delete")
	}
	after, err := db.GetTool(tool.ID)
	if err != nil {
		t.Fatalf("tool gone after deleting its creator: %v", err)
	}
	if after.CreatorID != boss.ID {
		t.Errorf("tool creator = %q, want the remaining admin %q", after.CreatorID, boss.ID)
	}
}

func TestUsersCLIDeleteWithExplicitHeir(t *testing.T) {
	e := newCLIEnv(t)
	e.mustRun("create", "boss@example.com", "-password", "password123", "-role", "admin")
	e.mustRun("create", "dev@example.com", "-password", "password123")
	e.mustRun("create", "ops@example.com", "-password", "password123")
	db := e.db()
	dev := e.user("dev@example.com")
	ops := e.user("ops@example.com")
	tool, err := db.CreateTool(store.Tool{Name: "Postgres", Address: "10.0.0.6", Port: 5432, CreatorID: dev.ID})
	if err != nil {
		t.Fatalf("create tool: %v", err)
	}

	e.mustRun("delete", "dev@example.com", "-reassign-to", "OPS@example.com", "-yes")
	after, err := db.GetTool(tool.ID)
	if err != nil {
		t.Fatalf("get tool: %v", err)
	}
	if after.CreatorID != ops.ID {
		t.Errorf("tool creator = %q, want %q", after.CreatorID, ops.ID)
	}

	for name, args := range map[string][]string{
		"unknown heir": {"delete", "ops@example.com", "-reassign-to", "nobody@example.com", "-yes"},
		"self heir":    {"delete", "ops@example.com", "-reassign-to", "ops@example.com", "-yes"},
		"unknown user": {"delete", "nobody@example.com", "-yes"},
	} {
		if _, err := e.run(args...); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}
