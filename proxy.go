package main

import (
	"encoding/json"
	"errors"
	"fmt"
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
	if _, err := a.readState(); err != nil {
		return fmt.Errorf("读取启动器状态失败：%w", err)
	}
	if settings == a.readProxySettings() {
		return nil
	}
	previousProxy, proxyExisted, err := readFileSnapshot(a.proxyPath())
	if err != nil {
		return err
	}
	previousCompose, composeExisted, err := readFileSnapshot(a.composePath())
	if err != nil {
		return err
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
	if err := atomicWriteFile(a.proxyPath(), append(data, '\n'), 0600); err != nil {
		return err
	}
	rollback := func(cause error) error {
		proxyErr := restoreFileSnapshot(a.proxyPath(), previousProxy, proxyExisted, 0600)
		composeErr := restoreFileSnapshot(a.composePath(), previousCompose, composeExisted, 0644)
		return errors.Join(cause, proxyErr, composeErr)
	}
	compose, err := a.readComposeSettings()
	if err != nil {
		return rollback(err)
	}
	if err := a.writeCompose(compose, "before-proxy-compose-save"); err != nil {
		return rollback(err)
	}
	if err := a.updateState(func(state *LauncherState) error {
		state.ComposeHash = fileSHA256(a.composePath())
		state.NeedsRebuild = true
		state.PendingDufsOnly = false
		state.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		return nil
	}); err != nil {
		return rollback(err)
	}
	return nil
}

func readFileSnapshot(path string) ([]byte, bool, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	return data, err == nil, err
}

func restoreFileSnapshot(path string, data []byte, existed bool, mode os.FileMode) error {
	if existed {
		return atomicWriteFile(path, data, mode)
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
