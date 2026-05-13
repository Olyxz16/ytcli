package youtrack

import (
	"github.com/Olyxz16/tkt/internal/model"
)

// toIssue maps a YouTrack API response to a domain model Issue.
func toIssue(yt *ytIssue) *model.Issue {
	if yt == nil {
		return nil
	}
	issue := &model.Issue{
		ID:          yt.IDReadable,
		DatabaseID:  yt.ID,
		Summary:     yt.Summary,
		Description: yt.Description,
		State:       extractState(yt),
		Priority:    extractPriority(yt),
		Created:     yt.Created,
		Updated:     yt.Updated,
		Resolved:    yt.Resolved,
	}
	if yt.Project != nil {
		issue.Project = &model.Project{
			ID:          yt.Project.ID,
			Name:        yt.Project.Name,
			ShortName:   yt.Project.ShortName,
			Description: yt.Project.Description,
		}
	}
	if yt.Reporter != nil {
		issue.Reporter = &model.User{
			ID:        yt.Reporter.ID,
			Login:     yt.Reporter.Login,
			Name:       yt.Reporter.Name,
			FullName:   yt.Reporter.FullName,
			Email:     yt.Reporter.Email,
			AvatarURL: yt.Reporter.AvatarURL,
		}
	}
	assignee := extractAssignee(yt)
	if assignee != nil {
		issue.Assignee = toUser(assignee)
	}
	for _, t := range yt.Tags {
		issue.Tags = append(issue.Tags, model.Tag{ID: t.ID, Name: t.Name})
	}
	for _, c := range yt.Comments {
		comment := model.Comment{
			ID:      c.ID,
			Text:    c.Text,
			Created: c.Created,
			Updated: c.Updated,
		}
		if c.Author != nil {
			comment.Author = &model.User{
				ID:        c.Author.ID,
				Login:     c.Author.Login,
				Name:       c.Author.Name,
				FullName:   c.Author.FullName,
				Email:     c.Author.Email,
				AvatarURL: c.Author.AvatarURL,
			}
		}
		issue.Comments = append(issue.Comments, comment)
	}
	for _, l := range yt.Links {
		link := model.IssueLink{
			ID:        l.ID,
			LinkType:  l.LinkType,
			Direction: l.Direction,
		}
		for _, li := range l.Issues {
			link.Issues = append(link.Issues, *toIssue(&li))
		}
		issue.Links = append(issue.Links, link)
	}
	return issue
}

// toIssues maps a slice of YouTrack issues to domain model Issues.
func toIssues(ytIssues []ytIssue) []model.Issue {
	issues := make([]model.Issue, len(ytIssues))
	for i := range ytIssues {
		issues[i] = *toIssue(&ytIssues[i])
	}
	return issues
}

// extractState extracts the State custom field from a YouTrack issue.
func extractState(yt *ytIssue) string {
	if yt.Resolved != nil {
		return "Resolved"
	}
	for _, cf := range yt.CustomFields {
		if cf.Name == "State" {
			return cf.stringValue()
		}
	}
	return ""
}

// extractPriority extracts the Priority custom field from a YouTrack issue.
func extractPriority(yt *ytIssue) string {
	for _, cf := range yt.CustomFields {
		if cf.Name == "Priority" {
			return cf.stringValue()
		}
	}
	return ""
}

// extractAssignee extracts the Assignee custom field from a YouTrack issue.
func extractAssignee(yt *ytIssue) *ytUser {
	for _, cf := range yt.CustomFields {
		if cf.Name == "Assignee" {
			if u, ok := cf.Value.(ytUser); ok {
				return &u
			}
		}
	}
	return nil
}

// toUser maps a YouTrack user to a domain model User.
func toUser(yt *ytUser) *model.User {
	if yt == nil {
		return nil
	}
	return &model.User{
		ID:        yt.ID,
		Login:     yt.Login,
		Name:      yt.Name,
		FullName:   yt.FullName,
		Email:     yt.Email,
		AvatarURL: yt.AvatarURL,
	}
}

// toComment maps a YouTrack comment to a domain model Comment.
func toComment(yt *ytComment) *model.Comment {
	if yt == nil {
		return nil
	}
	c := &model.Comment{
		ID:      yt.ID,
		Text:    yt.Text,
		Created: yt.Created,
		Updated: yt.Updated,
	}
	if yt.Author != nil {
		c.Author = toUser(yt.Author)
	}
	return c
}

// toArticle maps a YouTrack article to a domain model Article.
func toArticle(yt *ytArticle) *model.Article {
	if yt == nil {
		return nil
	}
	a := &model.Article{
		ID:              yt.ID,
		IDReadable:      yt.IDReadable,
		Summary:         yt.Summary,
		Content:         yt.Content,
		Created:         yt.Created,
		Updated:         yt.Updated,
		Ordinal:         yt.Ordinal,
		CommentsCount:   yt.CommentsCount,
	}
	if yt.Project != nil {
		a.Project = &model.Project{
			ID:          yt.Project.ID,
			Name:        yt.Project.Name,
			ShortName:   yt.Project.ShortName,
			Description: yt.Project.Description,
		}
	}
	if yt.Reporter != nil {
		a.Reporter = toUser(yt.Reporter)
	}
	if yt.ParentArticle != nil {
		a.ParentArticle = &model.Article{
			ID:          yt.ParentArticle.ID,
			IDReadable:  yt.ParentArticle.IDReadable,
			Summary:     yt.ParentArticle.Summary,
		}
	}
	for _, t := range yt.Tags {
		a.Tags = append(a.Tags, model.Tag{ID: t.ID, Name: t.Name})
	}
	for _, c := range yt.Comments {
		a.Comments = append(a.Comments, *toComment(&c))
	}
	return a
}

// toArticles maps a slice of YouTrack articles to domain model Articles.
func toArticles(ytArticles []ytArticle) []model.Article {
	articles := make([]model.Article, len(ytArticles))
	for i := range ytArticles {
		articles[i] = *toArticle(&ytArticles[i])
	}
	return articles
}

// toWorkItem maps a YouTrack work item to a domain model WorkItem.
func toWorkItem(yt *ytWorkItem) *model.WorkItem {
	if yt == nil {
		return nil
	}
	wi := &model.WorkItem{
		ID:      yt.ID,
		Created: yt.Created,
		Date:    yt.Date,
		Text:    yt.Text,
	}
	if yt.Author != nil {
		wi.Author = toUser(yt.Author)
	}
	if yt.Duration != nil {
		wi.Duration = &model.Duration{
			Minutes:      yt.Duration.Minutes,
			Presentation: yt.Duration.Presentation,
		}
	}
	if yt.Type != nil {
		wi.Type = &model.WorkItemType{
			ID:   yt.Type.ID,
			Name: yt.Type.Name,
		}
	}
	return wi
}

// toCommandResult maps a YouTrack command result to a domain model.
func toCommandResult(yt *ytCommandResult) *model.CommandResult {
	if yt == nil {
		return nil
	}
	cr := &model.CommandResult{
		Query: yt.Query,
	}
	for i := range yt.Issues {
		cr.Issues = append(cr.Issues, *toIssue(&yt.Issues[i]))
	}
	return cr
}