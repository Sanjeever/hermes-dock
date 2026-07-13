package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveProxySettingsWritesConfigAndCompose(t *testing.T) {
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

	settings := ProxySettings{
		Enabled:    true,
		HTTPProxy:  "http://host.docker.internal:7890",
		HTTPSProxy: "http://host.docker.internal:7890",
		NoProxy:    "localhost,127.0.0.1,host.docker.internal",
	}
	if err := app.SaveProxySettings(settings); err != nil {
		t.Fatal(err)
	}

	proxyPath := filepath.Join(home, ".hermes-dock", "launcher", "proxy.json")
	if _, err := os.Stat(proxyPath); err != nil {
		t.Fatalf("expected proxy config: %v", err)
	}
	compose, err := os.ReadFile(filepath.Join(home, ".hermes-dock", "docker-compose.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(compose)
	for _, want := range []string{
		`extra_hosts:`,
		`"host.docker.internal:host-gateway"`,
		`HTTP_PROXY: "http://host.docker.internal:7890"`,
		`HTTPS_PROXY: "http://host.docker.internal:7890"`,
		`NO_PROXY: "localhost,127.0.0.1,host.docker.internal"`,
		`http_proxy: "http://host.docker.internal:7890"`,
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("compose missing %q:\n%s", want, content)
		}
	}
}

func TestSaveProxySettingsBacksUpExistingConfig(t *testing.T) {
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
	first := ProxySettings{Enabled: true, HTTPProxy: "http://host.docker.internal:7890"}
	if err := app.SaveProxySettings(first); err != nil {
		t.Fatal(err)
	}
	backupsBefore := backupCount(t, app)

	second := ProxySettings{Enabled: true, HTTPProxy: "http://host.docker.internal:7891"}
	if err := app.SaveProxySettings(second); err != nil {
		t.Fatal(err)
	}
	state, err := app.readState()
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Backups) != backupsBefore+2 {
		t.Fatalf("backup count = %d, want %d", len(state.Backups), backupsBefore+2)
	}
	if got := state.Backups[len(state.Backups)-2].Reason; got != "before-proxy-save" {
		t.Fatalf("proxy backup reason = %q", got)
	}
}
