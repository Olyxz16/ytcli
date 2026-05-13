package provider

import (
	"context"
	"time"

	"github.com/Olyxz16/tkt/internal/model"
)

// RemoteProvider is the interface that all issue tracker providers must implement.
// It abstracts the remote API so the sync engine and CLI commands can work
// with any backend (YouTrack, GitHub, Jira, etc.) without knowing the details.
type RemoteProvider interface {
	// Name returns the provider identifier (e.g., "youtrack", "github").
	Name() string

	// Ping verifies connectivity and authentication with the remote service.
	Ping(ctx context.Context) error

	// Auth validates stored credentials for the provider.
	// Returns an AuthError if credentials are missing or expired.
	Auth(ctx context.Context) error

	// FetchIssues retrieves issues matching the given query from the remote provider.
	// top controls pagination (0 means use default), skip is the offset.
	FetchIssues(ctx context.Context, query Query, top, skip int) ([]model.Issue, error)

	// FetchIssue retrieves a single issue by its provider reference.
	FetchIssue(ctx context.Context, ref string) (*model.Issue, error)

	// CreateIssue creates a new issue on the remote provider.
	CreateIssue(ctx context.Context, issue model.Issue) (*model.Issue, error)

	// UpdateIssue updates an existing issue on the remote provider.
	UpdateIssue(ctx context.Context, ref string, updates map[string]interface{}) (*model.Issue, error)

	// DeleteIssue deletes an issue on the remote provider.
	DeleteIssue(ctx context.Context, ref string) error

	// FetchSchema returns the provider's workflow schema (states, priorities, etc.).
	FetchSchema(ctx context.Context) (*SchemaConfig, error)

	// PushOp pushes a queued operation to the remote provider.
	PushOp(ctx context.Context, op QueueOp) error
}

// Query represents a query for remote issues.
// It is provider-agnostic; each provider translates it into its native query language.
type Query struct {
	// Project filters by project short name (e.g., "PROJ").
	Project string

	// Assignee filters by assignee login (e.g., "me" for current user).
	Assignee string

	// State filters by issue state (e.g., "Open", "Done").
	State string

	// Text is a full-text search query.
	Text string
}

// QueueOp represents a queued sync operation to be pushed to the remote provider.
type QueueOp struct {
	// Operation is the type of operation: "create", "update", "comment", "state", "tag", "untag".
	Operation string

	// EntityType is the type of entity: "issue", "comment", etc.
	EntityType string

	// LocalID is the local database ID of the entity.
	LocalID int64

	// Payload contains operation-specific data as a JSON-serializable map.
	Payload map[string]interface{}

	// ProviderName is the name of the provider this op is for.
	ProviderName string

	// CreatedAt is when the operation was created.
	CreatedAt time.Time

	// Attempts is how many times this operation has been attempted.
	Attempts int

	// LastError is the error message from the last attempt, if any.
	LastError string
}

// SchemaConfig describes the remote provider's workflow configuration.
type SchemaConfig struct {
	// States are the valid state values for issues.
	States []string

	// Priorities are the valid priority values for issues.
	Priorities []string

	// DefaultState is the default state for new issues.
	DefaultState string

	// DefaultPriority is the default priority for new issues.
	DefaultPriority string

	// DoneStates are the states that mark an issue as completed.
	DoneStates []string
}

// AuthError indicates an authentication failure with the remote provider.
type AuthError struct {
	Provider string
	Message  string
}

func (e *AuthError) Error() string {
	return "auth error (" + e.Provider + "): " + e.Message
}

// NetworkError indicates a network or connectivity failure.
type NetworkError struct {
	Provider string
	Message  string
	Retry    bool
}

func (e *NetworkError) Error() string {
	return "network error (" + e.Provider + "): " + e.Message
}

// ValidationError indicates that the remote provider rejected the operation.
type ValidationError struct {
	Provider string
	Message  string
	Field    string
}

func (e *ValidationError) Error() string {
	return "validation error (" + e.Provider + "): " + e.Message
}

// NotFoundError indicates that the requested resource was not found.
type NotFoundError struct {
	Provider string
	Message  string
}

func (e *NotFoundError) Error() string {
	return "not found (" + e.Provider + "): " + e.Message
}

// ConflictError indicates a sync conflict between local and remote data.
type ConflictError struct {
	Provider   string
	LocalRef   string
	RemoteRef  string
	Message    string
}

func (e *ConflictError) Error() string {
	return "conflict (" + e.Provider + "): " + e.Message
}