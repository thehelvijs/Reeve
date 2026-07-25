package store

import (
	"database/sql"
	"time"
)

// Collection is a named, owned set of tools used to group the catalog.
type Collection struct {
	ID          string
	Name        string
	Description string
	Visibility  string
	CreatorID   string
	IconPath    string
	CreatedAt   time.Time
}

const collectionSelect = `SELECT id, name, description, visibility, creator_id, icon_path, created_at
	FROM collections`

// CreateCollection inserts a collection, defaulting to public visibility.
func (db *DB) CreateCollection(c Collection) (Collection, error) {
	c.ID = NewID()
	c.CreatedAt = time.Now().UTC()
	if c.Visibility == "" {
		c.Visibility = VisibilityPublic
	}
	_, err := db.sql.Exec(
		`INSERT INTO collections(id, name, description, visibility, creator_id, icon_path, created_at)
		 VALUES (?,?,?,?,?,?,?)`,
		c.ID, c.Name, c.Description, c.Visibility, c.CreatorID, c.IconPath,
		c.CreatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return Collection{}, err
	}
	return c, nil
}

// GetCollection returns a collection by id, or ErrNotFound.
func (db *DB) GetCollection(id string) (Collection, error) {
	row := db.sql.QueryRow(collectionSelect+` WHERE id = ?`, id)
	c, err := scanCollection(row)
	if err == sql.ErrNoRows {
		return Collection{}, ErrNotFound
	}
	return c, err
}

// UpdateCollection overwrites the editable fields of a collection.
func (db *DB) UpdateCollection(c Collection) error {
	return db.exec1(
		`UPDATE collections SET name=?, description=?, visibility=? WHERE id=?`,
		c.Name, c.Description, c.Visibility, c.ID)
}

// DeleteCollection removes a collection; membership and grants cascade.
func (db *DB) DeleteCollection(id string) error {
	return db.exec1(`DELETE FROM collections WHERE id = ?`, id)
}

// SetCollectionIconPath sets (or clears, when empty) a collection's icon path.
func (db *DB) SetCollectionIconPath(id, path string) error {
	return db.exec1(`UPDATE collections SET icon_path = ? WHERE id = ?`, path, id)
}

// ListCollectionsVisibleTo returns the collections a user may see, ordered by
// name. Admins see everything.
func (db *DB) ListCollectionsVisibleTo(userID string, isAdmin bool) ([]Collection, error) {
	q := collectionSelect
	var args []any
	if !isAdmin {
		q += ` WHERE ` + collectionVisibleClause
		args = collectionVisibleArgs(userID)
	}
	return db.queryCollections(q+` ORDER BY name`, args...)
}

// ListPublicCollections returns anonymous-visible collections ordered by name.
func (db *DB) ListPublicCollections() ([]Collection, error) {
	return db.queryCollections(collectionSelect + ` WHERE visibility = 'public' ORDER BY name`)
}

// CanSeeCollection reports whether the user may see the collection.
func (db *DB) CanSeeCollection(userID string, isAdmin bool, id string) (bool, error) {
	q := `SELECT 1 FROM collections WHERE id = ?`
	args := []any{id}
	if !isAdmin {
		q += ` AND ` + collectionVisibleClause
		args = append(args, collectionVisibleArgs(userID)...)
	}
	var one int
	if err := db.sql.QueryRow(q, args...).Scan(&one); err != nil {
		return false, nil
	}
	return true, nil
}

// collectionVisibleClause matches collections a non-admin may see. Bind with
// collectionVisibleArgs: public, creator, editor or granted principal.
const collectionVisibleClause = `(
	visibility = 'public'
	OR creator_id = ?
	OR id IN (SELECT collection_id FROM collection_editors
	          WHERE principal_type='user' AND principal_id = ?)
	OR id IN (SELECT collection_id FROM collection_editors
	          WHERE principal_type='group'
	            AND principal_id IN (SELECT group_id FROM group_members WHERE user_id = ?))
	OR id IN (SELECT collection_id FROM collection_visibility
	          WHERE principal_type='user' AND principal_id = ?)
	OR id IN (SELECT collection_id FROM collection_visibility
	          WHERE principal_type='group'
	            AND principal_id IN (SELECT group_id FROM group_members WHERE user_id = ?))
)`

func collectionVisibleArgs(userID string) []any {
	return []any{userID, userID, userID, userID, userID}
}

func (db *DB) queryCollections(q string, args ...any) ([]Collection, error) {
	rows, err := db.sql.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Collection{}
	for rows.Next() {
		c, err := scanCollection(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func scanCollection(row scanner) (Collection, error) {
	var c Collection
	var created string
	if err := row.Scan(&c.ID, &c.Name, &c.Description, &c.Visibility,
		&c.CreatorID, &c.IconPath, &created); err != nil {
		return Collection{}, err
	}
	c.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	return c, nil
}

// SetCollectionTools replaces a collection's whole membership set.
func (db *DB) SetCollectionTools(collectionID string, toolIDs []string) error {
	return db.replaceMembership(`collection_id`, collectionID, `tool_id`, toolIDs)
}

// SetToolCollections replaces the set of collections a tool belongs to.
func (db *DB) SetToolCollections(toolID string, collectionIDs []string) error {
	return db.replaceMembership(`tool_id`, toolID, `collection_id`, collectionIDs)
}

func (db *DB) replaceMembership(keyCol, keyID, valCol string, valIDs []string) error {
	return db.inTx(func(tx *sql.Tx) error {
		if _, err := tx.Exec(`DELETE FROM collection_tools WHERE `+keyCol+` = ?`, keyID); err != nil {
			return err
		}
		stmt := `INSERT INTO collection_tools(` + keyCol + `, ` + valCol + `) VALUES (?, ?) ON CONFLICT DO NOTHING`
		for _, v := range valIDs {
			if v == "" {
				continue
			}
			if _, err := tx.Exec(stmt, keyID, v); err != nil {
				return err
			}
		}
		return nil
	})
}

// AddCollectionTool puts a tool in a collection (idempotent).
func (db *DB) AddCollectionTool(collectionID, toolID string) error {
	_, err := db.sql.Exec(
		`INSERT INTO collection_tools(collection_id, tool_id) VALUES (?, ?) ON CONFLICT DO NOTHING`,
		collectionID, toolID)
	return err
}

// RemoveCollectionTool takes a tool out of a collection.
func (db *DB) RemoveCollectionTool(collectionID, toolID string) error {
	_, err := db.sql.Exec(
		`DELETE FROM collection_tools WHERE collection_id = ? AND tool_id = ?`, collectionID, toolID)
	return err
}

// ListCollectionToolIDs returns the tool ids in a collection.
func (db *DB) ListCollectionToolIDs(collectionID string) ([]string, error) {
	rows, err := db.sql.Query(`SELECT tool_id FROM collection_tools WHERE collection_id = ?`, collectionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// CountCollectionTools returns how many tools are in a collection.
func (db *DB) CountCollectionTools(collectionID string) (int, error) {
	var n int
	err := db.sql.QueryRow(`SELECT COUNT(*) FROM collection_tools WHERE collection_id = ?`,
		collectionID).Scan(&n)
	return n, err
}

// CollectionsForTools maps each given tool id to the collections it belongs to,
// without applying any visibility filter.
func (db *DB) CollectionsForTools(toolIDs []string) (map[string][]Collection, error) {
	out := map[string][]Collection{}
	if len(toolIDs) == 0 {
		return out, nil
	}
	q := `SELECT ct.tool_id, c.id, c.name, c.description, c.visibility, c.creator_id, c.icon_path, c.created_at
		FROM collection_tools ct JOIN collections c ON c.id = ct.collection_id
		WHERE ct.tool_id IN (` + placeholders(len(toolIDs)) + `) ORDER BY c.name`
	args := make([]any, len(toolIDs))
	for i, id := range toolIDs {
		args[i] = id
	}
	rows, err := db.sql.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var toolID string
		var c Collection
		var created string
		if err := rows.Scan(&toolID, &c.ID, &c.Name, &c.Description, &c.Visibility,
			&c.CreatorID, &c.IconPath, &created); err != nil {
			return nil, err
		}
		c.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		out[toolID] = append(out[toolID], c)
	}
	return out, rows.Err()
}

// AddCollectionEditor grants edit rights on a collection to a user or group.
func (db *DB) AddCollectionEditor(collectionID, principalType, principalID string) error {
	_, err := db.sql.Exec(
		`INSERT INTO collection_editors(collection_id, principal_type, principal_id)
		 VALUES (?, ?, ?) ON CONFLICT DO NOTHING`, collectionID, principalType, principalID)
	return err
}

// RemoveCollectionEditor revokes edit rights.
func (db *DB) RemoveCollectionEditor(collectionID, principalType, principalID string) error {
	_, err := db.sql.Exec(
		`DELETE FROM collection_editors WHERE collection_id = ? AND principal_type = ? AND principal_id = ?`,
		collectionID, principalType, principalID)
	return err
}

// ListCollectionEditors returns the principals with edit rights.
func (db *DB) ListCollectionEditors(collectionID string) ([]VisibilityGrant, error) {
	return db.listCollectionGrants(`collection_editors`, collectionID)
}

// AddCollectionVisibility grants sight of a restricted collection.
func (db *DB) AddCollectionVisibility(collectionID, principalType, principalID string) error {
	_, err := db.sql.Exec(
		`INSERT INTO collection_visibility(collection_id, principal_type, principal_id)
		 VALUES (?, ?, ?) ON CONFLICT DO NOTHING`, collectionID, principalType, principalID)
	return err
}

// RemoveCollectionVisibility revokes a visibility grant.
func (db *DB) RemoveCollectionVisibility(collectionID, principalType, principalID string) error {
	_, err := db.sql.Exec(
		`DELETE FROM collection_visibility WHERE collection_id = ? AND principal_type = ? AND principal_id = ?`,
		collectionID, principalType, principalID)
	return err
}

// ListCollectionVisibility returns the principals granted sight of a collection.
func (db *DB) ListCollectionVisibility(collectionID string) ([]VisibilityGrant, error) {
	return db.listCollectionGrants(`collection_visibility`, collectionID)
}

func (db *DB) listCollectionGrants(table, collectionID string) ([]VisibilityGrant, error) {
	rows, err := db.sql.Query(
		`SELECT principal_type, principal_id FROM `+table+` WHERE collection_id = ?`, collectionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []VisibilityGrant{}
	for rows.Next() {
		var g VisibilityGrant
		if err := rows.Scan(&g.PrincipalType, &g.PrincipalID); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

// CanEditCollection reports whether the user may modify the collection: admin,
// creator, or an editor principal.
func (db *DB) CanEditCollection(userID string, isAdmin bool, collectionID string) (bool, error) {
	var one int
	if isAdmin {
		err := db.sql.QueryRow(`SELECT 1 FROM collections WHERE id = ?`, collectionID).Scan(&one)
		return err == nil, nil
	}
	err := db.sql.QueryRow(`SELECT 1 FROM collections WHERE id = ? AND (`+
		collectionEditableClause+`)`, collectionID, userID, userID, userID).Scan(&one)
	if err != nil {
		return false, nil
	}
	return true, nil
}

// collectionEditableClause matches collections a non-admin may modify. Bind
// (userID x3): creator, user editor, or group editor.
const collectionEditableClause = `
	creator_id = ?
	OR id IN (SELECT collection_id FROM collection_editors
	          WHERE principal_type='user' AND principal_id = ?)
	OR id IN (SELECT collection_id FROM collection_editors
	          WHERE principal_type='group'
	            AND principal_id IN (SELECT group_id FROM group_members WHERE user_id = ?))`
