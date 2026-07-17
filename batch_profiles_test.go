package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestBatchCopyProfileConfigPreservesTargetAPIKeyAndRewritesSoul(t *testing.T) {
	app := newTestApp(t)
	for _, id := range []string{"sales", "support"} {
		if err := app.CreateProfile(CreateProfileRequest{ID: id, Name: id, Enabled: true, CopyMode: profileCopyClean}); err != nil {
			t.Fatal(err)
		}
	}
	sourceProviders, err := app.readProviderConfigForProfile(defaultProfileID)
	if err != nil {
		t.Fatal(err)
	}
	sourceEntry := sourceProviders.Providers["dashscope-payg"]
	sourceEntry.APIKey = "source-secret"
	sourceProviders.Providers["dashscope-payg"] = sourceEntry
	if err := app.SaveProviderConfigForProfile(defaultProfileID, sourceProviders); err != nil {
		t.Fatal(err)
	}
	sourceModel, err := app.readModelConfigForProfile(defaultProfileID)
	if err != nil {
		t.Fatal(err)
	}
	sourceModel.Default = "source-model"
	if err := app.SaveModelConfigForProfile(defaultProfileID, sourceModel); err != nil {
		t.Fatal(err)
	}
	targetProviders, err := app.readProviderConfigForProfile("sales")
	if err != nil {
		t.Fatal(err)
	}
	targetEntry := targetProviders.Providers["dashscope-payg"]
	targetEntry.APIKey = "target-secret"
	targetProviders.Providers["dashscope-payg"] = targetEntry
	if err := app.SaveProviderConfigForProfile("sales", targetProviders); err != nil {
		t.Fatal(err)
	}
	if err := atomicWriteFile(app.profileSoulPath(defaultProfileID), []byte("home=/opt/data\ntmp=/opt/data/tmp\n"), 0644); err != nil {
		t.Fatal(err)
	}
	sourceSkill := filepath.Join(app.profileDataDir(defaultProfileID), "skills", "custom", "copied", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(sourceSkill), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sourceSkill, []byte("---\nname: copied-skill\ndescription: copied\n---\n"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := app.BatchCopyProfileConfig(BatchProfileConfigRequest{
		SourceProfileID:  defaultProfileID,
		TargetProfileIDs: []string{"sales", "missing"},
		CopyMainModel:    true,
		CopySoul:         true,
		SkillPaths:       []string{"skills/custom/copied"},
		CopyProviders:    true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Succeeded != 1 || result.Failed != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
	targetModel, err := app.readModelConfigForProfile("sales")
	if err != nil {
		t.Fatal(err)
	}
	if targetModel.Default != "source-model" {
		t.Fatalf("target model = %q", targetModel.Default)
	}
	targetProviders, err = app.readProviderConfigForProfile("sales")
	if err != nil {
		t.Fatal(err)
	}
	if got := targetProviders.Providers["dashscope-payg"].APIKey; got != "target-secret" {
		t.Fatalf("target API key = %q, want preserved target secret", got)
	}
	soul, err := os.ReadFile(app.profileSoulPath("sales"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(soul), "/opt/data/profiles/sales/tmp") || strings.Contains(string(soul), "/opt/data/tmp") {
		t.Fatalf("target SOUL path was not rewritten:\n%s", soul)
	}
	if !fileExists(filepath.Join(app.profileDataDir("sales"), "skills", "custom", "copied", "SKILL.md")) {
		t.Fatal("selected skill was not copied")
	}
	state, err := app.readState()
	if err != nil {
		t.Fatal(err)
	}
	if !state.NeedsRebuild || len(state.Backups) == 0 {
		t.Fatalf("batch copy did not mark rebuild or create backups: %+v", state)
	}
}

func TestBatchCopyProfileConfigCopiesMainModelFallbackProviders(t *testing.T) {
	app := newTestApp(t)
	if err := app.CreateProfile(CreateProfileRequest{ID: "target", Name: "Target", CopyMode: profileCopyClean}); err != nil {
		t.Fatal(err)
	}
	source, err := app.readConfigMapForProfile(defaultProfileID)
	if err != nil {
		t.Fatal(err)
	}
	source["fallback_providers"] = []interface{}{"deepseek", "zhipu-payg"}
	data, err := yaml.Marshal(source)
	if err != nil {
		t.Fatal(err)
	}
	if err := atomicWriteFile(app.profileConfigPath(defaultProfileID), data, 0644); err != nil {
		t.Fatal(err)
	}
	target, err := app.readConfigMapForProfile("target")
	if err != nil {
		t.Fatal(err)
	}
	target["fallback_providers"] = []interface{}{"agnes"}
	data, err = yaml.Marshal(target)
	if err != nil {
		t.Fatal(err)
	}
	if err := atomicWriteFile(app.profileConfigPath("target"), data, 0644); err != nil {
		t.Fatal(err)
	}

	result, err := app.BatchCopyProfileConfig(BatchProfileConfigRequest{
		SourceProfileID:  defaultProfileID,
		TargetProfileIDs: []string{"target"},
		CopyMainModel:    true,
		CopyProviders:    true,
	})
	if err != nil || result.Failed != 0 {
		t.Fatalf("batch copy failed: %+v, %v", result, err)
	}
	model, err := app.readModelConfigForProfile("target")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(model.Fallbacks, ",") != "deepseek,zhipu-payg" {
		t.Fatalf("fallback providers = %#v", model.Fallbacks)
	}
}

func TestBatchCopyProfileConfigCopiesAPIKeyOnlyWhenExplicit(t *testing.T) {
	app := newTestApp(t)
	if err := app.CreateProfile(CreateProfileRequest{ID: "target", Name: "Target", CopyMode: profileCopyClean}); err != nil {
		t.Fatal(err)
	}
	providers, err := app.readProviderConfigForProfile(defaultProfileID)
	if err != nil {
		t.Fatal(err)
	}
	entry := providers.Providers["deepseek"]
	entry.APIKey = "explicit-secret"
	providers.Providers["deepseek"] = entry
	if err := app.SaveProviderConfigForProfile(defaultProfileID, providers); err != nil {
		t.Fatal(err)
	}
	result, err := app.BatchCopyProfileConfig(BatchProfileConfigRequest{
		SourceProfileID:  defaultProfileID,
		TargetProfileIDs: []string{"target"},
		CopyProviders:    true,
		IncludeAPIKeys:   true,
	})
	if err != nil || result.Failed != 0 {
		t.Fatalf("batch copy failed: %+v, %v", result, err)
	}
	target, err := app.readProviderConfigForProfile("target")
	if err != nil {
		t.Fatal(err)
	}
	if target.Providers["deepseek"].APIKey != "explicit-secret" {
		t.Fatal("explicit API key was not copied")
	}
}

func TestBatchCopyProfileConfigRejectsTargetKeyReuseWhenProviderEndpointChanges(t *testing.T) {
	app := newTestApp(t)
	if err := app.CreateProfile(CreateProfileRequest{ID: "target", Name: "Target", CopyMode: profileCopyClean}); err != nil {
		t.Fatal(err)
	}
	targetProviders, err := app.readProviderConfigForProfile("target")
	if err != nil {
		t.Fatal(err)
	}
	targetEntry := targetProviders.Providers["deepseek"]
	targetEntry.APIKey = "target-secret"
	targetProviders.Providers["deepseek"] = targetEntry
	if err := app.SaveProviderConfigForProfile("target", targetProviders); err != nil {
		t.Fatal(err)
	}

	sourceProviders, err := app.readProviderConfigForProfile(defaultProfileID)
	if err != nil {
		t.Fatal(err)
	}
	sourceEntry := sourceProviders.Providers["deepseek"]
	sourceEntry.BaseURL = "https://different.example/v1"
	sourceEntry.APIKey = "source-secret"
	sourceProviders.Providers["deepseek"] = sourceEntry
	if err := app.SaveProviderConfigForProfile(defaultProfileID, sourceProviders); err != nil {
		t.Fatal(err)
	}

	result, err := app.BatchCopyProfileConfig(BatchProfileConfigRequest{
		SourceProfileID:  defaultProfileID,
		TargetProfileIDs: []string{"target"},
		CopyProviders:    true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Failed != 1 || !strings.Contains(result.Results[0].Error, "避免把目标 API Key") {
		t.Fatalf("unsafe provider copy was not rejected: %+v", result)
	}
	targetProviders, err = app.readProviderConfigForProfile("target")
	if err != nil {
		t.Fatal(err)
	}
	if got := targetProviders.Providers["deepseek"]; got.APIKey != "target-secret" || got.BaseURL == "https://different.example/v1" {
		t.Fatalf("target provider changed after rejected copy: %+v", got)
	}
}

func TestBatchCopyProfileConfigIsolatesTargetFailures(t *testing.T) {
	app := newTestApp(t)
	for _, id := range []string{"good", "broken"} {
		if err := app.CreateProfile(CreateProfileRequest{ID: id, Name: id, CopyMode: profileCopyClean}); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(app.profileConfigPath("broken"), []byte("model: [\n"), 0644); err != nil {
		t.Fatal(err)
	}
	result, err := app.BatchCopyProfileConfig(BatchProfileConfigRequest{
		SourceProfileID:  defaultProfileID,
		TargetProfileIDs: []string{"broken", "good"},
		CopyMainModel:    true,
		CopyProviders:    true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Succeeded != 1 || result.Failed != 1 || !result.Results[1].Success {
		t.Fatalf("target failures were not isolated: %+v", result)
	}
}

func TestBatchCopyProfileConfigRequiresSelection(t *testing.T) {
	app := newTestApp(t)
	_, err := app.BatchCopyProfileConfig(BatchProfileConfigRequest{SourceProfileID: defaultProfileID, TargetProfileIDs: []string{"target"}})
	if err == nil || !strings.Contains(err.Error(), "至少选择一项") {
		t.Fatalf("expected selection error, got %v", err)
	}
}

func TestBatchCopyProfileConfigRejectsSourceSkillSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink test is unix-only")
	}
	app := newTestApp(t)
	if err := app.CreateProfile(CreateProfileRequest{ID: "target", Name: "Target", CopyMode: profileCopyClean}); err != nil {
		t.Fatal(err)
	}
	skillDir := filepath.Join(app.profileDataDir(defaultProfileID), "skills", "custom", "linked")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: linked\ndescription: linked\n---\n"), 0644); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "secret.txt")
	if err := os.WriteFile(outside, []byte("secret"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(skillDir, "secret.txt")); err != nil {
		t.Fatal(err)
	}

	_, err := app.BatchCopyProfileConfig(BatchProfileConfigRequest{
		SourceProfileID:  defaultProfileID,
		TargetProfileIDs: []string{"target"},
		SkillPaths:       []string{"skills/custom/linked"},
	})
	if err == nil || !strings.Contains(err.Error(), "符号链接") {
		t.Fatalf("expected source symlink rejection, got %v", err)
	}
	if fileExists(filepath.Join(app.profileDataDir("target"), "skills", "custom", "linked", "secret.txt")) {
		t.Fatal("source symlink content was copied")
	}
}

func TestBatchCopyProfileConfigRejectsTargetSkillSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink test is unix-only")
	}
	app := newTestApp(t)
	if err := app.CreateProfile(CreateProfileRequest{ID: "target", Name: "Target", CopyMode: profileCopyClean}); err != nil {
		t.Fatal(err)
	}
	sourceSkill := filepath.Join(app.profileDataDir(defaultProfileID), "skills", "custom", "linked", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(sourceSkill), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sourceSkill, []byte("---\nname: linked\ndescription: linked\n---\n"), 0644); err != nil {
		t.Fatal(err)
	}
	targetSkill := filepath.Join(app.profileDataDir("target"), "skills", "custom", "linked")
	if err := os.MkdirAll(filepath.Dir(targetSkill), 0755); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	if err := os.Symlink(outside, targetSkill); err != nil {
		t.Fatal(err)
	}

	result, err := app.BatchCopyProfileConfig(BatchProfileConfigRequest{
		SourceProfileID:  defaultProfileID,
		TargetProfileIDs: []string{"target"},
		SkillPaths:       []string{"skills/custom/linked"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Failed != 1 || !strings.Contains(result.Results[0].Error, "符号链接") {
		t.Fatalf("expected target symlink rejection: %+v", result)
	}
	if fileExists(filepath.Join(outside, "SKILL.md")) {
		t.Fatal("target symlink was followed outside the profile")
	}
}

func TestBatchCopyProfileConfigReportsAndMarksPartialTargetChange(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink test is unix-only")
	}
	app := newTestApp(t)
	if err := app.CreateProfile(CreateProfileRequest{ID: "target", Name: "Target", CopyMode: profileCopyClean}); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"a-first", "z-blocked"} {
		path := filepath.Join(app.profileDataDir(defaultProfileID), "skills", "custom", name, "SKILL.md")
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("---\nname: "+name+"\ndescription: test\n---\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	blocked := filepath.Join(app.profileDataDir("target"), "skills", "custom", "z-blocked")
	if err := os.MkdirAll(filepath.Dir(blocked), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(t.TempDir(), blocked); err != nil {
		t.Fatal(err)
	}
	state, _ := app.readState()
	state.NeedsRebuild = false
	if err := app.writeState(state); err != nil {
		t.Fatal(err)
	}

	result, err := app.BatchCopyProfileConfig(BatchProfileConfigRequest{
		SourceProfileID:  defaultProfileID,
		TargetProfileIDs: []string{"target"},
		SkillPaths:       []string{"skills/custom/a-first", "skills/custom/z-blocked"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Failed != 1 || !result.Results[0].Changed {
		t.Fatalf("partial target change was not reported: %+v", result)
	}
	state, _ = app.readState()
	if !state.NeedsRebuild {
		t.Fatal("partial target change did not mark configuration pending")
	}
	if !fileExists(filepath.Join(app.profileDataDir("target"), "skills", "custom", "a-first", "SKILL.md")) {
		t.Fatal("expected first skill to be committed before isolated failure")
	}
}
