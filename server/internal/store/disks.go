package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/thehelvijs/Reeve/contracts"
)

// HostDisks is a host's most recent filesystem capacity snapshot.
type HostDisks struct {
	TS    time.Time             `json:"ts"`
	Disks []contracts.DiskUsage `json:"disks"`
}

// replaceHostDisks overwrites a host's filesystem snapshot. One row per host:
// this is "what is mounted now and how full it is", not a series to chart, so
// there is no retention or rollup path behind it. The root filesystem is
// already charted in metric_samples for anyone who wants the trend.
func replaceHostDisks(x execer, hostID string, disks []contracts.DiskUsage, ts time.Time) error {
	if disks == nil {
		disks = []contracts.DiskUsage{}
	}
	body, err := json.Marshal(disks)
	if err != nil {
		return err
	}
	_, err = x.Exec(
		`INSERT OR REPLACE INTO host_disks(host_id, ts, disks) VALUES (?,?,?)`,
		hostID, ts.UTC().Format(metricTimeFmt), string(body))
	return err
}

// ReplaceHostDisks overwrites a host's filesystem snapshot.
func (db *DB) ReplaceHostDisks(hostID string, disks []contracts.DiskUsage, ts time.Time) error {
	return replaceHostDisks(db.sql, hostID, disks, ts)
}

// LatestHostDisks returns a host's stored snapshot, false when it has none.
func (db *DB) LatestHostDisks(hostID string) (HostDisks, bool) {
	var ts, body string
	err := db.sql.QueryRow(`SELECT ts, disks FROM host_disks WHERE host_id = ?`, hostID).Scan(&ts, &body)
	if errors.Is(err, sql.ErrNoRows) || err != nil {
		return HostDisks{}, false
	}
	out := HostDisks{Disks: []contracts.DiskUsage{}}
	out.TS, _ = time.Parse(metricTimeFmt, ts)
	if err := json.Unmarshal([]byte(body), &out.Disks); err != nil {
		return HostDisks{}, false
	}
	return out, true
}
