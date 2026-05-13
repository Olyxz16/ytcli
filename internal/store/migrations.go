package store

import (
	"database/sql"
	"fmt"
)

const currentVersion = 3

func migrate(db *sql.DB) error {
	var version int
	_ = db.QueryRow("SELECT value FROM schema_meta WHERE key = 'version'").Scan(&version)

	if version < 1 {
		if err := v1(db); err != nil {
			return fmt.Errorf("v1 migration: %w", err)
		}
		version = 1
	}

	if version < 2 {
		if err := v2(db); err != nil {
			return fmt.Errorf("v2 migration: %w", err)
		}
		version = 2
	}

	if version < 3 {
		if err := v3(db); err != nil {
			return fmt.Errorf("v3 migration: %w", err)
		}
		version = 3
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

func v2(db *sql.DB) error {
	// Migrate from remote_id/remote_db_id to provider_name/provider_key/provider_ref
	// Step 1: Create new table with updated schema
	stmts := []string{
		`CREATE TABLE issues_new (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			provider_name TEXT,
			provider_key TEXT,
			provider_ref TEXT,
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
			UNIQUE(provider_name, provider_ref)
		)`,
		// Copy data from old table, mapping remote_id → provider_ref, remote_db_id → provider_key
		`INSERT INTO issues_new (id, provider_name, provider_key, provider_ref, summary, description, state, priority, assignee, created_at, updated_at, synced_at, sync_status, remote_etag)
		 SELECT id, '', remote_db_id, remote_id, summary, description, state, priority, assignee, created_at, updated_at, synced_at, sync_status, remote_etag
		 FROM issues`,
		// Drop old FTS triggers and table
		`DROP TRIGGER IF EXISTS issues_ad`,
		`DROP TRIGGER IF EXISTS issues_au`,
		`DROP TRIGGER IF EXISTS issues_ai`,
		`DROP TABLE IF EXISTS issues_fts`,
		// Replace old table with new
		`DROP TABLE issues`,
		`ALTER TABLE issues_new RENAME TO issues`,
		// Recreate FTS
		`CREATE VIRTUAL TABLE issues_fts USING fts5(
			summary, description, state, priority, assignee,
			content=issues, content_rowid=id
		)`,
		`INSERT INTO issues_fts(rowid, summary, description, state, priority, assignee)
		 SELECT id, summary, description, state, priority, assignee FROM issues`,
		`CREATE TRIGGER issues_ai AFTER INSERT ON issues BEGIN
			INSERT INTO issues_fts(rowid, summary, description, state, priority, assignee)
			VALUES (new.id, new.summary, new.description, new.state, new.priority, new.assignee);
		END`,
		`CREATE TRIGGER issues_au AFTER UPDATE ON issues BEGIN
			INSERT INTO issues_fts(issues_fts, rowid, summary, description, state, priority, assignee)
			VALUES ('delete', old.id, old.summary, old.description, old.state, old.priority, old.assignee);
			INSERT INTO issues_fts(rowid, summary, description, state, priority, assignee)
			VALUES (new.id, new.summary, new.description, new.state, new.priority, new.assignee);
		END`,
		`CREATE TRIGGER issues_ad AFTER DELETE ON issues BEGIN
			INSERT INTO issues_fts(issues_fts, rowid, summary, description, state, priority, assignee)
			VALUES ('delete', old.id, old.summary, old.description, old.state, old.priority, old.assignee);
		END`,
		// Add provider_name column to sync_queue
		`ALTER TABLE sync_queue ADD COLUMN provider_name TEXT`,
	}

	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("v2 exec: %s: %w", stmt[:80], err)
		}
	}

	return nil
}

func v3(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS conflicts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			issue_id INTEGER NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
			provider_name TEXT,
			provider_ref TEXT,
			local_summary TEXT,
			local_description TEXT,
			local_state TEXT,
			local_priority TEXT,
			local_assignee TEXT,
			remote_summary TEXT,
			remote_description TEXT,
			remote_state TEXT,
			remote_priority TEXT,
			remote_assignee TEXT,
			detected_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			resolved_at DATETIME,
			resolution TEXT,
			UNIQUE(issue_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_conflicts_unresolved ON conflicts(resolution) WHERE resolution IS NULL`,
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("v3 exec: %s: %w", stmt[:50], err)
		}
	}
	return nil
}
