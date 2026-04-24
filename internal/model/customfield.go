package model

import (
	"encoding/json"
	"fmt"
)

// CustomField represents a unified custom field on an issue.
type CustomField struct {
	ID    string      `json:"id"`
	Name  string      `json:"name"`
	Value interface{} `json:"value"`
	Type  string      `json:"-"` // derived from $type
}

// customFieldRaw is used for intermediate unmarshaling.
type customFieldRaw struct {
	ID      json.RawMessage `json:"id"`
	Name    string          `json:"name"`
	Value   json.RawMessage `json:"value"`
	DollarType string       `json:"$type"`
}

// UnmarshalJSON implements polymorphic custom field unmarshaling.
func (cf *CustomField) UnmarshalJSON(data []byte) error {
	var raw customFieldRaw
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
		var v BundleElement
		if err := json.Unmarshal(raw.Value, &v); err == nil {
			cf.Value = v
		}
	case "MultiEnumIssueCustomField", "MultiBuildIssueCustomField":
		var v []BundleElement
		if err := json.Unmarshal(raw.Value, &v); err == nil {
			cf.Value = v
		}
	case "SingleUserIssueCustomField":
		var v User
		if err := json.Unmarshal(raw.Value, &v); err == nil {
			cf.Value = v
		}
	case "MultiUserIssueCustomField":
		var v []User
		if err := json.Unmarshal(raw.Value, &v); err == nil {
			cf.Value = v
		}
	case "SingleOwnedIssueCustomField":
		var v OwnedField
		if err := json.Unmarshal(raw.Value, &v); err == nil {
			cf.Value = v
		}
	case "SingleVersionIssueCustomField":
		var v Version
		if err := json.Unmarshal(raw.Value, &v); err == nil {
			cf.Value = v
		}
	case "MultiVersionIssueCustomField":
		var v []Version
		if err := json.Unmarshal(raw.Value, &v); err == nil {
			cf.Value = v
		}
	case "DateIssueCustomField":
		var v int64
		if err := json.Unmarshal(raw.Value, &v); err == nil {
			cf.Value = v
		}
	case "PeriodIssueCustomField":
		var v Duration
		if err := json.Unmarshal(raw.Value, &v); err == nil {
			cf.Value = v
		}
	case "TextIssueCustomField":
		var v string
		if err := json.Unmarshal(raw.Value, &v); err == nil {
			cf.Value = v
		}
	case "SimpleIssueCustomField":
		// Try string first, then int
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
		// Fallback: raw string
		cf.Value = string(raw.Value)
	}
	return nil
}

// StringValue returns a string representation of the custom field value.
func (cf CustomField) StringValue() string {
	if cf.Value == nil {
		return ""
	}
	switch v := cf.Value.(type) {
	case BundleElement:
		return v.Name
	case []BundleElement:
		var names []string
		for _, e := range v {
			names = append(names, e.Name)
		}
		return joinStrings(names, ", ")
	case User:
		return v.DisplayName()
	case []User:
		var names []string
		for _, u := range v {
			names = append(names, u.DisplayName())
		}
		return joinStrings(names, ", ")
	case OwnedField:
		return v.Name
	case Version:
		return v.Name
	case []Version:
		var names []string
		for _, e := range v {
			names = append(names, e.Name)
		}
		return joinStrings(names, ", ")
	case Duration:
		return v.Presentation
	case string:
		return v
	case int64:
		return fmt.Sprintf("%d", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// BundleElement represents an enum/build bundle value.
type BundleElement struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// OwnedField represents an owned field value.
type OwnedField struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Version represents a version field value.
type Version struct {
	ID   string `json:"id"`
	Name string `json:"name"`
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
