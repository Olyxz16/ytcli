package sync

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Olyxz16/tkt/internal/config"
	"github.com/Olyxz16/tkt/internal/model"
	"github.com/Olyxz16/tkt/internal/provider"
	"github.com/Olyxz16/tkt/internal/store"
)

// Manager orchestrates sync operations using a provider-agnostic RemoteProvider.
type Manager struct {
	db   *sql.DB
	prov provider.RemoteProvider
	cfg  *config.MergedConfig
	resolver ConflictResolver
}

// Result holds the outcome of a sync operation.
type Result struct {
	Pulled    int
	Pushed    int
	Conflicts int
	Failed    int
	Errors    []string
}

// retryPolicy defines how many times to retry and the backoff intervals.
const (
	maxRetries = 3
	baseDelay  = 1 * time.Second
)

// retryWithBackoff executes fn and retries on NetworkError with Retry=true.
// It does NOT retry AuthError. Returns the last error after maxRetries.
func retryWithBackoff(ctx context.Context, fn func() error) error {
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			delay := baseDelay * time.Duration(1<<(attempt-1))
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
		}
		err := fn()
		if err == nil {
			return nil
		}
		lastErr = err

		var netErr *provider.NetworkError
		if errors.As(err, &netErr) && netErr.Retry {
			continue
		}
		// Non-retryable error — stop immediately.
		return err
	}
	return fmt.Errorf("max retries exceeded: %w", lastErr)
}

// NewManager creates a sync manager with the default manual conflict resolver.
func NewManager(db *sql.DB, prov provider.RemoteProvider, cfg *config.MergedConfig) *Manager {
	return NewManagerWithResolver(db, prov, cfg, StrategyManual)
}

// NewManagerWithResolver creates a sync manager with the given conflict strategy.
func NewManagerWithResolver(db *sql.DB, prov provider.RemoteProvider, cfg *config.MergedConfig, strategy ConflictStrategy) *Manager {
	return &Manager{
		db:       db,
		prov:     prov,
		cfg:      cfg,
		resolver: NewConflictResolver(strategy),
	}
}

// Pull fetches remote issues and updates the local database.
func (m *Manager) Pull(ctx context.Context) (*Result, error) {
	if m.prov == nil {
		return nil, fmt.Errorf("no remote provider configured")
	}

	result := &Result{}
	localCfg, _, _ := config.LoadLocal()
	project := m.cfg.Project
	if project == "" && localCfg != nil {
		project = localCfg.Project
	}

	providerName := ""
	if localCfg != nil && localCfg.Provider.Name != "" {
		providerName = localCfg.Provider.Name
	}
	if providerName == "" && m.cfg != nil && m.cfg.ProviderName != "" {
		providerName = m.cfg.ProviderName
	}

	q := provider.Query{Project: project}

	var remoteIssues []model.Issue
	if err := retryWithBackoff(ctx, func() error {
		var err error
		remoteIssues, err = m.prov.FetchIssues(ctx, q, 0, 0)
		return err
	}); err != nil {
		return nil, fmt.Errorf("fetch remote issues: %w", err)
	}

	for _, ri := range remoteIssues {
		if err := m.syncRemoteIssue(ctx, ri, providerName); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("sync %s: %v", ri.ID, err))
			result.Failed++
			continue
		}
		result.Pulled++
	}

	return result, nil
}

// Push sends local pending changes to the remote server.
func (m *Manager) Push(ctx context.Context) (*Result, error) {
	if m.prov == nil {
		return nil, fmt.Errorf("no remote provider configured")
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

		if err := retryWithBackoff(ctx, func() error {
			return m.processQueueItem(ctx, item)
		}); err != nil {
			_ = store.MarkFailed(m.db, item.ID, err.Error())
			result.Errors = append(result.Errors, fmt.Sprintf("push %d (%s): %v", item.LocalID, item.Operation, err))
			result.Failed++
			continue
		}

		_ = store.MarkCompleted(m.db, item.ID)
		result.Pushed++
	}

	_ = store.PruneCompleted(m.db, 24*time.Hour)

	return result, nil
}

// syncRemoteIssue merges a single remote issue into the local database.
func (m *Manager) syncRemoteIssue(_ context.Context, ri model.Issue, providerName string) error {
	existing, err := store.GetIssueByProviderRef(m.db, providerName, ri.ID)
	if err != nil {
		return err
	}

	if existing == nil {
		assignee := ""
		if ri.Assignee != nil {
			assignee = ri.Assignee.Login
		}
		state := ri.State
		if state == "" && ri.Resolved != nil {
			state = "Resolved"
		}
		priority := ri.Priority
		if priority == "" {
			priority = "Normal"
		}

		result, err := m.db.Exec(
			`INSERT INTO issues(provider_name, provider_key, provider_ref, summary, description, state, priority, assignee, created_at, updated_at, synced_at, sync_status)
			 VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'synced')`,
			providerName, ri.DatabaseID, ri.ID, ri.Summary, ri.Description, state, priority, assignee,
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
		// Detect if there is a real divergence (not just a local edit with no remote change).
		if !detectConflict(existing, ri) {
			return nil
		}

		// There is a real conflict — resolve it.
		_, strategy, err := m.resolver.Resolve(*storeIssueToModel(existing), ri)
		if err != nil {
			// StrategyManual: store the conflict and return a marker error.
			if strategy == StrategyManual {
				localAssignee := ""
				if existing.Assignee != nil {
					localAssignee = *existing.Assignee
				}
				remoteAssignee := ""
				if ri.Assignee != nil {
					remoteAssignee = ri.Assignee.Login
				}
				_, cerr := store.CreateConflict(m.db, existing.ID, providerName, ri.ID,
					existing.Summary, existing.Description, existing.State, existing.Priority, localAssignee,
					ri.Summary, ri.Description, ri.State, ri.Priority, remoteAssignee,
				)
				if cerr != nil {
					return fmt.Errorf("store conflict: %w", cerr)
				}
				_, _ = m.db.Exec(`UPDATE issues SET sync_status = 'conflict' WHERE id = ?`, existing.ID)
				return fmt.Errorf("conflict: %s", ri.ID)
			}
			return err
		}

		if strategy == StrategyLocalWins {
			// Keep local — just update synced_at so we don't keep checking.
			_, err := m.db.Exec(`UPDATE issues SET synced_at = ? WHERE id = ?`, time.Now().UTC(), existing.ID)
			return err
		}
		// StrategyRemoteWins: fall through to normal update below.
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

// processQueueItem executes a single sync queue item against the remote provider.
func (m *Manager) processQueueItem(ctx context.Context, item store.QueueItem) error {
	issue, err := store.GetIssue(m.db, item.LocalID)
	if err != nil {
		return err
	}
	if issue == nil {
		return fmt.Errorf("local issue %d not found", item.LocalID)
	}

	// For create operations, we need to resolve the project and create via provider directly.
	if item.Operation == "create" {
		return m.pushCreate(ctx, issue)
	}

	// All other operations delegate to PushOp on the provider.
	payload := make(map[string]interface{})
	if item.Payload != "" {
		if err := json.Unmarshal([]byte(item.Payload), &payload); err != nil {
			return fmt.Errorf("parse payload: %w", err)
		}
	}
	payload["ref"] = issue.ProviderRef

	op := provider.QueueOp{
		Operation:    item.Operation,
		EntityType:   item.EntityType,
		LocalID:      item.LocalID,
		Payload:      payload,
		ProviderName: m.prov.Name(),
		CreatedAt:    time.Now().UTC(),
		Attempts:     item.Attempts,
	}

	if err := m.prov.PushOp(ctx, op); err != nil {
		return err
	}

	// After successful push, mark issue as synced for state-changing operations.
	if item.Operation == "update" || item.Operation == "state" {
		return store.SetSyncStatus(m.db, issue.ID, "synced")
	}
	return nil
}

// storeIssueToModel converts a store.LocalIssue to a model.Issue for the resolver.
func storeIssueToModel(li *store.LocalIssue) *model.Issue {
	issue := &model.Issue{
		ID:          li.ProviderRef,
		DatabaseID:  li.ProviderKey,
		Summary:     li.Summary,
		Description: li.Description,
		State:       li.State,
		Priority:    li.Priority,
		Created:     li.CreatedAt.UnixMilli(),
		Updated:     li.UpdatedAt.UnixMilli(),
	}
	if li.Assignee != nil && *li.Assignee != "" {
		issue.Assignee = &model.User{Login: *li.Assignee}
	}
	return issue
}

// pushCreate resolves the project and creates the issue via the provider.
func (m *Manager) pushCreate(ctx context.Context, issue *store.LocalIssue) error {
	projectID := m.cfg.Project
	if projectID == "" {
		return fmt.Errorf("no project configured for create")
	}

	// Resolve project short name to ID using FetchSchema or a direct create.
	// We create the issue with the project short name; the Provider handles resolution.
	newIssue := model.Issue{
		Summary:     issue.Summary,
		Description: issue.Description,
		Project:     &model.Project{ShortName: projectID},
	}

	created, err := m.prov.CreateIssue(ctx, newIssue)
	if err != nil {
		return err
	}

	providerName := ""
	localCfg, _, _ := config.LoadLocal()
	if localCfg != nil && localCfg.Provider.Name != "" {
		providerName = localCfg.Provider.Name
	}
	return store.SetProviderRef(m.db, issue.ID, providerName, created.DatabaseID, created.ID)
}