package api

import (
	"context"
	"fmt"
	"net/url"

	"github.com/Olyxz16/ytcli/internal/model"
)

// ListProjects returns all projects accessible to the current user.
func (c *Client) ListProjects(ctx context.Context) ([]model.Project, error) {
	q := url.Values{}
	q.Set("fields", ProjectList().String())

	var projects []model.Project
	if err := c.doJSON(ctx, "GET", "/api/admin/projects", q, nil, &projects); err != nil {
		return nil, err
	}
	return projects, nil
}

// GetProject returns a single project by ID or short name.
func (c *Client) GetProject(ctx context.Context, id string) (*model.Project, error) {
	q := url.Values{}
	q.Set("fields", ProjectList().String())

	var project model.Project
	if err := c.doJSON(ctx, "GET", fmt.Sprintf("/api/admin/projects/%s", url.PathEscape(id)), q, nil, &project); err != nil {
		return nil, err
	}
	return &project, nil
}
