package main

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestSaveWeComConfigNormalizesPoliciesAndClearsAllowlists(t *testing.T) {
	app := newTestAppWithDefaultEnv(t, []EnvVar{
		{Key: "WECOM_DM_POLICY", Value: "allowlist"},
		{Key: "WECOM_ALLOWED_USERS", Value: "user-a"},
		{Key: "WECOM_GROUP_POLICY", Value: "allowlist"},
		{Key: "WECOM_GROUP_ALLOWED_USERS", Value: "group-a"},
	})

	if err := app.SaveWeComConfig(WeComConfig{
		BotID:           "bot-id",
		Secret:          "secret",
		DMPolicy:        "allowlist",
		AllowedUsers:    "user-b",
		GroupPolicy:     "closed",
		GroupAllowUsers: "group-b",
	}); err != nil {
		t.Fatal(err)
	}

	env, err := readEnvFile(app.envPath())
	if err != nil {
		t.Fatal(err)
	}
	if got := envValue(env, "WECOM_DM_POLICY"); got != "closed" {
		t.Fatalf("WECOM_DM_POLICY = %q, want closed", got)
	}
	if got := envValue(env, "WECOM_GROUP_POLICY"); got != "closed" {
		t.Fatalf("WECOM_GROUP_POLICY = %q, want closed", got)
	}
	if got := envValue(env, "WECOM_ALLOWED_USERS"); got != "" {
		t.Fatalf("WECOM_ALLOWED_USERS = %q, want empty", got)
	}
	if got := envValue(env, "WECOM_GROUP_ALLOWED_USERS"); got != "" {
		t.Fatalf("WECOM_GROUP_ALLOWED_USERS = %q, want empty", got)
	}
}

func TestSaveWeComConfigPreservesExistingSecretWhenRequestIsMasked(t *testing.T) {
	app := newTestAppWithDefaultEnv(t, []EnvVar{
		{Key: "WECOM_BOT_ID", Value: "bot-id"},
		{Key: "WECOM_SECRET", Value: "real-secret"},
	})

	if err := app.SaveWeComConfig(WeComConfig{
		BotID:       "bot-id",
		Secret:      "******",
		DMPolicy:    "open",
		GroupPolicy: "open",
	}); err != nil {
		t.Fatal(err)
	}

	env, err := readEnvFile(app.envPath())
	if err != nil {
		t.Fatal(err)
	}
	if got := envValue(env, "WECOM_SECRET"); got != "real-secret" {
		t.Fatalf("WECOM_SECRET = %q, want real-secret", got)
	}
}

func TestSaveFeishuConfigNormalizesPolicyAndClearsAllowlist(t *testing.T) {
	app := newTestAppWithDefaultEnv(t, []EnvVar{
		{Key: "FEISHU_GROUP_POLICY", Value: "allowlist"},
		{Key: "FEISHU_ALLOWED_USERS", Value: "open-id-a"},
	})

	if err := app.SaveFeishuConfig(FeishuConfig{
		AppID:        "app-id",
		AppSecret:    "secret",
		Domain:       "feishu",
		AllowedUsers: "open-id-b",
		GroupPolicy:  "allowlist",
	}); err != nil {
		t.Fatal(err)
	}

	env, err := readEnvFile(app.envPath())
	if err != nil {
		t.Fatal(err)
	}
	if got := envValue(env, "FEISHU_GROUP_POLICY"); got != "disabled" {
		t.Fatalf("FEISHU_GROUP_POLICY = %q, want disabled", got)
	}
	if got := envValue(env, "FEISHU_ALLOWED_USERS"); got != "" {
		t.Fatalf("FEISHU_ALLOWED_USERS = %q, want empty", got)
	}
	if got := envValue(env, "FEISHU_ALLOW_ALL_USERS"); got != "true" {
		t.Fatalf("FEISHU_ALLOW_ALL_USERS = %q, want true", got)
	}
}

func TestSaveFeishuConfigPreservesExistingSecretWhenRequestIsRedacted(t *testing.T) {
	app := newTestAppWithDefaultEnv(t, []EnvVar{
		{Key: "FEISHU_APP_ID", Value: "app-id"},
		{Key: "FEISHU_APP_SECRET", Value: "real-secret"},
	})

	if err := app.SaveFeishuConfig(FeishuConfig{
		AppID:       "app-id",
		AppSecret:   "<redacted>",
		Domain:      "feishu",
		GroupPolicy: "open",
	}); err != nil {
		t.Fatal(err)
	}

	env, err := readEnvFile(app.envPath())
	if err != nil {
		t.Fatal(err)
	}
	if got := envValue(env, "FEISHU_APP_SECRET"); got != "real-secret" {
		t.Fatalf("FEISHU_APP_SECRET = %q, want real-secret", got)
	}
}

func TestSaveDingTalkConfigUsesOpenAccessAndPreservesMaskedSecret(t *testing.T) {
	app := newTestAppWithDefaultEnv(t, []EnvVar{
		{Key: "DINGTALK_CLIENT_ID", Value: "app-key"},
		{Key: "DINGTALK_CLIENT_SECRET", Value: "real-secret"},
		{Key: "DINGTALK_ALLOWED_USERS", Value: "user-a"},
	})

	if err := app.SaveDingTalkConfig(DingTalkConfig{
		ClientID:       "app-key",
		ClientSecret:   "******",
		RequireMention: false,
	}); err != nil {
		t.Fatal(err)
	}

	env, err := readEnvFile(app.envPath())
	if err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]string{
		"DINGTALK_CLIENT_SECRET":   "real-secret",
		"DINGTALK_ALLOW_ALL_USERS": "true",
		"DINGTALK_ALLOWED_USERS":   "",
		"DINGTALK_REQUIRE_MENTION": "false",
	} {
		if got := envValue(env, key); got != want {
			t.Fatalf("%s = %q, want %q", key, got, want)
		}
	}
}

func TestSaveDingTalkConfigStoresOptionalCardTemplateID(t *testing.T) {
	app := newTestApp(t)

	if err := app.SaveDingTalkConfig(DingTalkConfig{
		ClientID:       "app-key",
		ClientSecret:   "secret",
		RequireMention: true,
		CardTemplateID: "  card-template-1  ",
	}); err != nil {
		t.Fatal(err)
	}

	settings, err := app.readDingTalkSettingsForProfile(defaultProfileID)
	if err != nil {
		t.Fatal(err)
	}
	if settings.CardTemplateID != "card-template-1" {
		t.Fatalf("card template ID = %q", settings.CardTemplateID)
	}
	if !settings.RecommendedSettingsApplied {
		t.Fatal("saving a card template should preserve recommended settings")
	}
}

func TestSaveDingTalkConfigStoresCardTemplateWithoutCredentials(t *testing.T) {
	app := newTestApp(t)

	if err := app.SaveDingTalkConfig(DingTalkConfig{
		RequireMention: true,
		CardTemplateID: "card-template-only",
	}); err != nil {
		t.Fatal(err)
	}

	settings, err := app.readDingTalkSettingsForProfile(defaultProfileID)
	if err != nil {
		t.Fatal(err)
	}
	if settings.CardTemplateID != "card-template-only" {
		t.Fatalf("card template ID = %q", settings.CardTemplateID)
	}
}

func TestApplyRecommendedDingTalkSettingsBacksUpAndPreservesCardTemplate(t *testing.T) {
	app := newTestApp(t)
	configPath := app.profileConfigPath(defaultProfileID)
	cfg, err := app.readConfigMapForProfile(defaultProfileID)
	if err != nil {
		t.Fatal(err)
	}
	cfg["group_sessions_per_user"] = true
	cfg["custom_setting"] = "keep"
	display := asMap(cfg["display"])
	platforms := asMap(display["platforms"])
	delete(platforms, "dingtalk")
	display["platforms"] = platforms
	cfg["display"] = display
	dingTalk := asMap(asMap(cfg["platforms"])["dingtalk"])
	extra := asMap(dingTalk["extra"])
	extra["card_template_id"] = "card-template-1"
	dingTalk["extra"] = extra
	configPlatforms := asMap(cfg["platforms"])
	configPlatforms["dingtalk"] = dingTalk
	cfg["platforms"] = configPlatforms
	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := atomicWriteFile(configPath, data, 0644); err != nil {
		t.Fatal(err)
	}

	stateBefore, err := app.readState()
	if err != nil {
		t.Fatal(err)
	}
	if err := app.ApplyRecommendedDingTalkSettingsForProfile(defaultProfileID); err != nil {
		t.Fatal(err)
	}

	settings, err := app.readDingTalkSettingsForProfile(defaultProfileID)
	if err != nil {
		t.Fatal(err)
	}
	if !settings.RecommendedSettingsApplied {
		t.Fatal("recommended settings were not applied")
	}
	if settings.CardTemplateID != "card-template-1" {
		t.Fatalf("card template ID = %q", settings.CardTemplateID)
	}
	updated, err := app.readConfigMapForProfile(defaultProfileID)
	if err != nil {
		t.Fatal(err)
	}
	if asString(updated["custom_setting"]) != "keep" {
		t.Fatal("custom config was not preserved")
	}
	stateAfter, err := app.readState()
	if err != nil {
		t.Fatal(err)
	}
	if len(stateAfter.Backups) != len(stateBefore.Backups)+1 {
		t.Fatalf("backups = %d, want %d", len(stateAfter.Backups), len(stateBefore.Backups)+1)
	}
	backupPath := filepath.Join(app.instanceRoot, stateAfter.Backups[len(stateAfter.Backups)-1].Path)
	if _, err := os.Stat(backupPath); err != nil {
		t.Fatalf("recommended settings backup missing: %v", err)
	}
}

func TestUnbindPlatformClearsWeixinBinding(t *testing.T) {
	app := newTestAppWithDefaultEnv(t, []EnvVar{
		{Key: "WEIXIN_ACCOUNT_ID", Value: "wx-account"},
		{Key: "WEIXIN_TOKEN", Value: "wx-token"},
		{Key: "WEIXIN_BASE_URL", Value: "https://weixin.example"},
		{Key: "WEIXIN_HOME_CHANNEL", Value: "home-channel"},
		{Key: "WEIXIN_ALLOWED_USERS", Value: "user-a"},
		{Key: "WEIXIN_GROUP_ALLOWED_USERS", Value: "group-a"},
	})

	if err := app.UnbindPlatform("weixin"); err != nil {
		t.Fatal(err)
	}

	env, err := readEnvFile(app.envPath())
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"WEIXIN_ACCOUNT_ID", "WEIXIN_TOKEN", "WEIXIN_BASE_URL", "WEIXIN_HOME_CHANNEL", "WEIXIN_ALLOWED_USERS", "WEIXIN_GROUP_ALLOWED_USERS"} {
		if got := envValue(env, key); got != "" {
			t.Fatalf("%s = %q, want empty", key, got)
		}
	}
	if got := envValue(env, "WEIXIN_DM_POLICY"); got != "open" {
		t.Fatalf("WEIXIN_DM_POLICY = %q, want open", got)
	}
}

func TestUnbindPlatformClearsWeComBinding(t *testing.T) {
	app := newTestAppWithDefaultEnv(t, []EnvVar{
		{Key: "WECOM_BOT_ID", Value: "bot-id"},
		{Key: "WECOM_SECRET", Value: "secret"},
		{Key: "WECOM_ALLOWED_USERS", Value: "user-a"},
		{Key: "WECOM_GROUP_ALLOWED_USERS", Value: "group-a"},
	})

	if err := app.UnbindPlatform("wecom"); err != nil {
		t.Fatal(err)
	}

	env, err := readEnvFile(app.envPath())
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"WECOM_BOT_ID", "WECOM_SECRET", "WECOM_ALLOWED_USERS", "WECOM_GROUP_ALLOWED_USERS"} {
		if got := envValue(env, key); got != "" {
			t.Fatalf("%s = %q, want empty", key, got)
		}
	}
	if got := envValue(env, "WECOM_WEBSOCKET_URL"); got != "wss://openws.work.weixin.qq.com" {
		t.Fatalf("WECOM_WEBSOCKET_URL = %q", got)
	}
}

func TestUnbindPlatformClearsFeishuBinding(t *testing.T) {
	app := newTestAppWithDefaultEnv(t, []EnvVar{
		{Key: "FEISHU_APP_ID", Value: "app-id"},
		{Key: "FEISHU_APP_SECRET", Value: "secret"},
		{Key: "FEISHU_DOMAIN", Value: "lark"},
		{Key: "FEISHU_ALLOWED_USERS", Value: "user-a"},
	})

	if err := app.UnbindPlatform("feishu"); err != nil {
		t.Fatal(err)
	}

	env, err := readEnvFile(app.envPath())
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"FEISHU_APP_ID", "FEISHU_APP_SECRET", "FEISHU_ALLOWED_USERS"} {
		if got := envValue(env, key); got != "" {
			t.Fatalf("%s = %q, want empty", key, got)
		}
	}
	if got := envValue(env, "FEISHU_DOMAIN"); got != "feishu" {
		t.Fatalf("FEISHU_DOMAIN = %q, want feishu", got)
	}
	if got := envValue(env, "FEISHU_CONNECTION_MODE"); got != "websocket" {
		t.Fatalf("FEISHU_CONNECTION_MODE = %q, want websocket", got)
	}
	if got := envValue(env, "FEISHU_ALLOW_ALL_USERS"); got != "true" {
		t.Fatalf("FEISHU_ALLOW_ALL_USERS = %q, want true", got)
	}
}

func TestUnbindPlatformClearsDingTalkBinding(t *testing.T) {
	app := newTestApp(t)
	if err := app.SaveDingTalkConfig(DingTalkConfig{
		ClientID:       "app-key",
		ClientSecret:   "secret",
		RequireMention: false,
		CardTemplateID: "card-template-1",
	}); err != nil {
		t.Fatal(err)
	}
	stateBefore, err := app.readState()
	if err != nil {
		t.Fatal(err)
	}

	if err := app.UnbindPlatform("dingtalk"); err != nil {
		t.Fatal(err)
	}

	env, err := readEnvFile(app.envPath())
	if err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]string{
		"DINGTALK_CLIENT_ID":       "",
		"DINGTALK_CLIENT_SECRET":   "",
		"DINGTALK_ALLOWED_USERS":   "",
		"DINGTALK_ALLOW_ALL_USERS": "true",
		"DINGTALK_REQUIRE_MENTION": "true",
	} {
		if got := envValue(env, key); got != want {
			t.Fatalf("%s = %q, want %q", key, got, want)
		}
	}
	settings, err := app.readDingTalkSettingsForProfile(defaultProfileID)
	if err != nil {
		t.Fatal(err)
	}
	if settings.CardTemplateID != "" {
		t.Fatalf("card template ID = %q, want empty", settings.CardTemplateID)
	}
	stateAfter, err := app.readState()
	if err != nil {
		t.Fatal(err)
	}
	configBackupFound := false
	for _, backup := range stateAfter.Backups[len(stateBefore.Backups):] {
		if backup.Reason != "before-dingtalk-card-template-save" {
			continue
		}
		configBackupFound = true
		if _, err := os.Stat(filepath.Join(app.instanceRoot, backup.Path)); err != nil {
			t.Fatalf("dingtalk config backup missing: %v", err)
		}
	}
	if !configBackupFound {
		t.Fatal("dingtalk unbind did not record a config backup")
	}
}

func TestUnbindPlatformRejectsUnknownPlatform(t *testing.T) {
	app := newTestAppWithDefaultEnv(t, nil)
	if err := app.UnbindPlatform("unknown"); err == nil {
		t.Fatal("expected error for unknown platform")
	}
}

func newTestAppWithDefaultEnv(t *testing.T, vars []EnvVar) *App {
	t.Helper()
	app := NewApp()
	app.instanceRoot = t.TempDir()
	if err := ensureDir(app.defaultDataDir()); err != nil {
		t.Fatal(err)
	}
	if err := writeEnvFile(filepath.Join(app.defaultDataDir(), ".env"), vars); err != nil {
		t.Fatal(err)
	}
	return app
}
