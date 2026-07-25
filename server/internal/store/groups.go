package store

import (
	"time"
)

// Group is a named set of users used as a single principal for access grants.
type Group struct {
	ID        string
	Name      string
	CreatedAt time.Time
}

// CreateGroup inserts a group.
func (db *DB) CreateGroup(name string) (Group, error) {
	g := Group{ID: NewID(), Name: name, CreatedAt: time.Now().UTC()}
	_, err := db.sql.Exec(`INSERT INTO groups(id, name, created_at) VALUES (?, ?, ?)`,
		g.ID, g.Name, g.CreatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return Group{}, err
	}
	return g, nil
}

// ListGroups returns all groups ordered by name.
func (db *DB) ListGroups() ([]Group, error) {
	rows, err := db.sql.Query(`SELECT id, name, created_at FROM groups ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Group
	for rows.Next() {
		var g Group
		var created string
		if err := rows.Scan(&g.ID, &g.Name, &created); err != nil {
			return nil, err
		}
		g.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		out = append(out, g)
	}
	return out, rows.Err()
}

// RenameGroup changes a group's name.
func (db *DB) RenameGroup(id, name string) error {
	return db.exec1(`UPDATE groups SET name = ? WHERE id = ?`, name, id)
}

// DeleteGroup removes a group. Membership cascades via FK; visibility,
// credential-access and collection grants reference the group polymorphically,
// so they are cleared explicitly.
func (db *DB) DeleteGroup(id string) error {
	db.sql.Exec(`DELETE FROM tool_visibility WHERE principal_type='group' AND principal_id = ?`, id)
	db.sql.Exec(`DELETE FROM credential_access WHERE principal_type='group' AND principal_id = ?`, id)
	db.sql.Exec(`DELETE FROM collection_editors WHERE principal_type='group' AND principal_id = ?`, id)
	db.sql.Exec(`DELETE FROM collection_visibility WHERE principal_type='group' AND principal_id = ?`, id)
	return db.exec1(`DELETE FROM groups WHERE id = ?`, id)
}

// AddGroupMember adds a user to a group (idempotent).
func (db *DB) AddGroupMember(groupID, userID string) error {
	_, err := db.sql.Exec(
		`INSERT INTO group_members(group_id, user_id) VALUES (?, ?)
		 ON CONFLICT DO NOTHING`, groupID, userID)
	return err
}

// RemoveGroupMember removes a user from a group.
func (db *DB) RemoveGroupMember(groupID, userID string) error {
	_, err := db.sql.Exec(
		`DELETE FROM group_members WHERE group_id = ? AND user_id = ?`, groupID, userID)
	return err
}

// ListGroupMembers returns the user ids in a group.
func (db *DB) ListGroupMembers(groupID string) ([]string, error) {
	rows, err := db.sql.Query(`SELECT user_id FROM group_members WHERE group_id = ?`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}
