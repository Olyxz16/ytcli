package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// LocalIssue represents an issue in the local SQLite database.
type LocalIssue struct {
	ID          int64      `json:"id"`
	RemoteID    *string    `json:"remote_id,omitempty"`
	RemoteDBID  *string    `json:"remote_db_id,omitempty"`
	Summary     string     `json:"summary"`
	Description string     `json:"description"`
	State       string     `json:"state"`
	Priority    string     `json:"priority"`
	Assignee    *string    `json:"assignee,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	SyncedAt    *time.Time `json:"synced_at,omitempty"`
	SyncStatus  string     `json:"sync_status"`
	RemoteETag  *string    `json:"remote_etag,omitempty"`
	Tags        []string   `json:"tags"`
	Comments    int        `json:"comments"`
}

// CreateIssue inserts a new local issue.
func CreateIssue(db *sql.DB, summary, description, state, priority, assignee string) (*LocalIssue, error) {
	now := time.Now().UTC()
	var assigneePtr *string
	if assignee != "" {
		assigneePtr = &assignee
	}

	res, err := db.Exec(
		`INSERT INTO issues(summary, description, state, priority, assignee, created_at, updated_at, sync_status)
		 VALUES(?, ?, ?, ?, ?, ?, ?, 'local')`,
		summary, description, state, priority, assigneePtr, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("insert issue: %w", err)
	}

	id, _ := res.LastInsertId()
	return GetIssue(db, id)
}

// GetIssue retrieves an issue by its local ID.
func GetIssue(db *sql.DB, id int64) (*LocalIssue, error) {
	row := db.QueryRow(`
		SELECT id, remote_id, remote_db_id, summary, description, state, priority, assignee,
		       created_at, updated_at, synced_at, sync_status, remote_etag
		FROM issues WHERE id = ?`, id)
	return scanIssue(row)
}

// GetIssueByRemoteID retrieves an issue by its YouTrack remote ID.
func GetIssueByRemoteID(db *sql.DB, remoteID string) (*LocalIssue, error) {
	row := db.QueryRow(`
		SELECT id, remote_id, remote_db_id, summary, description, state, priority, assignee,
		       created_at, updated_at, synced_at, sync_status, remote_etag
		FROM issues WHERE remote_id = ?`, remoteID)
	return scanIssue(row)
}

// UpdateIssue updates fields of a local issue and marks it modified.
func UpdateIssue(db *sql.DB, id int64, updates map[string]interface{}) (*LocalIssue, error) {
	now := time.Now().UTC()
	allowed := map[string]bool{"summary": true, "description": true, "state": true, "priority": true, "assignee": true}

	var setClauses []string
	var args []interface{}
	for key, val := range updates {
		if !allowed[key] {
			continue
		}
		setClauses = append(setClauses, fmt.Sprintf("%s = ?", key))
		args = append(args, val)
	}
	if len(setClauses) == 0 {
		return GetIssue(db, id)
	}

	setClauses = append(setClauses, "updated_at = ?", "sync_status = CASE WHEN sync_status = 'synced' THEN 'modified' ELSE sync_status END")
	args = append(args, now)
	args = append(args, id)

	query := fmt.Sprintf("UPDATE issues SET %s WHERE id = ?", strings.Join(setClauses, ", "))
	if _, err := db.Exec(query, args...); err != nil {
		return nil, fmt.Errorf("update issue: %w", err)
	}

	return GetIssue(db, id)
}

// ListIssues lists all local issues, optionally filtered by query.
func ListIssues(db *sql.DB, query string, limit int) ([]LocalIssue, error) {
	var rows *sql.Rows
	var err error

	if query != "" {
		// Use FTS5 for text search
		ftsQuery := strings.Join(strings.Fields(query), " OR ")
		rows, err = db.Query(`
			SELECT i.id, i.remote_id, i.remote_db_id, i.summary, i.description, i.state, i.priority, i.assignee,
			       i.created_at, i.updated_at, i.synced_at, i.sync_status, i.remote_etag
			FROM issues i
			JOIN issues_fts fts ON i.id = fts.rowid
			WHERE issues_fts MATCH ?
			ORDER BY i.updated_at DESC
			LIMIT ?`, ftsQuery, limit)
	} else {
		rows, err = db.Query(`
			SELECT id, remote_id, remote_db_id, summary, description, state, priority, assignee,
			       created_at, updated_at, synced_at, sync_status, remote_etag
			FROM issues
			ORDER BY updated_at DESC
			LIMIT ?`, limit)
	}

	if err != nil {
		return nil, fmt.Errorf("list issues: %w", err)
	}
	defer rows.Close()

	var issues []LocalIssue
	for rows.Next() {
		issue, err := scanIssueRow(rows)
		if err != nil {
			return nil, err
		}
		issues = append(issues, *issue)
	}
	return issues, nil
}

// DeleteIssue removes a local issue and its related data.
func DeleteIssue(db *sql.DB, id int64) error {
	_, err := db.Exec("DELETE FROM issues WHERE id = ?", id)
	return err
}

// CountIssues returns the total number of local issues.
func CountIssues(db *sql.DB) (int, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM issues").Scan(&count)
	return count, err
}

// SetRemoteID maps a local issue to a remote YouTrack ID after sync.
func SetRemoteID(db *sql.DB, localID int64, remoteID, remoteDBID string) error {
	_, err := db.Exec(
		"UPDATE issues SET remote_id = ?, remote_db_id = ?, sync_status = 'synced', synced_at = ? WHERE id = ?",
		remoteID, remoteDBID, time.Now().UTC(), localID,
	)
	return err
}

// SetSyncStatus updates the sync status of an issue.
func SetSyncStatus(db *sql.DB, id int64, status string) error {
	_, err := db.Exec("UPDATE issues SET sync_status = ? WHERE id = ?", status, id)
	return err
}

// scanIssue scans a single issue from a sql.Row.
func scanIssue(row *sql.Row) (*LocalIssue, error) {
	var issue LocalIssue
	var remoteID, remoteDBID, assignee, remoteETag sql.NullString
	var syncedAt sql.NullTime

	err := row.Scan(
		&issue.ID, &remoteID, &remoteDBID, &issue.Summary, &issue.Description,
		&issue.State, &issue.Priority, &assignee,
		&issue.CreatedAt, &issue.UpdatedAt, &syncedAt, &issue.SyncStatus, &remoteETag,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if remoteID.Valid {
		issue.RemoteID = &remoteID.String
	}
	if remoteDBID.Valid {
		issue.RemoteDBID = &remoteDBID.String
	}
	if assignee.Valid {
		issue.Assignee = &assignee.String
	}
	if syncedAt.Valid {
		issue.SyncedAt = &syncedAt.Time
	}
	if remoteETag.Valid {
		issue.RemoteETag = &remoteETag.String
	}

	return &issue, nil
}

// scanIssueRow scans a single issue from sql.Rows.
func scanIssueRow(rows *sql.Rows) (*LocalIssue, error) {
	var issue LocalIssue
	var remoteID, remoteDBID, assignee, remoteETag sql.NullString
	var syncedAt sql.NullTime

	err := rows.Scan(
		&issue.ID, &remoteID, &remoteDBID, &issue.Summary, &issue.Description,
		&issue.State, &issue.Priority, &assignee,
		&issue.CreatedAt, &issue.UpdatedAt, &syncedAt, &issue.SyncStatus, &remoteETag,
	)
	if err != nil {
		return nil, err
	}

	if remoteID.Valid {
		issue.RemoteID = &remoteID.String
	}
	if remoteDBID.Valid {
		issue.RemoteDBID = &remoteDBID.String
	}
	if assignee.Valid {
		issue.Assignee = &assignee.String
	}
	if syncedAt.Valid {
		issue.SyncedAt = &syncedAt.Time
	}
	if remoteETag.Valid {
		issue.RemoteETag = &remoteETag.String
	}

	return &issue, nil
}

// LoadIssueTags populates the Tags field for a slice of issues.
func LoadIssueTags(db *sql.DB, issues []LocalIssue) error {
	for i := range issues {
		tags, err := GetIssueTags(db, issues[i].ID)
		if err != nil {
			return err
		}
		issues[i].Tags = tags
	}
	return nil
}

// LoadIssueComments counts comments for a slice of issues.
func LoadIssueComments(db *sql.DB, issues []LocalIssue) error {
	for i := range issues {
		count, err := CountComments(db, issues[i].ID)
		if err != nil {
			return err
		}
		issues[i].Comments = count
	}
	return nil
}

// ToJSON serializes a local issue to JSON (for sync queue payload).
func (i LocalIssue) ToJSON() (string, error) {
	data, err := json.Marshal(i)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
