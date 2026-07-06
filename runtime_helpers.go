package main

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed scripts/install-feishu-deps.sh
var runtimeHelperFS embed.FS

func (a *App) feishuDepsHelperPath() string {
	return filepath.Join(a.hermesDockDir(), "helpers", "install-feishu-deps")
}

func (a *App) ensureFeishuDepsHelper() error {
	content, err := runtimeHelperFS.ReadFile("scripts/install-feishu-deps.sh")
	if err != nil {
		return fmt.Errorf("读取飞书依赖初始化脚本失败：%w", err)
	}
	target := a.feishuDepsHelperPath()
	if err := ensureDir(filepath.Dir(target)); err != nil {
		return fmt.Errorf("创建运行时 helper 目录失败：%w", err)
	}
	if existing, err := os.ReadFile(target); err == nil && string(existing) == string(content) {
		if err := os.Chmod(target, 0755); err != nil {
			return fmt.Errorf("设置飞书依赖 helper 可执行权限失败：%w", err)
		}
		return nil
	}
	if err := os.WriteFile(target, content, 0755); err != nil {
		return fmt.Errorf("写入飞书依赖 helper 失败：%w", err)
	}
	if err := os.Chmod(target, 0755); err != nil {
		return fmt.Errorf("设置飞书依赖 helper 可执行权限失败：%w", err)
	}
	return nil
}
