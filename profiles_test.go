package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newTestApp(t *testing.T) *App {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	app := NewApp()
	app.startup(context.Background())
	if app.startupErr != nil {
		t.Fatal(app.startupErr)
	}
	app.ctx = nil
	return app
}

func TestProfileRegistryInitializesDefault(t *testing.T) {
	app := newTestApp(t)
	registry, err := app.readProfileRegistry()
	if err != nil {
		t.Fatal(err)
	}
	if len(registry.Profiles) != 1 {
		t.Fatalf("profiles = %d, want 1", len(registry.Profiles))
	}
	profile := registry.Profiles[0]
	if profile.ID != "default" || !profile.Enabled {
		t.Fatalf("default profile = %+v", profile)
	}
	if !fileExists(filepath.Join(app.instanceRoot, "launcher", "profiles.json")) {
		t.Fatalf("profiles.json not created")
	}
}

func TestValidateProfileID(t *testing.T) {
	valid := []string{"ab", "sales", "sales-1", "a1-b2"}
	for _, id := range valid {
		if err := validateProfileID(id, false); err != nil {
			t.Fatalf("%s should be valid: %v", id, err)
		}
	}
	invalid := []string{"default", "A", "a_", "-abc", "abc-", "中文", "../x"}
	for _, id := range invalid {
		if err := validateProfileID(id, false); err == nil {
			t.Fatalf("%s should be invalid", id)
		}
	}
}

func TestCreateProfileRewritesProfileHomeHints(t *testing.T) {
	app := newTestApp(t)
	if err := app.CreateProfile(CreateProfileRequest{ID: "sales", Name: "销售助手", Enabled: true, CopyMode: "clean"}); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(app.instanceRoot, "data", "profiles", "sales")
	config, err := os.ReadFile(filepath.Join(dir, "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(config), "cwd: /opt/data/profiles/sales") {
		t.Fatalf("config cwd not rewritten")
	}
	soul, err := os.ReadFile(filepath.Join(dir, "SOUL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(soul), "/opt/data/profiles/sales/tmp") {
		t.Fatalf("SOUL tmp path not rewritten")
	}
	env, err := readEnvFile(filepath.Join(dir, ".env"))
	if err != nil {
		t.Fatal(err)
	}
	if got := envValue(env, "HERMES_DOCK_PROFILE"); got != "sales" {
		t.Fatalf("HERMES_DOCK_PROFILE = %q", got)
	}
	if got := envValue(env, "HERMES_DOCK_PROFILE_HOME"); got != "/opt/data/profiles/sales" {
		t.Fatalf("HERMES_DOCK_PROFILE_HOME = %q", got)
	}
}

func TestCreateProfileCopyPersonalityRewritesSoulHome(t *testing.T) {
	app := newTestApp(t)
	if err := app.CreateProfile(CreateProfileRequest{ID: "writer", Name: "写作助手", Enabled: true, CopyMode: "personality-skills", CopyFrom: "default"}); err != nil {
		t.Fatal(err)
	}
	soul, err := os.ReadFile(filepath.Join(app.profileDataDir("writer"), "SOUL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(soul), "/opt/data/tmp") {
		t.Fatalf("copied SOUL still points at default tmp")
	}
	if !strings.Contains(string(soul), "/opt/data/profiles/writer/tmp") {
		t.Fatalf("copied SOUL does not point at writer tmp")
	}
}

func TestValidateRuntimeProfilesRejectsDuplicateEnabledPlatformIdentity(t *testing.T) {
	app := newTestApp(t)
	for _, id := range []string{"sales", "support"} {
		if err := app.CreateProfile(CreateProfileRequest{ID: id, Name: id, Enabled: true, CopyMode: "clean"}); err != nil {
			t.Fatal(err)
		}
		envPath := filepath.Join(app.profileDataDir(id), ".env")
		if err := writeEnvFile(envPath, mergeEnv(defaultEnvVars(), []EnvVar{
			{Key: "WECOM_BOT_ID", Value: "bot-1"},
			{Key: "WECOM_SECRET", Value: "secret"},
		})); err != nil {
			t.Fatal(err)
		}
	}
	registry, err := app.readProfileRegistry()
	if err != nil {
		t.Fatal(err)
	}
	err = app.validateRuntimeProfiles(registry)
	if err == nil || !strings.Contains(err.Error(), "企业微信 Bot") {
		t.Fatalf("expected duplicate wecom error, got %v", err)
	}
}

func TestBuildRuntimeManifestSkipsUnboundProfile(t *testing.T) {
	app := newTestApp(t)
	if err := app.CreateProfile(CreateProfileRequest{ID: "sales", Name: "销售助手", Enabled: true, CopyMode: "clean"}); err != nil {
		t.Fatal(err)
	}
	registry, err := app.readProfileRegistry()
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := app.buildRuntimeManifest(registry)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, profile := range manifest.Profiles {
		if profile.ID == "sales" {
			found = true
			if profile.Runnable {
				t.Fatalf("unbound profile should not be runnable")
			}
			if profile.Reason != "not_configured" {
				t.Fatalf("reason = %q", profile.Reason)
			}
		}
	}
	if !found {
		t.Fatalf("sales not found in manifest")
	}
}
