package store

import (
	"database/sql"
	"time"
)

// PipelineGroup is a named set of repo paths an operator watches together, such
// as every firmware repo regardless of which GitLab group or GitHub org holds
// it.
type PipelineGroup struct {
	ID   string
	Name string
	// Provider is "gitlab" or "github": which forge the paths belong to.
	Provider  string
	CreatedAt time.Time
	// Projects are full paths, "group/subgroup/project" or "owner/repo",
	// ordered by path.
	Projects []string
}

// ListPipelineGroups returns every group with its members, ordered by name. The
// membership comes back in one query rather than one per group.
func (db *DB) ListPipelineGroups() ([]PipelineGroup, error) {
	rows, err := db.sql.Query(`SELECT id, name, provider, created_at FROM pipeline_groups ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	groups := []PipelineGroup{}
	byID := map[string]int{}
	for rows.Next() {
		var g PipelineGroup
		var created string
		if err := rows.Scan(&g.ID, &g.Name, &g.Provider, &created); err != nil {
			return nil, err
		}
		g.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		g.Projects = []string{}
		byID[g.ID] = len(groups)
		groups = append(groups, g)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	members, err := db.sql.Query(
		`SELECT group_id, project_path FROM pipeline_group_projects ORDER BY project_path`)
	if err != nil {
		return nil, err
	}
	defer members.Close()
	for members.Next() {
		var groupID, path string
		if err := members.Scan(&groupID, &path); err != nil {
			return nil, err
		}
		if i, ok := byID[groupID]; ok {
			groups[i].Projects = append(groups[i].Projects, path)
		}
	}
	return groups, members.Err()
}

// CreatePipelineGroup inserts an empty group on one provider.
func (db *DB) CreatePipelineGroup(name, provider string) (PipelineGroup, error) {
	g := PipelineGroup{
		ID: NewID(), Name: name, Provider: provider,
		CreatedAt: time.Now().UTC(), Projects: []string{},
	}
	_, err := db.sql.Exec(`INSERT INTO pipeline_groups(id, name, provider, created_at) VALUES (?,?,?,?)`,
		g.ID, g.Name, g.Provider, g.CreatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return PipelineGroup{}, err
	}
	return g, nil
}

// RenamePipelineGroup renames a group, or returns ErrNotFound.
func (db *DB) RenamePipelineGroup(id, name string) error {
	return db.exec1(`UPDATE pipeline_groups SET name = ? WHERE id = ?`, name, id)
}

// DeletePipelineGroup removes a group; its membership cascades.
func (db *DB) DeletePipelineGroup(id string) error {
	return db.exec1(`DELETE FROM pipeline_groups WHERE id = ?`, id)
}

// AddPipelineGroupProject puts a project path in a group (idempotent).
func (db *DB) AddPipelineGroupProject(groupID, path string) error {
	_, err := db.sql.Exec(
		`INSERT INTO pipeline_group_projects(group_id, project_path) VALUES (?,?) ON CONFLICT DO NOTHING`,
		groupID, path)
	return err
}

// RemovePipelineGroupProject takes a project path out of a group.
func (db *DB) RemovePipelineGroupProject(groupID, path string) error {
	_, err := db.sql.Exec(
		`DELETE FROM pipeline_group_projects WHERE group_id = ? AND project_path = ?`, groupID, path)
	return err
}

// CountPipelineGroupProjects returns how many projects a group holds.
func (db *DB) CountPipelineGroupProjects(groupID string) (int, error) {
	var n int
	err := db.sql.QueryRow(
		`SELECT COUNT(*) FROM pipeline_group_projects WHERE group_id = ?`, groupID).Scan(&n)
	return n, err
}

// PipelineGroupExists reports whether a group id is real, so a write against a
// deleted group fails as not-found rather than inserting an orphan row that the
// foreign key would reject with a less useful error.
func (db *DB) PipelineGroupExists(id string) (bool, error) {
	var one int
	err := db.sql.QueryRow(`SELECT 1 FROM pipeline_groups WHERE id = ?`, id).Scan(&one)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}
