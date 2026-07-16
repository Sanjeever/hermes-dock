package main

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type weixinEvent struct {
	Type      string `json:"type"`
	ProfileID string `json:"profile_id,omitempty"`
	ScanData  string `json:"scan_data,omitempty"`
	Status    string `json:"status,omitempty"`
	AccountID string `json:"account_id,omitempty"`
	Token     string `json:"token,omitempty"`
	BaseURL   string `json:"base_url,omitempty"`
	UserID    string `json:"user_id,omitempty"`
	Message   string `json:"message,omitempty"`
}

func (a *App) StartWeixinLogin() error {
	if err := ensureDir(a.currentProfileDataDir()); err != nil {
		return err
	}
	ctx, err := a.startLoginSession("weixin", 10*time.Minute)
	if err != nil {
		return err
	}
	helperPath, err := a.writeWeixinHelper()
	if err != nil {
		a.finishLoginSession("weixin")
		return err
	}
	settings := a.readComposeSettings()
	profileID := a.currentProfileID()
	go a.runWeixinLogin(ctx, helperPath, settings.Image, profileID)
	return nil
}

func (a *App) CancelWeixinLogin() {
	a.cancelLoginSession("weixin")
}

func (a *App) runWeixinLogin(ctx context.Context, helperPath string, image string, profileID string) {
	defer a.finishLoginSession("weixin")
	profileHome := "/opt/data"
	if profileID != defaultProfileID {
		profileHome = "/opt/data/profiles/" + profileID
	}
	args := []string{
		"run", "--rm",
		"-v", a.dataDir() + ":/opt/data",
		"-v", helperPath + ":/opt/hermes-dock/weixin_login.py:ro",
		"-e", "HERMES_HOME=/opt/data",
		"-e", "HERMES_DOCK_PROFILE=" + profileID,
		"-e", "HERMES_DOCK_PROFILE_HOME=" + profileHome,
		image,
		"python", "/opt/hermes-dock/weixin_login.py",
	}
	cmd := backgroundCommandContext(ctx, "docker", args...)
	cmd.Dir = a.instanceRoot
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		a.emit("weixin-login:error", map[string]string{"profile_id": profileID, "message": err.Error()})
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		a.emit("weixin-login:error", map[string]string{"profile_id": profileID, "message": err.Error()})
		return
	}
	if err := cmd.Start(); err != nil {
		a.emit("weixin-login:error", map[string]string{"profile_id": profileID, "message": err.Error()})
		return
	}

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			a.forwardWeixinHelperLine(scanner.Text())
		}
	}()

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		var event weixinEvent
		line := scanner.Bytes()
		if err := json.Unmarshal(line, &event); err != nil {
			a.forwardWeixinHelperLine(string(line))
			continue
		}
		event.ProfileID = profileID
		switch event.Type {
		case "confirmed":
			if err := a.persistWeixinCredentials(profileID, event); err != nil {
				a.emit("weixin-login:error", map[string]string{"profile_id": profileID, "message": err.Error()})
				continue
			}
			a.emit("weixin-login:status", map[string]string{"profile_id": profileID, "status": "微信配置已保存，请应用配置后生效"})
			event.Token = ""
			a.emit("weixin-login:confirmed", event)
		case "qr_ready":
			a.emit("weixin-login:qr", event)
		case "error":
			a.emit("weixin-login:error", map[string]string{"profile_id": profileID, "message": redact(event.Message)})
		default:
			a.emit("weixin-login:status", event)
		}
	}
	if err := cmd.Wait(); err != nil && ctx.Err() == nil {
		a.emit("weixin-login:error", map[string]string{"profile_id": profileID, "message": redact(err.Error())})
	}
}

func (a *App) forwardWeixinHelperLine(line string) {
	line = strings.TrimSpace(line)
	if line == "" || isWeixinHelperNoise(line) {
		return
	}
	a.emit("docker:progress", StreamEvent{Line: redact(line)})
}

func isWeixinHelperNoise(line string) bool {
	noisePrefixes := []string{
		"/package/admin/s6-overlay/",
		"s6-rc:",
		"cont-init:",
		"[stage2]",
		"[supervise-perms]",
		"Syncing bundled skills",
		"Done:",
		"→ gateway is now running",
		"dashboard supervised alongside",
		"gateway will keep running",
		"Use `--no-supervise`",
		"HERMES_DASHBOARD_READY",
		"Hermes Web UI",
		"WARNING gateway.run: Shutdown context:",
	}
	for _, prefix := range noisePrefixes {
		if strings.HasPrefix(line, prefix) {
			return true
		}
	}
	noiseContains := []string{
		"Hermes Gateway Starting",
		"Messaging platforms + cron scheduler",
		"Press Ctrl+C to stop",
		"ocr-and-documents: bundled version shipped",
	}
	for _, fragment := range noiseContains {
		if strings.Contains(line, fragment) {
			return true
		}
	}
	return false
}

func (a *App) persistWeixinCredentials(profileID string, event weixinEvent) error {
	path := filepath.Join(a.profileDataDir(profileID), ".env")
	env, _ := readEnvFile(path)
	updates := []EnvVar{
		{Key: "WEIXIN_ACCOUNT_ID", Value: event.AccountID},
		{Key: "WEIXIN_TOKEN", Value: event.Token},
		{Key: "WEIXIN_BASE_URL", Value: event.BaseURL},
		{Key: "WEIXIN_CDN_BASE_URL", Value: "https://novac2c.cdn.weixin.qq.com/c2c"},
		{Key: "WEIXIN_DM_POLICY", Value: "open"},
		{Key: "WEIXIN_ALLOW_ALL_USERS", Value: "true"},
		{Key: "WEIXIN_ALLOWED_USERS", Value: ""},
		{Key: "WEIXIN_GROUP_POLICY", Value: "open"},
		{Key: "WEIXIN_GROUP_ALLOWED_USERS", Value: ""},
		{Key: "WEIXIN_HOME_CHANNEL", Value: event.UserID},
	}
	return a.saveEnvironmentTo(path, mergeEnv(env, updates))
}

func (a *App) writeWeixinHelper() (string, error) {
	dir := filepath.Join(a.hermesDockDir(), "helpers")
	if err := ensureDir(dir); err != nil {
		return "", err
	}
	path := filepath.Join(dir, "weixin_login.py")
	return path, os.WriteFile(path, []byte(weixinLoginHelper), 0644)
}

const weixinLoginHelper = `import asyncio
import json
import os
import sys
import time

from gateway.platforms.weixin import (
    EP_GET_BOT_QR,
    EP_GET_QR_STATUS,
    ILINK_BASE_URL,
    QR_TIMEOUT_MS,
    _api_get,
    _make_ssl_connector,
    save_weixin_account,
)

import aiohttp


def emit(payload):
    print(json.dumps(payload, ensure_ascii=False), flush=True)


async def main():
    hermes_home = os.environ.get("HERMES_DOCK_PROFILE_HOME") or "/opt/data"
    bot_type = "3"
    timeout_seconds = 480
    async with aiohttp.ClientSession(trust_env=True, connector=_make_ssl_connector()) as session:
        qr_resp = await _api_get(
            session,
            base_url=ILINK_BASE_URL,
            endpoint=f"{EP_GET_BOT_QR}?bot_type={bot_type}",
            timeout_ms=QR_TIMEOUT_MS,
        )
        qrcode_value = str(qr_resp.get("qrcode") or "")
        qrcode_url = str(qr_resp.get("qrcode_img_content") or "")
        if not qrcode_value:
            emit({"type": "error", "message": "二维码响应缺少 qrcode"})
            return 1
        emit({"type": "qr_ready", "scan_data": qrcode_url or qrcode_value, "status": "waiting"})
        deadline = time.monotonic() + timeout_seconds
        current_base_url = ILINK_BASE_URL
        refresh_count = 0
        last_status = ""
        while time.monotonic() < deadline:
            try:
                status_resp = await _api_get(
                    session,
                    base_url=current_base_url,
                    endpoint=f"{EP_GET_QR_STATUS}?qrcode={qrcode_value}",
                    timeout_ms=QR_TIMEOUT_MS,
                )
            except asyncio.TimeoutError:
                await asyncio.sleep(1)
                continue
            except Exception as exc:
                emit({"type": "status", "status": "poll_error", "message": str(exc)})
                await asyncio.sleep(1)
                continue
            status = str(status_resp.get("status") or "wait")
            if status != last_status:
                emit({"type": "status", "status": status})
                last_status = status
            if status == "scaned_but_redirect":
                redirect_host = str(status_resp.get("redirect_host") or "")
                if redirect_host:
                    current_base_url = f"https://{redirect_host}"
            elif status == "expired":
                refresh_count += 1
                if refresh_count > 3:
                    emit({"type": "error", "message": "二维码多次过期，请重新开始扫码登录"})
                    return 1
                qr_resp = await _api_get(
                    session,
                    base_url=ILINK_BASE_URL,
                    endpoint=f"{EP_GET_BOT_QR}?bot_type={bot_type}",
                    timeout_ms=QR_TIMEOUT_MS,
                )
                qrcode_value = str(qr_resp.get("qrcode") or "")
                qrcode_url = str(qr_resp.get("qrcode_img_content") or "")
                emit({"type": "qr_ready", "scan_data": qrcode_url or qrcode_value, "status": "waiting"})
            elif status == "confirmed":
                account_id = str(status_resp.get("ilink_bot_id") or "")
                token = str(status_resp.get("bot_token") or "")
                base_url = str(status_resp.get("baseurl") or ILINK_BASE_URL)
                user_id = str(status_resp.get("ilink_user_id") or "")
                if not account_id or not token:
                    emit({"type": "error", "message": "扫码已确认，但返回凭据不完整"})
                    return 1
                save_weixin_account(
                    hermes_home,
                    account_id=account_id,
                    token=token,
                    base_url=base_url,
                    user_id=user_id,
                )
                emit({
                    "type": "confirmed",
                    "account_id": account_id,
                    "token": token,
                    "base_url": base_url,
                    "user_id": user_id,
                })
                return 0
            await asyncio.sleep(1)
    emit({"type": "error", "message": "扫码登录超时"})
    return 1


if __name__ == "__main__":
    try:
        raise SystemExit(asyncio.run(main()))
    except Exception as exc:
        emit({"type": "error", "message": str(exc)})
        raise
`
