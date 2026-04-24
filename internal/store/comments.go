package store

import (
	"database/sql"
	"fmt"
	"time"
)

// LocalComment represents a comment in the local database.
type LocalComment struct {
	ID         int64      `json:"id"`
	IssueID    int64      `json:"issue_id"`
	RemoteID   *string    `json:"remote_id,omitempty"`
	Text       string     `json:"text"`
	Author     string     `json:"author"`
	CreatedAt  time.Time  `json:"created_at"`
	SyncStatus string     `json:"sync_status"`
}

// CreateComment inserts a new local comment.
func CreateComment(db *sql.DB, issueID int64, text string) (*LocalComment, error) {
	now := time.Now().UTC()
	res, err := db.Exec(
		`INSERT INTO comments(issue_id, text, created_at, sync_status)
		 VALUES(?, ?, ?, 'local')`,
		issueID, text, now,
	)
	if err != nil {
		return nil, fmt.Errorf("insert comment: %w", err)
	}
	id, _ := res.LastInsertId()
	return GetComment(db, id)
}

// GetComment retrieves a comment by its local ID.
func GetComment(db *sql.DB, id int64) (*LocalComment, error) {
	row := db.QueryRow(`
		SELECT id, issue_id, remote_id, text, author, created_at, sync_status
		FROM comments WHERE id = ?`, id)
	return scanComment(row)
}

// ListComments retrieves all comments for an issue.
func ListComments(db *sql.DB, issueID int64) ([]LocalComment, error) {
	rows, err := db.Query(`
		SELECT id, issue_id, remote_id, text, author, created_at, sync_status
		FROM comments WHERE issue_id = ? ORDER BY created_at ASC`, issueID)
	if err != nil {
		return nil, fmt.Errorf("list comments: %w", err)
	}
	defer rows.Close()

	var comments []LocalComment
	for rows.Next() {
		c, err := scanCommentRow(rows)
		if err != nil {
			return nil, err
		}
		comments = append(comments, *c)
	}
	return comments, nil
}

// CountComments returns the number of comments for an issue.
func CountComments(db *sql.DB, issueID int64) (int, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM comments WHERE issue_id = ?", issueID).Scan(&count)
	return count, err
}

// scanComment scans a single comment from a sql.Row.
func scanComment(row *sql.Row) (*LocalComment, error) {
	var c LocalComment
	var remoteID sql.NullString
	err := row.Scan(&c.ID, &c.IssueID, &remoteID, &c.Text, &c.Author, &c.CreatedAt, &c.SyncStatus)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if remoteID.Valid {
		c.RemoteID = &remoteID.String
	}
	return &c, nil
}

// scanCommentRow scans a single comment from sql.Rows.
func scanCommentRow(rows *sql.Rows) (*LocalComment, error) {
	var c LocalComment
	var remoteID sql.NullString
	err := rows.Scan(&c.ID, &c.IssueID, &remoteID, &c.Text, &c.Author, &c.CreatedAt, &c.SyncStatus)
	if err != nil {
		return nil, err
	}
	if remoteID.Valid {
		c.RemoteID = &remoteID.String
	}
	return &c, nil
}
