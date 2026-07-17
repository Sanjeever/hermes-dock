//go:build darwin

package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

const (
	updateLaunchdLabel    = "com.qizhihe.hermes-dock.update"
	updateLaunchdPath     = "/Library/LaunchDaemons/com.qizhihe.hermes-dock.update.plist"
	updateLaunchAgentPath = "/Library/LaunchAgents/com.qizhihe.hermes-dock.update-relaunch.plist"
)

func (a *App) registerUpdateTask() error {
	if os.Geteuid() != 0 {
		return errors.New("开启自动更新需要管理员权限")
	}
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>Label</key><string>%s</string>
<key>ProgramArguments</key><array><string>%s</string><string>--scheduled-update</string><string>--instance-root</string><string>%s</string></array>
<key>StartCalendarInterval</key><dict><key>Hour</key><integer>2</integer><key>Minute</key><integer>0</integer></dict>
<key>ProcessType</key><string>Background</string>
</dict></plist>
`, xmlEscape(updateLaunchdLabel), xmlEscape(executable), xmlEscape(a.instanceRoot))
	agent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>Label</key><string>com.qizhihe.hermes-dock.update-relaunch</string>
<key>ProgramArguments</key><array><string>%s</string><string>--launch-after-update</string><string>--instance-root</string><string>%s</string></array>
<key>RunAtLoad</key><true/>
</dict></plist>
`, xmlEscape(executable), xmlEscape(a.instanceRoot))
	_ = backgroundCommand("launchctl", "bootout", "system/"+updateLaunchdLabel).Run()
	if err := atomicWriteFile(updateLaunchdPath, []byte(plist), 0644); err != nil {
		return fmt.Errorf("写入自动更新任务失败：%w", err)
	}
	if err := atomicWriteFile(updateLaunchAgentPath, []byte(agent), 0644); err != nil {
		return fmt.Errorf("写入更新后启动项失败：%w", err)
	}
	if output, err := backgroundCommand("launchctl", "bootstrap", "system", updateLaunchdPath).CombinedOutput(); err != nil {
		return fmt.Errorf("注册自动更新任务失败：%s", strings.TrimSpace(string(output)))
	}
	return nil
}

func (a *App) unregisterUpdateTask() error {
	if !fileExists(updateLaunchdPath) && !fileExists(updateLaunchAgentPath) {
		return nil
	}
	if os.Geteuid() != 0 {
		return errors.New("关闭自动更新需要管理员权限")
	}
	if output, err := backgroundCommand("launchctl", "bootout", "system/"+updateLaunchdLabel).CombinedOutput(); err != nil && !strings.Contains(string(output), "Could not find service") {
		return fmt.Errorf("停用自动更新任务失败：%s", strings.TrimSpace(string(output)))
	}
	if err := os.Remove(updateLaunchdPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.Remove(updateLaunchAgentPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (a *App) updateTaskRegistered() (bool, error) {
	if !fileExists(updateLaunchdPath) || !fileExists(updateLaunchAgentPath) {
		return false, nil
	}
	err := backgroundCommand("launchctl", "print", "system/"+updateLaunchdLabel).Run()
	return err == nil, nil
}

func scheduledRelaunchAllowed() bool {
	return false
}

func xmlEscape(value string) string {
	replacer := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&apos;")
	return replacer.Replace(value)
}
