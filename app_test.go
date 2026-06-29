package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStartupCreatesHomeInstance(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	app := NewApp()
	app.startup(context.Background())
	if app.startupErr != nil {
		t.Fatal(app.startupErr)
	}

	root := filepath.Join(home, ".hermes-dock")
	if app.instanceRoot != root {
		t.Fatalf("instance root = %q, want %q", app.instanceRoot, root)
	}
	for _, path := range []string{
		"docker-compose.yaml",
		"docker-compose.override.yaml",
		"data/config.yaml",
		"data/.env",
		"launcher/state.json",
	} {
		if _, err := os.Stat(filepath.Join(root, path)); err != nil {
			t.Fatalf("expected %s: %v", path, err)
		}
	}
	if _, err := os.Stat(filepath.Join(root, ".hermes-dock")); !os.IsNotExist(err) {
		t.Fatalf("unexpected nested .hermes-dock directory: %v", err)
	}
}

func TestStartupComposeIncludesInitPermissions(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	app := NewApp()
	app.startup(context.Background())
	if app.startupErr != nil {
		t.Fatal(app.startupErr)
	}

	composePath := filepath.Join(home, ".hermes-dock", "docker-compose.yaml")
	data, err := os.ReadFile(composePath)
	if err != nil {
		t.Fatal(err)
	}
	compose := string(data)
	for _, want := range []string{
		"  init-permissions:",
		"    image: alpine:3.22",
		"    user: \"0:0\"",
		"    command: chown -R 10000:10000 /opt/data",
		"    restart: \"no\"",
		"    depends_on:\n      init-permissions:\n        condition: service_completed_successfully",
		"    command: /opt/hermes-dock/hermes-profile-runner",
		"      HERMES_HOME: \"/opt/data\"",
		"      - ./launcher/helpers/hermes-profile-runner:/opt/hermes-dock/hermes-profile-runner:ro",
	} {
		if !strings.Contains(compose, want) {
			t.Fatalf("compose missing %q:\n%s", want, compose)
		}
	}
}

func TestEnsureInstanceReadyMigratesLegacyCompose(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	app := NewApp()
	app.startup(context.Background())
	if app.startupErr != nil {
		t.Fatal(app.startupErr)
	}

	composePath := filepath.Join(home, ".hermes-dock", "docker-compose.yaml")
	content := []byte("services:\n  hermes:\n    image: local/test:latest\n")
	if err := os.WriteFile(composePath, content, 0644); err != nil {
		t.Fatal(err)
	}
	if err := app.ensureInstanceReady(); err != nil {
		t.Fatal(err)
	}
	actual, err := os.ReadFile(composePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(actual) == string(content) {
		t.Fatalf("legacy compose was not migrated")
	}
	if !strings.Contains(string(actual), "hermes-profile-runner") {
		t.Fatalf("migrated compose missing runner:\n%s", actual)
	}
}

func TestNormalizeDashScopeUsesCompatiblePayAsYouGoEndpoint(t *testing.T) {
	app := NewApp()
	model := app.normalizeModelConfigForSave(ModelConfig{
		Provider: "dashscope",
		Default:  "qwen3.7-max",
		BaseURL:  "https://dashscope.aliyuncs.com/apps/anthropic",
		APIMode:  "anthropic_messages",
	})
	if model.Provider != "custom" {
		t.Fatalf("provider = %q, want custom", model.Provider)
	}
	if model.BaseURL != "https://dashscope.aliyuncs.com/compatible-mode/v1" {
		t.Fatalf("base URL = %q", model.BaseURL)
	}
	if model.APIMode != "chat_completions" {
		t.Fatalf("api mode = %q", model.APIMode)
	}
}

func TestNormalizeDeepSeekDefaults(t *testing.T) {
	app := NewApp()
	model := app.normalizeModelConfigForSave(ModelConfig{Provider: "deepseek"})
	if model.Provider != "deepseek" {
		t.Fatalf("provider = %q, want deepseek", model.Provider)
	}
	if model.BaseURL != "https://api.deepseek.com" {
		t.Fatalf("base URL = %q", model.BaseURL)
	}
	if model.APIMode != "chat_completions" {
		t.Fatalf("api mode = %q", model.APIMode)
	}
}
