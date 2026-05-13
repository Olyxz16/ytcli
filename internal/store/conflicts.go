package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// Conflict represents a detected sync conflict.
type Conflict struct {
	ID               int64     `json:"id"`
	IssueID          int64     `json:"issue_id"`
	ProviderName     string    `json:"provider_name"`
	ProviderRef      string    `json:"provider_ref"`
	LocalSummary     string    `json:"local_summary"`
	LocalDescription string    `json:"local_description"`
	LocalState       string    `json:"local_state"`
	LocalPriority    string    `json:"local_priority"`
	LocalAssignee    string    `json:"local_assignee"`
	RemoteSummary     string    `json:"remote_summary"`
	RemoteDescription string    `json:"remote_description"`
	RemoteState       string    `json:"remote_state"`
	RemotePriority    string    `json:"remote_priority"`
	RemoteAssignee    string    `json:"remote_assignee"`
	DetectedAt        time.Time `json:"detected_at"`
	ResolvedAt        *time.Time `json:"resolved_at,omitempty"`
	Resolution        *string   `json:"resolution,omitempty"`
}

// CreateConflict stores a new conflict record.
func CreateConflict(db *sql.DB, issueID int64, providerName, providerRef string,
	localSummary, localDesc, localState, localPriority, localAssignee string,
	remoteSummary, remoteDesc, remoteState, remotePriority, remoteAssignee string,
) (*Conflict, error) {
	res, err := db.Exec(
		`INSERT INTO conflicts(issue_id, provider_name, provider_ref,
			local_summary, local_description, local_state, local_priority, local_assignee,
			remote_summary, remote_description, remote_state, remote_priority, remote_assignee)
		 VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(issue_id) DO UPDATE SET
			provider_name=excluded.provider_name,
			provider_ref=excluded.provider_ref,
			local_summary=excluded.local_summary,
			local_description=excluded.local_description,
			local_state=excluded.local_state,
			local_priority=excluded.local_priority,
			local_assignee=excluded.local_assignee,
			remote_summary=excluded.remote_summary,
			remote_description=excluded.remote_description,
			remote_state=excluded.remote_state,
			remote_priority=excluded.remote_priority,
			remote_assignee=excluded.remote_assignee,
			detected_at=excluded.detected_at,
			resolved_at=NULL,
			resolution=NULL`,
		issueID, providerName, providerRef,
		localSummary, localDesc, localState, localPriority, localAssignee,
		remoteSummary, remoteDesc, remoteState, remotePriority, remoteAssignee,
	)
	if err != nil {
		return nil, fmt.Errorf("insert conflict: %w", err)
	}
	id, _ := res.LastInsertId()
	if id == 0 {
		id = issueID
	}
	return GetConflict(db, id)
}

// GetConflict retrieves a conflict by ID.
func GetConflict(db *sql.DB, id int64) (*Conflict, error) {
	var c Conflict
	var resolvedAt sql.NullTime
	var resolution sql.NullString
	err := db.QueryRow(`
		SELECT id, issue_id, provider_name, provider_ref,
			local_summary, local_description, local_state, local_priority, local_assignee,
			remote_summary, remote_description, remote_state, remote_priority, remote_assignee,
			detected_at, resolved_at, resolution
		 FROM conflicts WHERE id = ?`, id).Scan(
		&c.ID, &c.IssueID, &c.ProviderName, &c.ProviderRef,
		&c.LocalSummary, &c.LocalDescription, &c.LocalState, &c.LocalPriority, &c.LocalAssignee,
		&c.RemoteSummary, &c.RemoteDescription, &c.RemoteState, &c.RemotePriority, &c.RemoteAssignee,
		&c.DetectedAt, &resolvedAt, &resolution,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if resolvedAt.Valid {
		c.ResolvedAt = &resolvedAt.Time
	}
	if resolution.Valid {
		c.Resolution = &resolution.String
	}
	return &c, nil
}

// GetConflicts returns all unresolved conflicts.
func GetConflicts(db *sql.DB) ([]Conflict, error) {
	rows, err := db.Query(`
		SELECT id, issue_id, provider_name, provider_ref,
			local_summary, local_description, local_state, local_priority, local_assignee,
			remote_summary, remote_description, remote_state, remote_priority, remote_assignee,
			detected_at, resolved_at, resolution
		 FROM conflicts WHERE resolution IS NULL
		 ORDER BY detected_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanConflicts(rows)
}

// ResolveConflict marks a conflict as resolved.
func ResolveConflict(db *sql.DB, id int64, resolution string) error {
	_, err := db.Exec(
		`UPDATE conflicts SET resolution = ?, resolved_at = ? WHERE id = ?`,
		resolution, time.Now().UTC(), id,
	)
	return err
}

// DeleteConflict removes a conflict record.
func DeleteConflict(db *sql.DB, id int64) error {
	_, err := db.Exec(`DELETE FROM conflicts WHERE id = ?`, id)
	return err
}

func scanConflicts(rows *sql.Rows) ([]Conflict, error) {
	var conflicts []Conflict
	for rows.Next() {
		var c Conflict
		var resolvedAt sql.NullTime
		var resolution sql.NullString
		err := rows.Scan(
			&c.ID, &c.IssueID, &c.ProviderName, &c.ProviderRef,
			&c.LocalSummary, &c.LocalDescription, &c.LocalState, &c.LocalPriority, &c.LocalAssignee,
			&c.RemoteSummary, &c.RemoteDescription, &c.RemoteState, &c.RemotePriority, &c.RemoteAssignee,
			&c.DetectedAt, &resolvedAt, &resolution,
		)
		if err != nil {
			return nil, err
		}
		if resolvedAt.Valid {
			c.ResolvedAt = &resolvedAt.Time
		}
		if resolution.Valid {
			c.Resolution = &resolution.String
		}
		conflicts = append(conflicts, c)
	}
	return conflicts, rows.Err()
}

// ConflictCount returns the number of unresolved conflicts.
func ConflictCount(db *sql.DB) (int, error) {
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM conflicts WHERE resolution IS NULL`).Scan(&count)
	return count, err
}

// DiffConflictData holds the diff between local and remote for JSON export.
type DiffConflictData struct {
	Local  map[string]interface{} `json:"local"`
	Remote map[string]interface{} `json:"remote"`
}

// ConflictDiffJSON returns a JSON representation of the conflict diff.
func (c *Conflict) DiffJSON() ([]byte, error) {
	data := DiffConflictData{
		Local: map[string]interface{}{
			"summary":     c.LocalSummary,
			"description": c.LocalDescription,
			"state":       c.LocalState,
			"priority":    c.LocalPriority,
			"assignee":    c.LocalAssignee,
		},
		Remote: map[string]interface{}{
			"summary":     c.RemoteSummary,
			"description": c.RemoteDescription,
			"state":       c.RemoteState,
			"priority":    c.RemotePriority,
			"assignee":    c.RemoteAssignee,
		},
	}
	return json.Marshal(data)
}