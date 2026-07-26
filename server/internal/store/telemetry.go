package store

import (
	"database/sql"
	"time"

	"github.com/thehelvijs/Reeve/contracts"
)

// writer is the subset of *sql.DB and *sql.Tx the write helpers need, so one
// implementation serves both a standalone call and a batched transaction.
type writer interface {
	Exec(query string, args ...any) (sql.Result, error)
	Prepare(query string) (*sql.Stmt, error)
}

// ApplyPush persists one agent push: the systemd, container and cron snapshots,
// the metric samples, the log events, and the host's heartbeat. It runs as a
// single transaction, so SQLite's commit cost scales with pushes rather than
// with inventory size, and a partially stored push is never visible.
func (db *DB) ApplyPush(hostID string, p contracts.Push, now time.Time) error {
	return db.inTx(func(tx *sql.Tx) error {
		if err := replaceServiceStatus(tx, hostID, p.Services, now); err != nil {
			return err
		}
		if err := replaceContainerStatus(tx, hostID, p.Containers, now); err != nil {
			return err
		}
		if err := replaceCronJobs(tx, hostID, p.CronJobs, now); err != nil {
			return err
		}
		if err := insertHostMetric(tx, hostID, p.Metrics, now); err != nil {
			return err
		}
		if err := insertContainerStats(tx, hostID, p.ContainerStats, now); err != nil {
			return err
		}
		if err := insertLogEvents(tx, hostID, p.LogEvents); err != nil {
			return err
		}
		// A host that vetoes locally will never take the update, so its slot is
		// released in the same write that records the veto. An empty reported
		// address keeps the last known one: a collector that could not answer
		// this tick is not evidence the host moved.
		_, err := tx.Exec(
			`UPDATE hosts
			 SET last_seen_at = ?, agent_version = ?, auto_update_vetoed = ?,
			     ip_address = CASE WHEN ? = '' THEN ip_address ELSE ? END,
			     update_started_at = CASE WHEN ? THEN NULL ELSE update_started_at END
			 WHERE id = ?`,
			now.UTC().Format(time.RFC3339Nano), p.AgentVersion, p.AutoUpdateVetoed,
			p.IPAddress, p.IPAddress, p.AutoUpdateVetoed, hostID)
		return err
	})
}

// ReplaceContainerStatus overwrites a host's container snapshot.
func (db *DB) ReplaceContainerStatus(hostID string, states []contracts.ContainerState) error {
	return db.inTx(func(tx *sql.Tx) error {
		return replaceContainerStatus(tx, hostID, states, time.Now().UTC())
	})
}

func replaceServiceStatus(w writer, hostID string, states []contracts.ServiceState, now time.Time) error {
	if _, err := w.Exec(`DELETE FROM service_status WHERE host_id = ?`, hostID); err != nil {
		return err
	}
	if len(states) == 0 {
		return nil
	}
	stmt, err := w.Prepare(
		`INSERT INTO service_status(host_id, unit, active_state, sub_state, updated_at) VALUES (?,?,?,?,?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	ts := now.UTC().Format(time.RFC3339Nano)
	for _, s := range states {
		if _, err := stmt.Exec(hostID, s.Unit, s.ActiveState, s.SubState, ts); err != nil {
			return err
		}
	}
	return nil
}

func replaceContainerStatus(w writer, hostID string, states []contracts.ContainerState, now time.Time) error {
	if _, err := w.Exec(`DELETE FROM container_status WHERE host_id = ?`, hostID); err != nil {
		return err
	}
	if len(states) == 0 {
		return nil
	}
	stmt, err := w.Prepare(
		`INSERT INTO container_status(host_id, container_id, name, image, state, health, updated_at)
		 VALUES (?,?,?,?,?,?,?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	ts := now.UTC().Format(time.RFC3339Nano)
	for _, c := range states {
		if _, err := stmt.Exec(hostID, c.ID, c.Name, c.Image, c.State, c.Health, ts); err != nil {
			return err
		}
	}
	return nil
}

func replaceCronJobs(w writer, hostID string, jobs []contracts.CronState, now time.Time) error {
	if _, err := w.Exec(`DELETE FROM cron_jobs WHERE host_id = ?`, hostID); err != nil {
		return err
	}
	if len(jobs) == 0 {
		return nil
	}
	stmt, err := w.Prepare(
		`INSERT INTO cron_jobs(host_id, name, schedule, last_run_at, last_exit, updated_at)
		 VALUES (?,?,?,?,?,?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	ts := now.UTC().Format(time.RFC3339Nano)
	for _, j := range jobs {
		var lastRun *string
		if j.LastRunAt != nil {
			s := j.LastRunAt.UTC().Format(time.RFC3339Nano)
			lastRun = &s
		}
		if _, err := stmt.Exec(hostID, j.Name, j.Schedule, lastRun, j.LastExit, ts); err != nil {
			return err
		}
	}
	return nil
}

func insertLogEvents(w writer, hostID string, events []contracts.LogEvent) error {
	if len(events) == 0 {
		return nil
	}
	stmt, err := w.Prepare(
		`INSERT INTO log_events(id, host_id, source, level, message, at) VALUES (?,?,?,?,?,?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, e := range events {
		if _, err := stmt.Exec(
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

// LookupServiceState returns a systemd unit's active state for a host.
func (db *DB) LookupServiceState(hostID, unit string) (activeState string, ok bool) {
	err := db.sql.QueryRow(
		`SELECT active_state FROM service_status WHERE host_id = ? AND unit = ?`, hostID, unit).
		Scan(&activeState)
	return activeState, err == nil
}

// LookupContainerState returns a container's state and health for a host.
func (db *DB) LookupContainerState(hostID, containerID string) (state, health string, ok bool) {
	err := db.sql.QueryRow(
		`SELECT state, health FROM container_status WHERE host_id = ? AND container_id = ?`,
		hostID, containerID).Scan(&state, &health)
	return state, health, err == nil
}

// LookupCronExists reports whether a cron job is present on a host.
func (db *DB) LookupCronExists(hostID, name string) bool {
	var one int
	err := db.sql.QueryRow(
		`SELECT 1 FROM cron_jobs WHERE host_id = ? AND name = ?`, hostID, name).Scan(&one)
	return err == nil
}
