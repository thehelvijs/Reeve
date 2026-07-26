package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/thehelvijs/Reeve/contracts"
)

// HostProcesses is a host's most recent process snapshot.
type HostProcesses struct {
	TS    time.Time                 `json:"ts"`
	Procs []contracts.ProcessSample `json:"procs"`
}

// ReplaceHostProcesses overwrites a host's process snapshot. One row per host:
// this is "what is running now", not a series to chart, so there is no
// retention or rollup path behind it.
func (db *DB) ReplaceHostProcesses(hostID string, procs []contracts.ProcessSample, ts time.Time) error {
	return replaceHostProcesses(db.sql, hostID, procs, ts)
}

type execer interface {
	Exec(query string, args ...any) (sql.Result, error)
}

func replaceHostProcesses(x execer, hostID string, procs []contracts.ProcessSample, ts time.Time) error {
	if procs == nil {
		procs = []contracts.ProcessSample{}
	}
	body, err := json.Marshal(procs)
	if err != nil {
		return err
	}
	_, err = x.Exec(
		`INSERT OR REPLACE INTO host_processes(host_id, ts, procs) VALUES (?,?,?)`,
		hostID, ts.UTC().Format(metricTimeFmt), string(body))
	return err
}

// LatestHostProcesses returns a host's stored snapshot, false when it has none.
func (db *DB) LatestHostProcesses(hostID string) (HostProcesses, bool) {
	var ts, body string
	err := db.sql.QueryRow(`SELECT ts, procs FROM host_processes WHERE host_id = ?`, hostID).
		Scan(&ts, &body)
	if errors.Is(err, sql.ErrNoRows) || err != nil {
		return HostProcesses{}, false
	}
	out := HostProcesses{Procs: []contracts.ProcessSample{}}
	out.TS, _ = time.Parse(metricTimeFmt, ts)
	if err := json.Unmarshal([]byte(body), &out.Procs); err != nil {
		return HostProcesses{}, false
	}
	return out, true
}
