package service

import (
	"context"
	"fmt"

	"github.com/Olyxz16/ytcli/internal/api"
	"github.com/Olyxz16/ytcli/internal/config"
	"github.com/Olyxz16/ytcli/internal/model"
)

// Service provides the business logic layer.
type Service struct {
	client *api.Client
}

// NewService creates a Service from a merged config.
func NewService(cfg *config.MergedConfig) (*Service, error) {
	if cfg.InstanceURL == "" {
		return nil, fmt.Errorf("no instance URL configured")
	}
	token, err := config.GetToken(cfg.Instance)
	if err != nil {
		return nil, fmt.Errorf("auth: %w", err)
	}
	return &Service{
		client: api.NewClient(cfg.InstanceURL, token),
	}, nil
}

// NewServiceWithClient creates a Service with an explicit API client (useful for tests).
func NewServiceWithClient(client *api.Client) *Service {
	return &Service{client: client}
}

const pageSize = 50

// ListIssues lists issues matching the query.
// If top is 0, all matching issues are fetched (auto-pagination).
func (s *Service) ListIssues(ctx context.Context, query string, top, skip int) ([]model.Issue, error) {
	if top > 0 {
		return s.client.ListIssues(ctx, query, top, skip, nil)
	}

	// Auto-pagination
	var all []model.Issue
	for {
		page, err := s.client.ListIssues(ctx, query, pageSize, skip, nil)
		if err != nil {
			return nil, err
		}
		if len(page) == 0 {
			break
		}
		all = append(all, page...)
		if len(page) < pageSize {
			break
		}
		skip += pageSize
	}
	return all, nil
}

// GetIssue retrieves an issue by ID.
func (s *Service) GetIssue(ctx context.Context, id string, withComments bool) (*model.Issue, error) {
	return s.client.GetIssue(ctx, id, withComments)
}

// CreateIssue creates a new issue.
func (s *Service) CreateIssue(ctx context.Context, issue model.Issue) (*model.Issue, error) {
	if issue.Project.ID == "" && issue.Project.ShortName == "" {
		return nil, fmt.Errorf("project is required")
	}
	if issue.Summary == "" {
		return nil, fmt.Errorf("summary is required")
	}

	// Resolve project short name to ID if needed
	if issue.Project.ID == "" && issue.Project.ShortName != "" {
		projects, err := s.client.ListProjects(ctx)
		if err != nil {
			return nil, fmt.Errorf("resolve project: %w", err)
		}
		for _, p := range projects {
			if p.ShortName == issue.Project.ShortName || p.Name == issue.Project.ShortName {
				issue.Project.ID = p.ID
				break
			}
		}
		if issue.Project.ID == "" {
			return nil, fmt.Errorf("project %q not found", issue.Project.ShortName)
		}
	}

	return s.client.CreateIssue(ctx, issue)
}

// UpdateIssue updates an issue.
func (s *Service) UpdateIssue(ctx context.Context, id string, updates map[string]interface{}) (*model.Issue, error) {
	return s.client.UpdateIssue(ctx, id, updates)
}

// DeleteIssue deletes an issue.
func (s *Service) DeleteIssue(ctx context.Context, id string) error {
	return s.client.DeleteIssue(ctx, id)
}

// IssueCount returns the count of issues matching a query.
func (s *Service) IssueCount(ctx context.Context, query string) (int, error) {
	return s.client.IssueCount(ctx, query)
}

// ExecuteCommand applies a YouTrack command.
func (s *Service) ExecuteCommand(ctx context.Context, query string, issueIDs []string, silent bool) (*model.CommandResult, error) {
	if query == "" {
		return nil, fmt.Errorf("command query is required")
	}
	if len(issueIDs) == 0 {
		return nil, fmt.Errorf("at least one issue ID is required")
	}
	issues := make([]model.Issue, len(issueIDs))
	for i, id := range issueIDs {
		issues[i] = model.Issue{IDReadable: id}
	}
	return s.client.ExecuteCommand(ctx, model.CommandResult{Query: query, Issues: issues}, silent)
}

// ListComments lists comments on an issue.
func (s *Service) ListComments(ctx context.Context, issueID string) ([]model.Comment, error) {
	return s.client.ListComments(ctx, issueID)
}

// AddComment adds a comment to an issue.
func (s *Service) AddComment(ctx context.Context, issueID, text string) (*model.Comment, error) {
	if text == "" {
		return nil, fmt.Errorf("comment text is required")
	}
	return s.client.AddComment(ctx, issueID, text)
}

// ListProjects lists all accessible projects.
func (s *Service) ListProjects(ctx context.Context) ([]model.Project, error) {
	return s.client.ListProjects(ctx)
}

// GetProject retrieves a project.
func (s *Service) GetProject(ctx context.Context, id string) (*model.Project, error) {
	return s.client.GetProject(ctx, id)
}

// Me returns the current user.
func (s *Service) Me(ctx context.Context) (*model.User, error) {
	return s.client.Me(ctx)
}

// ListLinks lists links for an issue.
func (s *Service) ListLinks(ctx context.Context, issueID string) ([]model.IssueLink, error) {
	return s.client.ListLinks(ctx, issueID)
}

// AddLink creates a link between issues.
func (s *Service) AddLink(ctx context.Context, issueID, targetID, linkTypeID string) error {
	return s.client.AddLink(ctx, issueID, targetID, linkTypeID)
}

// ListWorkItems lists logged work for an issue.
func (s *Service) ListWorkItems(ctx context.Context, issueID string) ([]model.WorkItem, error) {
	return s.client.ListWorkItems(ctx, issueID)
}

// AddWorkItem logs work time.
func (s *Service) AddWorkItem(ctx context.Context, issueID string, item model.WorkItem) (*model.WorkItem, error) {
	return s.client.AddWorkItem(ctx, issueID, item)
}

// ListTags returns all tags.
func (s *Service) ListTags(ctx context.Context) ([]model.Tag, error) {
	return s.client.ListTags(ctx)
}

// AddTagToIssue adds a tag to an issue.
func (s *Service) AddTagToIssue(ctx context.Context, issueID string, tag model.Tag) error {
	return s.client.AddTagToIssue(ctx, issueID, tag)
}

// RemoveTagFromIssue removes a tag from an issue by tag name.
// It resolves the tag name to its ID by fetching the issue's tags first.
func (s *Service) RemoveTagFromIssue(ctx context.Context, issueID, tagName string) error {
	issue, err := s.client.GetIssue(ctx, issueID, false)
	if err != nil {
		return fmt.Errorf("fetch issue tags: %w", err)
	}
	for _, tag := range issue.Tags {
		if tag.Name == tagName {
			return s.client.RemoveTagFromIssue(ctx, issueID, tag.ID)
		}
	}
	return fmt.Errorf("tag %q not found on issue %s", tagName, issueID)
}
