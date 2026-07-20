package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSyncBundledContentRejectsTargetSkillSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink test is unix-only")
	}
	app := newTestApp(t)
	target := filepath.Join(app.profileDataDir(defaultProfileID), "skills", "hermes-dock")
	if err := os.RemoveAll(target); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	if err := os.Symlink(outside, target); err != nil {
		t.Fatal(err)
	}
	result, err := app.SyncBundledContent(BundledContentSyncRequest{
		TargetProfileIDs: []string{defaultProfileID},
		SyncSkills:       true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Failed != 1 || !strings.Contains(result.Results[0].Error, "符号链接") {
		t.Fatalf("expected target symlink rejection: %+v", result)
	}
	if fileExists(filepath.Join(outside, "SKILL.md")) {
		t.Fatal("bundled sync followed a symlink outside the profile")
	}
}

func TestClassifyBundledFile(t *testing.T) {
	tests := []struct {
		name      string
		exists    bool
		current   string
		template  string
		previous  string
		collision bool
		want      string
	}{
		{name: "new", want: "added"},
		{name: "latest", exists: true, current: "new", template: "new", want: "unchanged"},
		{name: "safe update", exists: true, current: "old", template: "new", previous: "old", want: "updated"},
		{name: "modified", exists: true, current: "local", template: "new", previous: "old", want: "skipped"},
		{name: "custom collision", collision: true, want: "skipped"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := classifyBundledFile(tt.exists, tt.current, tt.template, tt.previous, tt.collision); got != tt.want {
				t.Fatalf("classifyBundledFile() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestReleaseSeedDataDoesNotReplaceExistingBundledBaseline(t *testing.T) {
	app := newTestApp(t)
	state, err := app.readBundledContentState("default")
	if err != nil {
		t.Fatal(err)
	}
	originalSoulHash := state.Files["SOUL.md"]
	if originalSoulHash == "" {
		t.Fatal("missing initial SOUL.md baseline")
	}
	modified := []byte("user customized soul\n")
	if err := atomicWriteFile(app.profileSoulPath("default"), modified, 0644); err != nil {
		t.Fatal(err)
	}
	if err := app.releaseSeedData(); err != nil {
		t.Fatal(err)
	}
	after, err := app.readBundledContentState("default")
	if err != nil {
		t.Fatal(err)
	}
	if after.Files["SOUL.md"] != originalSoulHash {
		t.Fatalf("bundled baseline changed from %q to %q", originalSoulHash, after.Files["SOUL.md"])
	}
	if after.Files["SOUL.md"] == contentHash(modified) {
		t.Fatal("user customization was recorded as a safe-to-overwrite bundled baseline")
	}
}

func TestReleaseSeedDataDoesNotAdoptExistingContentWithoutBaseline(t *testing.T) {
	app := NewApp()
	app.instanceRoot = t.TempDir()
	if err := ensureDir(app.dataDir()); err != nil {
		t.Fatal(err)
	}
	if err := atomicWriteFile(app.profileSoulPath("default"), []byte("existing user soul\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := app.releaseSeedData(); err != nil {
		t.Fatal(err)
	}
	if fileExists(app.bundledContentStatePath("default")) {
		t.Fatal("existing content without a baseline was incorrectly adopted as bundled content")
	}
}

func TestSyncBundledContentResetsModifiedSoul(t *testing.T) {
	app := newTestApp(t)
	modified := []byte("user customized soul\n")
	if err := atomicWriteFile(app.profileSoulPath(defaultProfileID), modified, 0644); err != nil {
		t.Fatal(err)
	}

	result, err := app.SyncBundledContent(BundledContentSyncRequest{
		TargetProfileIDs: []string{defaultProfileID},
		SyncSoul:         true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Updated != 1 || result.Skipped != 0 {
		t.Fatalf("unexpected sync result: %+v", result)
	}
	want, err := seedData.ReadFile("templates/seed-data/SOUL.md")
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(app.profileSoulPath(defaultProfileID))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Fatal("modified SOUL.md was not reset to the bundled template")
	}
	state, err := app.readState()
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Backups) == 0 || state.Backups[len(state.Backups)-1].Reason != "before-sync-bundled-soul-default" {
		t.Fatal("modified SOUL.md was not backed up before reset")
	}
}

func TestSyncBundledContentAddsMissingAndPreservesModifiedAndCustomSkills(t *testing.T) {
	app := newTestApp(t)
	modified := filepath.Join(app.profileDataDir(defaultProfileID), "skills", "hermes-dock", "SKILL.md")
	if err := os.WriteFile(modified, []byte("local edit"), 0644); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(app.profileDataDir(defaultProfileID), "skills", "hermes-dock-host", "SKILL.md")
	if err := os.Remove(missing); err != nil {
		t.Fatal(err)
	}
	missingOCRModel := filepath.Join(app.profileDataDir(defaultProfileID), "skills", "productivity", "image-text-ocr", "assets", "models", "PP-OCRv6_small_det_infer", "inference.json")
	if err := os.Remove(missingOCRModel); err != nil {
		t.Fatal(err)
	}
	custom := filepath.Join(app.profileDataDir(defaultProfileID), "skills", "custom", "keep", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(custom), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(custom, []byte("custom skill"), 0644); err != nil {
		t.Fatal(err)
	}
	oldSkill := filepath.Join(app.profileDataDir(defaultProfileID), "skills", "old-bundled", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(oldSkill), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(oldSkill, []byte("old skill"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := app.SyncBundledContent(BundledContentSyncRequest{TargetProfileIDs: []string{defaultProfileID}, SyncSkills: true})
	if err != nil {
		t.Fatal(err)
	}
	if result.Added == 0 || result.Skipped == 0 {
		t.Fatalf("unexpected sync result: %+v", result)
	}
	if data, _ := os.ReadFile(modified); string(data) != "local edit" {
		t.Fatal("modified bundled skill was overwritten")
	}
	if !fileExists(missing) {
		t.Fatal("missing bundled file was not added")
	}
	if !fileExists(missingOCRModel) {
		t.Fatal("missing bundled OCR model was not added")
	}
	if data, _ := os.ReadFile(custom); string(data) != "custom skill" {
		t.Fatal("custom skill was modified")
	}
	if data, _ := os.ReadFile(oldSkill); string(data) != "old skill" {
		t.Fatal("old skill was deleted or modified")
	}
}

func TestSyncBundledContentIsolatesProfileFailures(t *testing.T) {
	app := newTestApp(t)
	for _, id := range []string{"good", "broken"} {
		if err := app.CreateProfile(CreateProfileRequest{ID: id, Name: id, CopyMode: profileCopyClean}); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Remove(app.profileSoulPath("good")); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(app.profileSoulPath("broken")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(app.profileSoulPath("broken"), 0755); err != nil {
		t.Fatal(err)
	}
	result, err := app.SyncBundledContent(BundledContentSyncRequest{TargetProfileIDs: []string{"broken", "good"}, SyncSoul: true})
	if err != nil {
		t.Fatal(err)
	}
	if result.Succeeded != 1 || result.Failed != 1 || !result.Results[1].Success {
		t.Fatalf("profile failures were not isolated: %+v", result)
	}
	if !fileExists(app.profileSoulPath("good")) {
		t.Fatal("good profile was not synchronized")
	}
}
