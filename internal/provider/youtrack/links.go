package youtrack

import (
	"context"
	"fmt"
	"net/url"

	"github.com/Olyxz16/tkt/internal/model"
)

// ListLinks returns issue links for an issue.
func (c *Client) ListLinks(ctx context.Context, issueID string) ([]model.IssueLink, error) {
	q := url.Values{}
	q.Set("fields", "id,linkType(id,name),direction,issues(id,idReadable,summary)")

	var ytLinks []ytIssueLink
	if err := c.doJSON(ctx, "GET", fmt.Sprintf("/api/issues/%s/links", url.PathEscape(issueID)), q, nil, &ytLinks); err != nil {
		return nil, err
	}

	links := make([]model.IssueLink, len(ytLinks))
	for i, l := range ytLinks {
		links[i] = model.IssueLink{
			ID:        l.ID,
			LinkType:  l.LinkType,
			Direction: l.Direction,
		}
		for _, li := range l.Issues {
			links[i].Issues = append(links[i].Issues, *toIssue(&li))
		}
	}
	return links, nil
}

// AddLink creates a link between two issues.
func (c *Client) AddLink(ctx context.Context, issueID, targetID, linkTypeID string) error {
	payload := map[string]interface{}{
		"issues":   []map[string]string{{"idReadable": targetID}},
		"linkType": map[string]string{"id": linkTypeID},
	}
	_, err := c.do(ctx, "POST", fmt.Sprintf("/api/issues/%s/links", url.PathEscape(issueID)), nil, payload)
	return err
}

// ListWorkItems returns work items for an issue.
func (c *Client) ListWorkItems(ctx context.Context, issueID string) ([]model.WorkItem, error) {
	q := url.Values{}
	q.Set("fields", "id,author(id,login,name),created,date,duration(minutes,presentation),text,type(id,name)")

	var ytItems []ytWorkItem
	if err := c.doJSON(ctx, "GET", fmt.Sprintf("/api/issues/%s/timeTracking/workItems", url.PathEscape(issueID)), q, nil, &ytItems); err != nil {
		return nil, err
	}

	items := make([]model.WorkItem, len(ytItems))
	for i := range ytItems {
		items[i] = *toWorkItem(&ytItems[i])
	}
	return items, nil
}

// AddWorkItem logs work time for an issue.
func (c *Client) AddWorkItem(ctx context.Context, issueID string, item model.WorkItem) (*model.WorkItem, error) {
	q := url.Values{}
	q.Set("fields", "id,author(id,login,name),created,date,duration(minutes,presentation),text,type(id,name)")

	payload := map[string]interface{}{
		"date": item.Date,
		"duration": map[string]interface{}{
			"minutes": item.Duration.Minutes,
		},
	}
	if item.Text != "" {
		payload["text"] = item.Text
	}
	if item.Type != nil {
		payload["type"] = map[string]string{"id": item.Type.ID}
	}

	var ytItem ytWorkItem
	if err := c.doJSON(ctx, "POST", fmt.Sprintf("/api/issues/%s/timeTracking/workItems", url.PathEscape(issueID)), q, payload, &ytItem); err != nil {
		return nil, err
	}
	return toWorkItem(&ytItem), nil
}