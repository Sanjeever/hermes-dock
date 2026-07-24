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
		"shared",
		"data/config.yaml",
		"data/.env",
		"launcher/state.json",
		"launcher/dufs/config.yaml",
	} {
		if _, err := os.Stat(filepath.Join(root, path)); err != nil {
			t.Fatalf("expected %s: %v", path, err)
		}
	}
	assertRuntimeHelpers(t, root)
	if runtime.GOOS != "windows" {
		info, err := os.Stat(filepath.Join(root, "data", ".dock"))
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode()&os.ModeSticky == 0 || info.Mode().Perm() != 0777 {
			t.Fatalf("data/.dock mode = %v, want sticky writable runtime directory", info.Mode())
		}
	}
	if _, err := os.Stat(filepath.Join(root, ".hermes-dock")); !os.IsNotExist(err) {
		t.Fatalf("unexpected nested .hermes-dock directory: %v", err)
	}
}

func TestEnsureInstanceReadyDoesNotRewriteExistingProfileConfigs(t *testing.T) {
	app := newTestApp(t)
	if err := app.CreateProfile(CreateProfileRequest{ID: "sales", Name: "销售助手", Enabled: true, CopyMode: profileCopyClean}); err != nil {
		t.Fatal(err)
	}

	configs := map[string][]byte{
		defaultProfileID: []byte("# 保留用户原始格式\ngroup_sessions_per_user: true\nstreaming:\n  enabled: false\ndisplay:\n  platforms:\n    dingtalk:\n      show_reasoning: false\n      streaming: false\ncustom_setting: keep\n"),
		"sales":          []byte("display:\n  platforms:\n    feishu:\n      streaming: false\n"),
	}
	for id, content := range configs {
		if err := atomicWriteFile(filepath.Join(app.profileDataDir(id), "config.yaml"), content, 0644); err != nil {
			t.Fatal(err)
		}
	}
	state, err := app.readState()
	if err != nil {
		t.Fatal(err)
	}
	migrations := state.Migrations[:0]
	for _, migration := range state.Migrations {
		if migration.ID != "profile-streaming-v2" {
			migrations = append(migrations, migration)
		}
	}
	state.Migrations = migrations
	state.NeedsRebuild = false
	if err := app.writeState(state); err != nil {
		t.Fatal(err)
	}

	if err := app.ensureInstanceReady(); err != nil {
		t.Fatal(err)
	}
	for id, want := range configs {
		got, err := os.ReadFile(filepath.Join(app.profileDataDir(id), "config.yaml"))
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != string(want) {
			t.Fatalf("%s config.yaml was rewritten:\n%s", id, got)
		}
	}
	state, err = app.readState()
	if err != nil {
		t.Fatal(err)
	}
	if state.NeedsRebuild {
		t.Fatal("instance preparation marked unchanged profile configs for rebuild")
	}
	if migrationApplied(state.Migrations, "profile-streaming-v2") {
		t.Fatal("instance preparation recorded the removed profile config migration")
	}
}

func TestStartupComposeUsesTargetedImagePermissions(t *testing.T) {
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
		"    command: /opt/hermes-dock/hermes-profile-runner",
		"      HERMES_HOME: \"/opt/data\"",
		"      HERMES_WRITE_SAFE_ROOT: \"/opt/data\"",
		"      HERMES_DASHBOARD: \"0\"",
		"      HERMES_DOCK_SUPPRESS_HOME_CHANNEL_PROMPT: \"true\"",
		"      AGENT_BROWSER_EXECUTABLE_PATH: \"" + mustBundledChromiumExecutablePath(t, runtime.GOARCH) + "\"",
		"      - ./launcher/runtime-deps/" + runtimeDependencyBundleVersion + ":/opt/hermes-dock/runtime-deps:ro",
		"      - \"" + filepath.Join(home, ".hermes-dock", "shared") + ":/opt/data/.dock/shared\"",
		"      - ./launcher/helpers/verify-runtime-deps:/etc/cont-init.d/016-verify-runtime-deps:ro",
		"      - ./launcher/helpers/patch-wecom-filenames:/etc/cont-init.d/017-patch-wecom-filenames:ro",
		"      - ./launcher/helpers/install-feishu-deps:/etc/cont-init.d/018-install-feishu-deps:ro",
		"      - ./launcher/helpers/patch-home-channel-prompt:/etc/cont-init.d/019-patch-home-channel-prompt:ro",
		"      - ./launcher/helpers/install-dingtalk-deps:/etc/cont-init.d/020-install-dingtalk-deps:ro",
		"      - ./launcher/helpers/patch-dingtalk-media:/etc/cont-init.d/021-patch-dingtalk-media:ro",
		"      - ./launcher/helpers/hermes-profile-runner:/opt/hermes-dock/hermes-profile-runner:ro",
		"      - ./launcher/helpers/hostctl:/usr/local/bin/hostctl:ro",
		"      - ./launcher/host-bridge.token:/opt/hermes-dock/host-bridge.token:ro",
		"      - \"host.docker.internal:host-gateway\"",
		"  dufs:",
		"    image: sigoden/dufs:v0.46.0",
		"      - \"0.0.0.0:9878:5000\"",
		"      - ./launcher/dufs/config.yaml:/etc/dufs.yaml:ro",
		"      - hermes_runtime",
		"      - file_management",
	} {
		if !strings.Contains(compose, want) {
			t.Fatalf("compose missing %q:\n%s", want, compose)
		}
	}
	if strings.Contains(compose, "init-permissions") {
		t.Fatalf("compose must not recursively chown the data directory:\n%s", compose)
	}
	if strings.Contains(compose, "entrypoint:") {
		t.Fatalf("compose must not override entrypoint:\n%s", compose)
	}
	for _, forbidden := range []string{"127.0.0.1:8642", "127.0.0.1:9119", "HERMES_DASHBOARD_BASIC_AUTH_", "install-paddleocr-deps"} {
		if strings.Contains(compose, forbidden) {
			t.Fatalf("compose must not expose Hermes native service %q:\n%s", forbidden, compose)
		}
	}
}

func TestFactoryResetPreservesDefaultSharedDirectory(t *testing.T) {
	app := newTestApp(t)
	installFakeDocker(t, false)
	sharedFile := filepath.Join(app.sharedDir(), "team", "report.txt")
	mustWriteFile(t, sharedFile, "keep\n", 0644)
	mustWriteFile(t, filepath.Join(app.dataDir(), "remove-me.txt"), "remove\n", 0644)

	if err := app.FactoryResetInstance(); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(sharedFile); err != nil || string(data) != "keep\n" {
		t.Fatalf("shared file was not preserved: %q, %v", data, err)
	}
	if fileExists(filepath.Join(app.dataDir(), "remove-me.txt")) {
		t.Fatal("profile data was not reset")
	}
}

func TestCurrentComposeMigrationDoesNotRequireRebuild(t *testing.T) {
	app := newTestApp(t)
	if err := app.ensureInstanceReady(); err != nil {
		t.Fatal(err)
	}
	state, err := app.readState()
	if err != nil {
		t.Fatal(err)
	}
	if !migrationApplied(state.Migrations, composeRuntimeMigrationID) {
		t.Fatal("current compose baseline should be recorded")
	}
	if state.NeedsRebuild {
		t.Fatal("current compose baseline should not require applying configuration")
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
	content := []byte(`services:
  init-permissions:
    image: alpine:3.22
    command: chown -R 10000:10000 /opt/data
  hermes:
    image: local/test:latest
    depends_on:
      init-permissions:
        condition: service_completed_successfully
`)
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
	if !strings.Contains(string(actual), "/etc/cont-init.d/019-patch-home-channel-prompt") {
		t.Fatalf("migrated compose missing home channel prompt patch helper:\n%s", actual)
	}
	if !strings.Contains(string(actual), "/etc/cont-init.d/020-install-dingtalk-deps") {
		t.Fatalf("migrated compose missing dingtalk dependency helper:\n%s", actual)
	}
	if !strings.Contains(string(actual), "/etc/cont-init.d/021-patch-dingtalk-media") {
		t.Fatalf("migrated compose missing dingtalk media patch helper:\n%s", actual)
	}
	if strings.Contains(string(actual), "init-permissions") {
		t.Fatalf("migrated compose still includes full data chown:\n%s", actual)
	}
	state, err := app.readState()
	if err != nil {
		t.Fatal(err)
	}
	if !state.NeedsRebuild {
		t.Fatal("compose migration should require applying the new container configuration")
	}
	if !migrationApplied(state.Migrations, composeRuntimeMigrationID) {
		t.Fatal("compose migration should be recorded")
	}
	if err := app.markRebuildApplied("applied-compose-hash"); err != nil {
		t.Fatal(err)
	}

	reopened := NewApp()
	reopened.instanceRoot = filepath.Join(home, ".hermes-dock")
	if err := reopened.ensureInstanceReady(); err != nil {
		t.Fatal(err)
	}
	reopenedState, err := reopened.readState()
	if err != nil {
		t.Fatal(err)
	}
	if reopenedState.NeedsRebuild {
		t.Fatal("applied compose migration should stay cleared after reopening")
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
		"./launcher/runtime-deps/" + runtimeDependencyBundleVersion + ":/opt/hermes-dock/runtime-deps:ro",
		"./launcher/helpers/verify-runtime-deps:/etc/cont-init.d/016-verify-runtime-deps:ro",
		"./launcher/helpers/patch-wecom-filenames:/etc/cont-init.d/017-patch-wecom-filenames:ro",
		"./launcher/helpers/install-feishu-deps:/etc/cont-init.d/018-install-feishu-deps:ro",
		"./launcher/helpers/patch-home-channel-prompt:/etc/cont-init.d/019-patch-home-channel-prompt:ro",
		"./launcher/helpers/install-dingtalk-deps:/etc/cont-init.d/020-install-dingtalk-deps:ro",
		"./launcher/helpers/patch-dingtalk-media:/etc/cont-init.d/021-patch-dingtalk-media:ro",
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
	if got := state.Backups[len(state.Backups)-1].Reason; got != "before-compose-runtime-v8-migration" {
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
	if !strings.Contains(migratedCompose, "./launcher/helpers/install-dingtalk-deps:/etc/cont-init.d/020-install-dingtalk-deps:ro") {
		t.Fatalf("migrated compose missing dingtalk helper:\n%s", migratedCompose)
	}
	if !strings.Contains(migratedCompose, "./launcher/helpers/patch-dingtalk-media:/etc/cont-init.d/021-patch-dingtalk-media:ro") {
		t.Fatalf("migrated compose missing dingtalk media patch helper:\n%s", migratedCompose)
	}
	if strings.Contains(migratedCompose, "install-paddleocr-deps") {
		t.Fatalf("migrated compose retained paddleocr startup helper:\n%s", migratedCompose)
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
	dingtalkHelper := filepath.Join(root, "launcher", "helpers", "install-dingtalk-deps")
	if err := os.Remove(dingtalkHelper); err != nil {
		t.Fatal(err)
	}
	dingtalkMediaPatch := filepath.Join(root, "launcher", "helpers", "patch-dingtalk-media")
	if err := os.Remove(dingtalkMediaPatch); err != nil {
		t.Fatal(err)
	}
	runtimeDepsVerifier := filepath.Join(root, "launcher", "helpers", "verify-runtime-deps")
	if err := os.Remove(runtimeDepsVerifier); err != nil {
		t.Fatal(err)
	}
	wecomHelper := filepath.Join(root, "launcher", "helpers", "patch-wecom-filenames")
	if err := os.Remove(wecomHelper); err != nil {
		t.Fatal(err)
	}
	homeChannelHelper := filepath.Join(root, "launcher", "helpers", "patch-home-channel-prompt")
	if err := os.Remove(homeChannelHelper); err != nil {
		t.Fatal(err)
	}
	if err := app.ensureInstanceReady(); err != nil {
		t.Fatal(err)
	}
	assertRuntimeHelpers(t, root)
}

func assertRuntimeHelpers(t *testing.T, root string) {
	t.Helper()
	assertRuntimeDepsVerifierHelper(t, root)
	assertFeishuDepsHelper(t, root)
	assertDingTalkDepsHelper(t, root)
	assertDingTalkMediaPatchHelper(t, root)
	assertWecomFilenamePatchHelper(t, root)
	assertHomeChannelPromptPatchHelper(t, root)
	assertHostctlHelper(t, root)
	assertHostBridgeToken(t, root)
}

func assertRuntimeDepsVerifierHelper(t *testing.T, root string) {
	t.Helper()
	helper := filepath.Join(root, "launcher", "helpers", "verify-runtime-deps")
	data, err := os.ReadFile(helper)
	if err != nil {
		t.Fatalf("expected verify-runtime-deps helper: %v", err)
	}
	content := string(data)
	assertUnixRuntimeHelper(t, "verify-runtime-deps", content)
	for _, want := range []string{"python-version", "uname -m", "sha256sum -c SHA256SUMS"} {
		if !strings.Contains(content, want) {
			t.Fatalf("verify-runtime-deps missing %q:\n%s", want, content)
		}
	}
}

func assertHostctlHelper(t *testing.T, root string) {
	t.Helper()
	helper := filepath.Join(root, "launcher", "helpers", "hostctl")
	data, err := os.ReadFile(helper)
	if err != nil {
		t.Fatalf("expected hostctl helper: %v", err)
	}
	content := string(data)
	assertUnixRuntimeHelper(t, "hostctl", content)
	if !strings.Contains(content, "host.docker.internal:9877") {
		t.Fatalf("hostctl helper missing Host Bridge address")
	}
	for _, command := range []string{"/v1/files/read", "/v1/clipboard/text", "/v1/processes", "/v1/screenshot", "/v1/rpa/windows", "/v1/rpa/mouse", "/v1/rpa/keyboard"} {
		if !strings.Contains(content, command) {
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
	assertUnixRuntimeHelper(t, "install-feishu-deps", content)
	for _, want := range []string{
		"lark-oapi==1.5.3",
		"qrcode==7.4.2",
		"/opt/hermes/.venv/bin/python",
		"--offline",
		"--no-index",
		"--find-links \"$DEPS/wheels\"",
		"importlib.metadata",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("install-feishu-deps missing %q:\n%s", want, content)
		}
	}
	if strings.Contains(content, "https://") {
		t.Fatalf("install-feishu-deps must not use the network:\n%s", content)
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

func assertDingTalkDepsHelper(t *testing.T, root string) {
	t.Helper()
	helper := filepath.Join(root, "launcher", "helpers", "install-dingtalk-deps")
	data, err := os.ReadFile(helper)
	if err != nil {
		t.Fatalf("expected install-dingtalk-deps helper: %v", err)
	}
	content := string(data)
	assertUnixRuntimeHelper(t, "install-dingtalk-deps", content)
	for _, want := range []string{
		"dingtalk-stream==0.24.3",
		"alibabacloud-dingtalk==2.2.42",
		"qrcode==7.4.2",
		"/opt/hermes/.venv/bin/python",
		"--offline",
		"--no-index",
		"--find-links \"$DEPS/wheels\"",
		"importlib.metadata",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("install-dingtalk-deps missing %q:\n%s", want, content)
		}
	}
	if strings.Contains(content, "https://") {
		t.Fatalf("install-dingtalk-deps must not use the network:\n%s", content)
	}
}

func assertDingTalkMediaPatchHelper(t *testing.T, root string) {
	t.Helper()
	helper := filepath.Join(root, "launcher", "helpers", "patch-dingtalk-media")
	data, err := os.ReadFile(helper)
	if err != nil {
		t.Fatalf("expected patch-dingtalk-media helper: %v", err)
	}
	content := string(data)
	assertUnixRuntimeHelper(t, "patch-dingtalk-media", content)
	for _, want := range []string{
		"/opt/hermes/gateway/platforms/dingtalk.py",
		"444915b052ae9c922fcc76708ad73acc8844e928a811b162f98e4a67b9f22d19",
		"842ee304cacec43ea3369c79d0bc10c73b7fdc207dfb5928e96183c0a5f0a04f",
		"7f2dfd4044ef536d68742a39b57b3e4df6923bdbe2830b9afacaf1bc2679263c",
		"HERMES_DOCK_DINGTALK_MEDIA_PATCH_V2",
		"HERMES_WRITE_SAFE_ROOT",
		"path.relative_to(root)",
		"https://oapi.dingtalk.com/media/upload",
		"https://api.dingtalk.com/v1.0/robot/groupMessages/send",
		"https://api.dingtalk.com/v1.0/robot/oToMessages/batchSend",
		"sampleImageMsg",
		`params={"access_token": token, "type": media_type}`,
		"async def send_document(",
		"sampleFile",
		`"mediaId": media_id`,
		`"fileName": display_name`,
		`"fileType": file_type`,
		"async def send_voice(",
		"sampleAudio",
		"async def send_video(",
		"sampleVideo",
		`"videoMediaId": video_media_id`,
		`"picMediaId": cover_media_id`,
		"发送失败，请稍后重试或改用共享文件下载。",
		"py_compile",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("patch-dingtalk-media missing %q:\n%s", want, content)
		}
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(helper)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode()&0111 == 0 {
			t.Fatalf("patch-dingtalk-media mode = %v, want executable bit", info.Mode())
		}
	}
}

func TestDingTalkMediaPatchUpgradePathsAndUploadOrdering(t *testing.T) {
	data, err := runtimeHelperFS.ReadFile("scripts/patch-dingtalk-media.sh")
	if err != nil {
		t.Fatal(err)
	}
	script := string(data)
	importNeedle := pythonAssignment(t, script, "import_needle", `"""`)
	importReplacement := pythonAssignment(t, script, "import_replacement", `"""`)
	methodNeedle := pythonAssignment(t, script, "method_needle", `'''`)
	methodReplacement := pythonAssignment(t, script, "method_replacement", `'''`)

	const getChatInfo = "    async def get_chat_info("
	base := importNeedle + "\nclass DingTalkAdapter:\n" + methodNeedle + "\n" + getChatInfo
	basePatched := strings.Replace(base, importNeedle, importReplacement, 1)
	basePatched = strings.Replace(basePatched, methodNeedle, methodReplacement, 1)
	v1 := importReplacement + "\nclass DingTalkAdapter:\n    # HERMES_DOCK_DINGTALK_IMAGE_PATCH_V1\n    old_patch = True\n" + getChatInfo
	v1Start := strings.Index(v1, "    # HERMES_DOCK_DINGTALK_IMAGE_PATCH_V1\n")
	v1End := strings.Index(v1[v1Start:], getChatInfo)
	if v1Start < 0 || v1End < 0 {
		t.Fatal("invalid V1 patch fixture")
	}
	v1End += v1Start
	v1Patched := v1[:v1Start] + methodReplacement + "\n" + v1[v1End:]
	if basePatched != v1Patched {
		t.Fatal("base and V1 upgrade paths produce different adapters")
	}
	if strings.Count(basePatched, "HERMES_DOCK_DINGTALK_MEDIA_PATCH_V2") != 1 {
		t.Fatal("patched adapter does not contain exactly one V2 marker")
	}

	for _, item := range []struct {
		name string
		next string
	}{
		{name: "send_image_file", next: "_send_dingtalk_file"},
		{name: "_send_dingtalk_file", next: "send_document"},
		{name: "send_voice", next: "send_video"},
		{name: "send_video"},
	} {
		section := pythonMethodSection(t, methodReplacement, item.name, item.next)
		targetIndex := strings.Index(section, "destination = self._dingtalk_media_target(chat_id)")
		uploadIndex := strings.Index(section, "await self._upload_dingtalk_media(")
		if targetIndex < 0 || uploadIndex < 0 || targetIndex > uploadIndex {
			t.Fatalf("%s must resolve the DingTalk target before uploading", item.name)
		}
	}
}

func pythonAssignment(t *testing.T, script string, name string, quote string) string {
	t.Helper()
	token := name + " = " + quote
	start := strings.Index(script, token)
	if start < 0 {
		t.Fatalf("missing Python assignment %s", name)
	}
	start += len(token)
	end := strings.Index(script[start:], quote)
	if end < 0 {
		t.Fatalf("unterminated Python assignment %s", name)
	}
	return script[start : start+end]
}

func pythonMethodSection(t *testing.T, source string, name string, next string) string {
	t.Helper()
	startToken := "    async def " + name + "("
	start := strings.Index(source, startToken)
	if start < 0 {
		t.Fatalf("missing Python method %s", name)
	}
	if next == "" {
		return source[start:]
	}
	end := strings.Index(source[start+len(startToken):], "\n    async def "+next+"(")
	if end < 0 {
		t.Fatalf("missing Python method %s after %s", next, name)
	}
	return source[start : start+len(startToken)+end]
}

func assertWecomFilenamePatchHelper(t *testing.T, root string) {
	t.Helper()
	helper := filepath.Join(root, "launcher", "helpers", "patch-wecom-filenames")
	data, err := os.ReadFile(helper)
	if err != nil {
		t.Fatalf("expected patch-wecom-filenames helper: %v", err)
	}
	content := string(data)
	assertUnixRuntimeHelper(t, "patch-wecom-filenames", content)
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

func assertHomeChannelPromptPatchHelper(t *testing.T, root string) {
	t.Helper()
	helper := filepath.Join(root, "launcher", "helpers", "patch-home-channel-prompt")
	data, err := os.ReadFile(helper)
	if err != nil {
		t.Fatalf("expected patch-home-channel-prompt helper: %v", err)
	}
	content := string(data)
	assertUnixRuntimeHelper(t, "patch-home-channel-prompt", content)
	for _, want := range []string{
		"/opt/hermes/gateway/run.py",
		"HERMES_DOCK_SUPPRESS_HOME_CHANNEL_PROMPT",
		"home channel prompt marker not found",
		"py_compile",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("patch-home-channel-prompt missing %q:\n%s", want, content)
		}
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(helper)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode()&0111 == 0 {
			t.Fatalf("patch-home-channel-prompt mode = %v, want executable bit", info.Mode())
		}
	}
}

func assertUnixRuntimeHelper(t *testing.T, name string, content string) {
	t.Helper()
	if !strings.HasPrefix(content, "#!") {
		t.Fatalf("%s helper missing shebang", name)
	}
	if strings.Contains(content, "\r\n") {
		t.Fatalf("%s helper contains CRLF line endings", name)
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

func TestTextFileEditorRejectsOversizedContent(t *testing.T) {
	app := NewApp()
	app.instanceRoot = t.TempDir()
	large := strings.Repeat("x", textFileLimit+1)
	path := filepath.Join(app.instanceRoot, "large.txt")
	if err := os.WriteFile(path, []byte(large), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := app.ReadTextFile("large.txt"); err == nil {
		t.Fatal("oversized file should be rejected")
	}
	if err := app.SaveTextFile(TextFileRequest{Path: "new.txt", Content: large}); err == nil {
		t.Fatal("oversized content should be rejected")
	}
}
