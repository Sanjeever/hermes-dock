package main

import (
	"encoding/json"
	"regexp"
	"strings"
)

var redactionPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(token|secret|password|api[_-]?key|authorization)(["'\s:=]+)([^"'\s,}]+)`),
	regexp.MustCompile(`(?i)(Bearer\s+)[A-Za-z0-9._~+/=-]+`),
}

func redact(input string) string {
	output := input
	for _, pattern := range redactionPatterns {
		output = pattern.ReplaceAllString(output, `$1$2<redacted>`)
	}
	return output
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func yamlQuote(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	return value
}

func appendIfMissingMigration(records []MigrationRecord, next MigrationRecord) []MigrationRecord {
	for _, record := range records {
		if record.ID == next.ID {
			return records
		}
	}
	return append(records, next)
}

func toJSONMap(value string) (map[string]interface{}, error) {
	if strings.TrimSpace(value) == "" {
		return map[string]interface{}{}, nil
	}
	var out map[string]interface{}
	if err := json.Unmarshal([]byte(value), &out); err != nil {
		return nil, err
	}
	return out, nil
}
