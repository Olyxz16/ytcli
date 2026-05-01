package sync

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Olyxz16/ytcli/internal/api"
	"github.com/Olyxz16/ytcli/internal/config"
	"github.com/Olyxz16/ytcli/internal/service"
	"github.com/Olyxz16/ytcli/internal/store"
	_ "modernc.org/sqlite"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	db.SetMaxOpenConns(1)
	if err := storeMigrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

// storeMigrate is a wrapper to call the unexported migrate from store package.
// Since we cannot access it directly, we use Open which applies migrations.
func storeMigrate(db *sql.DB) error {
	// Manually apply the same migrations as store.migrate
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
			return err
		}
	}
	return nil
}

func newTestManager(t *testing.T, handler http.HandlerFunc) (*Manager, *sql.DB, *httptest.Server) {
	db := openTestDB(t)
	srv := httptest.NewServer(handler)
	client := api.NewClient(srv.URL, "token")
	svc := service.NewServiceWithClient(client)
	cfg := &config.MergedConfig{Project: "PROJ"}
	mgr := NewManager(db, svc, cfg)
	return mgr, db, srv
}

func TestNewManager(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	mgr := NewManager(db, nil, nil)
	if mgr == nil {
		t.Fatal("expected manager")
	}
}

func TestPullNoService(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	mgr := NewManager(db, nil, &config.MergedConfig{})
	_, err := mgr.Pull(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestPullNewRemoteIssue(t *testing.T) {
	mgr, db, srv := newTestManager(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/issues" {
			w.Write([]byte(`[{"id":"1-1","idReadable":"PROJ-1","summary":"Remote","created":1000,"updated":2000}]`))
		} else {
			w.Write([]byte(`[]`))
		}
	})
	defer srv.Close()
	defer db.Close()

	result, err := mgr.Pull(context.Background())
	if err != nil {
		t.Fatalf("pull: %v", err)
	}
	if result.Pulled != 1 {
		t.Errorf("pulled = %d", result.Pulled)
	}

	issue, err := store.GetIssueByRemoteID(db, "PROJ-1")
	if err != nil {
		t.Fatalf("get issue by remote id: %v", err)
	}
	if issue == nil {
		t.Fatal("expected issue to be inserted")
	}
	if issue.SyncStatus != "synced" {
		t.Errorf("sync_status = %q", issue.SyncStatus)
	}
}

func TestPullExistingSynced(t *testing.T) {
	mgr, db, srv := newTestManager(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/issues" {
			w.Write([]byte(`[{"id":"1-1","idReadable":"PROJ-1","summary":"Updated","created":1000,"updated":2000}]`))
		} else {
			w.Write([]byte(`[]`))
		}
	})
	defer srv.Close()
	defer db.Close()

	// Insert existing synced issue
	issue, _ := store.CreateIssue(db, "Old", "Desc", "Open", "Normal", "")
	store.SetRemoteID(db, issue.ID, "PROJ-1", "1-1")
	store.SetSyncStatus(db, issue.ID, "synced")

	result, err := mgr.Pull(context.Background())
	if err != nil {
		t.Fatalf("pull: %v", err)
	}
	if result.Pulled != 1 {
		t.Errorf("pulled = %d", result.Pulled)
	}

	updated, _ := store.GetIssue(db, issue.ID)
	if updated.Summary != "Updated" {
		t.Errorf("summary = %q", updated.Summary)
	}
}

func TestPullSyncsTags(t *testing.T) {
	mgr, db, srv := newTestManager(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/issues" {
			w.Write([]byte(`[{"id":"1-1","idReadable":"PROJ-1","summary":"Tagged","created":1000,"updated":2000,"tags":[{"id":"6-1","name":"bug"},{"id":"6-2","name":"feature"}]}]`))
		} else {
			w.Write([]byte(`[]`))
		}
	})
	defer srv.Close()
	defer db.Close()

	result, err := mgr.Pull(context.Background())
	if err != nil {
		t.Fatalf("pull: %v", err)
	}
	if result.Pulled != 1 {
		t.Errorf("pulled = %d", result.Pulled)
	}

	issue, _ := store.GetIssueByRemoteID(db, "PROJ-1")
	if issue == nil {
		t.Fatal("expected issue to be inserted")
	}
	tags, err := store.GetIssueTags(db, issue.ID)
	if err != nil {
		t.Fatalf("get issue tags: %v", err)
	}
	if len(tags) != 2 {
		t.Errorf("expected 2 tags, got %d: %v", len(tags), tags)
	}

	tagNames := map[string]bool{}
	for _, t := range tags {
		tagNames[t] = true
	}
	if !tagNames["bug"] || !tagNames["feature"] {
		t.Errorf("expected tags bug and feature, got %v", tags)
	}
}

func TestPullSkipModified(t *testing.T) {
	mgr, db, srv := newTestManager(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/issues" {
			w.Write([]byte(`[{"id":"1-1","idReadable":"PROJ-1","summary":"Remote","created":1000,"updated":2000}]`))
		} else {
			w.Write([]byte(`[]`))
		}
	})
	defer srv.Close()
	defer db.Close()

	issue, _ := store.CreateIssue(db, "Local", "Desc", "Open", "Normal", "")
	store.SetRemoteID(db, issue.ID, "PROJ-1", "1-1")
	store.SetSyncStatus(db, issue.ID, "modified")

	result, err := mgr.Pull(context.Background())
	if err != nil {
		t.Fatalf("pull: %v", err)
	}
	if result.Pulled != 1 {
		t.Errorf("pulled = %d", result.Pulled)
	}

	updated, _ := store.GetIssue(db, issue.ID)
	if updated.Summary != "Local" {
		t.Errorf("summary was overwritten: %q", updated.Summary)
	}
}

func TestPullSkipConflict(t *testing.T) {
	mgr, db, srv := newTestManager(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/issues" {
			w.Write([]byte(`[{"id":"1-1","idReadable":"PROJ-1","summary":"Remote","created":1000,"updated":2000}]`))
		} else {
			w.Write([]byte(`[]`))
		}
	})
	defer srv.Close()
	defer db.Close()

	issue, _ := store.CreateIssue(db, "Local", "Desc", "Open", "Normal", "")
	store.SetRemoteID(db, issue.ID, "PROJ-1", "1-1")
	store.SetSyncStatus(db, issue.ID, "conflict")

	result, err := mgr.Pull(context.Background())
	if err != nil {
		t.Fatalf("pull: %v", err)
	}
	if result.Pulled != 1 {
		t.Errorf("pulled = %d", result.Pulled)
	}

	updated, _ := store.GetIssue(db, issue.ID)
	if updated.Summary != "Local" {
		t.Errorf("summary was overwritten: %q", updated.Summary)
	}
}

func TestPullWithProjectFilter(t *testing.T) {
	mgr, db, srv := newTestManager(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/issues" {
			q := r.URL.Query().Get("query")
			if q != "project: {PROJ}" {
				t.Errorf("query = %q", q)
			}
			w.Write([]byte(`[]`))
		} else {
			w.Write([]byte(`[]`))
		}
	})
	defer srv.Close()
	defer db.Close()

	_, err := mgr.Pull(context.Background())
	if err != nil {
		t.Fatalf("pull: %v", err)
	}
}

func TestPushNoService(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	mgr := NewManager(db, nil, &config.MergedConfig{})
	_, err := mgr.Push(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestPushCreate(t *testing.T) {
	mgr, db, srv := newTestManager(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/admin/projects":
			w.Write([]byte(`[{"id":"p1","shortName":"PROJ","name":"Project"}]`))
		case "/api/issues":
			w.Write([]byte(`{"id":"1-1","idReadable":"PROJ-1","summary":"New"}`))
		default:
			w.Write([]byte(`[]`))
		}
	})
	defer srv.Close()
	defer db.Close()

	issue, _ := store.CreateIssue(db, "New", "Desc", "Open", "Normal", "")
	store.Enqueue(db, "create", "issue", issue.ID, map[string]interface{}{})

	result, err := mgr.Push(context.Background())
	if err != nil {
		t.Fatalf("push: %v", err)
	}
	if result.Pushed != 1 {
		t.Errorf("pushed = %d", result.Pushed)
	}

	updated, _ := store.GetIssue(db, issue.ID)
	if updated.RemoteID == nil || *updated.RemoteID != "PROJ-1" {
		t.Errorf("remote_id = %v", updated.RemoteID)
	}
}

func TestPushUpdate(t *testing.T) {
	mgr, db, srv := newTestManager(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"id":"1-1","summary":"Updated"}`))
	})
	defer srv.Close()
	defer db.Close()

	issue, _ := store.CreateIssue(db, "Old", "Desc", "Open", "Normal", "")
	store.SetRemoteID(db, issue.ID, "PROJ-1", "1-1")
	store.Enqueue(db, "update", "issue", issue.ID, map[string]interface{}{"summary": "Updated"})

	result, err := mgr.Push(context.Background())
	if err != nil {
		t.Fatalf("push: %v", err)
	}
	if result.Pushed != 1 {
		t.Errorf("pushed = %d", result.Pushed)
	}

	updated, _ := store.GetIssue(db, issue.ID)
	if updated.SyncStatus != "synced" {
		t.Errorf("sync_status = %q", updated.SyncStatus)
	}
}

func TestPushComment(t *testing.T) {
	mgr, db, srv := newTestManager(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"id":"c1","text":"Hello"}`))
	})
	defer srv.Close()
	defer db.Close()

	issue, _ := store.CreateIssue(db, "Issue", "Desc", "Open", "Normal", "")
	store.SetRemoteID(db, issue.ID, "PROJ-1", "1-1")
	store.Enqueue(db, "comment", "issue", issue.ID, map[string]interface{}{"text": "Hello"})

	result, err := mgr.Push(context.Background())
	if err != nil {
		t.Fatalf("push: %v", err)
	}
	if result.Pushed != 1 {
		t.Errorf("pushed = %d", result.Pushed)
	}
}

func TestPushState(t *testing.T) {
	mgr, db, srv := newTestManager(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/commands" {
			w.Write([]byte(`{"query":"State: Done","issues":[{"idReadable":"PROJ-1"}]}`))
		}
	})
	defer srv.Close()
	defer db.Close()

	issue, _ := store.CreateIssue(db, "Issue", "Desc", "Open", "Normal", "")
	store.SetRemoteID(db, issue.ID, "PROJ-1", "1-1")
	store.Enqueue(db, "state", "issue", issue.ID, map[string]interface{}{"state": "Done"})

	result, err := mgr.Push(context.Background())
	if err != nil {
		t.Fatalf("push: %v", err)
	}
	if result.Pushed != 1 {
		t.Errorf("pushed = %d", result.Pushed)
	}

	updated, _ := store.GetIssue(db, issue.ID)
	if updated.SyncStatus != "synced" {
		t.Errorf("sync_status = %q", updated.SyncStatus)
	}
}

func TestPushTag(t *testing.T) {
	mgr, db, srv := newTestManager(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/commands" {
			w.Write([]byte(`{"query":"tag bug"}`))
		}
	})
	defer srv.Close()
	defer db.Close()

	issue, _ := store.CreateIssue(db, "Issue", "Desc", "Open", "Normal", "")
	store.SetRemoteID(db, issue.ID, "PROJ-1", "1-1")
	store.Enqueue(db, "tag", "issue", issue.ID, map[string]interface{}{"tag": "bug"})

	result, err := mgr.Push(context.Background())
	if err != nil {
		t.Fatalf("push: %v", err)
	}
	if result.Pushed != 1 {
		t.Errorf("pushed = %d", result.Pushed)
	}
}

func TestPushUntag(t *testing.T) {
	mgr, db, srv := newTestManager(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/commands" {
			w.Write([]byte(`{"query":"untag bug"}`))
		}
	})
	defer srv.Close()
	defer db.Close()

	issue, _ := store.CreateIssue(db, "Issue", "Desc", "Open", "Normal", "")
	store.SetRemoteID(db, issue.ID, "PROJ-1", "1-1")
	store.Enqueue(db, "untag", "issue", issue.ID, map[string]interface{}{"tag": "bug"})

	result, err := mgr.Push(context.Background())
	if err != nil {
		t.Fatalf("push: %v", err)
	}
	if result.Pushed != 1 {
		t.Errorf("pushed = %d", result.Pushed)
	}
}

func TestPushUnknownOperation(t *testing.T) {
	mgr, db, srv := newTestManager(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{}`))
	})
	defer srv.Close()
	defer db.Close()

	issue, _ := store.CreateIssue(db, "Issue", "Desc", "Open", "Normal", "")
	store.SetRemoteID(db, issue.ID, "PROJ-1", "1-1")
	store.Enqueue(db, "unknown", "issue", issue.ID, map[string]interface{}{})

	result, err := mgr.Push(context.Background())
	if err != nil {
		t.Fatalf("push: %v", err)
	}
	if result.Failed != 1 {
		t.Errorf("failed = %d", result.Failed)
	}
}

func TestPushCreateNoProject(t *testing.T) {
	mgr, db, srv := newTestManager(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[]`))
	})
	defer srv.Close()
	defer db.Close()

	// Override config with empty project
	mgr.cfg = &config.MergedConfig{}

	issue, _ := store.CreateIssue(db, "New", "Desc", "Open", "Normal", "")
	store.Enqueue(db, "create", "issue", issue.ID, map[string]interface{}{})

	result, err := mgr.Push(context.Background())
	if err != nil {
		t.Fatalf("push: %v", err)
	}
	if result.Failed != 1 {
		t.Errorf("failed = %d", result.Failed)
	}
}

func TestPushCreateProjectNotFound(t *testing.T) {
	mgr, db, srv := newTestManager(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/admin/projects" {
			w.Write([]byte(`[]`))
		}
	})
	defer srv.Close()
	defer db.Close()

	issue, _ := store.CreateIssue(db, "New", "Desc", "Open", "Normal", "")
	store.Enqueue(db, "create", "issue", issue.ID, map[string]interface{}{})

	result, err := mgr.Push(context.Background())
	if err != nil {
		t.Fatalf("push: %v", err)
	}
	if result.Failed != 1 {
		t.Errorf("failed = %d", result.Failed)
	}
}

func TestPushMissingLocalIssue(t *testing.T) {
	mgr, db, srv := newTestManager(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{}`))
	})
	defer srv.Close()
	defer db.Close()

	store.Enqueue(db, "update", "issue", 9999, map[string]interface{}{})

	result, err := mgr.Push(context.Background())
	if err != nil {
		t.Fatalf("push: %v", err)
	}
	if result.Failed != 1 {
		t.Errorf("failed = %d", result.Failed)
	}
}

func TestPushNoRemoteID(t *testing.T) {
	mgr, db, srv := newTestManager(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{}`))
	})
	defer srv.Close()
	defer db.Close()

	issue, _ := store.CreateIssue(db, "Issue", "Desc", "Open", "Normal", "")
	store.Enqueue(db, "update", "issue", issue.ID, map[string]interface{}{})

	result, err := mgr.Push(context.Background())
	if err != nil {
		t.Fatalf("push: %v", err)
	}
	if result.Failed != 1 {
		t.Errorf("failed = %d", result.Failed)
	}
}

func TestPushPullRemoteError(t *testing.T) {
	mgr, db, srv := newTestManager(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`error`))
	})
	defer srv.Close()
	defer db.Close()

	_, err := mgr.Pull(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}
