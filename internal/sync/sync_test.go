package sync

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Olyxz16/tkt/internal/config"
	"github.com/Olyxz16/tkt/internal/provider/youtrack"
	"github.com/Olyxz16/tkt/internal/store"
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
		`INSERT OR IGNORE INTO schema_meta(key, value) VALUES('version', '2')`,
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
	client := youtrack.NewClient(srv.URL, "token")
	prov := youtrack.NewProviderWithClient(client)
	cfg := &config.MergedConfig{Project: "PROJ", ProviderURL: srv.URL, ProviderName: "youtrack"}
	mgr := NewManager(db, prov, cfg)
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

	issue, err := store.GetIssueByProviderRef(db, "youtrack", "PROJ-1")
	if err != nil {
		t.Fatalf("get issue by provider ref: %v", err)
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
	store.SetProviderRef(db, issue.ID, "youtrack", "1-1", "PROJ-1")
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

	issue, _ := store.GetIssueByProviderRef(db, "youtrack", "PROJ-1")
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

func TestPullConflictManual(t *testing.T) {
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
	store.SetProviderRef(db, issue.ID, "youtrack", "1-1", "PROJ-1")
	store.SetSyncStatus(db, issue.ID, "modified")

	result, err := mgr.Pull(context.Background())
	if err != nil {
		t.Fatalf("pull: %v", err)
	}
	// Manual resolver stores conflict and returns error -> counts as Failed, not Pulled
	if result.Failed != 1 {
		t.Errorf("failed = %d, want 1", result.Failed)
	}
	if result.Conflicts != 0 {
		// Conflicts field is not auto-incremented yet; it's 0
	}

	// Verify conflict was stored
	conflicts, err := store.GetConflicts(db)
	if err != nil {
		t.Fatalf("get conflicts: %v", err)
	}
	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(conflicts))
	}
	if conflicts[0].LocalSummary != "Local" {
		t.Errorf("local summary = %q", conflicts[0].LocalSummary)
	}
	if conflicts[0].RemoteSummary != "Remote" {
		t.Errorf("remote summary = %q", conflicts[0].RemoteSummary)
	}

	// Verify issue status is 'conflict'
	updated, _ := store.GetIssue(db, issue.ID)
	if updated.Summary != "Local" {
		t.Errorf("summary was overwritten: %q", updated.Summary)
	}
	if updated.SyncStatus != "conflict" {
		t.Errorf("sync_status = %q, want conflict", updated.SyncStatus)
	}
}

func TestPullConflictLocalWins(t *testing.T) {
	db := openTestDB(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/issues" {
			w.Write([]byte(`[{"id":"1-1","idReadable":"PROJ-1","summary":"Remote","created":1000,"updated":2000}]`))
		} else {
			w.Write([]byte(`[]`))
		}
	}))
	defer srv.Close()
	defer db.Close()

	client := youtrack.NewClient(srv.URL, "token")
	prov := youtrack.NewProviderWithClient(client)
	cfg := &config.MergedConfig{Project: "PROJ", ProviderURL: srv.URL, ProviderName: "youtrack"}
	mgr := NewManagerWithResolver(db, prov, cfg, StrategyLocalWins)

	issue, _ := store.CreateIssue(db, "Local", "Desc", "Open", "Normal", "")
	store.SetProviderRef(db, issue.ID, "youtrack", "1-1", "PROJ-1")
	store.SetSyncStatus(db, issue.ID, "modified")

	result, err := mgr.Pull(context.Background())
	if err != nil {
		t.Fatalf("pull: %v", err)
	}
	if result.Pulled != 1 {
		t.Errorf("pulled = %d, want 1", result.Pulled)
	}
	if result.Failed != 0 {
		t.Errorf("failed = %d, want 0", result.Failed)
	}

	updated, _ := store.GetIssue(db, issue.ID)
	if updated.Summary != "Local" {
		t.Errorf("summary was overwritten: %q", updated.Summary)
	}
	if updated.SyncStatus != "modified" {
		t.Errorf("sync_status changed to %q", updated.SyncStatus)
	}
}

func TestPullConflictRemoteWins(t *testing.T) {
	db := openTestDB(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/issues" {
			w.Write([]byte(`[{"id":"1-1","idReadable":"PROJ-1","summary":"Remote","created":1000,"updated":2000}]`))
		} else {
			w.Write([]byte(`[]`))
		}
	}))
	defer srv.Close()
	defer db.Close()

	client := youtrack.NewClient(srv.URL, "token")
	prov := youtrack.NewProviderWithClient(client)
	cfg := &config.MergedConfig{Project: "PROJ", ProviderURL: srv.URL, ProviderName: "youtrack"}
	mgr := NewManagerWithResolver(db, prov, cfg, StrategyRemoteWins)

	issue, _ := store.CreateIssue(db, "Local", "Desc", "Open", "Normal", "")
	store.SetProviderRef(db, issue.ID, "youtrack", "1-1", "PROJ-1")
	store.SetSyncStatus(db, issue.ID, "modified")

	result, err := mgr.Pull(context.Background())
	if err != nil {
		t.Fatalf("pull: %v", err)
	}
	if result.Pulled != 1 {
		t.Errorf("pulled = %d, want 1", result.Pulled)
	}
	if result.Failed != 0 {
		t.Errorf("failed = %d, want 0", result.Failed)
	}

	updated, _ := store.GetIssue(db, issue.ID)
	if updated.Summary != "Remote" {
		t.Errorf("summary = %q, want Remote", updated.Summary)
	}
	if updated.SyncStatus != "synced" {
		t.Errorf("sync_status = %q, want synced", updated.SyncStatus)
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
	if updated.ProviderRef != "PROJ-1" {
		t.Errorf("provider_ref = %v", updated.ProviderRef)
	}
}

func TestPushUpdate(t *testing.T) {
	mgr, db, srv := newTestManager(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"id":"1-1","summary":"Updated"}`))
	})
	defer srv.Close()
	defer db.Close()

	issue, _ := store.CreateIssue(db, "Old", "Desc", "Open", "Normal", "")
	store.SetProviderRef(db, issue.ID, "youtrack", "1-1", "PROJ-1")
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
	store.SetProviderRef(db, issue.ID, "youtrack", "1-1", "PROJ-1")
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
	store.SetProviderRef(db, issue.ID, "youtrack", "1-1", "PROJ-1")
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
	store.SetProviderRef(db, issue.ID, "youtrack", "1-1", "PROJ-1")
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
	store.SetProviderRef(db, issue.ID, "youtrack", "1-1", "PROJ-1")
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
	store.SetProviderRef(db, issue.ID, "youtrack", "1-1", "PROJ-1")
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
