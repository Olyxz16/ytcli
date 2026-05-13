package model

// Issue represents a project issue in the domain model.
type Issue struct {
	ID          string        `json:"id"`
	DatabaseID  string        `json:"databaseId,omitempty"`
	Summary     string        `json:"summary"`
	Description string        `json:"description"`
	State       string        `json:"state"`
	Priority    string        `json:"priority"`
	Project     *Project      `json:"project,omitempty"`
	Reporter    *User         `json:"reporter,omitempty"`
	Assignee    *User         `json:"assignee,omitempty"`
	Created     int64         `json:"created"`
	Updated     int64         `json:"updated"`
	Resolved    *int64        `json:"resolved,omitempty"`
	Tags        []Tag         `json:"tags,omitempty"`
	Comments    []Comment     `json:"comments,omitempty"`
	Links       []IssueLink   `json:"links,omitempty"`
}

// Project represents a project.
type Project struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	ShortName   string `json:"shortName"`
	Description string `json:"description"`
}

// User represents a user.
type User struct {
	ID        string `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	FullName   string `json:"fullName"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatarUrl"`
}

// DisplayName returns the best human-readable name for the user.
func (u *User) DisplayName() string {
	if u.FullName != "" {
		return u.FullName
	}
	if u.Name != "" {
		return u.Name
	}
	return u.Login
}

// Comment represents a comment on an issue.
type Comment struct {
	ID      string `json:"id"`
	Text    string `json:"text"`
	Author  *User  `json:"author"`
	Created int64  `json:"created"`
	Updated int64  `json:"updated"`
}

// Tag represents an issue tag.
type Tag struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// IssueLink represents a link between issues.
type IssueLink struct {
	ID        string  `json:"id"`
	LinkType  string  `json:"linkType"`
	Direction string  `json:"direction"`
	Issues    []Issue `json:"issues"`
}

// WorkItem represents a logged work item.
type WorkItem struct {
	ID       string        `json:"id"`
	Author   *User         `json:"author"`
	Created  int64         `json:"created"`
	Date     int64         `json:"date"`
	Duration *Duration     `json:"duration"`
	Text     string        `json:"text"`
	Type     *WorkItemType `json:"type"`
}

// Duration represents a time duration.
type Duration struct {
	Minutes      int    `json:"minutes"`
	Presentation string `json:"presentation"`
}

// WorkItemType represents a type of work item.
type WorkItemType struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// CommandResult represents the result of applying a remote command.
type CommandResult struct {
	Query  string  `json:"query"`
	Issues []Issue `json:"issues"`
}

// Article represents a knowledge base article.
type Article struct {
	ID            string    `json:"id"`
	IDReadable   string    `json:"idReadable"`
	Summary       string    `json:"summary"`
	Content       string    `json:"content"`
	Project       *Project  `json:"project,omitempty"`
	Reporter      *User     `json:"reporter,omitempty"`
	Created       int64     `json:"created"`
	Updated       int64     `json:"updated"`
	Ordinal       int       `json:"ordinal"`
	ParentArticle *Article  `json:"parentArticle,omitempty"`
	Tags          []Tag     `json:"tags,omitempty"`
	CommentsCount int       `json:"commentsCount"`
	Comments      []Comment `json:"comments,omitempty"`
}