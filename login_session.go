package main

import (
	"context"
	"fmt"
	"time"
)

func (a *App) startLoginSession(platform string, timeout time.Duration) (context.Context, error) {
	a.loginMu.Lock()
	defer a.loginMu.Unlock()
	if a.loginCancel != nil {
		return nil, fmt.Errorf("当前正在进行%s扫码绑定，请先完成或取消", loginPlatformLabel(a.loginPlatform))
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	a.loginCancel = cancel
	a.loginPlatform = platform
	return ctx, nil
}

func (a *App) finishLoginSession(platform string) {
	a.loginMu.Lock()
	if a.loginPlatform != platform {
		a.loginMu.Unlock()
		return
	}
	cancel := a.loginCancel
	a.loginCancel = nil
	a.loginPlatform = ""
	a.loginMu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (a *App) cancelLoginSession(platform string) {
	a.loginMu.Lock()
	defer a.loginMu.Unlock()
	if a.loginCancel == nil || (platform != "" && a.loginPlatform != platform) {
		return
	}
	a.loginCancel()
}

func loginPlatformLabel(platform string) string {
	switch platform {
	case "weixin":
		return "个人微信"
	case "feishu":
		return "飞书 / Lark"
	default:
		return "平台"
	}
}
