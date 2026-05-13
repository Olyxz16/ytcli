package youtrack

import (
	"encoding/json"
	"fmt"
)

// ytIssue is the YouTrack API response type for issues.
type ytIssue struct {
	ID          string          `json:"id"`
	IDReadable  string          `json:"idReadable"`
	Summary     string          `json:"summary"`
	Description string          `json:"description"`
	Project     *ytProject      `json:"project"`
	Reporter    *ytUser         `json:"reporter"`
	Created     int64           `json:"created"`
	Updated     int64           `json:"updated"`
	Resolved    *int64           `json:"resolved,omitempty"`
	CustomFields []ytCustomField `json:"customFields"`
	Tags        []ytTag         `json:"tags"`
	Comments    []ytComment     `json:"comments"`
	Links       []ytIssueLink   `json:"links"`
}

// ytProject is the YouTrack API response type for projects.
type ytProject struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	ShortName   string `json:"shortName"`
	Description string `json:"description"`
}

// ytUser is the YouTrack API response type for users.
type ytUser struct {
	ID        string `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	FullName  string `json:"fullName"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatarUrl"`
}

func (u *ytUser) displayName() string {
	if u == nil {
		return ""
	}
	if u.FullName != "" {
		return u.FullName
	}
	if u.Name != "" {
		return u.Name
	}
	return u.Login
}

// ytTag is the YouTrack API response type for tags.
type ytTag struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ytComment is the YouTrack API response type for comments.
type ytComment struct {
	ID      string  `json:"id"`
	Text    string  `json:"text"`
	Author  *ytUser `json:"author"`
	Created int64   `json:"created"`
	Updated int64   `json:"updated"`
}

// ytIssueLink is the YouTrack API response type for issue links.
type ytIssueLink struct {
	ID        string    `json:"id"`
	LinkType  string    `json:"linkType"`
	Direction string    `json:"direction"`
	Issues    []ytIssue `json:"issues"`
}

// ytCustomField represents a YouTrack custom field.
type ytCustomField struct {
	ID    string      `json:"id"`
	Name  string      `json:"name"`
	Value interface{} `json:"value"`
	Type  string      `json:"-"`
}

// ytCustomFieldRaw is used for intermediate unmarshaling.
type ytCustomFieldRaw struct {
	ID         json.RawMessage `json:"id"`
	Name       string          `json:"name"`
	Value      json.RawMessage `json:"value"`
	DollarType string          `json:"$type"`
}

func (cf *ytCustomField) UnmarshalJSON(data []byte) error {
	var raw ytCustomFieldRaw
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	cf.Name = raw.Name
	cf.Type = raw.DollarType

	var idStr string
	if err := json.Unmarshal(raw.ID, &idStr); err == nil {
		cf.ID = idStr
	}

	if raw.Value == nil || string(raw.Value) == "null" {
		cf.Value = nil
		return nil
	}

	switch raw.DollarType {
	case "SingleEnumIssueCustomField", "StateIssueCustomField", "SingleBuildIssueCustomField":
		var v ytBundleElement
		if err := json.Unmarshal(raw.Value, &v); err == nil {
			cf.Value = v
		}
	case "MultiEnumIssueCustomField", "MultiBuildIssueCustomField":
		var v []ytBundleElement
		if err := json.Unmarshal(raw.Value, &v); err == nil {
			cf.Value = v
		}
	case "SingleUserIssueCustomField":
		var v ytUser
		if err := json.Unmarshal(raw.Value, &v); err == nil {
			cf.Value = v
		}
	case "MultiUserIssueCustomField":
		var v []ytUser
		if err := json.Unmarshal(raw.Value, &v); err == nil {
			cf.Value = v
		}
	case "SingleOwnedIssueCustomField":
		var v ytOwnedField
		if err := json.Unmarshal(raw.Value, &v); err == nil {
			cf.Value = v
		}
	case "SingleVersionIssueCustomField":
		var v ytVersion
		if err := json.Unmarshal(raw.Value, &v); err == nil {
			cf.Value = v
		}
	case "MultiVersionIssueCustomField":
		var v []ytVersion
		if err := json.Unmarshal(raw.Value, &v); err == nil {
			cf.Value = v
		}
	case "DateIssueCustomField":
		var v int64
		if err := json.Unmarshal(raw.Value, &v); err == nil {
			cf.Value = v
		}
	case "PeriodIssueCustomField":
		var v ytDuration
		if err := json.Unmarshal(raw.Value, &v); err == nil {
			cf.Value = v
		}
	case "TextIssueCustomField":
		var v string
		if err := json.Unmarshal(raw.Value, &v); err == nil {
			cf.Value = v
		}
	case "SimpleIssueCustomField":
		var s string
		if err := json.Unmarshal(raw.Value, &s); err == nil {
			cf.Value = s
		} else {
			var i int64
			if err := json.Unmarshal(raw.Value, &i); err == nil {
				cf.Value = i
			} else {
				cf.Value = string(raw.Value)
			}
		}
	default:
		cf.Value = string(raw.Value)
	}
	return nil
}

func (cf ytCustomField) stringValue() string {
	if cf.Value == nil {
		return ""
	}
	switch v := cf.Value.(type) {
	case ytBundleElement:
		return v.Name
	case []ytBundleElement:
		var names []string
		for _, e := range v {
			names = append(names, e.Name)
		}
		return joinStrings(names, ", ")
	case ytUser:
		return v.displayName()
	case []ytUser:
		var names []string
		for _, u := range v {
			names = append(names, u.displayName())
		}
		return joinStrings(names, ", ")
	case ytOwnedField:
		return v.Name
	case ytVersion:
		return v.Name
	case []ytVersion:
		var names []string
		for _, e := range v {
			names = append(names, e.Name)
		}
		return joinStrings(names, ", ")
	case ytDuration:
		return v.Presentation
	case string:
		return v
	case int64:
		return fmt.Sprintf("%d", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// ytBundleElement represents an enum/build bundle value.
type ytBundleElement struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ytOwnedField represents an owned field value.
type ytOwnedField struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ytVersion represents a version field value.
type ytVersion struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ytDuration represents a time duration in YouTrack.
type ytDuration struct {
	Minutes      int    `json:"minutes"`
	Presentation string `json:"presentation"`
}

// ytWorkItem represents a YouTrack work item.
type ytWorkItem struct {
	ID       string        `json:"id"`
	Author   *ytUser       `json:"author"`
	Created  int64         `json:"created"`
	Date     int64         `json:"date"`
	Duration *ytDuration   `json:"duration"`
	Text     string        `json:"text"`
	Type     *ytWorkItemType `json:"type"`
}

// ytWorkItemType represents a type of work item.
type ytWorkItemType struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ytCommandResult represents the result of applying a YouTrack command.
type ytCommandResult struct {
	Query  string    `json:"query"`
	Issues []ytIssue `json:"issues"`
}

// ytSearchSuggestions holds autocomplete suggestions.
type ytSearchSuggestions struct {
	Query       string            `json:"query"`
	Suggestions []ytSuggestion    `json:"suggestions"`
}

// ytSuggestion is a single autocomplete suggestion.
type ytSuggestion struct {
	Option      string `json:"option"`
	Completion  string `json:"completionStart"`
	Description string `json:"description"`
	Prefix      string `json:"prefix"`
	Suffix      string `json:"suffix"`
}

// ytArticle represents a YouTrack knowledge base article.
type ytArticle struct {
	ID            string      `json:"id"`
	IDReadable    string      `json:"idReadable"`
	Summary       string      `json:"summary"`
	Content       string      `json:"content"`
	Project       *ytProject  `json:"project"`
	Reporter      *ytUser     `json:"reporter"`
	Created       int64       `json:"created"`
	Updated       int64       `json:"updated"`
	Ordinal       int         `json:"ordinal"`
	ParentArticle *ytArticle  `json:"parentArticle,omitempty"`
	Tags          []ytTag     `json:"tags"`
	CommentsCount int         `json:"commentsCount"`
	Comments      []ytComment `json:"comments"`
}

func joinStrings(ss []string, sep string) string {
	if len(ss) == 0 {
		return ""
	}
	result := ss[0]
	for i := 1; i < len(ss); i++ {
		result += sep + ss[i]
	}
	return result
}