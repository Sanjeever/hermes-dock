package main

import (
	"crypto/rand"
	"embed"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed scripts/hostctl.py scripts/install-dingtalk-deps.sh scripts/install-feishu-deps.sh scripts/install-paddleocr-deps.sh scripts/patch-home-channel-prompt.sh scripts/patch-wecom-filenames.sh scripts/verify-runtime-deps.sh
var runtimeHelperFS embed.FS

func (a *App) runtimeDepsVerifierHelperPath() string {
	return filepath.Join(a.hermesDockDir(), "helpers", "verify-runtime-deps")
}

func (a *App) feishuDepsHelperPath() string {
	return filepath.Join(a.hermesDockDir(), "helpers", "install-feishu-deps")
}

func (a *App) dingtalkDepsHelperPath() string {
	return filepath.Join(a.hermesDockDir(), "helpers", "install-dingtalk-deps")
}

func (a *App) paddleOCRDepsHelperPath() string {
	return filepath.Join(a.hermesDockDir(), "helpers", "install-paddleocr-deps")
}

func (a *App) wecomFilenamePatchHelperPath() string {
	return filepath.Join(a.hermesDockDir(), "helpers", "patch-wecom-filenames")
}

func (a *App) homeChannelPromptPatchHelperPath() string {
	return filepath.Join(a.hermesDockDir(), "helpers", "patch-home-channel-prompt")
}

func (a *App) hostctlHelperPath() string {
	return filepath.Join(a.hermesDockDir(), "helpers", "hostctl")
}

func (a *App) ensureFeishuDepsHelper() error {
	return a.ensureRuntimeHelper("scripts/install-feishu-deps.sh", a.feishuDepsHelperPath(), "飞书依赖 helper")
}

func (a *App) ensureRuntimeDepsVerifierHelper() error {
	return a.ensureRuntimeHelper("scripts/verify-runtime-deps.sh", a.runtimeDepsVerifierHelperPath(), "运行依赖校验 helper")
}

func (a *App) ensureDingTalkDepsHelper() error {
	return a.ensureRuntimeHelper("scripts/install-dingtalk-deps.sh", a.dingtalkDepsHelperPath(), "钉钉依赖 helper")
}

func (a *App) ensurePaddleOCRDepsHelper() error {
	return a.ensureRuntimeHelper("scripts/install-paddleocr-deps.sh", a.paddleOCRDepsHelperPath(), "OCR 依赖 helper")
}

func (a *App) ensureWecomFilenamePatchHelper() error {
	return a.ensureRuntimeHelper("scripts/patch-wecom-filenames.sh", a.wecomFilenamePatchHelperPath(), "企业微信文件名修复 helper")
}

func (a *App) ensureHomeChannelPromptPatchHelper() error {
	return a.ensureRuntimeHelper("scripts/patch-home-channel-prompt.sh", a.homeChannelPromptPatchHelperPath(), "默认通道提示修复 helper")
}

func (a *App) ensureHostctlHelper() error {
	return a.ensureRuntimeHelper("scripts/hostctl.py", a.hostctlHelperPath(), "宿主机控制 helper")
}

func (a *App) ensureContainerInitHelpers() error {
	if err := a.ensureHostBridgeToken(); err != nil {
		return err
	}
	if err := a.ensureRuntimeDepsVerifierHelper(); err != nil {
		return err
	}
	if err := a.ensureFeishuDepsHelper(); err != nil {
		return err
	}
	if err := a.ensureDingTalkDepsHelper(); err != nil {
		return err
	}
	if err := a.ensurePaddleOCRDepsHelper(); err != nil {
		return err
	}
	if err := a.ensureWecomFilenamePatchHelper(); err != nil {
		return err
	}
	if err := a.ensureHomeChannelPromptPatchHelper(); err != nil {
		return err
	}
	return a.ensureHostctlHelper()
}

func (a *App) ensureHostBridgeToken() error {
	path := a.hostBridgeTokenPath()
	if data, err := os.ReadFile(path); err == nil && len(data) >= 64 {
		return os.Chmod(path, 0600)
	}
	if err := ensureDir(filepath.Dir(path)); err != nil {
		return err
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return fmt.Errorf("生成宿主机控制令牌失败：%w", err)
	}
	return atomicWriteFile(path, []byte(hex.EncodeToString(raw)+"\n"), 0600)
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
