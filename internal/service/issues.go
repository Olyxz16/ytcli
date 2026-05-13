package service

import (
	"context"
	"fmt"

	"github.com/Olyxz16/tkt/internal/config"
	"github.com/Olyxz16/tkt/internal/model"
	"github.com/Olyxz16/tkt/internal/provider"
	"github.com/Olyxz16/tkt/internal/provider/youtrack"
)

// Service provides the business logic layer.
type Service struct {
	provider *youtrack.Provider
}

// RemoteProvider returns the underlying provider as a RemoteProvider interface.
func (s *Service) RemoteProvider() provider.RemoteProvider {
	return s.provider
}

// NewService creates a Service from a merged config.
func NewService(cfg *config.MergedConfig) (*Service, error) {
	if cfg.ProviderURL == "" {
		return nil, fmt.Errorf("no provider URL configured")
	}
	token, err := config.GetToken(cfg.ProviderName)
	if err != nil {
		return nil, fmt.Errorf("auth: %w", err)
	}
	return &Service{
		provider: youtrack.NewProvider(cfg.ProviderURL, token),
	}, nil
}

// NewServiceWithClient creates a Service with an explicit YouTrack client (useful for tests).
func NewServiceWithClient(client *youtrack.Client) *Service {
	return &Service{provider: youtrack.NewProviderWithClient(client)}
}

const pageSize = 50

// ListIssues lists issues matching the query.
// If top is 0, all matching issues are fetched (auto-pagination).
func (s *Service) ListIssues(ctx context.Context, query string, top, skip int) ([]model.Issue, error) {
	q := buildProviderQuery(query)
	if top > 0 {
		return s.provider.FetchIssues(ctx, q, top, skip)
	}

	var all []model.Issue
	offset := skip
	for {
		issues, err := s.provider.FetchIssues(ctx, q, pageSize, offset)
		if err != nil {
			return nil, err
		}
		if len(issues) == 0 {
			break
		}
		all = append(all, issues...)
		if len(issues) < pageSize {
			break
		}
		offset += pageSize
	}
	return all, nil
}

// GetIssue retrieves an issue by ID.
func (s *Service) GetIssue(ctx context.Context, id string, withComments bool) (*model.Issue, error) {
	return s.provider.FetchIssue(ctx, id)
}

// CreateIssue creates a new issue.
func (s *Service) CreateIssue(ctx context.Context, issue model.Issue) (*model.Issue, error) {
	if issue.Project.ID == "" && issue.Project.ShortName == "" {
		return nil, fmt.Errorf("project is required")
	}
	if issue.Summary == "" {
		return nil, fmt.Errorf("summary is required")
	}
	if issue.Project.ID == "" && issue.Project.ShortName != "" {
		projects, err := s.ListProjects(ctx)
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
	return s.provider.CreateIssue(ctx, issue)
}

// UpdateIssue updates an issue.
func (s *Service) UpdateIssue(ctx context.Context, id string, updates map[string]interface{}) (*model.Issue, error) {
	return s.provider.UpdateIssue(ctx, id, updates)
}

// DeleteIssue deletes an issue.
func (s *Service) DeleteIssue(ctx context.Context, id string) error {
	return s.provider.DeleteIssue(ctx, id)
}

// IssueCount returns the count of issues matching a query.
func (s *Service) IssueCount(ctx context.Context, query string) (int, error) {
	return s.provider.Client().IssueCount(ctx, query)
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
		issues[i] = model.Issue{ID: id}
	}
	return s.provider.Client().ExecuteCommand(ctx, model.CommandResult{Query: query, Issues: issues}, silent)
}

// ListComments lists comments on an issue.
func (s *Service) ListComments(ctx context.Context, issueID string) ([]model.Comment, error) {
	return s.provider.Client().ListComments(ctx, issueID)
}

// AddComment adds a comment to an issue.
func (s *Service) AddComment(ctx context.Context, issueID, text string) (*model.Comment, error) {
	if text == "" {
		return nil, fmt.Errorf("comment text is required")
	}
	return s.provider.Client().AddComment(ctx, issueID, text)
}

// ListProjects lists all accessible projects.
func (s *Service) ListProjects(ctx context.Context) ([]model.Project, error) {
	return s.provider.Client().ListProjects(ctx)
}

// GetProject retrieves a project.
func (s *Service) GetProject(ctx context.Context, id string) (*model.Project, error) {
	return s.provider.Client().GetProject(ctx, id)
}

// Me returns the current user.
func (s *Service) Me(ctx context.Context) (*model.User, error) {
	return s.provider.Client().Me(ctx)
}

// ListLinks lists links for an issue.
func (s *Service) ListLinks(ctx context.Context, issueID string) ([]model.IssueLink, error) {
	return s.provider.Client().ListLinks(ctx, issueID)
}

// AddLink creates a link between issues.
func (s *Service) AddLink(ctx context.Context, issueID, targetID, linkTypeID string) error {
	return s.provider.Client().AddLink(ctx, issueID, targetID, linkTypeID)
}

// ListWorkItems lists logged work for an issue.
func (s *Service) ListWorkItems(ctx context.Context, issueID string) ([]model.WorkItem, error) {
	return s.provider.Client().ListWorkItems(ctx, issueID)
}

// AddWorkItem logs work time.
func (s *Service) AddWorkItem(ctx context.Context, issueID string, item model.WorkItem) (*model.WorkItem, error) {
	return s.provider.Client().AddWorkItem(ctx, issueID, item)
}

// ListTags returns all tags.
func (s *Service) ListTags(ctx context.Context) ([]model.Tag, error) {
	return s.provider.Client().ListTags(ctx)
}

// AddTagToIssue adds a tag to an issue.
func (s *Service) AddTagToIssue(ctx context.Context, issueID string, tag model.Tag) error {
	return s.provider.Client().AddTagToIssue(ctx, issueID, tag)
}

// RemoveTagFromIssue removes a tag from an issue by tag name.
func (s *Service) RemoveTagFromIssue(ctx context.Context, issueID, tagName string) error {
	issue, err := s.provider.FetchIssue(ctx, issueID)
	if err != nil {
		return fmt.Errorf("fetch issue tags: %w", err)
	}
	for _, tag := range issue.Tags {
		if tag.Name == tagName {
			return s.provider.Client().RemoveTagFromIssue(ctx, issueID, tag.ID)
		}
	}
	return fmt.Errorf("tag %q not found on issue %s", tagName, issueID)
}

// ListArticles lists articles matching a query.
func (s *Service) ListArticles(ctx context.Context, query string, top, skip int) ([]model.Article, error) {
	if top > 0 {
		return s.provider.Client().ListArticles(ctx, query, top, skip)
	}
	var all []model.Article
	for {
		page, err := s.provider.Client().ListArticles(ctx, query, pageSize, skip)
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

// GetArticle retrieves an article by ID.
func (s *Service) GetArticle(ctx context.Context, id string, withComments bool) (*model.Article, error) {
	return s.provider.Client().GetArticle(ctx, id, withComments)
}

// CreateArticle creates a new article.
func (s *Service) CreateArticle(ctx context.Context, article model.Article) (*model.Article, error) {
	if article.Project == nil || (article.Project.ID == "" && article.Project.ShortName == "") {
		return nil, fmt.Errorf("project is required")
	}
	if article.Summary == "" {
		return nil, fmt.Errorf("summary is required")
	}
	if article.Project.ID == "" && article.Project.ShortName != "" {
		projects, err := s.ListProjects(ctx)
		if err != nil {
			return nil, fmt.Errorf("resolve project: %w", err)
		}
		for _, p := range projects {
			if p.ShortName == article.Project.ShortName || p.Name == article.Project.ShortName {
				article.Project.ID = p.ID
				break
			}
		}
		if article.Project.ID == "" {
			return nil, fmt.Errorf("project %q not found", article.Project.ShortName)
		}
	}
	return s.provider.Client().CreateArticle(ctx, article)
}

// UpdateArticle updates an article.
func (s *Service) UpdateArticle(ctx context.Context, id string, updates map[string]interface{}) (*model.Article, error) {
	return s.provider.Client().UpdateArticle(ctx, id, updates)
}

// DeleteArticle deletes an article.
func (s *Service) DeleteArticle(ctx context.Context, id string) error {
	return s.provider.Client().DeleteArticle(ctx, id)
}

// buildProviderQuery converts a raw YouTrack query string to a provider.Query.
// This preserves backward compatibility for direct query strings.
func buildProviderQuery(rawQuery string) provider.Query {
	return provider.Query{Text: rawQuery}
}