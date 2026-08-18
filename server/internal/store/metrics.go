package store

import (
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"github.com/thehelvijs/Reeve/contracts"
)

// MetricPoint is one host metric sample returned for charting.
type MetricPoint struct {
	TS          time.Time          `json:"ts"`
	CPUPct      float64            `json:"cpu_pct"`
	MemUsed     uint64             `json:"mem_used"`
	MemTotal    uint64             `json:"mem_total"`
	DiskUsed    uint64             `json:"disk_used"`
	DiskTotal   uint64             `json:"disk_total"`
	DiskRead    uint64             `json:"disk_read"`
	DiskWrite   uint64             `json:"disk_write"`
	NetRx       uint64             `json:"net_rx"`
	NetTx       uint64             `json:"net_tx"`
	Load1       float64            `json:"load1"`
	Load5       float64            `json:"load5"`
	Load15      float64            `json:"load15"`
	Temps       map[string]float64 `json:"temps"`
	GPUUtil     float64            `json:"gpu_util"`
	GPUMemUsed  uint64             `json:"gpu_mem_used"`
	GPUMemTotal uint64             `json:"gpu_mem_total"`
}

// ContainerPoint is one per-container sample returned for charting.
type ContainerPoint struct {
	ContainerID string    `json:"container_id"`
	Name        string    `json:"name"`
	TS          time.Time `json:"ts"`
	CPUPct      float64   `json:"cpu_pct"`
	MemUsed     uint64    `json:"mem_used"`
	MemLimit    uint64    `json:"mem_limit"`
}

// metricTimeFmt uses second precision so SQLite's strftime can bucket by epoch
// during rollup. INSERT OR REPLACE keeps a same-second re-push idempotent.
const metricTimeFmt = time.RFC3339

// intCols reads byte and second counters back as integers. A rollup averages
// them, SQLite keeps a fractional mean as REAL even in an INTEGER column, and
// scanning that into a uint64 fails — which took every metrics range down on a
// host whose rollup had run. Casting on read also rescues rows already stored.
func intCols(names ...string) string {
	out := make([]string, len(names))
	for i, n := range names {
		out[i] = "CAST(" + n + " AS INTEGER)"
	}
	return strings.Join(out, ", ")
}

// InsertHostMetric stores a raw host metric sample from a push.
func (db *DB) InsertHostMetric(hostID string, m contracts.HostMetrics, ts time.Time) error {
	return insertHostMetric(db.sql, hostID, m, ts)
}

func insertHostMetric(w writer, hostID string, m contracts.HostMetrics, ts time.Time) error {
	temps, _ := json.Marshal(m.Temps)
	_, err := w.Exec(
		`INSERT OR REPLACE INTO metric_samples(host_id, ts, resolution, cpu_pct, mem_used, mem_total,
			disk_used, disk_total, disk_read, disk_write, net_rx, net_tx, uptime_secs, temps,
			load1, load5, load15, gpu_util, gpu_mem_used, gpu_mem_total)
		 VALUES (?,?, 'raw', ?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		hostID, ts.UTC().Format(metricTimeFmt), m.CPUPct, m.MemUsed, m.MemTotal,
		m.DiskUsed, m.DiskTotal, m.DiskRead, m.DiskWrite, m.NetRx, m.NetTx, m.UptimeSecs, string(temps),
		m.Load1, m.Load5, m.Load15, m.GPUUtil, m.GPUMemUsed, m.GPUMemTotal)
	return err
}

// latestMetricSelect reads the newest raw sample per host. The correlated MAX
// rides the (host_id, resolution, ts) index, so it costs one seek per host.
const latestMetricSelect = `SELECT host_id, ts, cpu_pct, mem_used, mem_total, disk_used, disk_total,
		temps, load1, load5, load15, gpu_util, gpu_mem_used, gpu_mem_total
	 FROM metric_samples m WHERE resolution = 'raw' AND ts = (
		SELECT MAX(ts) FROM metric_samples WHERE host_id = m.host_id AND resolution = 'raw')`

func scanLatestMetric(row scanner) (string, MetricPoint, error) {
	var hostID string
	var p MetricPoint
	var ts, temps string
	if err := row.Scan(&hostID, &ts, &p.CPUPct, &p.MemUsed, &p.MemTotal, &p.DiskUsed, &p.DiskTotal,
		&temps, &p.Load1, &p.Load5, &p.Load15, &p.GPUUtil, &p.GPUMemUsed, &p.GPUMemTotal); err != nil {
		return "", MetricPoint{}, err
	}
	p.TS, _ = time.Parse(metricTimeFmt, ts)
	json.Unmarshal([]byte(temps), &p.Temps)
	return hostID, p, nil
}

// LatestHostMetric returns the newest raw sample for a host.
func (db *DB) LatestHostMetric(hostID string) (MetricPoint, bool) {
	_, p, err := scanLatestMetric(db.sql.QueryRow(latestMetricSelect+` AND host_id = ?`, hostID))
	if err != nil {
		return MetricPoint{}, false
	}
	return p, true
}

// LatestHostMetrics returns the newest raw sample for every host that has one,
// keyed by host id, so a listing does not query per host.
func (db *DB) LatestHostMetrics() (map[string]MetricPoint, error) {
	rows, err := db.sql.Query(latestMetricSelect)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]MetricPoint{}
	for rows.Next() {
		hostID, p, err := scanLatestMetric(rows)
		if err != nil {
			return nil, err
		}
		out[hostID] = p
	}
	return out, rows.Err()
}

// HostNetRate returns the max(rx,tx) byte/sec rate from the two newest raw
// samples; ok is false with fewer than two samples or a non-positive interval.
// A counter that went backwards (reboot / nic reset) contributes no rate.
func (db *DB) HostNetRate(hostID string) (float64, bool) {
	rows, err := db.sql.Query(
		`SELECT ts, net_rx, net_tx FROM metric_samples
		 WHERE host_id = ? AND resolution = 'raw' ORDER BY ts DESC LIMIT 2`, hostID)
	if err != nil {
		return 0, false
	}
	defer rows.Close()
	type sample struct {
		ts     time.Time
		rx, tx uint64
	}
	var samples []sample
	for rows.Next() {
		var ts string
		var rx, tx uint64
		if err := rows.Scan(&ts, &rx, &tx); err != nil {
			return 0, false
		}
		parsed, _ := time.Parse(metricTimeFmt, ts)
		samples = append(samples, sample{ts: parsed, rx: rx, tx: tx})
	}
	if len(samples) < 2 {
		return 0, false
	}
	newer, older := samples[0], samples[1]
	dt := newer.ts.Sub(older.ts).Seconds()
	if dt <= 0 {
		return 0, false
	}
	var rate float64
	if newer.rx >= older.rx {
		rate = float64(newer.rx-older.rx) / dt
	}
	if newer.tx >= older.tx {
		if txRate := float64(newer.tx-older.tx) / dt; txRate > rate {
			rate = txRate
		}
	}
	return rate, true
}

// DeleteHostMetrics removes all stored metric history for a host (host samples
// and per-container stats, every resolution). Current status is untouched.
func (db *DB) DeleteHostMetrics(hostID string) error {
	if _, err := db.sql.Exec(`DELETE FROM metric_samples WHERE host_id = ?`, hostID); err != nil {
		return err
	}
	if _, err := db.sql.Exec(`DELETE FROM container_stats WHERE host_id = ?`, hostID); err != nil {
		return err
	}
	if _, err := db.sql.Exec(`DELETE FROM host_processes WHERE host_id = ?`, hostID); err != nil {
		return err
	}
	if _, err := db.sql.Exec(`DELETE FROM host_disks WHERE host_id = ?`, hostID); err != nil {
		return err
	}
	_, err := db.sql.Exec(`DELETE FROM process_usage WHERE host_id = ?`, hostID)
	return err
}

// InsertContainerStats stores raw per-container samples from a push.
func (db *DB) InsertContainerStats(hostID string, stats []contracts.ContainerSample, ts time.Time) error {
	if len(stats) == 0 {
		return nil
	}
	return db.inTx(func(tx *sql.Tx) error { return insertContainerStats(tx, hostID, stats, ts) })
}

func insertContainerStats(w writer, hostID string, stats []contracts.ContainerSample, ts time.Time) error {
	if len(stats) == 0 {
		return nil
	}
	stmt, err := w.Prepare(
		`INSERT OR REPLACE INTO container_stats(host_id, container_id, ts, resolution, cpu_pct, mem_used, mem_limit)
		 VALUES (?,?,?, 'raw', ?,?,?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	t := ts.UTC().Format(metricTimeFmt)
	for _, s := range stats {
		if _, err := stmt.Exec(hostID, s.ContainerID, t, s.CPUPct, s.MemUsed, s.MemLimit); err != nil {
			return err
		}
	}
	return nil
}

// QueryHostMetrics returns samples for a host at the given resolution since a
// cutoff, oldest first.
func (db *DB) QueryHostMetrics(hostID, resolution string, since time.Time) ([]MetricPoint, error) {
	rows, err := db.sql.Query(
		`SELECT ts, cpu_pct, `+intCols(
			"mem_used", "mem_total", "disk_used", "disk_total", "disk_read", "disk_write",
			"net_rx", "net_tx")+`, temps, load1, load5, load15, gpu_util, `+
			intCols("gpu_mem_used", "gpu_mem_total")+`
		 FROM metric_samples WHERE host_id = ? AND resolution = ? AND ts >= ?
		 ORDER BY ts`, hostID, resolution, since.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []MetricPoint
	for rows.Next() {
		var p MetricPoint
		var ts, temps string
		if err := rows.Scan(&ts, &p.CPUPct, &p.MemUsed, &p.MemTotal, &p.DiskUsed, &p.DiskTotal,
			&p.DiskRead, &p.DiskWrite, &p.NetRx, &p.NetTx, &temps, &p.Load1, &p.Load5, &p.Load15,
			&p.GPUUtil, &p.GPUMemUsed, &p.GPUMemTotal); err != nil {
			return nil, err
		}
		p.TS, _ = time.Parse(time.RFC3339Nano, ts)
		json.Unmarshal([]byte(temps), &p.Temps)
		out = append(out, p)
	}
	return out, rows.Err()
}

// QueryContainerStats returns per-container samples for a host at a resolution
// since a cutoff, oldest first.
func (db *DB) QueryContainerStats(hostID, resolution string, since time.Time) ([]ContainerPoint, error) {
	rows, err := db.sql.Query(
		`SELECT container_stats.container_id, COALESCE(NULLIF(cs.display_name, ''), cs.name), container_stats.ts,
			container_stats.cpu_pct, `+intCols("container_stats.mem_used", "container_stats.mem_limit")+`
		 FROM container_stats
		 LEFT JOIN container_status cs ON cs.host_id = container_stats.host_id
			AND cs.container_id = container_stats.container_id
		 WHERE container_stats.host_id = ? AND container_stats.resolution = ? AND container_stats.ts >= ?
		 ORDER BY container_stats.ts`, hostID, resolution, since.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ContainerPoint
	for rows.Next() {
		var p ContainerPoint
		var ts string
		var name sql.NullString
		if err := rows.Scan(&p.ContainerID, &name, &ts, &p.CPUPct, &p.MemUsed, &p.MemLimit); err != nil {
			return nil, err
		}
		p.Name = name.String
		p.TS, _ = time.Parse(time.RFC3339Nano, ts)
		out = append(out, p)
	}
	return out, rows.Err()
}
