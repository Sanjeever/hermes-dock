package main

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed scripts/install-feishu-deps.sh scripts/patch-wecom-filenames.sh
var runtimeHelperFS embed.FS

func (a *App) feishuDepsHelperPath() string {
	return filepath.Join(a.hermesDockDir(), "helpers", "install-feishu-deps")
}

func (a *App) wecomFilenamePatchHelperPath() string {
	return filepath.Join(a.hermesDockDir(), "helpers", "patch-wecom-filenames")
}

func (a *App) ensureFeishuDepsHelper() error {
	return a.ensureRuntimeHelper("scripts/install-feishu-deps.sh", a.feishuDepsHelperPath(), "飞书依赖 helper")
}

func (a *App) ensureWecomFilenamePatchHelper() error {
	return a.ensureRuntimeHelper("scripts/patch-wecom-filenames.sh", a.wecomFilenamePatchHelperPath(), "企业微信文件名修复 helper")
}

func (a *App) ensureContainerInitHelpers() error {
	if err := a.ensureFeishuDepsHelper(); err != nil {
		return err
	}
	return a.ensureWecomFilenamePatchHelper()
}

func (a *App) ensureRuntimeHelper(source string, target string, label string) error {
	content, err := runtimeHelperFS.ReadFile(source)
	if err != nil {
		return fmt.Errorf("读取%s失败：%w", label, err)
	}
	if err := ensureDir(filepath.Dir(target)); err != nil {
		return fmt.Errorf("创建运行时 helper 目录失败：%w", err)
	}
	if existing, err := os.ReadFile(target); err == nil && string(existing) == string(content) {
		if err := os.Chmod(target, 0755); err != nil {
			return fmt.Errorf("设置%s可执行权限失败：%w", label, err)
		}
		return nil
	}
	if err := os.WriteFile(target, content, 0755); err != nil {
		return fmt.Errorf("写入%s失败：%w", label, err)
	}
	if err := os.Chmod(target, 0755); err != nil {
		return fmt.Errorf("设置%s可执行权限失败：%w", label, err)
	}
	return nil
}
