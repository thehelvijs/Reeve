package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/thehelvijs/Reeve/server/internal/auth"
	"github.com/thehelvijs/Reeve/server/internal/store"
)

// usersUsage is printed for `server users` with no or an unknown action.
const usersUsage = `manage accounts from a shell on the server, without the web UI:

  server users list
  server users create <email> -password <pw> [-role admin|basic] [-name "Full Name"]
  server users set-password <email> -password <pw>
  server users set-role <email> <admin|basic>
  server users set-name <email> "Full Name"
  server users enable <email>
  server users disable <email>
  server users delete <email> [-reassign-to <email>] -yes

The database is $REEVE_DB (default reeve.db). No master key is needed: none
of these actions touch encrypted data. Changing or clearing a password signs that
account out everywhere.
`

// runUsersCLI dispatches `server users <action>`. It opens the database
// directly, so it works while the server is running and before the master key is
// available.
func runUsersCLI(out io.Writer, dbPath string, args []string) error {
	if len(args) == 0 {
		fmt.Fprint(out, usersUsage)
		return errors.New("users: an action is required")
	}
	action, rest := args[0], args[1:]

	db, err := store.Open(dbPath)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	switch action {
	case "list":
		return usersList(out, db)
	case "create":
		return usersCreate(out, db, rest)
	case "set-password":
		return usersSetPassword(out, db, rest)
	case "set-role":
		return usersSetRole(out, db, rest)
	case "set-name":
		return usersSetName(out, db, rest)
	case "enable":
		return usersSetActive(out, db, rest, true)
	case "disable":
		return usersSetActive(out, db, rest, false)
	case "delete":
		return usersDelete(out, db, rest)
	default:
		fmt.Fprint(out, usersUsage)
		return fmt.Errorf("users: unknown action %q", action)
	}
}

// firstArg pulls the leading positional argument (an email) and returns the
// remaining flag arguments.
func firstArg(args []string) (string, []string, error) {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return "", nil, errors.New("an email address is required")
	}
	return normalizeEmail(args[0]), args[1:], nil
}

func normalizeEmail(s string) string {
	return strings.TrimSpace(strings.ToLower(s))
}

// lookup resolves an email to a user with a message naming the address, since a
// typo is the likeliest failure here.
func lookup(db *store.DB, email string) (store.User, error) {
	u, err := db.GetUserByEmail(email)
	if err != nil {
		return store.User{}, fmt.Errorf("no account with email %q", email)
	}
	return u, nil
}

func usersList(out io.Writer, db *store.DB) error {
	users, err := db.ListUsers()
	if err != nil {
		return err
	}
	if len(users) == 0 {
		fmt.Fprintln(out, "no accounts yet; the first account created becomes an admin")
		return nil
	}
	tw := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "EMAIL\tROLE\tSTATUS\tNAME\tCREATED")
	for _, u := range users {
		status := "active"
		if !u.Active {
			status = "disabled"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
			u.Email, u.Role, status, u.DisplayName, u.CreatedAt.Format(time.RFC3339))
	}
	return tw.Flush()
}

func usersCreate(out io.Writer, db *store.DB, args []string) error {
	email, rest, err := firstArg(args)
	if err != nil {
		return err
	}
	fs := flag.NewFlagSet("users create", flag.ContinueOnError)
	fs.SetOutput(out)
	password := fs.String("password", "", "the account's password")
	role := fs.String("role", store.RoleBasic, "admin or basic")
	name := fs.String("name", "", "display name")
	if err := fs.Parse(rest); err != nil {
		return err
	}
	if !strings.Contains(email, "@") {
		return fmt.Errorf("%q is not an email address", email)
	}
	if len(*password) < minPasswordLen {
		return fmt.Errorf("-password must be at least %d characters", minPasswordLen)
	}
	if *role != store.RoleAdmin && *role != store.RoleBasic {
		return fmt.Errorf("-role must be %q or %q", store.RoleAdmin, store.RoleBasic)
	}
	if _, err := db.GetUserByEmail(email); err == nil {
		return fmt.Errorf("an account with email %q already exists", email)
	}
	hash, err := auth.HashPassword(*password)
	if err != nil {
		return err
	}
	u, err := db.CreateUser(email, hash, *role)
	if err != nil {
		return err
	}
	if *name != "" {
		if err := db.SetUserDisplayName(u.ID, *name); err != nil {
			return err
		}
	}
	fmt.Fprintf(out, "created %s as %s\n", u.Email, u.Role)
	return nil
}

func usersSetPassword(out io.Writer, db *store.DB, args []string) error {
	email, rest, err := firstArg(args)
	if err != nil {
		return err
	}
	fs := flag.NewFlagSet("users set-password", flag.ContinueOnError)
	fs.SetOutput(out)
	password := fs.String("password", "", "the new password")
	if err := fs.Parse(rest); err != nil {
		return err
	}
	if *password == "" {
		return errors.New("-password is required")
	}
	u, err := lookup(db, email)
	if err != nil {
		return err
	}
	if err := setPassword(db, u.ID, *password); err != nil {
		return err
	}
	fmt.Fprintf(out, "password for %s set; that account's sessions were signed out\n", u.Email)
	return nil
}

func usersSetRole(out io.Writer, db *store.DB, args []string) error {
	email, rest, err := firstArg(args)
	if err != nil {
		return err
	}
	if len(rest) == 0 {
		return fmt.Errorf("a role is required: %s or %s", store.RoleAdmin, store.RoleBasic)
	}
	role := rest[0]
	if role != store.RoleAdmin && role != store.RoleBasic {
		return fmt.Errorf("role must be %q or %q", store.RoleAdmin, store.RoleBasic)
	}
	u, err := lookup(db, email)
	if err != nil {
		return err
	}
	if u.Role == role {
		fmt.Fprintf(out, "%s is already %s\n", u.Email, role)
		return nil
	}
	if role == store.RoleBasic {
		if err := guardLastActiveAdmin(db, u, "demote"); err != nil {
			return err
		}
	}
	if err := db.SetUserRole(u.ID, role); err != nil {
		return err
	}
	fmt.Fprintf(out, "%s is now %s\n", u.Email, role)
	return nil
}

func usersSetName(out io.Writer, db *store.DB, args []string) error {
	email, rest, err := firstArg(args)
	if err != nil {
		return err
	}
	if len(rest) == 0 {
		return errors.New("a display name is required (use \"\" to clear it)")
	}
	u, err := lookup(db, email)
	if err != nil {
		return err
	}
	name := strings.TrimSpace(strings.Join(rest, " "))
	if len(name) > maxDisplayNameLen {
		return fmt.Errorf("display name must be at most %d characters", maxDisplayNameLen)
	}
	if err := db.SetUserDisplayName(u.ID, name); err != nil {
		return err
	}
	fmt.Fprintf(out, "display name for %s set to %q\n", u.Email, name)
	return nil
}

func usersSetActive(out io.Writer, db *store.DB, args []string, active bool) error {
	email, _, err := firstArg(args)
	if err != nil {
		return err
	}
	u, err := lookup(db, email)
	if err != nil {
		return err
	}
	if u.Active == active {
		state := "already enabled"
		if !active {
			state = "already disabled"
		}
		fmt.Fprintf(out, "%s is %s\n", u.Email, state)
		return nil
	}
	if !active {
		if err := guardLastActiveAdmin(db, u, "disable"); err != nil {
			return err
		}
	}
	if err := db.SetUserActive(u.ID, active); err != nil {
		return err
	}
	word := "enabled"
	if !active {
		word = "disabled"
	}
	fmt.Fprintf(out, "%s %s\n", u.Email, word)
	return nil
}

func usersDelete(out io.Writer, db *store.DB, args []string) error {
	email, rest, err := firstArg(args)
	if err != nil {
		return err
	}
	fs := flag.NewFlagSet("users delete", flag.ContinueOnError)
	fs.SetOutput(out)
	reassignTo := fs.String("reassign-to", "", "account that inherits their tools and credentials (default: another admin)")
	yes := fs.Bool("yes", false, "confirm the deletion")
	if err := fs.Parse(rest); err != nil {
		return err
	}
	u, err := lookup(db, email)
	if err != nil {
		return err
	}
	if !*yes {
		return fmt.Errorf("refusing to delete %s without -yes; their tools and credentials are reassigned and cannot be restored", u.Email)
	}

	heir, err := deletionHeir(db, u, normalizeEmail(*reassignTo))
	if err != nil {
		return err
	}
	if u.Role == store.RoleAdmin {
		n, err := db.CountAdmins()
		if err != nil {
			return err
		}
		if n <= 1 {
			return errors.New("refusing to delete the only admin account; create another admin first")
		}
	}
	if err := db.DeleteUser(u.ID, heir.ID); err != nil {
		return err
	}
	fmt.Fprintf(out, "deleted %s; their tools and credentials now belong to %s\n", u.Email, heir.Email)
	return nil
}

// deletionHeir resolves who inherits the deleted account's tools and
// credentials: the caller's choice, or the oldest remaining admin.
func deletionHeir(db *store.DB, target store.User, reassignTo string) (store.User, error) {
	if reassignTo != "" {
		heir, err := lookup(db, reassignTo)
		if err != nil {
			return store.User{}, fmt.Errorf("-reassign-to: %w", err)
		}
		if heir.ID == target.ID {
			return store.User{}, errors.New("-reassign-to must be a different account")
		}
		return heir, nil
	}
	heir, err := db.OldestAdminExcluding(target.ID)
	if err != nil {
		return store.User{}, errors.New("no other admin to inherit their tools; pass -reassign-to <email>")
	}
	return heir, nil
}

// guardLastActiveAdmin refuses a change that would leave nobody able to
// administer the instance, which is the lockout this CLI exists to fix.
func guardLastActiveAdmin(db *store.DB, u store.User, verb string) error {
	if u.Role != store.RoleAdmin || !u.Active {
		return nil
	}
	n, err := db.CountActiveAdmins()
	if err != nil {
		return err
	}
	if n <= 1 {
		return fmt.Errorf("refusing to %s the only active admin; promote or enable another admin first", verb)
	}
	return nil
}
