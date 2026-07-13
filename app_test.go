package main

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestStartupCreatesHomeInstance(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	app := NewApp()
	app.hostBridgeAddr = "127.0.0.1:0"
	app.startup(context.Background())
	t.Cleanup(func() { app.stopHostBridge(context.Background()) })
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
	assertRuntimeHelpers(t, root)
	if _, err := os.Stat(filepath.Join(root, ".hermes-dock")); !os.IsNotExist(err) {
		t.Fatalf("unexpected nested .hermes-dock directory: %v", err)
	}
}

func TestStartupComposeIncludesInitPermissions(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	app := NewApp()
	app.hostBridgeAddr = "127.0.0.1:0"
	app.startup(context.Background())
	t.Cleanup(func() { app.stopHostBridge(context.Background()) })
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
		"      - ./launcher/helpers/patch-wecom-filenames:/etc/cont-init.d/017-patch-wecom-filenames:ro",
		"      - ./launcher/helpers/install-feishu-deps:/etc/cont-init.d/018-install-feishu-deps:ro",
		"      - ./launcher/helpers/hermes-profile-runner:/opt/hermes-dock/hermes-profile-runner:ro",
		"      - ./launcher/helpers/hostctl:/usr/local/bin/hostctl:ro",
		"      - ./launcher/host-bridge.token:/opt/hermes-dock/host-bridge.token:ro",
		"      - \"host.docker.internal:host-gateway\"",
	} {
		if !strings.Contains(compose, want) {
			t.Fatalf("compose missing %q:\n%s", want, compose)
		}
	}
	if strings.Contains(compose, "entrypoint:") {
		t.Fatalf("compose must not override entrypoint:\n%s", compose)
	}
}

func TestEnsureInstanceReadyMigratesLegacyCompose(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	app := NewApp()
	app.hostBridgeAddr = "127.0.0.1:0"
	app.startup(context.Background())
	t.Cleanup(func() { app.stopHostBridge(context.Background()) })
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
	if !strings.Contains(string(actual), "/etc/cont-init.d/018-install-feishu-deps") {
		t.Fatalf("migrated compose missing feishu dependency helper:\n%s", actual)
	}
	if !strings.Contains(string(actual), "/etc/cont-init.d/017-patch-wecom-filenames") {
		t.Fatalf("migrated compose missing wecom filename patch helper:\n%s", actual)
	}
}

func TestEnsureInstanceReadyMigratesRunnerComposeMissingRuntimeHelpers(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	app := NewApp()
	app.hostBridgeAddr = "127.0.0.1:0"
	app.startup(context.Background())
	t.Cleanup(func() { app.stopHostBridge(context.Background()) })
	if app.startupErr != nil {
		t.Fatal(app.startupErr)
	}

	root := filepath.Join(home, ".hermes-dock")
	composePath := filepath.Join(root, "docker-compose.yaml")
	oldCompose := `services:
  hermes:
    image: local/test:latest
    init: false
    command: /opt/hermes-dock/hermes-profile-runner
    volumes:
      - ./data:/opt/data
      - ./launcher/helpers/hermes-profile-runner:/opt/hermes-dock/hermes-profile-runner:ro
`
	if err := os.WriteFile(composePath, []byte(oldCompose), 0644); err != nil {
		t.Fatal(err)
	}
	backupsBefore := backupCount(t, app)

	if err := app.ensureInstanceReady(); err != nil {
		t.Fatal(err)
	}
	migrated, err := os.ReadFile(composePath)
	if err != nil {
		t.Fatal(err)
	}
	migratedCompose := string(migrated)
	if migratedCompose == oldCompose {
		t.Fatalf("runner compose missing feishu helper was not migrated")
	}
	for _, want := range []string{
		"command: /opt/hermes-dock/hermes-profile-runner",
		"./launcher/helpers/patch-wecom-filenames:/etc/cont-init.d/017-patch-wecom-filenames:ro",
		"./launcher/helpers/install-feishu-deps:/etc/cont-init.d/018-install-feishu-deps:ro",
	} {
		if !strings.Contains(migratedCompose, want) {
			t.Fatalf("migrated compose missing %q:\n%s", want, migratedCompose)
		}
	}
	if strings.Contains(migratedCompose, "entrypoint:") {
		t.Fatalf("migrated compose must not override entrypoint:\n%s", migratedCompose)
	}
	state, err := app.readState()
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Backups) != backupsBefore+1 {
		t.Fatalf("backup count = %d, want %d", len(state.Backups), backupsBefore+1)
	}
	if got := state.Backups[len(state.Backups)-1].Reason; got != "before-compose-runtime-helper-migration" {
		t.Fatalf("backup reason = %q", got)
	}

	backupsAfterMigration := backupCount(t, app)
	if err := app.ensureInstanceReady(); err != nil {
		t.Fatal(err)
	}
	afterIdempotent, err := os.ReadFile(composePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(afterIdempotent) != migratedCompose {
		t.Fatalf("compose changed on idempotent ensure")
	}
	if got := backupCount(t, app); got != backupsAfterMigration {
		t.Fatalf("backup count after idempotent ensure = %d, want %d", got, backupsAfterMigration)
	}
}

func TestEnsureInstanceReadyMigratesRunnerComposeMissingWecomPatchHelper(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	app := NewApp()
	app.hostBridgeAddr = "127.0.0.1:0"
	app.startup(context.Background())
	t.Cleanup(func() { app.stopHostBridge(context.Background()) })
	if app.startupErr != nil {
		t.Fatal(app.startupErr)
	}

	root := filepath.Join(home, ".hermes-dock")
	composePath := filepath.Join(root, "docker-compose.yaml")
	oldCompose := `services:
  hermes:
    image: local/test:latest
    init: false
    command: /opt/hermes-dock/hermes-profile-runner
    volumes:
      - ./data:/opt/data
      - ./launcher/helpers/install-feishu-deps:/etc/cont-init.d/018-install-feishu-deps:ro
      - ./launcher/helpers/hermes-profile-runner:/opt/hermes-dock/hermes-profile-runner:ro
`
	if err := os.WriteFile(composePath, []byte(oldCompose), 0644); err != nil {
		t.Fatal(err)
	}

	if err := app.ensureInstanceReady(); err != nil {
		t.Fatal(err)
	}
	migrated, err := os.ReadFile(composePath)
	if err != nil {
		t.Fatal(err)
	}
	migratedCompose := string(migrated)
	if migratedCompose == oldCompose {
		t.Fatalf("runner compose missing wecom patch helper was not migrated")
	}
	if !strings.Contains(migratedCompose, "./launcher/helpers/patch-wecom-filenames:/etc/cont-init.d/017-patch-wecom-filenames:ro") {
		t.Fatalf("migrated compose missing wecom patch helper:\n%s", migratedCompose)
	}
}

func TestEnsureInstanceReadyRestoresRuntimeHelpers(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	app := NewApp()
	app.hostBridgeAddr = "127.0.0.1:0"
	app.startup(context.Background())
	t.Cleanup(func() { app.stopHostBridge(context.Background()) })
	if app.startupErr != nil {
		t.Fatal(app.startupErr)
	}

	root := filepath.Join(home, ".hermes-dock")
	feishuHelper := filepath.Join(root, "launcher", "helpers", "install-feishu-deps")
	if err := os.Remove(feishuHelper); err != nil {
		t.Fatal(err)
	}
	wecomHelper := filepath.Join(root, "launcher", "helpers", "patch-wecom-filenames")
	if err := os.Remove(wecomHelper); err != nil {
		t.Fatal(err)
	}
	if err := app.ensureInstanceReady(); err != nil {
		t.Fatal(err)
	}
	assertRuntimeHelpers(t, root)
}

func assertRuntimeHelpers(t *testing.T, root string) {
	t.Helper()
	assertFeishuDepsHelper(t, root)
	assertWecomFilenamePatchHelper(t, root)
	assertHostctlHelper(t, root)
	assertHostBridgeToken(t, root)
}

func assertHostctlHelper(t *testing.T, root string) {
	t.Helper()
	helper := filepath.Join(root, "launcher", "helpers", "hostctl")
	data, err := os.ReadFile(helper)
	if err != nil {
		t.Fatalf("expected hostctl helper: %v", err)
	}
	if !strings.Contains(string(data), "host.docker.internal:9877") {
		t.Fatalf("hostctl helper missing Host Bridge address")
	}
	for _, command := range []string{"/v1/files/read", "/v1/clipboard/text", "/v1/processes", "/v1/screenshot"} {
		if !strings.Contains(string(data), command) {
			t.Fatalf("hostctl helper missing %q", command)
		}
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(helper)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode()&0111 == 0 {
			t.Fatalf("hostctl mode = %v, want executable bit", info.Mode())
		}
	}
}

func assertHostBridgeToken(t *testing.T, root string) {
	t.Helper()
	token := filepath.Join(root, "launcher", "host-bridge.token")
	data, err := os.ReadFile(token)
	if err != nil {
		t.Fatalf("expected Host Bridge token: %v", err)
	}
	if len(strings.TrimSpace(string(data))) != 64 {
		t.Fatalf("Host Bridge token length = %d, want 64", len(strings.TrimSpace(string(data))))
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(token)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0600 {
			t.Fatalf("Host Bridge token mode = %v, want 0600", info.Mode().Perm())
		}
	}
}

func assertFeishuDepsHelper(t *testing.T, root string) {
	t.Helper()
	helper := filepath.Join(root, "launcher", "helpers", "install-feishu-deps")
	data, err := os.ReadFile(helper)
	if err != nil {
		t.Fatalf("expected install-feishu-deps helper: %v", err)
	}
	content := string(data)
	for _, want := range []string{
		"lark-oapi==1.5.3",
		"qrcode==7.4.2",
		"/opt/hermes/.venv/bin/python",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("install-feishu-deps missing %q:\n%s", want, content)
		}
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(helper)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode()&0111 == 0 {
			t.Fatalf("install-feishu-deps mode = %v, want executable bit", info.Mode())
		}
	}
}

func assertWecomFilenamePatchHelper(t *testing.T, root string) {
	t.Helper()
	helper := filepath.Join(root, "launcher", "helpers", "patch-wecom-filenames")
	data, err := os.ReadFile(helper)
	if err != nil {
		t.Fatalf("expected patch-wecom-filenames helper: %v", err)
	}
	content := string(data)
	for _, want := range []string{
		"/opt/hermes/gateway/platforms/wecom.py",
		"MAX_WECOM_CACHE_BASENAME_BYTES",
		"_sanitize_inbound_filename",
		"unquote",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("patch-wecom-filenames missing %q:\n%s", want, content)
		}
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(helper)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode()&0111 == 0 {
			t.Fatalf("patch-wecom-filenames mode = %v, want executable bit", info.Mode())
		}
	}
}

func backupCount(t *testing.T, app *App) int {
	t.Helper()
	state, err := app.readState()
	if err != nil {
		t.Fatal(err)
	}
	return len(state.Backups)
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
