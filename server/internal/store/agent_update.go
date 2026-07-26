package store

import "time"

// Auto-update policy values a host row may carry.
const (
	AutoUpdateDefault = "default"
	AutoUpdateOn      = "on"
	AutoUpdateOff     = "off"
)

// slotStamp is fixed-width so SQLite's string comparison orders slots correctly;
// RFC3339Nano trims trailing zeros and would sort ".5Z" after ".50000001Z".
const slotStamp = time.RFC3339

// ValidAutoUpdatePolicy reports whether p is a policy the hosts table accepts.
func ValidAutoUpdatePolicy(p string) bool {
	return p == AutoUpdateDefault || p == AutoUpdateOn || p == AutoUpdateOff
}

// SetHostAutoUpdate sets a host's auto-update policy override.
func (db *DB) SetHostAutoUpdate(id, policy string) error {
	return db.exec1(`UPDATE hosts SET auto_update = ? WHERE id = ?`, policy, id)
}

// StartHostUpdate stamps a host as holding a rollout slot.
func (db *DB) StartHostUpdate(id string, at time.Time) error {
	return db.exec1(`UPDATE hosts SET update_started_at = ? WHERE id = ?`,
		at.UTC().Format(slotStamp), id)
}

// ClearHostUpdateSlot releases a host's rollout slot.
func (db *DB) ClearHostUpdateSlot(id string) error {
	return db.exec1(`UPDATE hosts SET update_started_at = NULL WHERE id = ?`, id)
}

// CountLiveUpdateSlots counts hosts stamped at or after cutoff, still inside the
// stall window and therefore occupying a slot.
func (db *DB) CountLiveUpdateSlots(cutoff time.Time) (int, error) {
	return db.countUpdateSlots(`update_started_at >= ?`, cutoff)
}

// CountStalledUpdates counts hosts stamped before cutoff: told to update and
// never seen again on the new version.
func (db *DB) CountStalledUpdates(cutoff time.Time) (int, error) {
	return db.countUpdateSlots(`update_started_at < ?`, cutoff)
}

func (db *DB) countUpdateSlots(cond string, cutoff time.Time) (int, error) {
	var n int
	err := db.sql.QueryRow(
		`SELECT COUNT(*) FROM hosts WHERE id != ? AND update_started_at IS NOT NULL AND `+cond,
		ServerHostID, cutoff.UTC().Format(slotStamp)).Scan(&n)
	return n, err
}

// ListStalledHostNames names the hosts that wedged a rollout, for the UI banner.
func (db *DB) ListStalledHostNames(cutoff time.Time) ([]string, error) {
	rows, err := db.sql.Query(
		`SELECT name FROM hosts
		 WHERE id != ? AND update_started_at IS NOT NULL AND update_started_at < ?
		 ORDER BY name`,
		ServerHostID, cutoff.UTC().Format(slotStamp))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		out = append(out, name)
	}
	return out, rows.Err()
}

// ClearStalledUpdates releases every slot stamped before cutoff so a paused
// rollout can resume.
func (db *DB) ClearStalledUpdates(cutoff time.Time) error {
	_, err := db.sql.Exec(
		`UPDATE hosts SET update_started_at = NULL
		 WHERE update_started_at IS NOT NULL AND update_started_at < ?`,
		cutoff.UTC().Format(slotStamp))
	return err
}
