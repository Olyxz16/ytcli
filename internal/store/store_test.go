package store

import (
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// openTestDB creates an in-memory SQLite DB with migrations applied.
func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	db.SetMaxOpenConns(1)
	if err := migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestCreateAndGetIssue(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	issue, err := CreateIssue(db, "Test summary", "Test description", "Open", "Normal", "")
	if err != nil {
		t.Fatalf("create issue: %v", err)
	}
	if issue.ID == 0 {
		t.Fatal("expected non-zero ID")
	}
	if issue.Summary != "Test summary" {
		t.Errorf("summary = %q, want Test summary", issue.Summary)
	}
	if issue.SyncStatus != "local" {
		t.Errorf("sync_status = %q, want local", issue.SyncStatus)
	}

	got, err := GetIssue(db, issue.ID)
	if err != nil {
		t.Fatalf("get issue: %v", err)
	}
	if got == nil {
		t.Fatal("expected issue, got nil")
	}
	if got.Summary != issue.Summary {
		t.Errorf("got summary %q, want %q", got.Summary, issue.Summary)
	}
}

func TestGetIssueNotFound(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	issue, err := GetIssue(db, 9999)
	if err != nil {
		t.Fatalf("get issue: %v", err)
	}
	if issue != nil {
		t.Fatalf("expected nil, got %+v", issue)
	}
}

func TestGetIssueByRemoteID(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	issue, _ := CreateIssue(db, "Remote", "Desc", "Open", "Normal", "")
	if err := SetRemoteID(db, issue.ID, "PROJ-1", "1-1"); err != nil {
		t.Fatalf("set remote id: %v", err)
	}

	got, err := GetIssueByRemoteID(db, "PROJ-1")
	if err != nil {
		t.Fatalf("get by remote id: %v", err)
	}
	if got == nil || got.ID != issue.ID {
		t.Fatal("issue not found by remote ID")
	}
}

func TestUpdateIssue(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	issue, _ := CreateIssue(db, "Before", "Desc", "Open", "Normal", "")
	updated, err := UpdateIssue(db, issue.ID, map[string]interface{}{
		"summary": "After",
		"state":   "In Progress",
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Summary != "After" {
		t.Errorf("summary = %q, want After", updated.Summary)
	}
	if updated.State != "In Progress" {
		t.Errorf("state = %q, want In Progress", updated.State)
	}
	// sync_status only transitions to 'modified' from 'synced'; local stays local
	if updated.SyncStatus != "local" {
		t.Errorf("sync_status = %q, want local", updated.SyncStatus)
	}
}

func TestUpdateIssueNoAllowedFields(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	issue, _ := CreateIssue(db, "Before", "Desc", "Open", "Normal", "")
	updated, err := UpdateIssue(db, issue.ID, map[string]interface{}{
		"unknown": "value",
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Summary != "Before" {
		t.Errorf("summary changed unexpectedly to %q", updated.Summary)
	}
}

func TestListIssues(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	CreateIssue(db, "Alpha", "Desc", "Open", "Normal", "")
	CreateIssue(db, "Beta", "Desc", "Open", "Normal", "")
	CreateIssue(db, "Gamma", "Desc", "Open", "Normal", "")

	issues, err := ListIssues(db, "", 10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(issues) != 3 {
		t.Errorf("len(issues) = %d, want 3", len(issues))
	}
}

func TestListIssuesByQuery(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	CreateIssue(db, "Search me", "Desc", "Open", "Normal", "")
	CreateIssue(db, "Ignore", "Desc", "Open", "Normal", "")

	issues, err := ListIssues(db, "Search", 10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(issues) != 1 {
		t.Errorf("len(issues) = %d, want 1", len(issues))
	}
	if issues[0].Summary != "Search me" {
		t.Errorf("summary = %q, want Search me", issues[0].Summary)
	}
}

func TestDeleteIssue(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	issue, _ := CreateIssue(db, "To delete", "Desc", "Open", "Normal", "")
	if err := DeleteIssue(db, issue.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	got, _ := GetIssue(db, issue.ID)
	if got != nil {
		t.Fatal("expected issue to be deleted")
	}
}

func TestCountIssues(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	count, err := CountIssues(db)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Errorf("count = %d, want 0", count)
	}

	CreateIssue(db, "One", "Desc", "Open", "Normal", "")
	count, _ = CountIssues(db)
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}
}

func TestSetRemoteIDAndSyncStatus(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	issue, _ := CreateIssue(db, "Remote", "Desc", "Open", "Normal", "")
	if err := SetRemoteID(db, issue.ID, "PROJ-99", "99-1"); err != nil {
		t.Fatalf("set remote id: %v", err)
	}
	if err := SetSyncStatus(db, issue.ID, "synced"); err != nil {
		t.Fatalf("set sync status: %v", err)
	}

	got, _ := GetIssue(db, issue.ID)
	if got.RemoteID == nil || *got.RemoteID != "PROJ-99" {
		t.Errorf("remote_id = %v, want PROJ-99", got.RemoteID)
	}
	if got.SyncStatus != "synced" {
		t.Errorf("sync_status = %q, want synced", got.SyncStatus)
	}
}

func TestLoadIssueTagsAndComments(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	issue, _ := CreateIssue(db, "Tagged", "Desc", "Open", "Normal", "")
	_ = AddIssueTag(db, issue.ID, "bug")
	_ = AddIssueTag(db, issue.ID, "urgent")
	_, _ = CreateComment(db, issue.ID, "First comment")
	_, _ = CreateComment(db, issue.ID, "Second comment")

	issues := []LocalIssue{*issue}
	if err := LoadIssueTags(db, issues); err != nil {
		t.Fatalf("load tags: %v", err)
	}
	if err := LoadIssueComments(db, issues); err != nil {
		t.Fatalf("load comments: %v", err)
	}

	if len(issues[0].Tags) != 2 {
		t.Errorf("tags = %v, want 2", issues[0].Tags)
	}
	if issues[0].Comments != 2 {
		t.Errorf("comments = %d, want 2", issues[0].Comments)
	}
}

func TestCreateAndGetComment(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	issue, _ := CreateIssue(db, "Issue", "Desc", "Open", "Normal", "")
	c, err := CreateComment(db, issue.ID, "Hello")
	if err != nil {
		t.Fatalf("create comment: %v", err)
	}
	if c.Text != "Hello" {
		t.Errorf("text = %q, want Hello", c.Text)
	}

	got, err := GetComment(db, c.ID)
	if err != nil {
		t.Fatalf("get comment: %v", err)
	}
	if got == nil || got.Text != "Hello" {
		t.Fatal("comment not found")
	}
}

func TestListComments(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	issue, _ := CreateIssue(db, "Issue", "Desc", "Open", "Normal", "")
	CreateComment(db, issue.ID, "A")
	CreateComment(db, issue.ID, "B")

	comments, err := ListComments(db, issue.ID)
	if err != nil {
		t.Fatalf("list comments: %v", err)
	}
	if len(comments) != 2 {
		t.Errorf("len = %d, want 2", len(comments))
	}
}

func TestCountComments(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	issue, _ := CreateIssue(db, "Issue", "Desc", "Open", "Normal", "")
	count, _ := CountComments(db, issue.ID)
	if count != 0 {
		t.Errorf("count = %d, want 0", count)
	}
	CreateComment(db, issue.ID, "C")
	count, _ = CountComments(db, issue.ID)
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}
}

func TestCreateAndGetTag(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	id, err := CreateTag(db, "feature")
	if err != nil {
		t.Fatalf("create tag: %v", err)
	}
	if id == 0 {
		t.Fatal("expected non-zero tag ID")
	}

	// Duplicate should return existing ID
	id2, err := CreateTag(db, "feature")
	if err != nil {
		t.Fatalf("create duplicate tag: %v", err)
	}
	if id2 != id {
		t.Errorf("duplicate tag id = %d, want %d", id2, id)
	}
}

func TestGetTagNotFound(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	_, err := GetTag(db, "missing")
	if err == nil {
		t.Fatal("expected error for missing tag")
	}
}

func TestListTags(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	CreateTag(db, "z-last")
	CreateTag(db, "a-first")

	tags, err := ListTags(db)
	if err != nil {
		t.Fatalf("list tags: %v", err)
	}
	if len(tags) != 2 {
		t.Errorf("len = %d, want 2", len(tags))
	}
	if tags[0] != "a-first" {
		t.Errorf("first tag = %q, want a-first", tags[0])
	}
}

func TestAddRemoveIssueTag(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	issue, _ := CreateIssue(db, "Issue", "Desc", "Open", "Normal", "")
	if err := AddIssueTag(db, issue.ID, "bug"); err != nil {
		t.Fatalf("add tag: %v", err)
	}
	tags, _ := GetIssueTags(db, issue.ID)
	if len(tags) != 1 || tags[0] != "bug" {
		t.Errorf("tags = %v, want [bug]", tags)
	}

	if err := RemoveIssueTag(db, issue.ID, "bug"); err != nil {
		t.Fatalf("remove tag: %v", err)
	}
	tags, _ = GetIssueTags(db, issue.ID)
	if len(tags) != 0 {
		t.Errorf("tags = %v, want []", tags)
	}
}

func TestRemoveIssueTagNotFound(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	issue, _ := CreateIssue(db, "Issue", "Desc", "Open", "Normal", "")
	err := RemoveIssueTag(db, issue.ID, "missing")
	if err == nil {
		t.Fatal("expected error removing non-existent tag")
	}
}

func TestEnqueueAndDequeue(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	issue, _ := CreateIssue(db, "Issue", "Desc", "Open", "Normal", "")
	if err := Enqueue(db, "update", "issue", issue.ID, map[string]interface{}{"summary": "New"}); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	items, err := DequeuePending(db)
	if err != nil {
		t.Fatalf("dequeue: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len = %d, want 1", len(items))
	}
	if items[0].Operation != "update" {
		t.Errorf("operation = %q, want update", items[0].Operation)
	}
}

func TestQueueMarking(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	issue, _ := CreateIssue(db, "Issue", "Desc", "Open", "Normal", "")
	Enqueue(db, "update", "issue", issue.ID, map[string]interface{}{})

	items, _ := DequeuePending(db)
	id := items[0].ID

	if err := MarkInProgress(db, id); err != nil {
		t.Fatalf("mark in_progress: %v", err)
	}
	// In-progress items are not returned by DequeuePending
	items, _ = DequeuePending(db)
	if len(items) != 0 {
		t.Errorf("pending after in_progress = %d, want 0", len(items))
	}

	if err := MarkFailed(db, id, "network error"); err != nil {
		t.Fatalf("mark failed: %v", err)
	}
	items, _ = DequeuePending(db)
	if len(items) != 1 {
		t.Errorf("pending after failed = %d, want 1", len(items))
	}
	if items[0].Attempts != 1 {
		t.Errorf("attempts = %d, want 1", items[0].Attempts)
	}
	if items[0].LastError == nil || *items[0].LastError != "network error" {
		t.Errorf("last_error = %v, want network error", items[0].LastError)
	}

	if err := MarkCompleted(db, id); err != nil {
		t.Fatalf("mark completed: %v", err)
	}
	items, _ = DequeuePending(db)
	if len(items) != 0 {
		t.Errorf("pending after completed = %d, want 0", len(items))
	}
}

func TestQueueCount(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	issue, _ := CreateIssue(db, "Issue", "Desc", "Open", "Normal", "")
	Enqueue(db, "a", "issue", issue.ID, map[string]interface{}{})
	Enqueue(db, "b", "issue", issue.ID, map[string]interface{}{})

	items, _ := DequeuePending(db)
	MarkCompleted(db, items[0].ID)
	MarkFailed(db, items[1].ID, "err")

	pending, failed, completed, err := QueueCount(db)
	if err != nil {
		t.Fatalf("queue count: %v", err)
	}
	// MarkFailed resets status to 'pending', completed stays completed
	if pending != 1 {
		t.Errorf("pending = %d, want 1", pending)
	}
	if failed != 0 {
		t.Errorf("failed = %d, want 0", failed)
	}
	if completed != 1 {
		t.Errorf("completed = %d, want 1", completed)
	}
}

func TestPruneCompleted(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	issue, _ := CreateIssue(db, "Issue", "Desc", "Open", "Normal", "")
	Enqueue(db, "a", "issue", issue.ID, map[string]interface{}{})
	items, _ := DequeuePending(db)
	MarkCompleted(db, items[0].ID)

	// Should not prune fresh items
	if err := PruneCompleted(db, 1*time.Hour); err != nil {
		t.Fatalf("prune: %v", err)
	}
	_, _, completed, _ := QueueCount(db)
	if completed != 1 {
		t.Errorf("completed = %d, want 1", completed)
	}

	// Manually age the item
	_, _ = db.Exec("UPDATE sync_queue SET created_at = datetime('now', '-2 days') WHERE id = ?", items[0].ID)
	if err := PruneCompleted(db, 1*time.Hour); err != nil {
		t.Fatalf("prune aged: %v", err)
	}
	_, _, completed, _ = QueueCount(db)
	if completed != 0 {
		t.Errorf("completed after prune = %d, want 0", completed)
	}
}

func TestMigrationIdempotent(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	// Running migrate again should not error
	if err := migrate(db); err != nil {
		t.Fatalf("second migrate: %v", err)
	}
}

func TestLocalIssueToJSON(t *testing.T) {
	issue := LocalIssue{
		ID:       1,
		Summary:  "Test",
		State:    "Open",
		Priority: "Normal",
	}
	jsonStr, err := issue.ToJSON()
	if err != nil {
		t.Fatalf("to json: %v", err)
	}
	if jsonStr == "" {
		t.Fatal("expected non-empty JSON")
	}
}
