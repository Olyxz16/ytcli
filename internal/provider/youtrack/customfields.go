package youtrack

// marshalCustomFields converts custom fields to API payload format with $type.
func marshalCustomFields(fields []ytCustomField) []map[string]interface{} {
	result := make([]map[string]interface{}, len(fields))
	for i, cf := range fields {
		m := map[string]interface{}{
			"name": cf.Name,
		}
		if cf.ID != "" {
			m["id"] = cf.ID
		}
		m["value"] = cf.Value

		switch cf.Value.(type) {
		case ytBundleElement:
			m["$type"] = "SingleEnumIssueCustomField"
		case []ytBundleElement:
			m["$type"] = "MultiEnumIssueCustomField"
		case ytUser:
			m["$type"] = "SingleUserIssueCustomField"
		case []ytUser:
			m["$type"] = "MultiUserIssueCustomField"
		case ytOwnedField:
			m["$type"] = "SingleOwnedIssueCustomField"
		case ytVersion:
			m["$type"] = "SingleVersionIssueCustomField"
		case []ytVersion:
			m["$type"] = "MultiVersionIssueCustomField"
		case ytDuration:
			m["$type"] = "PeriodIssueCustomField"
		case string:
			m["$type"] = "TextIssueCustomField"
		case int64:
			m["$type"] = "DateIssueCustomField"
		default:
			if cf.Type != "" {
				m["$type"] = cf.Type
			}
		}
		result[i] = m
	}
	return result
}