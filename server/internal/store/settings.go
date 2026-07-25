package store

// GetSetting returns a setting value and whether it was present.
func (db *DB) GetSetting(key string) (string, bool) {
	var v string
	err := db.sql.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&v)
	if err != nil {
		return "", false
	}
	return v, true
}

// SetSetting upserts a setting value.
func (db *DB) SetSetting(key, value string) error {
	_, err := db.sql.Exec(
		`INSERT INTO settings(key, value) VALUES (?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
	return err
}

// GetBoolSetting returns a boolean setting, defaulting when unset.
func (db *DB) GetBoolSetting(key string, def bool) bool {
	v, ok := db.GetSetting(key)
	if !ok {
		return def
	}
	return v == "true"
}
