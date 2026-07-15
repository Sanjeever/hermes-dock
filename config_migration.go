package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

const profileStreamingMigrationID = "profile-streaming-v1"

func (a *App) migrateProfileStreamingDefaults() error {
	state, err := a.readState()
	if err != nil {
		return err
	}
	if migrationApplied(state.Migrations, profileStreamingMigrationID) {
		return nil
	}

	registry, err := a.readProfileRegistry()
	if err != nil {
		return err
	}
	changedAny := false
	for _, profile := range registry.Profiles {
		path := filepath.Join(a.profileDataDir(profile.ID), "config.yaml")
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("读取 %s 的 config.yaml 失败：%w", firstNonEmpty(profile.Name, profile.ID), err)
		}
		next, changed, err := addProfileStreamingDefaults(data)
		if err != nil {
			return fmt.Errorf("迁移 %s 的 config.yaml 失败：%w", firstNonEmpty(profile.Name, profile.ID), err)
		}
		if !changed {
			continue
		}
		if err := a.backupFile(path, "before-profile-streaming-migration"); err != nil {
			return err
		}
		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		if err := atomicWriteFile(path, next, info.Mode().Perm()); err != nil {
			return err
		}
		changedAny = true
	}

	state, err = a.readState()
	if err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	state.Migrations = appendIfMissingMigration(state.Migrations, MigrationRecord{
		ID:        profileStreamingMigrationID,
		AppliedAt: now,
	})
	state.NeedsRebuild = state.NeedsRebuild || changedAny
	state.UpdatedAt = now
	return a.writeState(state)
}

func addProfileStreamingDefaults(data []byte) ([]byte, bool, error) {
	var validated map[string]interface{}
	if err := yaml.Unmarshal(data, &validated); err != nil {
		return nil, false, err
	}

	var document yaml.Node
	if err := yaml.Unmarshal(data, &document); err != nil {
		return nil, false, err
	}
	if len(document.Content) != 1 || document.Content[0].Kind != yaml.MappingNode {
		return nil, false, fmt.Errorf("顶层配置必须是 YAML mapping")
	}
	root := document.Content[0]
	changed := false

	streaming, added, err := ensureYAMLMapping(root, "streaming")
	if err != nil {
		return nil, false, err
	}
	changed = changed || added
	for _, item := range []struct {
		key   string
		tag   string
		value string
	}{
		{key: "enabled", tag: "!!bool", value: "true"},
		{key: "transport", tag: "!!str", value: "edit"},
		{key: "edit_interval", tag: "!!float", value: "0.8"},
		{key: "buffer_threshold", tag: "!!int", value: "24"},
		{key: "cursor", tag: "!!str", value: " ▉"},
	} {
		changed = ensureYAMLScalar(streaming, item.key, item.tag, item.value) || changed
	}

	display, added, err := ensureYAMLMapping(root, "display")
	if err != nil {
		return nil, false, err
	}
	changed = changed || added
	platforms, added, err := ensureYAMLMapping(display, "platforms")
	if err != nil {
		return nil, false, err
	}
	changed = changed || added

	feishu, added, err := ensureYAMLMapping(platforms, "feishu")
	if err != nil {
		return nil, false, err
	}
	changed = changed || added
	for _, item := range []struct {
		key   string
		tag   string
		value string
	}{
		{key: "streaming", tag: "!!bool", value: "true"},
		{key: "tool_progress", tag: "!!str", value: "off"},
		{key: "interim_assistant_messages", tag: "!!bool", value: "true"},
		{key: "long_running_notifications", tag: "!!bool", value: "false"},
		{key: "busy_ack_detail", tag: "!!bool", value: "false"},
	} {
		changed = ensureYAMLScalar(feishu, item.key, item.tag, item.value) || changed
	}

	for _, platform := range []string{"weixin", "wecom"} {
		platformConfig, added, err := ensureYAMLMapping(platforms, platform)
		if err != nil {
			return nil, false, err
		}
		changed = changed || added
		changed = ensureYAMLScalar(platformConfig, "streaming", "!!bool", "false") || changed
	}

	if !changed {
		return data, false, nil
	}
	next, err := yaml.Marshal(&document)
	if err != nil {
		return nil, false, err
	}
	return next, true, nil
}

func ensureYAMLMapping(parent *yaml.Node, key string) (*yaml.Node, bool, error) {
	value, found := yamlMappingValue(parent, key)
	if found {
		if value.Kind != yaml.MappingNode {
			return nil, false, fmt.Errorf("%s 必须是 YAML mapping", key)
		}
		return value, false, nil
	}
	value = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	parent.Content = append(parent.Content, yamlStringNode(key), value)
	return value, true, nil
}

func ensureYAMLScalar(parent *yaml.Node, key string, tag string, value string) bool {
	if _, found := yamlMappingValue(parent, key); found {
		return false
	}
	parent.Content = append(parent.Content, yamlStringNode(key), &yaml.Node{
		Kind:  yaml.ScalarNode,
		Tag:   tag,
		Value: value,
	})
	return true
}

func yamlMappingValue(parent *yaml.Node, key string) (*yaml.Node, bool) {
	for i := 0; i+1 < len(parent.Content); i += 2 {
		if parent.Content[i].Value == key {
			return parent.Content[i+1], true
		}
	}
	return nil, false
}

func yamlStringNode(value string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value}
}

func migrationApplied(records []MigrationRecord, id string) bool {
	for _, record := range records {
		if record.ID == id {
			return true
		}
	}
	return false
}
