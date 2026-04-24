package model

// Issue represents a YouTrack issue.
type Issue struct {
	ID               string             `json:"id"`
	IDReadable       string             `json:"idReadable"`
	Summary          string             `json:"summary"`
	Description      string             `json:"description"`
	WikifiedDescription string          `json:"wikifiedDescription"`
	Project          *Project           `json:"project"`
	Reporter         *User              `json:"reporter"`
	Updater          *User              `json:"updater"`
	Assignee         *User              `json:"-"` // derived from custom fields
	Created          int64              `json:"created"`
	Updated          int64              `json:"updated"`
	Resolved         *int64             `json:"resolved,omitempty"`
	CommentsCount    int                `json:"commentsCount"`
	CustomFields     []CustomField      `json:"customFields"`
	Tags             []Tag              `json:"tags"`
	Comments         []Comment          `json:"comments"`
	Links            []IssueLink        `json:"links"`
	Votes            int                `json:"votes"`
}

// Project represents a YouTrack project.
type Project struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	ShortName   string `json:"shortName"`
	Description string `json:"description"`
}

// User represents a YouTrack user.
type User struct {
	ID        string `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	FullName  string `json:"fullName"`
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
	ID       string `json:"id"`
	Text     string `json:"text"`
	Author   *User  `json:"author"`
	Created  int64  `json:"created"`
	Updated  int64  `json:"updated"`
}

// Tag represents an issue tag.
type Tag struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// IssueLink represents a link between issues.
type IssueLink struct {
	ID          string  `json:"id"`
	LinkType    string  `json:"linkType"`
	Direction   string  `json:"direction"`
	Issues      []Issue `json:"issues"`
}

// WorkItem represents a logged work item.
type WorkItem struct {
	ID       string `json:"id"`
	Author   *User  `json:"author"`
	Created  int64  `json:"created"`
	Date     int64  `json:"date"`
	Duration *Duration `json:"duration"`
	Text     string `json:"text"`
	Type     *WorkItemType `json:"type"`
}

// Duration represents a time duration in YouTrack.
type Duration struct {
	Minutes int `json:"minutes"`
	Presentation string `json:"presentation"`
}

// WorkItemType represents a type of work item.
type WorkItemType struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// CommandResult represents the result of applying a command.
type CommandResult struct {
	Query string  `json:"query"`
	Issues []Issue `json:"issues"`
}

// SearchSuggestions holds autocomplete suggestions.
type SearchSuggestions struct {
	Query       string        `json:"query"`
	Suggestions []Suggestion  `json:"suggestions"`
}

// Suggestion is a single autocomplete suggestion.
type Suggestion struct {
	Option string `json:"option"`
	Completion string `json:"completionStart"`
	Description string `json:"description"`
	Prefix string `json:"prefix"`
	Suffix string `json:"suffix"`
}
