package store

import (
	"encoding/json"
	"strings"
	"time"
)

// Visibility values for a tool.
const (
	VisibilityPublic     = "public"
	VisibilityRestricted = "restricted"
)

// Tool is a catalog entry.
type Tool struct {
	ID               string
	Name             string
	Description      string
	Category         string
	Tags             []string
	Scheme           string
	Address          string
	Port             int
	URL              string
	PhysicalLocation string
	HostID           string
	SourceType       string
	SourceRef        string
	Visibility       string
	CreatorID        string
	LogAlertEnabled  bool
	CreatedAt        time.Time
	IconPath         string
	ThumbnailPath    string
}

// SetToolIconPath sets (or clears, when empty) a tool's stored icon path.
func (db *DB) SetToolIconPath(id, path string) error {
	return db.exec1(`UPDATE tools SET icon_path = ? WHERE id = ?`, path, id)
}

// SetToolThumbnailPath sets (or clears, when empty) a tool's thumbnail path.
func (db *DB) SetToolThumbnailPath(id, path string) error {
	return db.exec1(`UPDATE tools SET thumbnail_path = ? WHERE id = ?`, path, id)
}

// ToolFilter narrows a catalog listing. Empty fields are ignored.
type ToolFilter struct {
	Search     string
	Category   string
	HostID     string
	SourceType string
}

// CountTools returns the total number of tools.
func (db *DB) CountTools() (int, error) {
	var n int
	err := db.sql.QueryRow("SELECT COUNT(*) FROM tools").Scan(&n)
	return n, err
}

// CreateTool inserts a tool with the given creator.
func (db *DB) CreateTool(t Tool) (Tool, error) {
	t.ID = NewID()
	t.CreatedAt = time.Now().UTC()
	if t.Visibility == "" {
		t.Visibility = VisibilityPublic
	}
	if t.SourceType == "" {
		t.SourceType = "manual"
	}
	tags, _ := json.Marshal(t.Tags)
	_, err := db.sql.Exec(
		`INSERT INTO tools(id, name, description, category, tags, scheme, address, port, url,
			physical_location, host_id, source_type, source_ref, visibility, creator_id,
			log_alert_enabled, created_at)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		t.ID, t.Name, t.Description, t.Category, string(tags), t.Scheme, t.Address, t.Port, t.URL,
		t.PhysicalLocation, nullable(t.HostID), t.SourceType, t.SourceRef, t.Visibility, t.CreatorID,
		boolToInt(t.LogAlertEnabled), t.CreatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return Tool{}, err
	}
	return t, nil
}

// UpdateTool overwrites the editable fields of a tool.
func (db *DB) UpdateTool(t Tool) error {
	tags, _ := json.Marshal(t.Tags)
	return db.exec1(
		`UPDATE tools SET name=?, description=?, category=?, tags=?, scheme=?, address=?, port=?, url=?,
			physical_location=?, host_id=?, source_type=?, source_ref=?, visibility=?,
			log_alert_enabled=? WHERE id=?`,
		t.Name, t.Description, t.Category, string(tags), t.Scheme, t.Address, t.Port, t.URL,
		t.PhysicalLocation, nullable(t.HostID), t.SourceType, t.SourceRef, t.Visibility,
		boolToInt(t.LogAlertEnabled), t.ID,
	)
}

// DeleteTool removes a tool (visibility rows cascade).
func (db *DB) DeleteTool(id string) error {
	return db.exec1(`DELETE FROM tools WHERE id = ?`, id)
}

// GetTool returns a tool by id, or ErrNotFound.
func (db *DB) GetTool(id string) (Tool, error) {
	rows, err := db.sql.Query(toolSelect+` WHERE id = ?`, id)
	if err != nil {
		return Tool{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		return Tool{}, ErrNotFound
	}
	return db.scanTool(rows)
}

// ListToolsVisibleTo returns the tools a user may see, honoring visibility and
// the optional filter. Admins see everything.
func (db *DB) ListToolsVisibleTo(userID string, isAdmin bool, f ToolFilter) ([]Tool, error) {
	var where []string
	var args []any

	if !isAdmin {
		where = append(where, toolVisibleClause)
		args = append(args, userID, userID, userID)
	}
	if f.Search != "" {
		where = append(where, `(name LIKE ? OR description LIKE ? OR tags LIKE ?)`)
		like := "%" + f.Search + "%"
		args = append(args, like, like, like)
	}
	if f.Category != "" {
		where = append(where, `category = ?`)
		args = append(args, f.Category)
	}
	if f.HostID != "" {
		where = append(where, `host_id = ?`)
		args = append(args, f.HostID)
	}
	if f.SourceType != "" {
		where = append(where, `source_type = ?`)
		args = append(args, f.SourceType)
	}

	q := toolSelect
	if len(where) > 0 {
		q += " WHERE " + strings.Join(where, " AND ")
	}
	q += " ORDER BY name"

	rows, err := db.sql.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Tool
	for rows.Next() {
		t, err := db.scanTool(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// ListPublicTools returns tools with public visibility (anonymous-visible),
// honoring the optional filter. Used by the unauthenticated portal endpoint.
func (db *DB) ListPublicTools(f ToolFilter) ([]Tool, error) {
	where := []string{`visibility = 'public'`}
	var args []any
	if f.Search != "" {
		where = append(where, `(name LIKE ? OR description LIKE ? OR tags LIKE ?)`)
		like := "%" + f.Search + "%"
		args = append(args, like, like, like)
	}
	if f.Category != "" {
		where = append(where, `category = ?`)
		args = append(args, f.Category)
	}
	if f.HostID != "" {
		where = append(where, `host_id = ?`)
		args = append(args, f.HostID)
	}
	if f.SourceType != "" {
		where = append(where, `source_type = ?`)
		args = append(args, f.SourceType)
	}

	q := toolSelect + " WHERE " + strings.Join(where, " AND ") + " ORDER BY name"
	rows, err := db.sql.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Tool
	for rows.Next() {
		t, err := db.scanTool(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// ListMonitoredTools returns tools bound to a host source (the alert engine's
// subjects), regardless of visibility.
func (db *DB) ListMonitoredTools() ([]Tool, error) {
	rows, err := db.sql.Query(toolSelect + ` WHERE host_id IS NOT NULL AND host_id != '' AND source_type != 'manual' AND source_ref != ''`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Tool
	for rows.Next() {
		t, err := db.scanTool(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// ToolIDsByCreator returns the ids of tools created by a user.
func (db *DB) ToolIDsByCreator(userID string) ([]string, error) {
	rows, err := db.sql.Query(`SELECT id FROM tools WHERE creator_id = ?`, userID)
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

// CanSeeTool reports whether the user may see the tool.
func (db *DB) CanSeeTool(userID string, isAdmin bool, toolID string) (bool, error) {
	if isAdmin {
		var one int
		err := db.sql.QueryRow(`SELECT 1 FROM tools WHERE id = ?`, toolID).Scan(&one)
		return err == nil, nil
	}
	var one int
	err := db.sql.QueryRow(
		`SELECT 1 FROM tools WHERE id = ? AND (`+toolVisibleClause+`)`,
		toolID, userID, userID, userID).Scan(&one)
	if err != nil {
		return false, nil
	}
	return one == 1, nil
}

// toolVisibleClause matches tools a non-admin user may see. Bind (userID x3):
// public OR own OR granted-to-user OR granted-to-one-of-user's-groups.
const toolVisibleClause = `(
	visibility = 'public'
	OR creator_id = ?
	OR id IN (SELECT tool_id FROM tool_visibility WHERE principal_type='user' AND principal_id = ?)
	OR id IN (
		SELECT tool_id FROM tool_visibility
		WHERE principal_type='group'
		  AND principal_id IN (SELECT group_id FROM group_members WHERE user_id = ?)
	)
)`

const toolSelect = `SELECT id, name, description, category, tags, scheme, address, port, url,
	physical_location, host_id, source_type, source_ref, visibility, creator_id,
	log_alert_enabled, created_at, icon_path, thumbnail_path FROM tools`

func (db *DB) scanTool(row scanner) (Tool, error) {
	var t Tool
	var tags, created string
	var hostID *string
	var logAlert int
	if err := row.Scan(&t.ID, &t.Name, &t.Description, &t.Category, &tags, &t.Scheme, &t.Address, &t.Port, &t.URL,
		&t.PhysicalLocation, &hostID, &t.SourceType, &t.SourceRef, &t.Visibility,
		&t.CreatorID, &logAlert, &created, &t.IconPath, &t.ThumbnailPath); err != nil {
		return Tool{}, err
	}
	json.Unmarshal([]byte(tags), &t.Tags)
	if t.Tags == nil {
		t.Tags = []string{}
	}
	if hostID != nil {
		t.HostID = *hostID
	}
	t.LogAlertEnabled = logAlert == 1
	t.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	return t, nil
}

func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
