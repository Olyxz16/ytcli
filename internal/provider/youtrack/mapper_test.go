package youtrack

import (
	"encoding/json"
	"testing"

	"github.com/Olyxz16/tkt/internal/provider"
)

func TestToIssueNil(t *testing.T) {
	if result := toIssue(nil); result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestToIssueBasic(t *testing.T) {
	yt := &ytIssue{
		ID:          "2-42",
		IDReadable:  "PROJ-1",
		Summary:     "Test issue",
		Description: "Body text",
		Created:     1000,
		Updated:     2000,
	}
	issue := toIssue(yt)
	if issue.ID != "PROJ-1" {
		t.Errorf("ID = %q, want PROJ-1", issue.ID)
	}
	if issue.DatabaseID != "2-42" {
		t.Errorf("DatabaseID = %q, want 2-42", issue.DatabaseID)
	}
	if issue.Summary != "Test issue" {
		t.Errorf("Summary = %q", issue.Summary)
	}
	if issue.Description != "Body text" {
		t.Errorf("Description = %q", issue.Description)
	}
	if issue.Created != 1000 {
		t.Errorf("Created = %d, want 1000", issue.Created)
	}
	if issue.Updated != 2000 {
		t.Errorf("Updated = %d, want 2000", issue.Updated)
	}
}

func TestToIssueWithProject(t *testing.T) {
	yt := &ytIssue{
		ID:         "2-42",
		IDReadable: "PROJ-1",
		Summary:    "Test",
		Project: &ytProject{
			ID:          "proj-1",
			Name:        "My Project",
			ShortName:   "PROJ",
			Description: "A project",
		},
	}
	issue := toIssue(yt)
	if issue.Project == nil {
		t.Fatal("expected project")
	}
	if issue.Project.ID != "proj-1" {
		t.Errorf("Project.ID = %q", issue.Project.ID)
	}
	if issue.Project.ShortName != "PROJ" {
		t.Errorf("Project.ShortName = %q", issue.Project.ShortName)
	}
}

func TestToIssueWithReporter(t *testing.T) {
	yt := &ytIssue{
		ID:         "2-42",
		IDReadable: "PROJ-1",
		Summary:    "Test",
		Reporter: &ytUser{
			ID:       "user-1",
			Login:    "john",
			Name:     "John",
			FullName: "John Doe",
			Email:    "john@example.com",
		},
	}
	issue := toIssue(yt)
	if issue.Reporter == nil {
		t.Fatal("expected reporter")
	}
	if issue.Reporter.Login != "john" {
		t.Errorf("Reporter.Login = %q", issue.Reporter.Login)
	}
}

func TestToIssueWithAssignee(t *testing.T) {
	yt := &ytIssue{
		ID:         "2-42",
		IDReadable: "PROJ-1",
		Summary:    "Test",
		CustomFields: []ytCustomField{
			{
				Name:  "Assignee",
				Value: ytUser{ID: "user-2", Login: "jane", FullName: "Jane Smith"},
			},
		},
	}
	issue := toIssue(yt)
	if issue.Assignee == nil {
		t.Fatal("expected assignee")
	}
	if issue.Assignee.Login != "jane" {
		t.Errorf("Assignee.Login = %q", issue.Assignee.Login)
	}
}

func TestToIssueWithState(t *testing.T) {
	yt := &ytIssue{
		ID:         "2-42",
		IDReadable: "PROJ-1",
		Summary:    "Test",
		CustomFields: []ytCustomField{
			{
				Name:  "State",
				Value: ytBundleElement{ID: "state-1", Name: "In Progress"},
			},
		},
	}
	issue := toIssue(yt)
	if issue.State != "In Progress" {
		t.Errorf("State = %q, want 'In Progress'", issue.State)
	}
}

func TestToIssueStateResolved(t *testing.T) {
	resolved := int64(3000)
	yt := &ytIssue{
		ID:         "2-42",
		IDReadable: "PROJ-1",
		Summary:    "Test",
		Resolved:   &resolved,
	}
	issue := toIssue(yt)
	if issue.State != "Resolved" {
		t.Errorf("State = %q, want Resolved", issue.State)
	}
	if issue.Resolved == nil || *issue.Resolved != 3000 {
		t.Errorf("Resolved = %v, want 3000", issue.Resolved)
	}
}

func TestToIssueWithPriority(t *testing.T) {
	yt := &ytIssue{
		ID:         "2-42",
		IDReadable: "PROJ-1",
		Summary:    "Test",
		CustomFields: []ytCustomField{
			{
				Name:  "Priority",
				Value: ytBundleElement{ID: "prio-1", Name: "Critical"},
			},
		},
	}
	issue := toIssue(yt)
	if issue.Priority != "Critical" {
		t.Errorf("Priority = %q, want Critical", issue.Priority)
	}
}

func TestToIssueWithTags(t *testing.T) {
	yt := &ytIssue{
		ID:         "2-42",
		IDReadable: "PROJ-1",
		Summary:    "Test",
		Tags: []ytTag{
			{ID: "tag-1", Name: "bug"},
			{ID: "tag-2", Name: "feature"},
		},
	}
	issue := toIssue(yt)
	if len(issue.Tags) != 2 {
		t.Fatalf("Tags len = %d, want 2", len(issue.Tags))
	}
	if issue.Tags[0].Name != "bug" {
		t.Errorf("Tags[0].Name = %q", issue.Tags[0].Name)
	}
	if issue.Tags[1].ID != "tag-2" {
		t.Errorf("Tags[1].ID = %q", issue.Tags[1].ID)
	}
}

func TestToIssueWithComments(t *testing.T) {
	yt := &ytIssue{
		ID:         "2-42",
		IDReadable: "PROJ-1",
		Summary:    "Test",
		Comments: []ytComment{
			{
				ID:      "c-1",
				Text:    "Hello",
				Author:  &ytUser{ID: "u-1", Login: "john"},
				Created: 1000,
				Updated: 2000,
			},
		},
	}
	issue := toIssue(yt)
	if len(issue.Comments) != 1 {
		t.Fatalf("Comments len = %d, want 1", len(issue.Comments))
	}
	if issue.Comments[0].Text != "Hello" {
		t.Errorf("Comment.Text = %q", issue.Comments[0].Text)
	}
	if issue.Comments[0].Author == nil || issue.Comments[0].Author.Login != "john" {
		t.Errorf("Comment.Author.Login = %v", issue.Comments[0].Author)
	}
}

func TestToIssues(t *testing.T) {
	ytIssues := []ytIssue{
		{ID: "2-1", IDReadable: "PROJ-1", Summary: "First"},
		{ID: "2-2", IDReadable: "PROJ-2", Summary: "Second"},
	}
	issues := toIssues(ytIssues)
	if len(issues) != 2 {
		t.Fatalf("len = %d, want 2", len(issues))
	}
	if issues[0].ID != "PROJ-1" {
		t.Errorf("issues[0].ID = %q", issues[0].ID)
	}
	if issues[1].Summary != "Second" {
		t.Errorf("issues[1].Summary = %q", issues[1].Summary)
	}
}

func TestToUserNil(t *testing.T) {
	if result := toUser(nil); result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestToUser(t *testing.T) {
	yt := &ytUser{
		ID:        "u-1",
		Login:     "john",
		Name:      "John",
		FullName:  "John Doe",
		Email:     "john@example.com",
		AvatarURL: "https://example.com/avatar.png",
	}
	user := toUser(yt)
	if user.ID != "u-1" {
		t.Errorf("ID = %q", user.ID)
	}
	if user.Login != "john" {
		t.Errorf("Login = %q", user.Login)
	}
	if user.AvatarURL != "https://example.com/avatar.png" {
		t.Errorf("AvatarURL = %q", user.AvatarURL)
	}
}

func TestToCommentNil(t *testing.T) {
	if result := toComment(nil); result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestToComment(t *testing.T) {
	yt := &ytComment{
		ID:      "c-1",
		Text:    "Hello world",
		Created: 1000,
		Updated: 2000,
	}
	c := toComment(yt)
	if c.ID != "c-1" {
		t.Errorf("ID = %q", c.ID)
	}
	if c.Text != "Hello world" {
		t.Errorf("Text = %q", c.Text)
	}
}

func TestToWorkItemNil(t *testing.T) {
	if result := toWorkItem(nil); result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestToWorkItem(t *testing.T) {
	minutes := 60
	yt := &ytWorkItem{
		ID:      "w-1",
		Text:    "Development",
		Created: 1000,
		Date:    2000,
		Author:  &ytUser{ID: "u-1", Login: "john"},
		Duration: &ytDuration{
			Minutes:      minutes,
			Presentation: "1h",
		},
		Type: &ytWorkItemType{ID: "wt-1", Name: "Development"},
	}
	wi := toWorkItem(yt)
	if wi.ID != "w-1" {
		t.Errorf("ID = %q", wi.ID)
	}
	if wi.Text != "Development" {
		t.Errorf("Text = %q", wi.Text)
	}
	if wi.Duration == nil || wi.Duration.Minutes != 60 {
		t.Errorf("Duration.Minutes = %v", wi.Duration)
	}
	if wi.Duration.Presentation != "1h" {
		t.Errorf("Duration.Presentation = %q", wi.Duration.Presentation)
	}
	if wi.Type == nil || wi.Type.Name != "Development" {
		t.Errorf("Type = %v", wi.Type)
	}
}

func TestToArticleNil(t *testing.T) {
	if result := toArticle(nil); result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestToArticle(t *testing.T) {
	yt := &ytArticle{
		ID:            "a-1",
		IDReadable:    "ART-1",
		Summary:       "Test article",
		Content:       "Article body",
		Created:        1000,
		Updated:        2000,
		Ordinal:        1,
		CommentsCount:  3,
		Project: &ytProject{
			ID:        "proj-1",
			ShortName: "PROJ",
		},
		Tags: []ytTag{
			{ID: "tag-1", Name: "docs"},
		},
	}
	article := toArticle(yt)
	if article.ID != "a-1" {
		t.Errorf("ID = %q", article.ID)
	}
	if article.IDReadable != "ART-1" {
		t.Errorf("IDReadable = %q", article.IDReadable)
	}
	if article.CommentsCount != 3 {
		t.Errorf("CommentsCount = %d", article.CommentsCount)
	}
	if article.Project == nil || article.Project.ShortName != "PROJ" {
		t.Errorf("Project = %v", article.Project)
	}
	if len(article.Tags) != 1 || article.Tags[0].Name != "docs" {
		t.Errorf("Tags = %v", article.Tags)
	}
}

func TestToArticleWithParent(t *testing.T) {
	yt := &ytArticle{
		ID:          "a-2",
		IDReadable:  "ART-2",
		Summary:     "Child article",
		ParentArticle: &ytArticle{
			ID:          "a-1",
			IDReadable:  "ART-1",
			Summary:     "Parent article",
		},
	}
	article := toArticle(yt)
	if article.ParentArticle == nil {
		t.Fatal("expected parent article")
	}
	if article.ParentArticle.ID != "a-1" {
		t.Errorf("ParentArticle.ID = %q", article.ParentArticle.ID)
	}
	if article.ParentArticle.Summary != "Parent article" {
		t.Errorf("ParentArticle.Summary = %q", article.ParentArticle.Summary)
	}
}

func TestToCommandResultNil(t *testing.T) {
	if result := toCommandResult(nil); result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestToCommandResult(t *testing.T) {
	yt := &ytCommandResult{
		Query: "State: Done",
		Issues: []ytIssue{
			{ID: "2-42", IDReadable: "PROJ-1", Summary: "Test"},
		},
	}
	cr := toCommandResult(yt)
	if cr.Query != "State: Done" {
		t.Errorf("Query = %q", cr.Query)
	}
	if len(cr.Issues) != 1 {
		t.Fatalf("Issues len = %d", len(cr.Issues))
	}
	if cr.Issues[0].ID != "PROJ-1" {
		t.Errorf("Issues[0].ID = %q", cr.Issues[0].ID)
	}
}

func TestExtractStateFromCustomField(t *testing.T) {
	yt := &ytIssue{
		ID:          "2-42",
		IDReadable:  "PROJ-1",
		Summary:     "Test",
		CustomFields: []ytCustomField{
			{
				Name:  "State",
				Value: ytBundleElement{ID: "s-1", Name: "Open"},
			},
		},
	}
	state := extractState(yt)
	if state != "Open" {
		t.Errorf("state = %q, want Open", state)
	}
}

func TestExtractPriorityFromCustomField(t *testing.T) {
	yt := &ytIssue{
		ID:          "2-42",
		IDReadable:  "PROJ-1",
		Summary:     "Test",
		CustomFields: []ytCustomField{
			{
				Name:  "Priority",
				Value: ytBundleElement{ID: "p-1", Name: "High"},
			},
		},
	}
	priority := extractPriority(yt)
	if priority != "High" {
		t.Errorf("priority = %q, want High", priority)
	}
}

func TestExtractAssigneeFromCustomField(t *testing.T) {
	yt := &ytIssue{
		ID:          "2-42",
		IDReadable:  "PROJ-1",
		Summary:     "Test",
		CustomFields: []ytCustomField{
			{
				Name: "Assignee",
				Value: ytUser{
					ID:       "u-1",
					Login:    "john",
					FullName: "John Doe",
				},
			},
		},
	}
	assignee := extractAssignee(yt)
	if assignee == nil {
		t.Fatal("expected assignee")
	}
	if assignee.Login != "john" {
		t.Errorf("assignee.Login = %q", assignee.Login)
	}
}

func TestExtractAssigneeNil(t *testing.T) {
	yt := &ytIssue{
		ID:         "2-42",
		IDReadable: "PROJ-1",
		Summary:    "Test",
	}
	assignee := extractAssignee(yt)
	if assignee != nil {
		t.Errorf("expected nil assignee, got %v", assignee)
	}
}

func TestExtractStateNoCustomFields(t *testing.T) {
	yt := &ytIssue{
		ID:         "2-42",
		IDReadable: "PROJ-1",
		Summary:    "Test",
	}
	state := extractState(yt)
	if state != "" {
		t.Errorf("expected empty state, got %q", state)
	}
}

func TestExtractPriorityNoCustomFields(t *testing.T) {
	yt := &ytIssue{
		ID:         "2-42",
		IDReadable: "PROJ-1",
		Summary:    "Test",
	}
	priority := extractPriority(yt)
	if priority != "" {
		t.Errorf("expected empty priority, got %q", priority)
	}
}

func TestYtCustomFieldUnmarshalSingleEnum(t *testing.T) {
	data := `{
		"id": "cf-1",
		"name": "State",
		"value": {"id": "s-1", "name": "Open"},
		"$type": "StateIssueCustomField"
	}`
	var cf ytCustomField
	if err := json.Unmarshal([]byte(data), &cf); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if cf.Name != "State" {
		t.Errorf("Name = %q", cf.Name)
	}
	if cf.Type != "StateIssueCustomField" {
		t.Errorf("Type = %q", cf.Type)
	}
	be, ok := cf.Value.(ytBundleElement)
	if !ok {
		t.Fatalf("Value type = %T, want ytBundleElement", cf.Value)
	}
	if be.Name != "Open" {
		t.Errorf("Value.Name = %q", be.Name)
	}
	str := cf.stringValue()
	if str != "Open" {
		t.Errorf("stringValue = %q, want Open", str)
	}
}

func TestYtCustomFieldUnmarshalMultiEnum(t *testing.T) {
	data := `{
		"id": "cf-2",
		"name": "Platforms",
		"value": [{"id": "p-1", "name": "Linux"}, {"id": "p-2", "name": "macOS"}],
		"$type": "MultiEnumIssueCustomField"
	}`
	var cf ytCustomField
	if err := json.Unmarshal([]byte(data), &cf); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	values, ok := cf.Value.([]ytBundleElement)
	if !ok {
		t.Fatalf("Value type = %T, want []ytBundleElement", cf.Value)
	}
	if len(values) != 2 {
		t.Errorf("len(values) = %d, want 2", len(values))
	}
	str := cf.stringValue()
	if str != "Linux, macOS" {
		t.Errorf("stringValue = %q, want 'Linux, macOS'", str)
	}
}

func TestYtCustomFieldUnmarshalUser(t *testing.T) {
	data := `{
		"id": "cf-3",
		"name": "Assignee",
		"value": {"id": "u-1", "login": "john", "name": "John", "fullName": "John Doe"},
		"$type": "SingleUserIssueCustomField"
	}`
	var cf ytCustomField
	if err := json.Unmarshal([]byte(data), &cf); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	user, ok := cf.Value.(ytUser)
	if !ok {
		t.Fatalf("Value type = %T, want ytUser", cf.Value)
	}
	if user.Login != "john" {
		t.Errorf("Value.Login = %q", user.Login)
	}
	str := cf.stringValue()
	if str != "John Doe" {
		t.Errorf("stringValue = %q, want 'John Doe'", str)
	}
}

func TestYtCustomFieldUnmarshalNullValue(t *testing.T) {
	data := `{
		"id": "cf-4",
		"name": "Assignee",
		"value": null,
		"$type": "SingleUserIssueCustomField"
	}`
	var cf ytCustomField
	if err := json.Unmarshal([]byte(data), &cf); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if cf.Value != nil {
		t.Errorf("Value = %v, want nil", cf.Value)
	}
	str := cf.stringValue()
	if str != "" {
		t.Errorf("stringValue = %q, want empty", str)
	}
}

func TestYtCustomFieldUnmarshalString(t *testing.T) {
	data := `{
		"id": "cf-5",
		"name": "Environment",
		"value": "production",
		"$type": "TextIssueCustomField"
	}`
	var cf ytCustomField
	if err := json.Unmarshal([]byte(data), &cf); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	str, ok := cf.Value.(string)
	if !ok {
		t.Fatalf("Value type = %T, want string", cf.Value)
	}
	if str != "production" {
		t.Errorf("Value = %q, want production", str)
	}
}

func TestYtCustomFieldUnmarshalDuration(t *testing.T) {
	data := `{
		"id": "cf-6",
		"name": "Estimation",
		"value": {"minutes": 120, "presentation": "2h"},
		"$type": "PeriodIssueCustomField"
	}`
	var cf ytCustomField
	if err := json.Unmarshal([]byte(data), &cf); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	dur, ok := cf.Value.(ytDuration)
	if !ok {
		t.Fatalf("Value type = %T, want ytDuration", cf.Value)
	}
	if dur.Minutes != 120 {
		t.Errorf("Minutes = %d, want 120", dur.Minutes)
	}
	str := cf.stringValue()
	if str != "2h" {
		t.Errorf("stringValue = %q, want 2h", str)
	}
}

func TestBuildYouTrackQuery(t *testing.T) {
	tests := []struct {
		name  string
		query provider.Query
		want  string
	}{
		{
			name:  "empty query",
			query: provider.Query{},
			want:  "",
		},
		{
			name:  "project only",
			query: provider.Query{Project: "PROJ"},
			want:  "project: {PROJ}",
		},
		{
			name:  "project and assignee",
			query: provider.Query{Project: "PROJ", Assignee: "me"},
			want:  "project: {PROJ} for: me",
		},
		{
			name:  "project and state",
			query: provider.Query{Project: "PROJ", State: "Open"},
			want:  "project: {PROJ} State: {Open}",
		},
		{
			name:  "text search",
			query: provider.Query{Text: "search term"},
			want:  "search term",
		},
		{
			name:  "all fields",
			query: provider.Query{Project: "PROJ", Assignee: "me", State: "Open", Text: "urgent"},
			want:  "project: {PROJ} for: me State: {Open} urgent",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildYouTrackQuery(tt.query)
			if got != tt.want {
				t.Errorf("buildYouTrackQuery() = %q, want %q", got, tt.want)
			}
		})
	}
}