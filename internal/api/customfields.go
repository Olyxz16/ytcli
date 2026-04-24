package api

import (
	"github.com/Olyxz16/ytcli/internal/model"
)

// marshalCustomFields converts model custom fields to API payload format with $type.
func marshalCustomFields(fields []model.CustomField) []map[string]interface{} {
	result := make([]map[string]interface{}, len(fields))
	for i, cf := range fields {
		m := map[string]interface{}{
			"name": cf.Name,
		}
		if cf.ID != "" {
			m["id"] = cf.ID
		}
		m["value"] = cf.Value

		// Infer $type from value type
		switch cf.Value.(type) {
		case model.BundleElement:
			m["$type"] = "SingleEnumIssueCustomField"
		case []model.BundleElement:
			m["$type"] = "MultiEnumIssueCustomField"
		case model.User:
			m["$type"] = "SingleUserIssueCustomField"
		case []model.User:
			m["$type"] = "MultiUserIssueCustomField"
		case model.OwnedField:
			m["$type"] = "SingleOwnedIssueCustomField"
		case model.Version:
			m["$type"] = "SingleVersionIssueCustomField"
		case []model.Version:
			m["$type"] = "MultiVersionIssueCustomField"
		case model.Duration:
			m["$type"] = "PeriodIssueCustomField"
		case string:
			m["$type"] = "TextIssueCustomField"
		case int64:
			m["$type"] = "DateIssueCustomField"
		default:
			// Try to use existing Type if set
			if cf.Type != "" {
				m["$type"] = cf.Type
			}
		}
		result[i] = m
	}
	return result
}
