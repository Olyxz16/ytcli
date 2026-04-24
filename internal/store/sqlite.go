package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// DBPath returns the path to the local SQLite database.
func DBPath() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Join(wd, ".ytcli", "store.db"), nil
}

// Open opens the local SQLite database, creating it if necessary.
func Open() (*sql.DB, error) {
	path, err := DBPath()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}
	if err := migrate(db); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return db, nil
}

// IsInitialized checks whether a local DB exists in the current directory.
func IsInitialized() bool {
	path, err := DBPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}
