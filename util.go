package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func atomicWriteFile(path string, data []byte, mode os.FileMode) error {
	file, err := os.CreateTemp(filepath.Dir(path), ".hermes-dock-write-*")
	if err != nil {
		return err
	}
	tmp := file.Name()
	defer os.Remove(tmp)

	if err := file.Chmod(mode); err != nil {
		file.Close()
		return err
	}
	if _, err := file.Write(data); err != nil {
		file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

type redactionPattern struct {
	regex       *regexp.Regexp
	replacement string
}

var redactionPatterns = []redactionPattern{
	{regexp.MustCompile(`(?i)(token|secret|password|api[_-]?key|authorization)(["'\s:=]+)([^"'\s,}]+)`), `$1$2<redacted>`},
	{regexp.MustCompile(`(?i)(Bearer\s+)[A-Za-z0-9._~+/=-]+`), `$1<redacted>`},
	{regexp.MustCompile(`(?i)((?:https?|socks5?)://[^:/@\s]+:)[^@\s]+(@)`), `$1<redacted>$2`},
}

func redact(input string) string {
	output := input
	for _, pattern := range redactionPatterns {
		output = pattern.regex.ReplaceAllString(output, pattern.replacement)
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

func migrationApplied(records []MigrationRecord, id string) bool {
	for _, record := range records {
		if record.ID == id {
			return true
		}
	}
	return false
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
