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
