package youtrack

import (
	"context"
	"fmt"

	"github.com/Olyxz16/tkt/internal/model"
	"github.com/Olyxz16/tkt/internal/provider"
)

// Provider implements the RemoteProvider interface for YouTrack.
type Provider struct {
	client *Client
}

// QueryOptions holds pagination and filtering options for YouTrack queries.
type QueryOptions struct {
	Query string
	Top   int
	Skip  int
}

// NewProvider creates a new YouTrack provider.
func NewProvider(baseURL, token string) *Provider {
	return &Provider{
		client: NewClient(baseURL, token),
	}
}

// NewProviderWithClient creates a YouTrack provider with an explicit client (for tests).
func NewProviderWithClient(client *Client) *Provider {
	return &Provider{client: client}
}

// Client returns the underlying YouTrack API client for direct access.
func (p *Provider) Client() *Client {
	return p.client
}

// Name returns the provider identifier.
func (p *Provider) Name() string {
	return "youtrack"
}

// Ping verifies connectivity and authentication.
func (p *Provider) Ping(ctx context.Context) error {
	_, err := p.client.Me(ctx)
	if err != nil {
		if IsAuthError(err) {
			return &provider.AuthError{Provider: "youtrack", Message: err.Error()}
		}
		return &provider.NetworkError{Provider: "youtrack", Message: err.Error()}
	}
	return nil
}

// Auth validates stored credentials.
func (p *Provider) Auth(ctx context.Context) error {
	return p.Ping(ctx)
}

// FetchIssues retrieves issues matching the given query.
func (p *Provider) FetchIssues(ctx context.Context, q provider.Query, top, skip int) ([]model.Issue, error) {
	query := buildYouTrackQuery(q)
	if top == 0 {
		top = 50
	}
	issues, err := p.client.ListIssues(ctx, query, top, skip, nil)
	if err != nil {
		return nil, wrapError(err)
	}
	return issues, nil
}

// FetchIssue retrieves a single issue by its provider reference.
func (p *Provider) FetchIssue(ctx context.Context, ref string) (*model.Issue, error) {
	issue, err := p.client.GetIssue(ctx, ref, false)
	if err != nil {
		return nil, wrapError(err)
	}
	return issue, nil
}

// CreateIssue creates a new issue on YouTrack.
// If issue.Project.ID is empty but ShortName is set, resolves the project ID first.
func (p *Provider) CreateIssue(ctx context.Context, issue model.Issue) (*model.Issue, error) {
	if issue.Project != nil && issue.Project.ID == "" && issue.Project.ShortName != "" {
		projects, err := p.client.ListProjects(ctx)
		if err != nil {
			return nil, wrapError(err)
		}
		for _, proj := range projects {
			if proj.ShortName == issue.Project.ShortName || proj.Name == issue.Project.ShortName {
				issue.Project.ID = proj.ID
				break
			}
		}
		if issue.Project.ID == "" {
			return nil, &provider.ValidationError{Provider: "youtrack", Message: fmt.Sprintf("project %q not found", issue.Project.ShortName)}
		}
	}
	created, err := p.client.CreateIssue(ctx, issue)
	if err != nil {
		return nil, wrapError(err)
	}
	return created, nil
}

// UpdateIssue updates an existing issue on YouTrack.
func (p *Provider) UpdateIssue(ctx context.Context, ref string, updates map[string]interface{}) (*model.Issue, error) {
	updated, err := p.client.UpdateIssue(ctx, ref, updates)
	if err != nil {
		return nil, wrapError(err)
	}
	return updated, nil
}

// DeleteIssue deletes an issue on YouTrack.
func (p *Provider) DeleteIssue(ctx context.Context, ref string) error {
	err := p.client.DeleteIssue(ctx, ref)
	if err != nil {
		return wrapError(err)
	}
	return nil
}

// FetchSchema returns the YouTrack project's workflow configuration.
func (p *Provider) FetchSchema(ctx context.Context) (*provider.SchemaConfig, error) {
	// YouTrack doesn't have a direct schema API, so we return nil
	// to indicate that the local schema should be used as-is.
	// Future: could fetch project custom fields to build the schema.
	return nil, nil
}

// PushOp pushes a queued operation to YouTrack.
func (p *Provider) PushOp(ctx context.Context, op provider.QueueOp) error {
	switch op.Operation {
	case "create":
		return p.pushCreate(ctx, op)
	case "update":
		return p.pushUpdate(ctx, op)
	case "comment":
		return p.pushComment(ctx, op)
	case "state":
		return p.pushState(ctx, op)
	case "tag":
		return p.pushTag(ctx, op)
	case "untag":
		return p.pushUntag(ctx, op)
	default:
		return fmt.Errorf("unknown operation: %s", op.Operation)
	}
}

func (p *Provider) pushCreate(ctx context.Context, op provider.QueueOp) error {
	summary, _ := op.Payload["summary"].(string)
	description, _ := op.Payload["description"].(string)
	projectID, _ := op.Payload["project_id"].(string)

	issue := model.Issue{
		Summary:     summary,
		Description: description,
		Project:     &model.Project{ID: projectID},
	}

	_, err := p.client.CreateIssue(ctx, issue)
	return err
}

func (p *Provider) pushUpdate(ctx context.Context, op provider.QueueOp) error {
	ref, _ := op.Payload["ref"].(string)
	if ref == "" {
		return fmt.Errorf("issue has no provider reference")
	}
	_, err := p.client.UpdateIssue(ctx, ref, op.Payload)
	return err
}

func (p *Provider) pushComment(ctx context.Context, op provider.QueueOp) error {
	ref, _ := op.Payload["ref"].(string)
	text, _ := op.Payload["text"].(string)
	if ref == "" {
		return fmt.Errorf("issue has no provider reference")
	}
	_, err := p.client.AddComment(ctx, ref, text)
	return err
}

func (p *Provider) pushState(ctx context.Context, op provider.QueueOp) error {
	ref, _ := op.Payload["ref"].(string)
	state, _ := op.Payload["state"].(string)
	if ref == "" {
		return fmt.Errorf("issue has no provider reference")
	}
	_, err := p.client.ExecuteCommand(ctx, model.CommandResult{Query: fmt.Sprintf("State: %s", state), Issues: []model.Issue{{ID: ref}}}, false)
	if err != nil {
		return wrapError(err)
	}
	return nil
}

func (p *Provider) pushTag(ctx context.Context, op provider.QueueOp) error {
	ref, _ := op.Payload["ref"].(string)
	tagName, _ := op.Payload["tag"].(string)
	if ref == "" {
		return fmt.Errorf("issue has no provider reference")
	}
	_, err := p.client.ExecuteCommand(ctx, model.CommandResult{Query: fmt.Sprintf("tag %s", tagName), Issues: []model.Issue{{ID: ref}}}, false)
	return err
}

func (p *Provider) pushUntag(ctx context.Context, op provider.QueueOp) error {
	ref, _ := op.Payload["ref"].(string)
	tagName, _ := op.Payload["tag"].(string)
	if ref == "" {
		return fmt.Errorf("issue has no provider reference")
	}
	_, err := p.client.ExecuteCommand(ctx, model.CommandResult{Query: fmt.Sprintf("untag %s", tagName), Issues: []model.Issue{{ID: ref}}}, false)
	return err
}

// buildYouTrackQuery converts a provider.Query to a YouTrack query string.
func buildYouTrackQuery(q provider.Query) string {
	var parts []string
	if q.Project != "" {
		parts = append(parts, fmt.Sprintf("project: {%s}", q.Project))
	}
	if q.Assignee != "" {
		parts = append(parts, fmt.Sprintf("for: %s", q.Assignee))
	}
	if q.State != "" {
		parts = append(parts, fmt.Sprintf("State: {%s}", q.State))
	}
	if q.Text != "" {
		parts = append(parts, q.Text)
	}
	result := ""
	for i, part := range parts {
		if i > 0 {
			result += " "
		}
		result += part
	}
	return result
}

// wrapError converts YouTrack API errors to provider errors.
func wrapError(err error) error {
	if err == nil {
		return nil
	}
	if IsAuthError(err) {
		return &provider.AuthError{Provider: "youtrack", Message: err.Error()}
	}
	if IsNotFoundError(err) {
		return &provider.NotFoundError{Provider: "youtrack", Message: err.Error()}
	}
	if IsValidationError(err) {
		return &provider.ValidationError{Provider: "youtrack", Message: err.Error()}
	}
	return &provider.NetworkError{Provider: "youtrack", Message: err.Error(), Retry: true}
}