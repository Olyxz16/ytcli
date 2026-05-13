package sync

import (
	"fmt"

	"github.com/Olyxz16/tkt/internal/model"
	"github.com/Olyxz16/tkt/internal/provider"
	"github.com/Olyxz16/tkt/internal/store"
)

// ConflictStrategy names the available conflict resolution strategies.
type ConflictStrategy string

const (
	// StrategyLocalWins keeps the local version when a conflict is detected.
	StrategyLocalWins ConflictStrategy = "local"

	// StrategyRemoteWins overwrites the local version with the remote version.
	StrategyRemoteWins ConflictStrategy = "remote"

	// StrategyManual stores the conflict and requires user resolution (default).
	StrategyManual ConflictStrategy = "manual"
)

// ConflictResolver picks a winner when a local issue and a remote issue diverge.
type ConflictResolver interface {
	// Strategy returns the resolver's strategy name.
	Strategy() ConflictStrategy

	// Resolve decides which version wins.
	// Returns the winning issue, the resolution string, and an error.
	// For StrategyManual it returns nil and an error so the conflict is stored.
	Resolve(local, remote model.Issue) (*model.Issue, ConflictStrategy, error)
}

// LocalWinsResolver always keeps the local version.
type LocalWinsResolver struct{}

func (r *LocalWinsResolver) Strategy() ConflictStrategy { return StrategyLocalWins }

func (r *LocalWinsResolver) Resolve(local, remote model.Issue) (*model.Issue, ConflictStrategy, error) {
	return &local, StrategyLocalWins, nil
}

// RemoteWinsResolver always overwrites with the remote version.
type RemoteWinsResolver struct{}

func (r *RemoteWinsResolver) Strategy() ConflictStrategy { return StrategyRemoteWins }

func (r *RemoteWinsResolver) Resolve(local, remote model.Issue) (*model.Issue, ConflictStrategy, error) {
	return &remote, StrategyRemoteWins, nil
}

// ManualResolver stores the conflict for user resolution.
type ManualResolver struct{}

func (r *ManualResolver) Strategy() ConflictStrategy { return StrategyManual }

func (r *ManualResolver) Resolve(local, remote model.Issue) (*model.Issue, ConflictStrategy, error) {
	return nil, StrategyManual, &provider.ConflictError{
		Provider:  "",
		LocalRef:  local.ID,
		RemoteRef: remote.ID,
		Message:   fmt.Sprintf("conflict on issue %s — manual resolution required", local.ID),
	}
}

// NewConflictResolver creates a resolver for the given strategy.
func NewConflictResolver(strategy ConflictStrategy) ConflictResolver {
	switch strategy {
	case StrategyLocalWins:
		return &LocalWinsResolver{}
	case StrategyRemoteWins:
		return &RemoteWinsResolver{}
	default:
		return &ManualResolver{}
	}
}

// detectConflict returns true if the local issue differs from the remote issue
// in a way that requires resolution. It compares summary, description, state,
// priority and assignee.
func detectConflict(local *store.LocalIssue, remote model.Issue) bool {
	if local.Summary != remote.Summary {
		return true
	}
	if local.Description != remote.Description {
		return true
	}
	if local.State != remote.State {
		return true
	}
	if local.Priority != remote.Priority {
		return true
	}
	localAssignee := ""
	if local.Assignee != nil {
		localAssignee = *local.Assignee
	}
	remoteAssignee := ""
	if remote.Assignee != nil {
		remoteAssignee = remote.Assignee.Login
	}
	if localAssignee != remoteAssignee {
		return true
	}
	return false
}