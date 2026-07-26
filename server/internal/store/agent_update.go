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

// TryStartHostUpdate grants a rollout slot in a single conditional write, so
// two concurrent pushes cannot both observe spare capacity and both stamp.
// It refuses if any host is stalled or the live count is already at cap.
func (db *DB) TryStartHostUpdate(id string, at, cutoff time.Time, concurrency int) (bool, error) {
	res, err := db.sql.Exec(`
		UPDATE hosts SET update_started_at = ?
		WHERE id = ?
		  AND NOT EXISTS (
		    SELECT 1 FROM hosts s
		    WHERE s.id != ? AND s.update_started_at IS NOT NULL AND s.update_started_at < ?
		  )
		  AND (
		    SELECT COUNT(*) FROM hosts l
		    WHERE l.id != ? AND l.update_started_at IS NOT NULL AND l.update_started_at >= ?
		  ) < ?`,
		at.UTC().Format(slotStamp), id,
		ServerHostID, cutoff.UTC().Format(slotStamp),
		ServerHostID, cutoff.UTC().Format(slotStamp), concurrency)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n == 1, nil
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
