package store

import (
	"database/sql"
	"errors"
	"time"
)

// Request statuses.
const (
	RequestPending  = "pending"
	RequestApproved = "approved"
	RequestDenied   = "denied"
)

// ErrDuplicateRequest is returned when a user already has an open request for a
// tool.
var ErrDuplicateRequest = errors.New("an open request already exists")

// AccessRequest is a user's request for a tool's credentials.
type AccessRequest struct {
	ID          string
	ToolID      string
	RequesterID string
	Status      string
	Note        string
	DecidedBy   *string
	DecidedAt   *time.Time
	CreatedAt   time.Time
}

// CreateAccessRequest opens a request, rejecting a duplicate open one.
func (db *DB) CreateAccessRequest(toolID, requesterID, note string) (AccessRequest, error) {
	var existing int
	err := db.sql.QueryRow(
		`SELECT 1 FROM access_requests WHERE tool_id = ? AND requester_id = ? AND status = 'pending'`,
		toolID, requesterID).Scan(&existing)
	if err == nil {
		return AccessRequest{}, ErrDuplicateRequest
	}
	r := AccessRequest{
		ID: NewID(), ToolID: toolID, RequesterID: requesterID,
		Status: RequestPending, Note: note, CreatedAt: time.Now().UTC(),
	}
	_, err = db.sql.Exec(
		`INSERT INTO access_requests(id, tool_id, requester_id, status, note, created_at)
		 VALUES (?,?,?,?,?,?)`,
		r.ID, r.ToolID, r.RequesterID, r.Status, r.Note, r.CreatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return AccessRequest{}, err
	}
	return r, nil
}

// GetAccessRequest returns a request by id.
func (db *DB) GetAccessRequest(id string) (AccessRequest, error) {
	r, err := scanRequest(db.sql.QueryRow(requestSelect+` WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return AccessRequest{}, ErrNotFound
	}
	return r, err
}

// ListRequestsByRequester returns a user's own requests, newest first.
func (db *DB) ListRequestsByRequester(userID string) ([]AccessRequest, error) {
	return db.queryRequests(requestSelect+` WHERE requester_id = ? ORDER BY created_at DESC`, userID)
}

// ListPendingRequestsForTools returns pending requests for the given tool ids.
func (db *DB) ListPendingRequestsForTools(toolIDs []string) ([]AccessRequest, error) {
	if len(toolIDs) == 0 {
		return nil, nil
	}
	q := requestSelect + ` WHERE status = 'pending' AND tool_id IN (` + placeholders(len(toolIDs)) + `) ORDER BY created_at DESC`
	args := make([]any, len(toolIDs))
	for i, id := range toolIDs {
		args[i] = id
	}
	return db.queryRequests(q, args...)
}

// ListAllPendingRequests returns every pending request (admin view).
func (db *DB) ListAllPendingRequests() ([]AccessRequest, error) {
	return db.queryRequests(requestSelect + ` WHERE status = 'pending' ORDER BY created_at DESC`)
}

// DecideAccessRequest sets a request's outcome. It must currently be pending.
func (db *DB) DecideAccessRequest(id, status, decidedBy string) error {
	return db.exec1(
		`UPDATE access_requests SET status = ?, decided_by = ?, decided_at = ?
		 WHERE id = ? AND status = 'pending'`,
		status, decidedBy, time.Now().UTC().Format(time.RFC3339Nano), id)
}

const requestSelect = `SELECT id, tool_id, requester_id, status, note, decided_by, decided_at, created_at FROM access_requests`

func (db *DB) queryRequests(q string, args ...any) ([]AccessRequest, error) {
	rows, err := db.sql.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AccessRequest
	for rows.Next() {
		r, err := scanRequest(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func scanRequest(row scanner) (AccessRequest, error) {
	var r AccessRequest
	var created string
	var decidedBy, decidedAt *string
	if err := row.Scan(&r.ID, &r.ToolID, &r.RequesterID, &r.Status, &r.Note, &decidedBy, &decidedAt, &created); err != nil {
		return AccessRequest{}, err
	}
	r.DecidedBy = decidedBy
	r.DecidedAt = parseNullableTime(decidedAt)
	r.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	return r, nil
}

func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	s := "?"
	for i := 1; i < n; i++ {
		s += ",?"
	}
	return s
}
