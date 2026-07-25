package store

// Threshold is a resource-alert limit for a host+metric. HostID "" is the
// global default; a per-host row overrides it.
type Threshold struct {
	HostID  string  `json:"-"`
	Metric  string  `json:"metric"`
	Enabled bool    `json:"enabled"`
	Value   float64 `json:"threshold"`
}

// SetThreshold upserts a threshold row.
func (db *DB) SetThreshold(hostID, metric string, enabled bool, value float64) error {
	_, err := db.sql.Exec(
		`INSERT INTO alert_thresholds(host_id, metric, enabled, threshold) VALUES (?,?,?,?)
		 ON CONFLICT(host_id, metric) DO UPDATE SET enabled = excluded.enabled, threshold = excluded.threshold`,
		hostID, metric, boolToInt(enabled), value)
	return err
}

// EffectiveThreshold returns the per-host row if present, else the global row.
func (db *DB) EffectiveThreshold(hostID, metric string) (Threshold, bool) {
	var t Threshold
	var en int
	err := db.sql.QueryRow(
		`SELECT host_id, metric, enabled, threshold FROM alert_thresholds WHERE host_id = ? AND metric = ?`,
		hostID, metric).Scan(&t.HostID, &t.Metric, &en, &t.Value)
	if err == nil {
		t.Enabled = en == 1
		return t, true
	}
	err = db.sql.QueryRow(
		`SELECT host_id, metric, enabled, threshold FROM alert_thresholds WHERE host_id = '' AND metric = ?`,
		metric).Scan(&t.HostID, &t.Metric, &en, &t.Value)
	if err != nil {
		return Threshold{}, false
	}
	t.Enabled = en == 1
	return t, true
}

// DeleteHostThresholds removes a host's per-host overrides (used on host delete).
func (db *DB) DeleteHostThresholds(hostID string) error {
	_, err := db.sql.Exec(`DELETE FROM alert_thresholds WHERE host_id = ?`, hostID)
	return err
}
