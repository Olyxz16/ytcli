package api

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/Olyxz16/ytcli/internal/model"
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

	var issues []model.Issue
	if err := c.doJSON(ctx, "GET", "/api/issues", q, nil, &issues); err != nil {
		return nil, err
	}
	return issues, nil
}

// GetIssue retrieves a single issue by ID (e.g., "PROJ-42" or "2-42").
func (c *Client) GetIssue(ctx context.Context, id string, withComments bool) (*model.Issue, error) {
	q := url.Values{}
	if withComments {
		q.Set("fields", IssueDetailWithComments().String())
	} else {
		q.Set("fields", IssueDetail().String())
	}

	var issue model.Issue
	if err := c.doJSON(ctx, "GET", fmt.Sprintf("/api/issues/%s", url.PathEscape(id)), q, nil, &issue); err != nil {
		return nil, err
	}
	return &issue, nil
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
	if len(issue.CustomFields) > 0 {
		payload["customFields"] = issue.CustomFields
	}
	if len(issue.Tags) > 0 {
		payload["tags"] = issue.Tags
	}

	var created model.Issue
	if err := c.doJSON(ctx, "POST", "/api/issues", q, payload, &created); err != nil {
		return nil, err
	}
	return &created, nil
}

// UpdateIssue updates fields of an existing issue.
func (c *Client) UpdateIssue(ctx context.Context, id string, updates map[string]interface{}) (*model.Issue, error) {
	q := url.Values{}
	q.Set("fields", IssueDetail().String())

	var updated model.Issue
	if err := c.doJSON(ctx, "POST", fmt.Sprintf("/api/issues/%s", url.PathEscape(id)), q, updates, &updated); err != nil {
		return nil, err
	}
	return &updated, nil
}

// DeleteIssue deletes an issue.
func (c *Client) DeleteIssue(ctx context.Context, id string) error {
	_, err := c.do(ctx, "DELETE", fmt.Sprintf("/api/issues/%s", url.PathEscape(id)), nil, nil)
	if err != nil {
		return err
	}
	return nil
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
