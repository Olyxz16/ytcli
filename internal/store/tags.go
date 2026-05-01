package store

import (
	"database/sql"
	"fmt"
)

// CreateTag inserts a new tag if it doesn't exist.
func CreateTag(db *sql.DB, name string) (int64, error) {
	res, err := db.Exec("INSERT OR IGNORE INTO tags(name) VALUES(?)", name)
	if err != nil {
		return 0, fmt.Errorf("insert tag: %w", err)
	}
	id, _ := res.LastInsertId()
	if id == 0 {
		var existingID int64
		err := db.QueryRow("SELECT id FROM tags WHERE name = ?", name).Scan(&existingID)
		if err != nil {
			return 0, err
		}
		return existingID, nil
	}
	return id, nil
}

// CreateTagWithRemoteID inserts a tag with its remote ID, or updates the remote ID if it already exists.
func CreateTagWithRemoteID(db *sql.DB, name, remoteID string) (int64, error) {
	var existingID int64
	err := db.QueryRow("SELECT id FROM tags WHERE name = ?", name).Scan(&existingID)
	if err == nil {
		_, err := db.Exec("UPDATE tags SET remote_id = ? WHERE id = ?", remoteID, existingID)
		return existingID, err
	}
	res, err := db.Exec("INSERT INTO tags(name, remote_id) VALUES(?, ?)", name, remoteID)
	if err != nil {
		return 0, fmt.Errorf("insert tag: %w", err)
	}
	id, _ := res.LastInsertId()
	return id, nil
}

// GetTag retrieves a tag by name.
func GetTag(db *sql.DB, name string) (int64, error) {
	var id int64
	err := db.QueryRow("SELECT id FROM tags WHERE name = ?", name).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("tag %q not found", name)
	}
	return id, err
}

// ListTags returns all tags.
func ListTags(db *sql.DB) ([]string, error) {
	rows, err := db.Query("SELECT name FROM tags ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tags = append(tags, name)
	}
	return tags, nil
}

// AddIssueTag associates a tag with an issue.
func AddIssueTag(db *sql.DB, issueID int64, tagName string) error {
	tagID, err := CreateTag(db, tagName)
	if err != nil {
		return err
	}
	_, err = db.Exec("INSERT OR IGNORE INTO issue_tags(issue_id, tag_id) VALUES(?, ?)", issueID, tagID)
	return err
}

// RemoveIssueTag removes a tag from an issue.
func RemoveIssueTag(db *sql.DB, issueID int64, tagName string) error {
	tagID, err := GetTag(db, tagName)
	if err != nil {
		return err
	}
	_, err = db.Exec("DELETE FROM issue_tags WHERE issue_id = ? AND tag_id = ?", issueID, tagID)
	return err
}

// GetIssueTags returns all tag names for an issue.
func GetIssueTags(db *sql.DB, issueID int64) ([]string, error) {
	rows, err := db.Query(`
		SELECT t.name FROM tags t
		JOIN issue_tags it ON t.id = it.tag_id
		WHERE it.issue_id = ?
		ORDER BY t.name`, issueID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tags = append(tags, name)
	}
	return tags, nil
}

// SyncIssueTags replaces all tags for an issue with the given tags.
// It also stores the remote_id mapping for each tag.
func SyncIssueTags(db *sql.DB, issueID int64, tags []struct{ Name, RemoteID string }) error {
	_, err := db.Exec("DELETE FROM issue_tags WHERE issue_id = ?", issueID)
	if err != nil {
		return fmt.Errorf("clear issue tags: %w", err)
	}
	for _, tag := range tags {
		tagID, err := CreateTagWithRemoteID(db, tag.Name, tag.RemoteID)
		if err != nil {
			return fmt.Errorf("create tag %q: %w", tag.Name, err)
		}
		_, err = db.Exec("INSERT OR IGNORE INTO issue_tags(issue_id, tag_id) VALUES(?, ?)", issueID, tagID)
		if err != nil {
			return fmt.Errorf("link tag %q: %w", tag.Name, err)
		}
	}
	return nil
}
