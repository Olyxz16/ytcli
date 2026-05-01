package sync

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Olyxz16/ytcli/internal/api"
	"github.com/Olyxz16/ytcli/internal/config"
	"github.com/Olyxz16/ytcli/internal/model"
	"github.com/Olyxz16/ytcli/internal/service"
	"github.com/Olyxz16/ytcli/internal/store"
)

// Manager orchestrates sync operations.
type Manager struct {
	db     *sql.DB
	client *api.Client
	svc    *service.Service
	cfg    *config.MergedConfig
}

// Result holds the outcome of a sync operation.
type Result struct {
	Pulled    int
	Pushed    int
	Conflicts int
	Failed    int
	Errors    []string
}

// NewManager creates a sync manager.
func NewManager(db *sql.DB, svc *service.Service, cfg *config.MergedConfig) *Manager {
	return &Manager{
		db:  db,
		svc: svc,
		cfg: cfg,
	}
}

// Pull fetches remote issues and updates the local database.
func (m *Manager) Pull(ctx context.Context) (*Result, error) {
	if m.svc == nil {
		return nil, fmt.Errorf("no remote service configured")
	}

	result := &Result{}
	localCfg, _, _ := config.LoadLocal()
	project := m.cfg.Project
	if project == "" && localCfg != nil {
		project = localCfg.Project
	}

	query := ""
	if project != "" {
		query = fmt.Sprintf("project: {%s}", project)
	}

	// Fetch all remote issues
	remoteIssues, err := m.svc.ListIssues(ctx, query, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("fetch remote issues: %w", err)
	}

	for _, ri := range remoteIssues {
		if err := m.syncRemoteIssue(ctx, ri); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("sync %s: %v", ri.IDReadable, err))
			result.Failed++
			continue
		}
		result.Pulled++
	}

	return result, nil
}

// Push sends local pending changes to the remote server.
func (m *Manager) Push(ctx context.Context) (*Result, error) {
	if m.svc == nil {
		return nil, fmt.Errorf("no remote service configured")
	}

	result := &Result{}
	items, err := store.DequeuePending(m.db)
	if err != nil {
		return nil, fmt.Errorf("dequeue: %w", err)
	}

	for _, item := range items {
		if err := store.MarkInProgress(m.db, item.ID); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("mark in-progress %d: %v", item.ID, err))
			continue
		}

		if err := m.processQueueItem(ctx, item); err != nil {
			_ = store.MarkFailed(m.db, item.ID, err.Error())
			result.Errors = append(result.Errors, fmt.Sprintf("push %d (%s): %v", item.LocalID, item.Operation, err))
			result.Failed++
			continue
		}

		_ = store.MarkCompleted(m.db, item.ID)
		result.Pushed++
	}

	// Prune old completed items
	_ = store.PruneCompleted(m.db, 24*time.Hour)

	return result, nil
}

// syncRemoteIssue merges a single remote issue into the local database.
func (m *Manager) syncRemoteIssue(ctx context.Context, ri model.Issue) error {
	existing, err := store.GetIssueByRemoteID(m.db, ri.IDReadable)
	if err != nil {
		return err
	}

	if existing == nil {
		assignee := ""
		if ri.Reporter != nil {
			assignee = ri.Reporter.Login
		}
		state := "Open"
		priority := "Normal"
		for _, cf := range ri.CustomFields {
			if cf.Name == "State" && cf.Value != nil {
				if s, ok := cf.Value.(map[string]interface{}); ok {
					if name, ok := s["name"].(string); ok {
						state = name
					}
				}
			}
			if cf.Name == "Priority" && cf.Value != nil {
				if s, ok := cf.Value.(map[string]interface{}); ok {
					if name, ok := s["name"].(string); ok {
						priority = name
					}
				}
			}
		}
		if ri.Resolved != nil {
			state = "Closed"
		}

		result, err := m.db.Exec(
			`INSERT INTO issues(remote_id, remote_db_id, summary, description, state, priority, assignee, created_at, updated_at, synced_at, sync_status)
			 VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'synced')`,
			ri.IDReadable, ri.ID, ri.Summary, ri.Description, state, priority, assignee,
			time.UnixMilli(ri.Created), time.UnixMilli(ri.Updated), time.Now().UTC(),
		)
		if err != nil {
			return err
		}
		localID, _ := result.LastInsertId()

		if len(ri.Tags) > 0 {
			tags := make([]struct{ Name, RemoteID string }, len(ri.Tags))
			for i, t := range ri.Tags {
				tags[i] = struct{ Name, RemoteID string }{t.Name, t.ID}
			}
			if err := store.SyncIssueTags(m.db, localID, tags); err != nil {
				return fmt.Errorf("sync tags: %w", err)
			}
		}
		return nil
	}

	if existing.SyncStatus == "modified" || existing.SyncStatus == "conflict" {
		return nil
	}

	_, err = m.db.Exec(
		`UPDATE issues SET summary = ?, description = ?, updated_at = ?, synced_at = ?, sync_status = 'synced'
		 WHERE id = ?`,
		ri.Summary, ri.Description, time.Now().UTC(), time.Now().UTC(), existing.ID,
	)
	if err != nil {
		return err
	}

	if len(ri.Tags) > 0 {
		tags := make([]struct{ Name, RemoteID string }, len(ri.Tags))
		for i, t := range ri.Tags {
			tags[i] = struct{ Name, RemoteID string }{t.Name, t.ID}
		}
		if err := store.SyncIssueTags(m.db, existing.ID, tags); err != nil {
			return fmt.Errorf("sync tags: %w", err)
		}
	} else {
		_, err := m.db.Exec("DELETE FROM issue_tags WHERE issue_id = ?", existing.ID)
		if err != nil {
			return fmt.Errorf("clear tags: %w", err)
		}
	}

	return nil
}

// processQueueItem executes a single sync queue item against the remote API.
func (m *Manager) processQueueItem(ctx context.Context, item store.QueueItem) error {
	issue, err := store.GetIssue(m.db, item.LocalID)
	if err != nil {
		return err
	}
	if issue == nil {
		return fmt.Errorf("local issue %d not found", item.LocalID)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(item.Payload), &payload); err != nil {
		return fmt.Errorf("parse payload: %w", err)
	}

	switch item.Operation {
	case "create":
		return m.pushCreate(ctx, issue, payload)
	case "update":
		return m.pushUpdate(ctx, issue, payload)
	case "comment":
		return m.pushComment(ctx, issue, payload)
	case "state":
		return m.pushState(ctx, issue, payload)
	case "tag":
		return m.pushTag(ctx, issue, payload)
	case "untag":
		return m.pushUntag(ctx, issue, payload)
	default:
		return fmt.Errorf("unknown operation: %s", item.Operation)
	}
}

func (m *Manager) pushCreate(ctx context.Context, issue *store.LocalIssue, payload map[string]interface{}) error {
	// Need project ID from config
	projectID := m.cfg.Project
	if projectID == "" {
		return fmt.Errorf("no project configured for create")
	}

	// Resolve project short name to ID via API
	projects, err := m.svc.ListProjects(ctx)
	if err != nil {
		return err
	}
	var ytProjectID string
	for _, p := range projects {
		if p.ShortName == projectID || p.Name == projectID {
			ytProjectID = p.ID
			break
		}
	}
	if ytProjectID == "" {
		return fmt.Errorf("project %q not found on remote", projectID)
	}

	newIssue := model.Issue{
		Summary:     issue.Summary,
		Description: issue.Description,
		Project:     &model.Project{ID: ytProjectID},
	}

	created, err := m.svc.CreateIssue(ctx, newIssue)
	if err != nil {
		return err
	}

	// Update local issue with remote IDs
	return store.SetRemoteID(m.db, issue.ID, created.IDReadable, created.ID)
}

func (m *Manager) pushUpdate(ctx context.Context, issue *store.LocalIssue, payload map[string]interface{}) error {
	if issue.RemoteID == nil {
		return fmt.Errorf("issue has no remote ID")
	}
	_, err := m.svc.UpdateIssue(ctx, *issue.RemoteID, payload)
	if err != nil {
		return err
	}
	return store.SetSyncStatus(m.db, issue.ID, "synced")
}

func (m *Manager) pushComment(ctx context.Context, issue *store.LocalIssue, payload map[string]interface{}) error {
	if issue.RemoteID == nil {
		return fmt.Errorf("issue has no remote ID")
	}
	text, _ := payload["text"].(string)
	_, err := m.svc.AddComment(ctx, *issue.RemoteID, text)
	return err
}

func (m *Manager) pushState(ctx context.Context, issue *store.LocalIssue, payload map[string]interface{}) error {
	if issue.RemoteID == nil {
		return fmt.Errorf("issue has no remote ID")
	}
	state, _ := payload["state"].(string)
	_, err := m.svc.ExecuteCommand(ctx, fmt.Sprintf("State: %s", state), []string{*issue.RemoteID}, false)
	if err != nil {
		return err
	}
	return store.SetSyncStatus(m.db, issue.ID, "synced")
}

func (m *Manager) pushTag(ctx context.Context, issue *store.LocalIssue, payload map[string]interface{}) error {
	if issue.RemoteID == nil {
		return fmt.Errorf("issue has no remote ID")
	}
	tagName, _ := payload["tag"].(string)
	// Tags need to be created on remote first, then added
	// For simplicity, use the commands API
	_, err := m.svc.ExecuteCommand(ctx, fmt.Sprintf("tag %s", tagName), []string{*issue.RemoteID}, false)
	return err
}

func (m *Manager) pushUntag(ctx context.Context, issue *store.LocalIssue, payload map[string]interface{}) error {
	if issue.RemoteID == nil {
		return fmt.Errorf("issue has no remote ID")
	}
	tagName, _ := payload["tag"].(string)
	_, err := m.svc.ExecuteCommand(ctx, fmt.Sprintf("untag %s", tagName), []string{*issue.RemoteID}, false)
	return err
}
