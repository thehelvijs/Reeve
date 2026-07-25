package store

// VisibilityGrant is a principal allowed to see a restricted tool.
type VisibilityGrant struct {
	PrincipalType string
	PrincipalID   string
}

// AddToolVisibility grants a user or group visibility of a restricted tool.
func (db *DB) AddToolVisibility(toolID, principalType, principalID string) error {
	_, err := db.sql.Exec(
		`INSERT INTO tool_visibility(tool_id, principal_type, principal_id) VALUES (?, ?, ?)
		 ON CONFLICT DO NOTHING`, toolID, principalType, principalID)
	return err
}

// RemoveToolVisibility revokes a visibility grant.
func (db *DB) RemoveToolVisibility(toolID, principalType, principalID string) error {
	_, err := db.sql.Exec(
		`DELETE FROM tool_visibility WHERE tool_id = ? AND principal_type = ? AND principal_id = ?`,
		toolID, principalType, principalID)
	return err
}

// ListToolVisibility returns the visibility grants for a tool.
func (db *DB) ListToolVisibility(toolID string) ([]VisibilityGrant, error) {
	rows, err := db.sql.Query(
		`SELECT principal_type, principal_id FROM tool_visibility WHERE tool_id = ?`, toolID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []VisibilityGrant
	for rows.Next() {
		var g VisibilityGrant
		if err := rows.Scan(&g.PrincipalType, &g.PrincipalID); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}
