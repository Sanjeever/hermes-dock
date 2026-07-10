package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"testing"
)

func TestRegisterFeishuBotUsesLarkDomainAndProbesBot(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case feishuRegistrationPath:
			if err := r.ParseForm(); err != nil {
				t.Fatal(err)
			}
			switch r.Form.Get("action") {
			case "init":
				_, _ = w.Write([]byte(`{"supported_auth_methods":["client_secret"]}`))
			case "begin":
				_, _ = w.Write([]byte(`{"device_code":"device-code","verification_uri_complete":"https://scan.example/path","interval":1,"expire_in":600}`))
			case "poll":
				_, _ = w.Write([]byte(`{"client_id":"cli_test","client_secret":"secret-test","user_info":{"tenant_brand":"lark"}}`))
			}
		case "/open-apis/auth/v3/tenant_access_token/internal":
			_, _ = w.Write([]byte(`{"tenant_access_token":"tenant-token"}`))
		case "/open-apis/bot/v3/info":
			if got := r.Header.Get("Authorization"); got != "Bearer tenant-token" {
				t.Fatalf("Authorization = %q", got)
			}
			_, _ = w.Write([]byte(`{"bot":{"app_name":"Lark Bot"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	originalAccounts := feishuAccountsURLs
	originalOpen := feishuOpenURLs
	feishuAccountsURLs = map[string]string{"feishu": server.URL, "lark": server.URL}
	feishuOpenURLs = map[string]string{"feishu": server.URL, "lark": server.URL}
	t.Cleanup(func() {
		feishuAccountsURLs = originalAccounts
		feishuOpenURLs = originalOpen
	})

	credentials, err := NewApp().registerFeishuBot(context.Background(), "default")
	if err != nil {
		t.Fatal(err)
	}
	if credentials.AppID != "cli_test" || credentials.AppSecret != "secret-test" {
		t.Fatalf("unexpected credentials: %#v", credentials)
	}
	if credentials.Domain != "lark" || credentials.BotName != "Lark Bot" {
		t.Fatalf("unexpected registration result: %#v", credentials)
	}
}

func TestPersistFeishuCredentialsUsesTargetProfileAndKeepsHomeChannel(t *testing.T) {
	app := NewApp()
	app.instanceRoot = t.TempDir()
	profileID := "sales"
	path := filepath.Join(app.profileDataDir(profileID), ".env")
	if err := app.saveEnvironmentTo(path, []EnvVar{{Key: "FEISHU_HOME_CHANNEL", Value: "oc_existing"}}); err != nil {
		t.Fatal(err)
	}
	if err := app.persistFeishuCredentials(profileID, feishuCredentials{AppID: "cli_test", AppSecret: "secret-test", Domain: "lark"}); err != nil {
		t.Fatal(err)
	}
	env, err := readEnvFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]string{
		"FEISHU_APP_ID":          "cli_test",
		"FEISHU_APP_SECRET":      "secret-test",
		"FEISHU_DOMAIN":          "lark",
		"FEISHU_CONNECTION_MODE": "websocket",
		"FEISHU_ALLOW_ALL_USERS": "true",
		"FEISHU_GROUP_POLICY":    "open",
		"FEISHU_HOME_CHANNEL":    "oc_existing",
	} {
		if got := envValue(env, key); got != want {
			t.Fatalf("%s = %q, want %q", key, got, want)
		}
	}
}

func TestLoginSessionAllowsOnlyOnePlatform(t *testing.T) {
	app := NewApp()
	if _, err := app.startLoginSession("weixin", feishuLoginTimeout); err != nil {
		t.Fatal(err)
	}
	if _, err := app.startLoginSession("feishu", feishuLoginTimeout); err == nil {
		t.Fatal("expected concurrent login session to be rejected")
	}
	app.cancelLoginSession("weixin")
	app.finishLoginSession("weixin")
	if _, err := app.startLoginSession("feishu", feishuLoginTimeout); err != nil {
		t.Fatal(err)
	}
}

func TestAppendFeishuQRTracking(t *testing.T) {
	got, err := url.Parse(appendFeishuQRTracking("https://scan.example/path?existing=value"))
	if err != nil {
		t.Fatal(err)
	}
	if got.Query().Get("existing") != "value" || got.Query().Get("from") != "hermes" || got.Query().Get("tp") != "hermes" {
		t.Fatalf("unexpected QR URL: %s", got.String())
	}
}
