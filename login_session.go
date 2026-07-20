package main

import (
	"context"
	"fmt"
	"time"
)

type loginSessionState struct {
	platform  string
	profileID string
	cancel    context.CancelFunc
	done      chan struct{}
	err       error
}

func (a *App) startLoginSession(platform string, profileID string, timeout time.Duration) (context.Context, error) {
	a.loginMu.Lock()
	defer a.loginMu.Unlock()
	if a.loginSession != nil {
		return nil, fmt.Errorf("当前正在进行%s扫码绑定，请先完成或取消", loginPlatformLabel(a.loginSession.platform))
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	a.loginSession = &loginSessionState{
		platform:  platform,
		profileID: profileID,
		cancel:    cancel,
		done:      make(chan struct{}),
	}
	return ctx, nil
}

func (a *App) finishLoginSession(platform string, sessionErr error) {
	a.loginMu.Lock()
	session := a.loginSession
	if session == nil || session.platform != platform {
		a.loginMu.Unlock()
		return
	}
	session.err = sessionErr
	a.loginSession = nil
	a.loginMu.Unlock()
	session.cancel()
	close(session.done)
}

func (a *App) cancelLoginSession(platform string) {
	a.loginMu.Lock()
	session := a.loginSession
	if session == nil || (platform != "" && session.platform != platform) {
		a.loginMu.Unlock()
		return
	}
	a.loginMu.Unlock()
	session.cancel()
}

func (a *App) cancelLoginSessionAndWait(platform string) error {
	a.loginMu.Lock()
	session := a.loginSession
	if session == nil || (platform != "" && session.platform != platform) {
		a.loginMu.Unlock()
		return nil
	}
	a.loginMu.Unlock()
	session.cancel()
	<-session.done
	return session.err
}

func (a *App) cancelProfileLoginSessionAndWait(profileID string) error {
	a.loginMu.Lock()
	session := a.loginSession
	if session == nil || session.profileID != profileID {
		a.loginMu.Unlock()
		return nil
	}
	a.loginMu.Unlock()
	session.cancel()
	<-session.done
	return session.err
}

func loginPlatformLabel(platform string) string {
	switch platform {
	case "weixin":
		return "个人微信"
	case "feishu":
		return "飞书 / Lark"
	case "dingtalk":
		return "钉钉"
	default:
		return "平台"
	}
}
