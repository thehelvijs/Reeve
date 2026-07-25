package store

import (
	"time"

	"github.com/thehelvijs/Reeve/contracts"
)

// ReplaceServiceStatus overwrites a host's systemd unit snapshot.
func (db *DB) ReplaceServiceStatus(hostID string, states []contracts.ServiceState) error {
	tx, err := db.sql.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM service_status WHERE host_id = ?`, hostID); err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	for _, s := range states {
		if _, err := tx.Exec(
			`INSERT INTO service_status(host_id, unit, active_state, sub_state, updated_at) VALUES (?,?,?,?,?)`,
			hostID, s.Unit, s.ActiveState, s.SubState, now); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// ReplaceContainerStatus overwrites a host's container snapshot.
func (db *DB) ReplaceContainerStatus(hostID string, states []contracts.ContainerState) error {
	tx, err := db.sql.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM container_status WHERE host_id = ?`, hostID); err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	for _, c := range states {
		if _, err := tx.Exec(
			`INSERT INTO container_status(host_id, container_id, name, image, state, health, updated_at)
			 VALUES (?,?,?,?,?,?,?)`,
			hostID, c.ID, c.Name, c.Image, c.State, c.Health, now); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// ReplaceCronJobs overwrites a host's cron snapshot.
func (db *DB) ReplaceCronJobs(hostID string, jobs []contracts.CronState) error {
	tx, err := db.sql.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM cron_jobs WHERE host_id = ?`, hostID); err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	for _, j := range jobs {
		var lastRun *string
		if j.LastRunAt != nil {
			s := j.LastRunAt.UTC().Format(time.RFC3339Nano)
			lastRun = &s
		}
		if _, err := tx.Exec(
			`INSERT INTO cron_jobs(host_id, name, schedule, last_run_at, last_exit, updated_at)
			 VALUES (?,?,?,?,?,?)`,
			hostID, j.Name, j.Schedule, lastRun, j.LastExit, now); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// InsertLogEvents appends error-level log events for a host.
func (db *DB) InsertLogEvents(hostID string, events []contracts.LogEvent) error {
	for _, e := range events {
		if _, err := db.sql.Exec(
			`INSERT INTO log_events(id, host_id, source, level, message, at) VALUES (?,?,?,?,?,?)`,
			NewID(), hostID, e.Source, e.Level, e.Message, e.At.UTC().Format(time.RFC3339Nano)); err != nil {
			return err
		}
	}
	return nil
}

// InventoryService is a systemd unit in a host's inventory.
type InventoryService struct {
	Unit        string `json:"unit"`
	ActiveState string `json:"active_state"`
	SubState    string `json:"sub_state"`
}

// InventoryContainer is a container in a host's inventory.
type InventoryContainer struct {
	ContainerID string `json:"container_id"`
	Name        string `json:"name"`
	Image       string `json:"image"`
	State       string `json:"state"`
	Health      string `json:"health"`
}

// InventoryCron is a cron job in a host's inventory.
type InventoryCron struct {
	Name     string `json:"name"`
	Schedule string `json:"schedule"`
}

// ListServiceStatus returns a host's systemd units.
func (db *DB) ListServiceStatus(hostID string) ([]InventoryService, error) {
	rows, err := db.sql.Query(
		`SELECT unit, active_state, sub_state FROM service_status WHERE host_id = ? ORDER BY unit`, hostID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []InventoryService
	for rows.Next() {
		var s InventoryService
		if err := rows.Scan(&s.Unit, &s.ActiveState, &s.SubState); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// ListContainerStatus returns a host's containers.
func (db *DB) ListContainerStatus(hostID string) ([]InventoryContainer, error) {
	rows, err := db.sql.Query(
		`SELECT container_id, name, image, state, health FROM container_status WHERE host_id = ? ORDER BY name`, hostID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []InventoryContainer
	for rows.Next() {
		var c InventoryContainer
		if err := rows.Scan(&c.ContainerID, &c.Name, &c.Image, &c.State, &c.Health); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// ListCronJobs returns a host's cron jobs.
func (db *DB) ListCronJobs(hostID string) ([]InventoryCron, error) {
	rows, err := db.sql.Query(
		`SELECT name, schedule FROM cron_jobs WHERE host_id = ? ORDER BY name`, hostID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []InventoryCron
	for rows.Next() {
		var c InventoryCron
		if err := rows.Scan(&c.Name, &c.Schedule); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// SourceRefsForHost returns the set of source_refs already linked to a tool for
// a host, so inventory can mark linked items.
func (db *DB) SourceRefsForHost(hostID string) (map[string]bool, error) {
	rows, err := db.sql.Query(
		`SELECT source_type, source_ref FROM tools WHERE host_id = ? AND source_ref != ''`, hostID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var stype, sref string
		if err := rows.Scan(&stype, &sref); err != nil {
			return nil, err
		}
		out[stype+":"+sref] = true
	}
	return out, rows.Err()
}
