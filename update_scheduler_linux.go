//go:build linux

package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	updateServicePath   = "/etc/systemd/system/hermes-dock-update.service"
	updateTimerPath     = "/etc/systemd/system/hermes-dock-update.timer"
	updateAutostartPath = "/etc/xdg/autostart/hermes-dock-update-relaunch.desktop"
)

func (a *App) registerUpdateTask() error {
	if os.Geteuid() != 0 {
		return errors.New("开启自动更新需要以 root 身份运行企智盒")
	}
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	service := fmt.Sprintf(`[Unit]
Description=Hermes Dock automatic update

[Service]
Type=oneshot
ExecStart=%s --scheduled-update --instance-root %s
`, systemdQuote(executable), systemdQuote(a.instanceRoot))
	timer := `[Unit]
Description=Check for Hermes Dock updates every night

[Timer]
OnCalendar=*-*-* 02:00:00
RandomizedDelaySec=30m
Persistent=true
Unit=hermes-dock-update.service

[Install]
WantedBy=timers.target
`
	autostart := fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=Hermes Dock Update Restart
Exec=%s --launch-after-update --instance-root %s
NoDisplay=true
`, desktopExecQuote(executable), desktopExecQuote(a.instanceRoot))
	if err := atomicWriteFile(updateServicePath, []byte(service), 0644); err != nil {
		return fmt.Errorf("写入自动更新服务失败：%w", err)
	}
	if err := atomicWriteFile(updateTimerPath, []byte(timer), 0644); err != nil {
		return fmt.Errorf("写入自动更新定时器失败：%w", err)
	}
	if err := atomicWriteFile(updateAutostartPath, []byte(autostart), 0644); err != nil {
		return fmt.Errorf("写入更新后启动项失败：%w", err)
	}
	if output, err := backgroundCommand("systemctl", "daemon-reload").CombinedOutput(); err != nil {
		return commandOutputError("重新加载 systemd 失败", err, output)
	}
	if output, err := backgroundCommand("systemctl", "enable", "--now", "hermes-dock-update.timer").CombinedOutput(); err != nil {
		return commandOutputError("启用自动更新定时器失败", err, output)
	}
	return nil
}

func (a *App) unregisterUpdateTask() error {
	if !fileExists(updateServicePath) && !fileExists(updateTimerPath) && !fileExists(updateAutostartPath) {
		return nil
	}
	if os.Geteuid() != 0 {
		return errors.New("关闭自动更新需要以 root 身份运行企智盒")
	}
	if output, err := backgroundCommand("systemctl", "disable", "--now", "hermes-dock-update.timer").CombinedOutput(); err != nil {
		return commandOutputError("停用自动更新定时器失败", err, output)
	}
	if err := os.Remove(updateTimerPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.Remove(updateServicePath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.Remove(updateAutostartPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if output, err := backgroundCommand("systemctl", "daemon-reload").CombinedOutput(); err != nil {
		return commandOutputError("重新加载 systemd 失败", err, output)
	}
	return nil
}

func (a *App) updateTaskRegistered() (bool, error) {
	if !fileExists(updateServicePath) || !fileExists(updateTimerPath) || !fileExists(updateAutostartPath) {
		return false, nil
	}
	err := backgroundCommand("systemctl", "is-enabled", "--quiet", "hermes-dock-update.timer").Run()
	return err == nil, nil
}

func scheduledRelaunchAllowed() bool {
	return false
}

func systemdQuote(value string) string {
	value = filepath.Clean(value)
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	return `"` + value + `"`
}

func desktopExecQuote(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	value = strings.ReplaceAll(value, `%`, `%%`)
	return `"` + value + `"`
}
