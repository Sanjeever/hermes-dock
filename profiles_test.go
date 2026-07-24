package main

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func newTestApp(t *testing.T) *App {
	t.Helper()
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

func TestRuntimeManifestUsesUniqueGeneration(t *testing.T) {
	app := newTestApp(t)
	registry, err := app.readProfileRegistry()
	if err != nil {
		t.Fatal(err)
	}
	first, err := app.buildRuntimeManifest(registry)
	if err != nil {
		t.Fatal(err)
	}
	second, err := app.buildRuntimeManifest(registry)
	if err != nil {
		t.Fatal(err)
	}
	if first.Generation == "" || second.Generation == "" {
		t.Fatal("runtime manifest generation must not be empty")
	}
	if first.Generation == second.Generation {
		t.Fatalf("runtime manifest generation was reused: %q", first.Generation)
	}
}

func TestRuntimeStatusReadyRejectsPreviousGeneration(t *testing.T) {
	manifest := RuntimeManifest{
		Generation: "current",
		Profiles: []RuntimeManifestProfile{{
			ID:       "sales",
			Name:     "销售助手",
			Runnable: true,
		}},
	}
	ready, err := runtimeStatusReady(manifest, RuntimeStatus{
		Generation: "previous",
		Profiles: map[string]RuntimeProfileStatus{
			"sales": {State: "exited"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if ready {
		t.Fatal("previous runtime generation must not be accepted")
	}

	ready, err = runtimeStatusReady(manifest, RuntimeStatus{
		Generation: "current",
		Profiles: map[string]RuntimeProfileStatus{
			"sales": {State: "running"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !ready {
		t.Fatal("current running generation should be ready")
	}
}

func TestReadRuntimeStatusDoesNotExposePreviousGenerationExit(t *testing.T) {
	app := newTestApp(t)
	env, err := readEnvFile(app.defaultEnvPath())
	if err != nil {
		t.Fatal(err)
	}
	if err := writeEnvFile(app.defaultEnvPath(), mergeEnv(env, []EnvVar{
		{Key: "WECOM_BOT_ID", Value: "bot-id"},
		{Key: "WECOM_SECRET", Value: "secret"},
	})); err != nil {
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
	manifestData, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := atomicWriteFile(app.runtimeManifestPath(), manifestData, 0644); err != nil {
		t.Fatal(err)
	}
	statusData, err := json.Marshal(RuntimeStatus{
		Generation: "previous",
		UpdatedAt:  time.Now().UTC().Format(time.RFC3339),
		Profiles: map[string]RuntimeProfileStatus{
			"default": {Enabled: true, State: "exited"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := atomicWriteFile(app.runtimeStatusPath(), statusData, 0644); err != nil {
		t.Fatal(err)
	}

	status := app.readRuntimeStatus("running").Profiles["default"]
	if status.State != "starting" {
		t.Fatalf("derived state = %q, want starting", status.State)
	}
}

func TestRuntimeStatusReadyAcceptsNoRunnableProfiles(t *testing.T) {
	manifest := RuntimeManifest{
		Generation: "current",
		Profiles: []RuntimeManifestProfile{{
			ID:       "default",
			Enabled:  true,
			Runnable: false,
		}},
	}
	ready, err := runtimeStatusReady(manifest, RuntimeStatus{
		Generation: "current",
		Profiles: map[string]RuntimeProfileStatus{
			"default": {State: "not_configured"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !ready {
		t.Fatal("current generation with no runnable profiles should be ready")
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
	configMap := map[string]interface{}{}
	if err := parseYAMLFile(filepath.Join(dir, "config.yaml"), &configMap); err != nil {
		t.Fatal(err)
	}
	if !asBool(asMap(configMap["streaming"])["enabled"]) {
		t.Fatal("gateway streaming should be enabled")
	}
	platforms := asMap(asMap(configMap["display"])["platforms"])
	if !asBool(asMap(platforms["feishu"])["streaming"]) {
		t.Fatal("feishu streaming should be enabled")
	}
	if asBool(asMap(platforms["weixin"])["streaming"]) {
		t.Fatal("weixin streaming should be disabled")
	}
	if asBool(asMap(platforms["wecom"])["streaming"]) {
		t.Fatal("wecom streaming should be disabled")
	}
	if groupSessionsPerUser, ok := configMap["group_sessions_per_user"].(bool); !ok || groupSessionsPerUser {
		t.Fatal("group sessions should be shared by default")
	}
	dingTalkDisplay := asMap(platforms["dingtalk"])
	if !asBool(dingTalkDisplay["show_reasoning"]) {
		t.Fatal("dingtalk reasoning should be shown")
	}
	if !asBool(dingTalkDisplay["streaming"]) {
		t.Fatal("dingtalk streaming should be enabled")
	}
	if asString(dingTalkDisplay["tool_progress"]) != "off" {
		t.Fatal("dingtalk tool progress should be disabled")
	}
	if interim, ok := dingTalkDisplay["interim_assistant_messages"].(bool); !ok || interim {
		t.Fatal("dingtalk interim assistant messages should be disabled")
	}
	soul, err := os.ReadFile(filepath.Join(dir, "SOUL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(soul), "/opt/data/profiles/sales/tmp") {
		t.Fatalf("SOUL tmp path not rewritten")
	}
	if !strings.Contains(string(soul), "MEDIA:/opt/data/profiles/sales/") {
		t.Fatalf("SOUL media delivery path not rewritten")
	}
	if !strings.Contains(string(soul), "/opt/data/.dock/shared") || strings.Contains(string(soul), "/opt/data/profiles/sales/.dock/shared") {
		t.Fatalf("SOUL shared directory path was rewritten for the profile")
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

func TestCreateProfileValidatesCopyModeBeforeCreatingDirectory(t *testing.T) {
	app := newTestApp(t)
	id := "invalid-mode"
	if err := app.CreateProfile(CreateProfileRequest{ID: id, CopyMode: "unsupported"}); err == nil {
		t.Fatal("unsupported copy mode should fail")
	}
	if fileExists(app.profileDataDir(id)) {
		t.Fatal("failed profile creation left a residual directory")
	}
}

func TestCreateProfileValidatesCopySourceBeforeCreatingDirectory(t *testing.T) {
	app := newTestApp(t)
	id := "missing-source"
	if err := app.CreateProfile(CreateProfileRequest{ID: id, CopyMode: profileCopyPersona, CopyFrom: "not-found"}); err == nil {
		t.Fatal("missing copy source should fail")
	}
	if fileExists(app.profileDataDir(id)) {
		t.Fatal("failed profile creation left a residual directory")
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
	if !strings.Contains(string(soul), "/opt/data/.dock/shared") || strings.Contains(string(soul), "/opt/data/profiles/writer/.dock/shared") {
		t.Fatalf("copied SOUL shared directory path was rewritten")
	}
}

func TestDeleteProfileStopsContainerAndBacksUpSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink test is unix-only")
	}
	app := newTestApp(t)
	if err := app.CreateProfile(CreateProfileRequest{ID: "guest-002", Name: "访客", Enabled: true, CopyMode: "clean"}); err != nil {
		t.Fatal(err)
	}
	dir := app.profileDataDir("guest-002")
	if err := os.WriteFile(filepath.Join(dir, "python3"), []byte("python"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("python3", filepath.Join(dir, "python")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/container-only/python3", filepath.Join(dir, "python-container")); err != nil {
		t.Fatal(err)
	}
	installFakeDocker(t, true)

	if err := app.DeleteProfile("guest-002"); err != nil {
		t.Fatal(err)
	}
	if fileExists(dir) {
		t.Fatalf("profile directory still exists")
	}
	registry, err := app.readProfileRegistry()
	if err != nil {
		t.Fatal(err)
	}
	if profileExists(registry, "guest-002") {
		t.Fatalf("deleted profile remains in registry")
	}
	logData, err := os.ReadFile(fakeDockerLogPath(t))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(logData), "compose stop") {
		t.Fatalf("docker compose stop was not called: %s", logData)
	}
	if strings.Contains(string(logData), "compose start") || strings.Contains(string(logData), "compose up") {
		t.Fatalf("container was restarted during profile deletion: %s", logData)
	}

	state, err := app.readState()
	if err != nil {
		t.Fatal(err)
	}
	var backupPath string
	for _, backup := range state.Backups {
		if backup.Reason == "before-profile-delete-guest-002" {
			backupPath = filepath.Join(app.instanceRoot, filepath.FromSlash(backup.Path))
		}
	}
	if backupPath == "" {
		t.Fatalf("profile backup was not recorded")
	}
	file, err := os.Open(backupPath)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	gz, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	found := map[string]string{}
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if header.Typeflag == tar.TypeSymlink {
			found[filepath.Base(header.Name)] = header.Linkname
		}
	}
	if found["python"] != "python3" {
		t.Fatalf("python symlink target = %q", found["python"])
	}
	if found["python-container"] != "/container-only/python3" {
		t.Fatalf("broken symlink target = %q", found["python-container"])
	}
}

func TestDeleteProfileWaitsForProfileLoginWorker(t *testing.T) {
	app := newTestApp(t)
	if err := app.CreateProfile(CreateProfileRequest{ID: "scanner", Name: "扫码助手", Enabled: true, CopyMode: "clean"}); err != nil {
		t.Fatal(err)
	}
	installFakeDocker(t, false)
	ctx, err := app.startLoginSession("weixin", "scanner", feishuLoginTimeout)
	if err != nil {
		t.Fatal(err)
	}
	canceled := make(chan struct{})
	release := make(chan struct{})
	go func() {
		<-ctx.Done()
		close(canceled)
		<-release
		app.finishLoginSession("weixin", nil)
	}()
	result := make(chan error, 1)
	go func() { result <- app.DeleteProfile("scanner") }()
	<-canceled
	select {
	case err := <-result:
		t.Fatalf("DeleteProfile returned before the login worker exited: %v", err)
	default:
	}
	if !fileExists(app.profileDataDir("scanner")) {
		t.Fatal("profile directory was moved before the login worker exited")
	}
	close(release)
	if err := <-result; err != nil {
		t.Fatal(err)
	}
	if fileExists(app.profileDataDir("scanner")) {
		t.Fatal("profile directory still exists after deletion")
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

func TestValidateRuntimeProfilesRejectsDuplicateDingTalkAppKey(t *testing.T) {
	app := newTestApp(t)
	for _, id := range []string{"sales", "support"} {
		if err := app.CreateProfile(CreateProfileRequest{ID: id, Name: id, Enabled: true, CopyMode: "clean"}); err != nil {
			t.Fatal(err)
		}
		envPath := filepath.Join(app.profileDataDir(id), ".env")
		if err := writeEnvFile(envPath, mergeEnv(defaultEnvVars(), []EnvVar{
			{Key: "DINGTALK_CLIENT_ID", Value: "app-key"},
			{Key: "DINGTALK_CLIENT_SECRET", Value: "secret"},
		})); err != nil {
			t.Fatal(err)
		}
	}
	registry, err := app.readProfileRegistry()
	if err != nil {
		t.Fatal(err)
	}
	if err := app.validateRuntimeProfiles(registry); err == nil || !strings.Contains(err.Error(), "钉钉 AppKey") {
		t.Fatalf("expected duplicate dingtalk error, got %v", err)
	}
}

func TestBuildRuntimeManifestIncludesDingTalkBinding(t *testing.T) {
	app := newTestApp(t)
	env, err := readEnvFile(app.defaultEnvPath())
	if err != nil {
		t.Fatal(err)
	}
	if err := writeEnvFile(app.defaultEnvPath(), mergeEnv(env, []EnvVar{
		{Key: "DINGTALK_CLIENT_ID", Value: "app-key"},
		{Key: "DINGTALK_CLIENT_SECRET", Value: "secret"},
	})); err != nil {
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
	if len(manifest.Profiles) != 1 || !manifest.Profiles[0].Runnable {
		t.Fatalf("dingtalk profile should be runnable: %#v", manifest.Profiles)
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
