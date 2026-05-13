package youtrack

import (
	"context"
	"fmt"
	"net/url"

	"github.com/Olyxz16/tkt/internal/model"
)

// ListProjects returns all projects accessible to the current user.
func (c *Client) ListProjects(ctx context.Context) ([]model.Project, error) {
	q := url.Values{}
	q.Set("fields", ProjectList().String())

	var ytProjects []ytProject
	if err := c.doJSON(ctx, "GET", "/api/admin/projects", q, nil, &ytProjects); err != nil {
		return nil, err
	}

	projects := make([]model.Project, len(ytProjects))
	for i, p := range ytProjects {
		projects[i] = model.Project{
			ID:          p.ID,
			Name:        p.Name,
			ShortName:   p.ShortName,
			Description: p.Description,
		}
	}
	return projects, nil
}

// GetProject returns a single project by ID or short name.
func (c *Client) GetProject(ctx context.Context, id string) (*model.Project, error) {
	q := url.Values{}
	q.Set("fields", ProjectList().String())

	var ytProject ytProject
	if err := c.doJSON(ctx, "GET", fmt.Sprintf("/api/admin/projects/%s", url.PathEscape(id)), q, nil, &ytProject); err != nil {
		return nil, err
	}
	return &model.Project{
		ID:          ytProject.ID,
		Name:        ytProject.Name,
		ShortName:   ytProject.ShortName,
		Description: ytProject.Description,
	}, nil
}