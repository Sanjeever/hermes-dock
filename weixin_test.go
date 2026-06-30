package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWeixinHelperNoiseFiltersSupervisorShutdown(t *testing.T) {
	line := "WARNING gateway.run: Shutdown context: signal=SIGTERM under_systemd=no parent_pid=126 parent_name=s6-supervise"
	if !isWeixinHelperNoise(line) {
		t.Fatalf("expected shutdown warning to be filtered")
	}
}

func TestWeixinHelperNoiseKeepsUsefulErrors(t *testing.T) {
	line := "failed to connect to Docker daemon"
	if isWeixinHelperNoise(line) {
		t.Fatalf("expected useful error to pass through")
	}
}

func TestPersistWeixinCredentialsUsesLoginProfile(t *testing.T) {
	app := NewApp()
	app.instanceRoot = t.TempDir()
	for _, dir := range []string{
		app.hermesDockDir(),
		app.profileDataDir(defaultProfileID),
		app.profileDataDir("sales"),
		app.profileDataDir("support"),
	} {
		if err := ensureDir(dir); err != nil {
			t.Fatal(err)
		}
	}
	state := defaultState()
	state.UI.LastProfile = "support"
	if err := app.writeState(state); err != nil {
		t.Fatal(err)
	}
	if err := app.writeProfileRegistry(ProfileRegistry{
		SchemaVersion: 1,
		Profiles: []ProfileEntry{
			{ID: defaultProfileID, Name: "默认助手", Enabled: true},
			{ID: "sales", Name: "Sales", Enabled: true},
			{ID: "support", Name: "Support", Enabled: true},
		},
	}); err != nil {
		t.Fatal(err)
	}

	err := app.persistWeixinCredentials("sales", weixinEvent{
		AccountID: "wx-account",
		Token:     "wx-token",
		BaseURL:   "https://weixin.example",
		UserID:    "wx-user",
	})
	if err != nil {
		t.Fatal(err)
	}

	salesEnv, err := readEnvFile(filepath.Join(app.profileDataDir("sales"), ".env"))
	if err != nil {
		t.Fatal(err)
	}
	if got := envValue(salesEnv, "WEIXIN_ACCOUNT_ID"); got != "wx-account" {
		t.Fatalf("sales WEIXIN_ACCOUNT_ID = %q, want wx-account", got)
	}
	if got := envValue(salesEnv, "WEIXIN_TOKEN"); got != "wx-token" {
		t.Fatalf("sales WEIXIN_TOKEN = %q, want wx-token", got)
	}
	if _, err := os.Stat(filepath.Join(app.profileDataDir("support"), ".env")); !os.IsNotExist(err) {
		t.Fatalf("support .env should not be created, stat err = %v", err)
	}
}
