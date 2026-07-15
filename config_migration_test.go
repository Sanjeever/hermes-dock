package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAddProfileStreamingDefaultsEnablesFeishuStreaming(t *testing.T) {
	input := []byte(`streaming:
  enabled: false
  transport: off
display:
  language: zh
  platforms:
    feishu:
      streaming: false
      tool_progress: verbose
    weixin:
      streaming: true
custom_setting: keep
`)

	next, changed, err := addProfileStreamingDefaults(input)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("streaming activation switches should be updated")
	}
	var config map[string]interface{}
	if err := parseYAML(next, &config); err != nil {
		t.Fatal(err)
	}
	streaming := asMap(config["streaming"])
	if !asBool(streaming["enabled"]) {
		t.Fatal("streaming.enabled should be enabled")
	}
	if got := asString(streaming["transport"]); got != "off" {
		t.Fatalf("streaming.transport = %q, want off", got)
	}
	platforms := asMap(asMap(config["display"])["platforms"])
	if !asBool(asMap(platforms["feishu"])["streaming"]) {
		t.Fatal("feishu streaming should be enabled")
	}
	if got := asString(asMap(platforms["feishu"])["tool_progress"]); got != "verbose" {
		t.Fatalf("feishu tool_progress = %q, want verbose", got)
	}
	if !asBool(asMap(platforms["weixin"])["streaming"]) {
		t.Fatal("existing weixin streaming setting should be preserved")
	}
	if asBool(asMap(platforms["wecom"])["streaming"]) {
		t.Fatal("missing wecom streaming setting should default to false")
	}
	if got := asString(config["custom_setting"]); got != "keep" {
		t.Fatalf("custom_setting = %q, want keep", got)
	}

	idempotent, changed, err := addProfileStreamingDefaults(next)
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatal("streaming defaults migration should be idempotent")
	}
	if string(idempotent) != string(next) {
		t.Fatal("idempotent migration changed config content")
	}
}

func TestMigrateProfileStreamingDefaultsUpdatesAllProfiles(t *testing.T) {
	app := newTestApp(t)
	if err := app.CreateProfile(CreateProfileRequest{ID: "sales", Name: "销售助手", Enabled: true, CopyMode: profileCopyClean}); err != nil {
		t.Fatal(err)
	}

	oldConfig := []byte("streaming:\n  enabled: false\ndisplay:\n  language: zh\n  platforms:\n    feishu:\n      streaming: false\ncustom_setting: keep\n")
	for _, id := range []string{defaultProfileID, "sales"} {
		path := filepath.Join(app.profileDataDir(id), "config.yaml")
		if err := atomicWriteFile(path, oldConfig, 0644); err != nil {
			t.Fatal(err)
		}
	}
	state, err := app.readState()
	if err != nil {
		t.Fatal(err)
	}
	state.Migrations = removeMigration(state.Migrations, profileStreamingMigrationID)
	state.Migrations = appendIfMissingMigration(state.Migrations, MigrationRecord{ID: "profile-streaming-v1", AppliedAt: "2026-07-15T00:00:00Z"})
	state.NeedsRebuild = false
	if err := app.writeState(state); err != nil {
		t.Fatal(err)
	}
	backupsBefore := len(state.Backups)

	if err := app.migrateProfileStreamingDefaults(); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{defaultProfileID, "sales"} {
		var config map[string]interface{}
		if err := parseYAMLFile(filepath.Join(app.profileDataDir(id), "config.yaml"), &config); err != nil {
			t.Fatal(err)
		}
		if !asBool(asMap(config["streaming"])["enabled"]) {
			t.Fatalf("%s streaming should be enabled", id)
		}
		platforms := asMap(asMap(config["display"])["platforms"])
		if !asBool(asMap(platforms["feishu"])["streaming"]) {
			t.Fatalf("%s feishu streaming should be enabled", id)
		}
		if got := asString(config["custom_setting"]); got != "keep" {
			t.Fatalf("%s custom_setting = %q, want keep", id, got)
		}
	}

	state, err = app.readState()
	if err != nil {
		t.Fatal(err)
	}
	if !migrationApplied(state.Migrations, profileStreamingMigrationID) {
		t.Fatal("profile streaming migration was not recorded")
	}
	if !state.NeedsRebuild {
		t.Fatal("profile streaming migration should require rebuild")
	}
	if got := len(state.Backups); got != backupsBefore+2 {
		t.Fatalf("backup count = %d, want %d", got, backupsBefore+2)
	}
	for _, backup := range state.Backups[backupsBefore:] {
		if backup.Reason != "before-profile-streaming-v2-migration" {
			t.Fatalf("backup reason = %q", backup.Reason)
		}
		if _, err := os.Stat(filepath.Join(app.instanceRoot, backup.Path)); err != nil {
			t.Fatal(err)
		}
	}

	if err := app.migrateProfileStreamingDefaults(); err != nil {
		t.Fatal(err)
	}
	afterIdempotent, err := app.readState()
	if err != nil {
		t.Fatal(err)
	}
	if len(afterIdempotent.Backups) != len(state.Backups) {
		t.Fatal("idempotent migration created additional backups")
	}
}

func TestMigrateProfileStreamingDefaultsDoesNotRequireRebuildWhenUnchanged(t *testing.T) {
	app := newTestApp(t)
	state, err := app.readState()
	if err != nil {
		t.Fatal(err)
	}
	state.Migrations = removeMigration(state.Migrations, profileStreamingMigrationID)
	state.NeedsRebuild = false
	if err := app.writeState(state); err != nil {
		t.Fatal(err)
	}
	backupsBefore := len(state.Backups)

	if err := app.migrateProfileStreamingDefaults(); err != nil {
		t.Fatal(err)
	}
	state, err = app.readState()
	if err != nil {
		t.Fatal(err)
	}
	if !migrationApplied(state.Migrations, profileStreamingMigrationID) {
		t.Fatal("profile streaming migration was not recorded")
	}
	if state.NeedsRebuild {
		t.Fatal("unchanged streaming config should not require rebuild")
	}
	if len(state.Backups) != backupsBefore {
		t.Fatal("unchanged streaming config should not create backups")
	}
}

func removeMigration(records []MigrationRecord, id string) []MigrationRecord {
	out := records[:0]
	for _, record := range records {
		if record.ID != id {
			out = append(out, record)
		}
	}
	return out
}
