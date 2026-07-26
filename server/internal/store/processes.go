package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"sort"
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

// usageBucket is the accumulation window. Coarse on purpose: it is the floor on
// how precisely an average can be sliced, and the reason a week of per-process
// usage costs a few thousand rows instead of a few million.
const usageBucket = 5 * time.Minute

// ProcessUsage is one command's usage over a window, averaged across the pushes
// that reported it and peaked at the worst single push.
type ProcessUsage struct {
	Command string  `json:"command"`
	CPUAvg  float64 `json:"cpu_avg"`
	CPUMax  float64 `json:"cpu_max"`
	MemAvg  float64 `json:"mem_avg"`
	MemMax  uint64  `json:"mem_max"`
	Samples int     `json:"samples"`
}

// accumulateProcessUsage folds one push's processes into their 5-minute bucket.
func accumulateProcessUsage(x execer, hostID string, procs []contracts.ProcessSample, ts time.Time) error {
	bucket := ts.UTC().Truncate(usageBucket).Format(metricTimeFmt)
	for _, p := range procs {
		if p.Command == "" {
			continue
		}
		if _, err := x.Exec(
			`INSERT INTO process_usage(host_id, bucket, command, samples, cpu_sum, cpu_max, mem_sum, mem_max)
			 VALUES (?,?,?,1,?,?,?,?)
			 ON CONFLICT(host_id, bucket, command) DO UPDATE SET
			   samples = samples + 1,
			   cpu_sum = cpu_sum + excluded.cpu_sum,
			   cpu_max = MAX(cpu_max, excluded.cpu_max),
			   mem_sum = mem_sum + excluded.mem_sum,
			   mem_max = MAX(mem_max, excluded.mem_max)`,
			hostID, bucket, p.Command, p.CPUPct, p.CPUPct, float64(p.MemRSS), int64(p.MemRSS)); err != nil {
			return err
		}
	}
	return nil
}

const processUsageSelect = `
	SELECT command,
	       SUM(cpu_sum) / SUM(samples), MAX(cpu_max),
	       SUM(mem_sum) / SUM(samples), MAX(mem_max),
	       SUM(samples)
	FROM process_usage
	WHERE host_id = ? AND bucket >= ?
	GROUP BY command`

// ProcessUsageSince returns the heaviest commands over a window: the top limit
// by average CPU and the top limit by average memory, merged. Both dimensions
// are asked for separately because a memory hog that never burns CPU would
// otherwise fall off the end of a CPU-ordered list, which is exactly the thing
// someone hunting for waste is looking for.
func (db *DB) ProcessUsageSince(hostID string, since time.Time, limit int) ([]ProcessUsage, error) {
	if limit <= 0 {
		return []ProcessUsage{}, nil
	}
	cutoff := since.UTC().Truncate(usageBucket).Format(metricTimeFmt)
	seen := map[string]bool{}
	out := []ProcessUsage{}
	for _, order := range []string{"cpu", "mem"} {
		rows, err := db.sql.Query(processUsageSelect+` ORDER BY SUM(`+order+`_sum) / SUM(samples) DESC LIMIT ?`,
			hostID, cutoff, limit)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var u ProcessUsage
			if err := rows.Scan(&u.Command, &u.CPUAvg, &u.CPUMax, &u.MemAvg, &u.MemMax, &u.Samples); err != nil {
				rows.Close()
				return nil, err
			}
			if seen[u.Command] {
				continue
			}
			seen[u.Command] = true
			out = append(out, u)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return nil, err
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CPUAvg > out[j].CPUAvg })
	return out, nil
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
