package store

import (
	"database/sql"
	"errors"
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

// GetGroup returns a group by id, or ErrNotFound.
func (db *DB) GetGroup(id string) (Group, error) {
	var g Group
	var created string
	err := db.sql.QueryRow(`SELECT id, name, created_at FROM groups WHERE id = ?`, id).
		Scan(&g.ID, &g.Name, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return Group{}, ErrNotFound
	}
	if err != nil {
		return Group{}, err
	}
	g.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
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

// Role values for a group membership. A moderator manages their own group's
// membership; a member is only carried by it.
const (
	GroupRoleModerator = "moderator"
	GroupRoleMember    = "member"
)

// GroupMember is one membership row: who, and what they may do in that group.
type GroupMember struct {
	UserID string
	Role   string
}

// AddGroupMember adds a user to a group, keeping the role an existing row
// already carries: adding someone twice must not silently demote them.
func (db *DB) AddGroupMember(groupID, userID, role string) error {
	_, err := db.sql.Exec(
		`INSERT INTO group_members(group_id, user_id, role) VALUES (?, ?, ?)
		 ON CONFLICT DO NOTHING`, groupID, userID, role)
	return err
}

// SetGroupMemberRole changes an existing membership's role, or ErrNotFound.
func (db *DB) SetGroupMemberRole(groupID, userID, role string) error {
	return db.exec1(`UPDATE group_members SET role = ? WHERE group_id = ? AND user_id = ?`,
		role, groupID, userID)
}

// RemoveGroupMember removes a user from a group.
func (db *DB) RemoveGroupMember(groupID, userID string) error {
	_, err := db.sql.Exec(
		`DELETE FROM group_members WHERE group_id = ? AND user_id = ?`, groupID, userID)
	return err
}

// ListGroupMembers returns a group's memberships, moderators first.
func (db *DB) ListGroupMembers(groupID string) ([]GroupMember, error) {
	rows, err := db.sql.Query(
		`SELECT user_id, role FROM group_members WHERE group_id = ?
		 ORDER BY role = 'moderator' DESC, user_id`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []GroupMember
	for rows.Next() {
		var m GroupMember
		if err := rows.Scan(&m.UserID, &m.Role); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// IsGroupModerator reports whether the user moderates that group.
func (db *DB) IsGroupModerator(groupID, userID string) (bool, error) {
	var n int
	err := db.sql.QueryRow(
		`SELECT COUNT(*) FROM group_members WHERE group_id = ? AND user_id = ? AND role = ?`,
		groupID, userID, GroupRoleModerator).Scan(&n)
	return n > 0, err
}

// ListGroupsModeratedBy returns the groups the user moderates, ordered by name.
func (db *DB) ListGroupsModeratedBy(userID string) ([]Group, error) {
	rows, err := db.sql.Query(
		`SELECT g.id, g.name, g.created_at FROM groups g
		 JOIN group_members m ON m.group_id = g.id
		 WHERE m.user_id = ? AND m.role = ? ORDER BY g.name`, userID, GroupRoleModerator)
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
