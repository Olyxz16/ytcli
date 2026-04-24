package store

import (
	"database/sql"
	"fmt"
)

const currentVersion = 1

func migrate(db *sql.DB) error {
	var version int
	_ = db.QueryRow("SELECT value FROM schema_meta WHERE key = 'version'").Scan(&version)

	if version < 1 {
		if err := v1(db); err != nil {
			return fmt.Errorf("v1 migration: %w", err)
		}
		version = 1
	}

	_, _ = db.Exec("INSERT OR REPLACE INTO schema_meta(key, value) VALUES('version', ?)", version)
	return nil
}

func v1(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS issues (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			remote_id TEXT,
			remote_db_id TEXT,
			summary TEXT NOT NULL,
			description TEXT,
			state TEXT NOT NULL DEFAULT 'Open',
			priority TEXT NOT NULL DEFAULT 'Normal',
			assignee TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			synced_at DATETIME,
			sync_status TEXT NOT NULL DEFAULT 'local',
			remote_etag TEXT,
			UNIQUE(remote_id)
		)`,
		`CREATE TABLE IF NOT EXISTS comments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			issue_id INTEGER NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
			remote_id TEXT,
			text TEXT NOT NULL,
			author TEXT NOT NULL DEFAULT 'local',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			sync_status TEXT NOT NULL DEFAULT 'local'
		)`,
		`CREATE TABLE IF NOT EXISTS tags (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			remote_id TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS issue_tags (
			issue_id INTEGER REFERENCES issues(id) ON DELETE CASCADE,
			tag_id INTEGER REFERENCES tags(id),
			PRIMARY KEY(issue_id, tag_id)
		)`,
		`CREATE TABLE IF NOT EXISTS sync_queue (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			operation TEXT NOT NULL,
			entity_type TEXT NOT NULL,
			local_id INTEGER NOT NULL,
			payload TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			attempts INTEGER DEFAULT 0,
			last_error TEXT,
			status TEXT NOT NULL DEFAULT 'pending'
		)`,
		`CREATE TABLE IF NOT EXISTS schema_meta (
			key TEXT PRIMARY KEY,
			value TEXT
		)`,
		`INSERT OR IGNORE INTO schema_meta(key, value) VALUES('version', '1')`,
	}

	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("exec: %s: %w", stmt[:50], err)
		}
	}

	// Create FTS5 virtual table and triggers
	ftsStmts := []string{
		`CREATE VIRTUAL TABLE IF NOT EXISTS issues_fts USING fts5(
			summary, description, state, priority, assignee,
			content=issues, content_rowid=id
		)`,
		`CREATE TRIGGER IF NOT EXISTS issues_ai AFTER INSERT ON issues BEGIN
			INSERT INTO issues_fts(rowid, summary, description, state, priority, assignee)
			VALUES (new.id, new.summary, new.description, new.state, new.priority, new.assignee);
		END`,
		`CREATE TRIGGER IF NOT EXISTS issues_au AFTER UPDATE ON issues BEGIN
			INSERT INTO issues_fts(issues_fts, rowid, summary, description, state, priority, assignee)
			VALUES ('delete', old.id, old.summary, old.description, old.state, old.priority, old.assignee);
			INSERT INTO issues_fts(rowid, summary, description, state, priority, assignee)
			VALUES (new.id, new.summary, new.description, new.state, new.priority, new.assignee);
		END`,
		`CREATE TRIGGER IF NOT EXISTS issues_ad AFTER DELETE ON issues BEGIN
			INSERT INTO issues_fts(issues_fts, rowid, summary, description, state, priority, assignee)
			VALUES ('delete', old.id, old.summary, old.description, old.state, old.priority, old.assignee);
		END`,
	}

	for _, stmt := range ftsStmts {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("exec fts: %w", err)
		}
	}

	return nil
}
