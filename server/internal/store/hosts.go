package store

import (
	"database/sql"
	"errors"
	"time"
)

// ServerHostID is the reserved host row for the machine Reeve itself runs on.
// It lists like any other host — a service on the server's own box is a service
// somebody has to watch — but it carries no agent, so the fleet update queries
// still skip it.
const ServerHostID = "__server__"

// EnsureServerHost creates the reserved self-monitoring host row if absent.
func (db *DB) EnsureServerHost(os string) error {
	_, err := db.sql.Exec(
		`INSERT OR IGNORE INTO hosts(id, name, os, physical_location, enroll_token_hash, offline_after_secs, created_at)
		 VALUES (?,?,?,?,?,?,?)`,
		ServerHostID, "Reeve server", os, "", "", 60, time.Now().UTC().Format(time.RFC3339Nano))
	return err
}

// Host is a monitored machine running an agent.
type Host struct {
	ID               string
	Name             string
	OS               string
	PhysicalLocation string
	// IPAddress is where the agent last reported this host to be, on the route
	// from the host to the server. Empty until the first push.
	IPAddress    string
	AgentVersion string
	// AgentChecksum is the sha256 of the binary the agent last reported.
	AgentChecksum    string
	AutoUpdate       string
	AutoUpdateVetoed bool
	UpdateStartedAt  *time.Time
	// UpdateForced reports that an operator granted the slot by hand.
	UpdateForced     bool
	LastSeenAt       *time.Time
	OfflineAfterSecs int
	// ControlEnabled is what the agent last reported: whether it will run
	// commands at all. False for an agent that never said so.
	ControlEnabled bool
	CreatedAt      time.Time
	IconPath       string
	ThumbnailPath  string
	Latitude       *float64
	Longitude      *float64
	// PinColor is #rrggbb for this host's map pin, empty for the brand accent.
	PinColor string
}

// SetHostThumbnailPath sets (or clears, when empty) a host's thumbnail path.
func (db *DB) SetHostThumbnailPath(id, path string) error {
	return db.exec1(`UPDATE hosts SET thumbnail_path = ? WHERE id = ?`, path, id)
}

// UpdateHostLocation sets a host's physical location text, optional map
// coordinates (nil clears a coordinate) and pin colour (empty for the default).
func (db *DB) UpdateHostLocation(id, location string, lat, lng *float64, pinColor string) error {
	return db.exec1(
		`UPDATE hosts SET physical_location = ?, latitude = ?, longitude = ?, pin_color = ? WHERE id = ?`,
		location, lat, lng, pinColor, id)
}

// SetHostIconPath sets (or clears, when empty) a host's stored icon path.
func (db *DB) SetHostIconPath(id, path string) error {
	return db.exec1(`UPDATE hosts SET icon_path = ? WHERE id = ?`, path, id)
}

// HostHasPublicTool reports whether a host is referenced by any public tool, so
// the anonymous portal may show its icon without leaking a private host.
func (db *DB) HostHasPublicTool(id string) bool {
	var one int
	err := db.sql.QueryRow(
		`SELECT 1 FROM tools WHERE host_id = ? AND visibility = 'public' LIMIT 1`, id).Scan(&one)
	return err == nil
}

// Online reports whether the host has pushed within its offline window.
func (h Host) Online(now time.Time) bool {
	if h.LastSeenAt == nil {
		return false
	}
	return now.Sub(*h.LastSeenAt) <= time.Duration(h.OfflineAfterSecs)*time.Second
}

// CountHosts returns the total number of hosts.
func (db *DB) CountHosts() (int, error) {
	var n int
	err := db.sql.QueryRow("SELECT COUNT(*) FROM hosts").Scan(&n)
	return n, err
}

// CreateHost inserts a host with its enrollment token hash.
func (db *DB) CreateHost(name, os, location, tokenHash string, offlineAfter int) (Host, error) {
	if offlineAfter <= 0 {
		offlineAfter = 60
	}
	h := Host{
		ID: NewID(), Name: name, OS: os, PhysicalLocation: location,
		AutoUpdate: AutoUpdateDefault, OfflineAfterSecs: offlineAfter, CreatedAt: time.Now().UTC(),
	}
	_, err := db.sql.Exec(
		`INSERT INTO hosts(id, name, os, physical_location, enroll_token_hash, offline_after_secs, created_at)
		 VALUES (?,?,?,?,?,?,?)`,
		h.ID, h.Name, h.OS, h.PhysicalLocation, tokenHash, h.OfflineAfterSecs,
		h.CreatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return Host{}, err
	}
	return h, nil
}

// ListHosts returns all hosts ordered by name.
func (db *DB) ListHosts() ([]Host, error) {
	rows, err := db.sql.Query(hostSelect + ` ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Host
	for rows.Next() {
		h, err := db.scanHost(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

// ListPublicHosts returns only hosts referenced by at least one public tool,
// so the anonymous portal never leaks hosts whose tools are all restricted.
func (db *DB) ListPublicHosts() ([]Host, error) {
	q := hostSelect + ` WHERE id IN (
		SELECT DISTINCT host_id FROM tools
		WHERE visibility = 'public' AND host_id IS NOT NULL AND host_id != ''
	) ORDER BY name`
	rows, err := db.sql.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Host
	for rows.Next() {
		h, err := db.scanHost(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

// GetHost returns a host by id.
func (db *DB) GetHost(id string) (Host, error) {
	h, err := db.scanHost(db.sql.QueryRow(hostSelect+` WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Host{}, ErrNotFound
	}
	return h, err
}

// GetHostByTokenHash resolves the host for an agent's enrollment token hash.
func (db *DB) GetHostByTokenHash(tokenHash string) (Host, error) {
	h, err := db.scanHost(db.sql.QueryRow(hostSelect+` WHERE enroll_token_hash = ?`, tokenHash))
	if errors.Is(err, sql.ErrNoRows) {
		return Host{}, ErrNotFound
	}
	return h, err
}

// SetHostEnrollTokenHash replaces a host's enrollment token hash, which revokes
// the previous token. Used when the server pushes an install over SSH and mints
// a fresh token for that host.
func (db *DB) SetHostEnrollTokenHash(id, tokenHash string) error {
	return db.exec1(`UPDATE hosts SET enroll_token_hash = ? WHERE id = ?`, tokenHash, id)
}

// MarkHostSeen stamps last_seen_at without claiming an agent version, for the
// server's own row: it heartbeats from inside the server until an agent is
// installed on that machine, and must not read as one that already has.
func (db *DB) MarkHostSeen(id string) error {
	return db.exec1(`UPDATE hosts SET last_seen_at = ? WHERE id = ?`,
		time.Now().UTC().Format(time.RFC3339Nano), id)
}

// TouchHost updates last_seen_at and agent version on a successful push.
func (db *DB) TouchHost(id, agentVersion string) error {
	return db.exec1(`UPDATE hosts SET last_seen_at = ?, agent_version = ? WHERE id = ?`,
		time.Now().UTC().Format(time.RFC3339Nano), agentVersion, id)
}

// DeleteHost removes a host (telemetry cascades).
func (db *DB) DeleteHost(id string) error {
	return db.exec1(`DELETE FROM hosts WHERE id = ?`, id)
}

const hostSelect = `SELECT id, name, os, physical_location, ip_address, agent_version, agent_checksum, auto_update, auto_update_vetoed, update_started_at, update_forced, last_seen_at, offline_after_secs, control_enabled, created_at, icon_path, thumbnail_path, latitude, longitude, pin_color FROM hosts`

func (db *DB) scanHost(row scanner) (Host, error) {
	var h Host
	var created string
	var lastSeen, updateStarted *string
	var lat, lng sql.NullFloat64
	if err := row.Scan(&h.ID, &h.Name, &h.OS, &h.PhysicalLocation, &h.IPAddress, &h.AgentVersion,
		&h.AgentChecksum, &h.AutoUpdate, &h.AutoUpdateVetoed, &updateStarted, &h.UpdateForced, &lastSeen,
		&h.OfflineAfterSecs, &h.ControlEnabled, &created, &h.IconPath, &h.ThumbnailPath, &lat, &lng, &h.PinColor); err != nil {
		return Host{}, err
	}
	h.LastSeenAt = parseNullableTime(lastSeen)
	h.UpdateStartedAt = parseNullableTime(updateStarted)
	h.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	if lat.Valid {
		h.Latitude = &lat.Float64
	}
	if lng.Valid {
		h.Longitude = &lng.Float64
	}
	return h, nil
}
