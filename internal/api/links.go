package api

import (
	"context"
	"fmt"
	"net/url"

	"github.com/Olyxz16/ytcli/internal/model"
)

// ListLinks returns issue links for an issue.
func (c *Client) ListLinks(ctx context.Context, issueID string) ([]model.IssueLink, error) {
	q := url.Values{}
	q.Set("fields", "id,linkType(id,name),direction,issues(id,idReadable,summary)")

	var links []model.IssueLink
	if err := c.doJSON(ctx, "GET", fmt.Sprintf("/api/issues/%s/links", url.PathEscape(issueID)), q, nil, &links); err != nil {
		return nil, err
	}
	return links, nil
}

// AddLink creates a link between two issues.
func (c *Client) AddLink(ctx context.Context, issueID, targetID, linkTypeID string) error {
	payload := map[string]interface{}{
		"issues": []map[string]string{{"idReadable": targetID}},
		"linkType": map[string]string{"id": linkTypeID},
	}
	_, err := c.do(ctx, "POST", fmt.Sprintf("/api/issues/%s/links", url.PathEscape(issueID)), nil, payload)
	return err
}

// ListWorkItems returns work items for an issue.
func (c *Client) ListWorkItems(ctx context.Context, issueID string) ([]model.WorkItem, error) {
	q := url.Values{}
	q.Set("fields", "id,author(id,login,name),created,date,duration(minutes,presentation),text,type(id,name)")

	var items []model.WorkItem
	if err := c.doJSON(ctx, "GET", fmt.Sprintf("/api/issues/%s/timeTracking/workItems", url.PathEscape(issueID)), q, nil, &items); err != nil {
		return nil, err
	}
	return items, nil
}

// AddWorkItem logs work time for an issue.
func (c *Client) AddWorkItem(ctx context.Context, issueID string, item model.WorkItem) (*model.WorkItem, error) {
	q := url.Values{}
	q.Set("fields", "id,author(id,login,name),created,date,duration(minutes,presentation),text,type(id,name)")

	payload := map[string]interface{}{
		"date":     item.Date,
		"duration": item.Duration,
	}
	if item.Text != "" {
		payload["text"] = item.Text
	}
	if item.Type != nil {
		payload["type"] = map[string]string{"id": item.Type.ID}
	}

	var created model.WorkItem
	if err := c.doJSON(ctx, "POST", fmt.Sprintf("/api/issues/%s/timeTracking/workItems", url.PathEscape(issueID)), q, payload, &created); err != nil {
		return nil, err
	}
	return &created, nil
}
