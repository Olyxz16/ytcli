package youtrack

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/Olyxz16/tkt/internal/model"
)

// ListIssues searches for issues matching the query.
func (c *Client) ListIssues(ctx context.Context, query string, top, skip int, customFields []string) ([]model.Issue, error) {
	q := url.Values{}
	q.Set("fields", IssueList().String())
	if query != "" {
		q.Set("query", query)
	}
	if top > 0 {
		q.Set("$top", strconv.Itoa(top))
	}
	if skip > 0 {
		q.Set("$skip", strconv.Itoa(skip))
	}
	for _, cf := range customFields {
		q.Add("customFields", cf)
	}

	var ytIssues []ytIssue
	if err := c.doJSON(ctx, "GET", "/api/issues", q, nil, &ytIssues); err != nil {
		return nil, err
	}
	return toIssues(ytIssues), nil
}

// GetIssue retrieves a single issue by ID (e.g., "PROJ-42" or "2-42").
func (c *Client) GetIssue(ctx context.Context, id string, withComments bool) (*model.Issue, error) {
	q := url.Values{}
	if withComments {
		q.Set("fields", IssueDetailWithComments().String())
	} else {
		q.Set("fields", IssueDetail().String())
	}

	var ytIssue ytIssue
	if err := c.doJSON(ctx, "GET", fmt.Sprintf("/api/issues/%s", url.PathEscape(id)), q, nil, &ytIssue); err != nil {
		return nil, err
	}
	return toIssue(&ytIssue), nil
}

// CreateIssue creates a new issue.
func (c *Client) CreateIssue(ctx context.Context, issue model.Issue) (*model.Issue, error) {
	q := url.Values{}
	q.Set("fields", IssueDetail().String())

	payload := map[string]interface{}{
		"project": map[string]string{"id": issue.Project.ID},
		"summary": issue.Summary,
	}
	if issue.Description != "" {
		payload["description"] = issue.Description
	}

	var ytIssue ytIssue
	if err := c.doJSON(ctx, "POST", "/api/issues", q, payload, &ytIssue); err != nil {
		return nil, err
	}
	return toIssue(&ytIssue), nil
}

// UpdateIssue updates fields of an existing issue.
func (c *Client) UpdateIssue(ctx context.Context, id string, updates map[string]interface{}) (*model.Issue, error) {
	q := url.Values{}
	q.Set("fields", IssueDetail().String())

	var ytIssue ytIssue
	if err := c.doJSON(ctx, "POST", fmt.Sprintf("/api/issues/%s", url.PathEscape(id)), q, updates, &ytIssue); err != nil {
		return nil, err
	}
	return toIssue(&ytIssue), nil
}

// DeleteIssue deletes an issue.
func (c *Client) DeleteIssue(ctx context.Context, id string) error {
	_, err := c.do(ctx, "DELETE", fmt.Sprintf("/api/issues/%s", url.PathEscape(id)), nil, nil)
	return err
}

// IssueCount returns the number of issues matching a query.
func (c *Client) IssueCount(ctx context.Context, query string) (int, error) {
	q := url.Values{}
	q.Set("fields", "count")
	if query != "" {
		q.Set("query", query)
	}

	var result struct {
		Count int `json:"count"`
	}
	if err := c.doJSON(ctx, "GET", "/api/issuesGetter/count", q, nil, &result); err != nil {
		return 0, err
	}
	return result.Count, nil
}