package api

import (
	"strings"
)

// Fields helps build the YouTrack fields parameter.
type Fields struct {
	parts []string
}

// NewFields creates a new Fields builder.
func NewFields(parts ...string) *Fields {
	return &Fields{parts: parts}
}

// Add adds top-level fields.
func (f *Fields) Add(parts ...string) *Fields {
	f.parts = append(f.parts, parts...)
	return f
}

// Nested adds a nested field expression like project(name).
func (f *Fields) Nested(field string, nested ...string) *Fields {
	f.parts = append(f.parts, field+"("+strings.Join(nested, ",")+")")
	return f
}

// String returns the fields parameter value.
func (f *Fields) String() string {
	return strings.Join(f.parts, ",")
}

// IssueList returns default fields for listing issues.
func IssueList() *Fields {
	return NewFields(
		"id",
		"idReadable",
		"summary",
		"created",
		"updated",
		"resolved",
		"commentsCount",
		"reporter(id,login,name,fullName)",
		"project(id,name,shortName)",
		"customFields(id,name,value(id,name,login,fullName,presentation,minutes))",
		"tags(id,name)",
	)
}

// IssueDetail returns default fields for issue detail.
func IssueDetail() *Fields {
	return NewFields(
		"id",
		"idReadable",
		"summary",
		"description",
		"wikifiedDescription",
		"created",
		"updated",
		"resolved",
		"reporter(id,login,name,fullName)",
		"updater(id,login,name,fullName)",
		"project(id,name,shortName)",
		"customFields(id,name,value(id,name,login,fullName,presentation,minutes))",
		"tags(id,name)",
		"votes",
	)
}

// IssueDetailWithComments returns fields for issue detail including comments.
func IssueDetailWithComments() *Fields {
	return IssueDetail().Add(
		"comments(id,text,created,updated,author(id,login,name,fullName))",
	)
}

// ProjectList returns default fields for listing projects.
func ProjectList() *Fields {
	return NewFields("id", "name", "shortName", "description")
}

// UserDetail returns default fields for user detail.
func UserDetail() *Fields {
	return NewFields("id", "login", "name", "fullName", "email", "avatarUrl")
}
