package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// QueueItem represents a pending sync operation.
type QueueItem struct {
	ID         int64     `json:"id"`
	Operation  string    `json:"operation"`
	EntityType string    `json:"entity_type"`
	LocalID    int64     `json:"local_id"`
	Payload    string    `json:"payload"`
	CreatedAt  time.Time `json:"created_at"`
	Attempts   int       `json:"attempts"`
	LastError  *string   `json:"last_error,omitempty"`
	Status     string    `json:"status"`
}

// Enqueue adds an operation to the sync queue.
func Enqueue(db *sql.DB, operation, entityType string, localID int64, payload map[string]interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}
	_, err = db.Exec(
		`INSERT INTO sync_queue(operation, entity_type, local_id, payload, status)
		 VALUES(?, ?, ?, ?, 'pending')`,
		operation, entityType, localID, string(data),
	)
	return err
}

// DequeuePending returns all pending queue items.
func DequeuePending(db *sql.DB) ([]QueueItem, error) {
	rows, err := db.Query(`
		SELECT id, operation, entity_type, local_id, payload, created_at, attempts, last_error, status
		FROM sync_queue
		WHERE status = 'pending' AND attempts < 3
		ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []QueueItem
	for rows.Next() {
		item, err := scanQueueItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, nil
}

// MarkInProgress sets a queue item to in-progress.
func MarkInProgress(db *sql.DB, id int64) error {
	_, err := db.Exec("UPDATE sync_queue SET status = 'in_progress' WHERE id = ?", id)
	return err
}

// MarkCompleted removes a queue item.
func MarkCompleted(db *sql.DB, id int64) error {
	_, err := db.Exec("UPDATE sync_queue SET status = 'completed' WHERE id = ?", id)
	return err
}

// MarkFailed increments attempts and stores the error.
func MarkFailed(db *sql.DB, id int64, errMsg string) error {
	_, err := db.Exec(
		"UPDATE sync_queue SET status = 'pending', attempts = attempts + 1, last_error = ? WHERE id = ?",
		errMsg, id,
	)
	return err
}

// QueueCount returns counts by status.
func QueueCount(db *sql.DB) (pending, failed, completed int, err error) {
	rows, err := db.Query("SELECT status, COUNT(*) FROM sync_queue GROUP BY status")
	if err != nil {
		return 0, 0, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return 0, 0, 0, err
		}
		switch status {
		case "pending":
			pending += count
		case "failed":
			failed += count
		case "completed":
			completed += count
		}
	}
	return pending, failed, completed, nil
}

// PruneCompleted removes completed queue items older than the given duration.
func PruneCompleted(db *sql.DB, olderThan time.Duration) error {
	cutoff := time.Now().UTC().Add(-olderThan)
	_, err := db.Exec("DELETE FROM sync_queue WHERE status = 'completed' AND created_at < ?", cutoff)
	return err
}

func scanQueueItem(rows *sql.Rows) (*QueueItem, error) {
	var item QueueItem
	var lastErr sql.NullString
	err := rows.Scan(&item.ID, &item.Operation, &item.EntityType, &item.LocalID,
		&item.Payload, &item.CreatedAt, &item.Attempts, &lastErr, &item.Status)
	if err != nil {
		return nil, err
	}
	if lastErr.Valid {
		item.LastError = &lastErr.String
	}
	return &item, nil
}
