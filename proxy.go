package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func defaultProxySettings() ProxySettings {
	return ProxySettings{
		NoProxy: "localhost,127.0.0.1,::1,host.docker.internal",
	}
}

func withProxyDefaults(settings ProxySettings) ProxySettings {
	defaults := defaultProxySettings()
	settings.HTTPProxy = strings.TrimSpace(settings.HTTPProxy)
	settings.HTTPSProxy = strings.TrimSpace(settings.HTTPSProxy)
	settings.ALLProxy = strings.TrimSpace(settings.ALLProxy)
	settings.NoProxy = strings.TrimSpace(settings.NoProxy)
	if settings.NoProxy == "" {
		settings.NoProxy = defaults.NoProxy
	}
	return settings
}

func (a *App) proxyPath() string {
	return filepath.Join(a.hermesDockDir(), "proxy.json")
}

func (a *App) readProxySettings() ProxySettings {
	data, err := os.ReadFile(a.proxyPath())
	if err != nil {
		return defaultProxySettings()
	}
	var settings ProxySettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return defaultProxySettings()
	}
	return withProxyDefaults(settings)
}

func (a *App) SaveProxySettings(settings ProxySettings) error {
	settings = withProxyDefaults(settings)
	if settings.Enabled && settings.HTTPProxy == "" && settings.HTTPSProxy == "" && settings.ALLProxy == "" {
		return errors.New("启用容器代理时，请至少填写一个代理地址")
	}
	if err := ensureDir(filepath.Dir(a.proxyPath())); err != nil {
		return err
	}
	if _, err := os.Stat(a.proxyPath()); err == nil {
		if err := a.backupFile(a.proxyPath(), "before-proxy-save"); err != nil {
			return err
		}
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(a.proxyPath(), append(data, '\n'), 0600); err != nil {
		return err
	}
	compose := a.readComposeSettings()
	if err := a.writeCompose(compose, "before-proxy-compose-save"); err != nil {
		return err
	}
	state, _ := a.readState()
	state.ComposeHash = fileSHA256(a.composePath())
	state.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	return a.writeState(state)
}
