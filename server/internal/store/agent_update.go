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

// blockingSlot matches, for the given table alias, a host whose rollout slot
// still counts toward the fleet. A host that can no longer update is excluded:
// its slot is dangling, and letting it read as stalled would halt every other
// host behind a row the UI shows as "updates off". A policy of `default`
// blocks only when fleetDefault is on, mirroring effectiveAutoUpdate, or a
// stale slot on a `default` host would keep blocking after an admin turns the
// fleet toggle off.
func blockingSlot(alias string, fleetDefault bool) string {
	def := "0"
	if fleetDefault {
		def = "1"
	}
	return alias + `.id != '` + ServerHostID + `'` +
		` AND ` + alias + `.update_started_at IS NOT NULL` +
		` AND ` + alias + `.auto_update != '` + AutoUpdateOff + `'` +
		` AND ` + alias + `.auto_update_vetoed = 0` +
		` AND (` + alias + `.auto_update = '` + AutoUpdateOn + `' OR ` + def + ` = 1)`
}

// SetHostAutoUpdate sets a host's auto-update policy override. Setting it off
// also releases any slot the host holds, or a host taken out of the rollout
// would keep halting it.
func (db *DB) SetHostAutoUpdate(id, policy string) error {
	if policy == AutoUpdateOff {
		return db.exec1(`UPDATE hosts SET auto_update = ?, update_started_at = NULL WHERE id = ?`, policy, id)
	}
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
func (db *DB) TryStartHostUpdate(id string, at, cutoff time.Time, concurrency int, fleetDefault bool) (bool, error) {
	res, err := db.sql.Exec(`
		UPDATE hosts SET update_started_at = ?
		WHERE id = ?
		  AND NOT EXISTS (
		    SELECT 1 FROM hosts s
		    WHERE `+blockingSlot("s", fleetDefault)+` AND s.update_started_at < ?
		  )
		  AND (
		    SELECT COUNT(*) FROM hosts l
		    WHERE `+blockingSlot("l", fleetDefault)+` AND l.update_started_at >= ?
		  ) < ?`,
		at.UTC().Format(slotStamp), id,
		cutoff.UTC().Format(slotStamp),
		cutoff.UTC().Format(slotStamp), concurrency)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n == 1, nil
}

// StalledHost identifies a host that wedged a rollout. The ID lets the UI link
// the banner straight to the page where an admin can take it out of the rollout.
type StalledHost struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ListStalledHosts names the hosts that wedged a rollout, for the UI banner.
func (db *DB) ListStalledHosts(cutoff time.Time, fleetDefault bool) ([]StalledHost, error) {
	rows, err := db.sql.Query(
		`SELECT id, name FROM hosts
		 WHERE `+blockingSlot("hosts", fleetDefault)+` AND update_started_at < ?
		 ORDER BY name`,
		cutoff.UTC().Format(slotStamp))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []StalledHost{}
	for rows.Next() {
		var h StalledHost
		if err := rows.Scan(&h.ID, &h.Name); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

// ClearStalledUpdates releases every slot stamped before cutoff so a paused
// rollout can resume. It is deliberately blind to policy: a dangling slot on a
// host that can no longer update is exactly what resume should also sweep away.
func (db *DB) ClearStalledUpdates(cutoff time.Time) error {
	_, err := db.sql.Exec(
		`UPDATE hosts SET update_started_at = NULL
		 WHERE id != ? AND update_started_at IS NOT NULL AND update_started_at < ?`,
		ServerHostID, cutoff.UTC().Format(slotStamp))
	return err
}
