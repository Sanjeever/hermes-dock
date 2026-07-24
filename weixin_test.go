package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

func TestPersistWeixinCredentialsDoesNotOverwriteUnreadableEnv(t *testing.T) {
	app := newTestApp(t)
	original := strings.Repeat("x", 70*1024)
	if err := os.WriteFile(app.envPath(), []byte(original), 0600); err != nil {
		t.Fatal(err)
	}
	err := app.persistWeixinCredentials(defaultProfileID, weixinEvent{AccountID: "account", Token: "secret"})
	if err == nil {
		t.Fatal("invalid .env should block credential save")
	}
	saved, readErr := os.ReadFile(app.envPath())
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(saved) != original {
		t.Fatal("invalid .env was overwritten")
	}
}

func TestRemoveWeixinLoginContainerConfirmsContainerIsGone(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake docker script is unix-only")
	}
	for _, tt := range []struct {
		name      string
		psOutput  string
		wantError bool
	}{
		{name: "already absent", psOutput: "", wantError: false},
		{name: "still present", psOutput: "container-id", wantError: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			dockerPath := filepath.Join(dir, "docker")
			script := "#!/bin/sh\n" +
				"if [ \"$1\" = \"rm\" ]; then exit 1; fi\n" +
				"if [ \"$1\" = \"ps\" ]; then printf '%s' \"$PS_OUTPUT\"; exit 0; fi\n" +
				"exit 1\n"
			if err := os.WriteFile(dockerPath, []byte(script), 0755); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
			t.Setenv("PS_OUTPUT", tt.psOutput)
			err := removeWeixinLoginContainer("test-container")
			if (err != nil) != tt.wantError {
				t.Fatalf("removeWeixinLoginContainer() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}
