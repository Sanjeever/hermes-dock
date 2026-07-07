package main

import (
	"path/filepath"
	"testing"
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
