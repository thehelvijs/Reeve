package store

import (
	"fmt"
	"strconv"
	"time"
)

// Retention defines how long each resolution tier is kept.
type Retention struct {
	Raw     time.Duration
	FiveMin time.Duration
	OneHour time.Duration
}

// DefaultRetention keeps raw for 48h, 5m for 7d, 1h for 30d.
var DefaultRetention = Retention{
	Raw:     48 * time.Hour,
	FiveMin: 7 * 24 * time.Hour,
	OneHour: 30 * 24 * time.Hour,
}

// EffectiveRetention returns the operator-configured windows, falling back to
// DefaultRetention for any key that is unset or unparseable.
func (db *DB) EffectiveRetention() Retention {
	ret := DefaultRetention
	if d, ok := db.retentionSetting("retention.raw_secs"); ok {
		ret.Raw = d
	}
	if d, ok := db.retentionSetting("retention.fivemin_secs"); ok {
		ret.FiveMin = d
	}
	if d, ok := db.retentionSetting("retention.onehour_secs"); ok {
		ret.OneHour = d
	}
	return ret
}

// retentionSetting reads one retention key as a duration in seconds.
func (db *DB) retentionSetting(key string) (time.Duration, bool) {
	v, ok := db.GetSetting(key)
	if !ok {
		return 0, false
	}
	secs, err := strconv.Atoi(v)
	if err != nil {
		return 0, false
	}
	return time.Duration(secs) * time.Second, true
}

// RollupAndPrune aggregates raw→5m→1h for complete buckets and prunes each tier
// past its retention window. It is idempotent: only complete buckets not already
// present are inserted, so repeated runs converge. `now` is injected for tests.
func (db *DB) RollupAndPrune(now time.Time, ret Retention) error {
	if err := db.rollupHostMetrics("raw", "5m", 300, now); err != nil {
		return err
	}
	if err := db.rollupHostMetrics("5m", "1h", 3600, now); err != nil {
		return err
	}
	if err := db.rollupContainerStats("raw", "5m", 300, now); err != nil {
		return err
	}
	if err := db.rollupContainerStats("5m", "1h", 3600, now); err != nil {
		return err
	}
	return db.prune(now, ret)
}

// rollupHostMetrics aggregates one host-metric tier into a coarser one. Only
// buckets whose window has fully elapsed (end <= now) are inserted.
func (db *DB) rollupHostMetrics(from, to string, bucketSecs int64, now time.Time) error {
	cutoff := now.UTC().Unix() - bucketSecs
	q := fmt.Sprintf(`
		INSERT OR IGNORE INTO metric_samples(host_id, ts, resolution, cpu_pct, mem_used, mem_total,
			disk_used, disk_total, disk_read, disk_write, net_rx, net_tx, uptime_secs, temps,
			load1, load5, load15, gpu_util, gpu_mem_used, gpu_mem_total)
		SELECT host_id,
			strftime('%%Y-%%m-%%dT%%H:%%M:%%SZ', (CAST(strftime('%%s', ts) AS INTEGER) / %d) * %d, 'unixepoch'),
			'%s',
			AVG(cpu_pct), AVG(mem_used), AVG(mem_total), AVG(disk_used), AVG(disk_total),
			AVG(disk_read), AVG(disk_write), AVG(net_rx), AVG(net_tx), AVG(uptime_secs), '{}',
			AVG(load1), AVG(load5), AVG(load15), AVG(gpu_util), AVG(gpu_mem_used), AVG(gpu_mem_total)
		FROM metric_samples
		WHERE resolution = '%s'
		GROUP BY host_id, (CAST(strftime('%%s', ts) AS INTEGER) / %d)
		HAVING (CAST(strftime('%%s', MIN(ts)) AS INTEGER) / %d) * %d <= %d`,
		bucketSecs, bucketSecs, to, from, bucketSecs, bucketSecs, bucketSecs, cutoff)
	_, err := db.sql.Exec(q)
	return err
}

func (db *DB) rollupContainerStats(from, to string, bucketSecs int64, now time.Time) error {
	cutoff := now.UTC().Unix() - bucketSecs
	q := fmt.Sprintf(`
		INSERT OR IGNORE INTO container_stats(host_id, container_id, ts, resolution, cpu_pct, mem_used, mem_limit)
		SELECT host_id, container_id,
			strftime('%%Y-%%m-%%dT%%H:%%M:%%SZ', (CAST(strftime('%%s', ts) AS INTEGER) / %d) * %d, 'unixepoch'),
			'%s',
			AVG(cpu_pct), AVG(mem_used), AVG(mem_limit)
		FROM container_stats
		WHERE resolution = '%s'
		GROUP BY host_id, container_id, (CAST(strftime('%%s', ts) AS INTEGER) / %d)
		HAVING (CAST(strftime('%%s', MIN(ts)) AS INTEGER) / %d) * %d <= %d`,
		bucketSecs, bucketSecs, to, from, bucketSecs, bucketSecs, bucketSecs, cutoff)
	_, err := db.sql.Exec(q)
	return err
}

func (db *DB) prune(now time.Time, ret Retention) error {
	tiers := []struct {
		res string
		age time.Duration
	}{
		{"raw", ret.Raw},
		{"5m", ret.FiveMin},
		{"1h", ret.OneHour},
	}
	for _, t := range tiers {
		cutoff := now.Add(-t.age).UTC().Format(time.RFC3339Nano)
		if _, err := db.sql.Exec(`DELETE FROM metric_samples WHERE resolution = ? AND ts < ?`, t.res, cutoff); err != nil {
			return err
		}
		if _, err := db.sql.Exec(`DELETE FROM container_stats WHERE resolution = ? AND ts < ?`, t.res, cutoff); err != nil {
			return err
		}
	}
	// Process usage has one tier, already bucketed at five minutes, so it
	// follows the 5m window rather than getting a rollup of its own.
	usageCutoff := now.Add(-ret.FiveMin).UTC().Format(metricTimeFmt)
	if _, err := db.sql.Exec(`DELETE FROM process_usage WHERE bucket < ?`, usageCutoff); err != nil {
		return err
	}
	return nil
}
