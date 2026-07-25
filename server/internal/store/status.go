package store

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
