package model

import (
	"encoding/json"
	"testing"
)

func TestCustomFieldUnmarshal(t *testing.T) {
	input := `{
		"id": "92-1",
		"name": "Priority",
		"value": {"name": "Major", "id": "67-2"},
		"$type": "SingleEnumIssueCustomField"
	}`

	var cf CustomField
	if err := json.Unmarshal([]byte(input), &cf); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if cf.Name != "Priority" {
		t.Errorf("name = %q, want Priority", cf.Name)
	}
	if cf.StringValue() != "Major" {
		t.Errorf("value = %q, want Major", cf.StringValue())
	}
}

func TestCustomFieldUnmarshalUser(t *testing.T) {
	input := `{
		"name": "Assignee",
		"value": {"login": "john.doe", "fullName": "John Doe"},
		"$type": "SingleUserIssueCustomField"
	}`

	var cf CustomField
	if err := json.Unmarshal([]byte(input), &cf); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if cf.StringValue() != "John Doe" {
		t.Errorf("value = %q, want John Doe", cf.StringValue())
	}
}

func TestCustomFieldUnmarshalMultiEnum(t *testing.T) {
	input := `{
		"name": "Tags",
		"value": [{"name": "bug"}, {"name": "urgent"}],
		"$type": "MultiEnumIssueCustomField"
	}`

	var cf CustomField
	if err := json.Unmarshal([]byte(input), &cf); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if cf.StringValue() != "bug, urgent" {
		t.Errorf("value = %q, want bug, urgent", cf.StringValue())
	}
}
