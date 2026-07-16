package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"
)

const (
	feishuRegistrationPath = "/oauth/v1/app/registration"
	feishuLoginTimeout     = 10 * time.Minute
)

var feishuAccountsURLs = map[string]string{
	"feishu": "https://accounts.feishu.cn",
	"lark":   "https://accounts.larksuite.com",
}

var feishuOpenURLs = map[string]string{
	"feishu": "https://open.feishu.cn",
	"lark":   "https://open.larksuite.com",
}

type feishuLoginEvent struct {
	Type      string `json:"type"`
	ProfileID string `json:"profile_id,omitempty"`
	ScanData  string `json:"scan_data,omitempty"`
	Status    string `json:"status,omitempty"`
	Message   string `json:"message,omitempty"`
	Domain    string `json:"domain,omitempty"`
	BotName   string `json:"bot_name,omitempty"`
}

type feishuCredentials struct {
	AppID     string
	AppSecret string
	Domain    string
	BotName   string
}

func (a *App) StartFeishuLogin() error {
	if err := ensureDir(a.currentProfileDataDir()); err != nil {
		return err
	}
	ctx, err := a.startLoginSession("feishu", feishuLoginTimeout)
	if err != nil {
		return err
	}
	profileID := a.currentProfileID()
	go a.runFeishuLogin(ctx, profileID)
	return nil
}

func (a *App) CancelFeishuLogin() {
	a.cancelLoginSession("feishu")
}

func (a *App) runFeishuLogin(ctx context.Context, profileID string) {
	defer a.finishLoginSession("feishu")
	credentials, err := a.registerFeishuBot(ctx, profileID)
	if err != nil {
		message := "飞书扫码绑定失败"
		switch ctx.Err() {
		case context.Canceled:
			message = "已取消飞书扫码绑定"
		case context.DeadlineExceeded:
			message = "飞书扫码绑定超时，请重新开始"
		default:
			message = redact(err.Error())
		}
		a.emit("feishu-login:error", feishuLoginEvent{Type: "error", ProfileID: profileID, Message: message})
		return
	}
	if err := a.persistFeishuCredentials(profileID, credentials); err != nil {
		a.emit("feishu-login:error", feishuLoginEvent{Type: "error", ProfileID: profileID, Message: redact(err.Error())})
		return
	}
	message := "飞书配置已保存，请应用配置后生效"
	if credentials.BotName == "" {
		message = "已绑定，暂时无法读取机器人名称；应用配置后验证运行状态"
	}
	a.emit("feishu-login:confirmed", feishuLoginEvent{
		Type:      "confirmed",
		ProfileID: profileID,
		Status:    message,
		Domain:    credentials.Domain,
		BotName:   credentials.BotName,
	})
}

func (a *App) registerFeishuBot(ctx context.Context, profileID string) (feishuCredentials, error) {
	domain := "feishu"
	init, err := postFeishuRegistration(ctx, domain, url.Values{"action": {"init"}})
	if err != nil {
		return feishuCredentials{}, err
	}
	if !stringListContains(init["supported_auth_methods"], "client_secret") {
		return feishuCredentials{}, fmt.Errorf("飞书扫码注册环境不支持 client_secret 授权")
	}
	begin, err := postFeishuRegistration(ctx, domain, url.Values{
		"action":            {"begin"},
		"archetype":         {"PersonalAgent"},
		"auth_method":       {"client_secret"},
		"request_user_info": {"open_id"},
	})
	if err != nil {
		return feishuCredentials{}, err
	}
	deviceCode := feishuStringValue(begin["device_code"])
	qrURL := feishuStringValue(begin["verification_uri_complete"])
	if deviceCode == "" || qrURL == "" {
		return feishuCredentials{}, fmt.Errorf("飞书二维码响应缺少必要信息")
	}
	qrURL = appendFeishuQRTracking(qrURL)
	a.emit("feishu-login:qr", feishuLoginEvent{Type: "qr_ready", ProfileID: profileID, ScanData: qrURL, Status: "等待飞书扫码"})

	interval := intValue(begin["interval"], 5)
	expiresIn := intValue(begin["expire_in"], int(feishuLoginTimeout.Seconds()))
	deadline := time.Now().Add(time.Duration(expiresIn) * time.Second)
	for time.Now().Before(deadline) {
		result, err := postFeishuRegistration(ctx, domain, url.Values{
			"action":      {"poll"},
			"device_code": {deviceCode},
			"tp":          {"ob_app"},
		})
		if err != nil {
			return feishuCredentials{}, err
		}
		userInfo, _ := result["user_info"].(map[string]interface{})
		if feishuStringValue(userInfo["tenant_brand"]) == "lark" {
			domain = "lark"
		}
		appID := feishuStringValue(result["client_id"])
		appSecret := feishuStringValue(result["client_secret"])
		if appID != "" && appSecret != "" {
			botName, _ := probeFeishuBot(ctx, appID, appSecret, domain)
			return feishuCredentials{AppID: appID, AppSecret: appSecret, Domain: domain, BotName: botName}, nil
		}
		switch feishuStringValue(result["error"]) {
		case "access_denied":
			return feishuCredentials{}, fmt.Errorf("飞书扫码授权已拒绝")
		case "expired_token":
			return feishuCredentials{}, fmt.Errorf("飞书二维码已过期，请重新开始")
		}
		if err := waitForFeishuPoll(ctx, time.Duration(interval)*time.Second); err != nil {
			return feishuCredentials{}, err
		}
	}
	return feishuCredentials{}, context.DeadlineExceeded
}

func (a *App) persistFeishuCredentials(profileID string, credentials feishuCredentials) error {
	path := filepath.Join(a.profileDataDir(profileID), ".env")
	env, _ := readEnvFile(path)
	updates := []EnvVar{
		{Key: "FEISHU_APP_ID", Value: credentials.AppID},
		{Key: "FEISHU_APP_SECRET", Value: credentials.AppSecret},
		{Key: "FEISHU_DOMAIN", Value: credentials.Domain},
		{Key: "FEISHU_CONNECTION_MODE", Value: "websocket"},
		{Key: "FEISHU_ALLOW_ALL_USERS", Value: "true"},
		{Key: "FEISHU_ALLOWED_USERS", Value: ""},
		{Key: "FEISHU_GROUP_POLICY", Value: "open"},
	}
	return a.saveEnvironmentTo(path, mergeEnv(env, updates))
}

func postFeishuRegistration(ctx context.Context, domain string, form url.Values) (map[string]interface{}, error) {
	baseURL := feishuAccountsURLs[domain]
	if baseURL == "" {
		return nil, fmt.Errorf("不支持的飞书平台区域：%s", domain)
	}
	return postFeishuForm(ctx, baseURL+feishuRegistrationPath, form)
}

func postFeishuForm(ctx context.Context, endpoint string, form url.Values) (map[string]interface{}, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	result := map[string]interface{}{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("飞书服务返回了无效响应（HTTP %d）", response.StatusCode)
	}
	return result, nil
}

func probeFeishuBot(ctx context.Context, appID string, appSecret string, domain string) (string, error) {
	baseURL := feishuOpenURLs[domain]
	if baseURL == "" {
		return "", fmt.Errorf("不支持的飞书平台区域：%s", domain)
	}
	payload, err := json.Marshal(map[string]string{"app_id": appID, "app_secret": appSecret})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/open-apis/auth/v3/tenant_access_token/internal", strings.NewReader(string(payload)))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	response, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}
	tokenResult := map[string]interface{}{}
	if err := json.Unmarshal(body, &tokenResult); err != nil {
		return "", err
	}
	accessToken := feishuStringValue(tokenResult["tenant_access_token"])
	if accessToken == "" {
		return "", fmt.Errorf("无法获取飞书机器人访问令牌")
	}
	req, err = http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/open-apis/bot/v3/info", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	response, err = (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	body, err = io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}
	botResult := map[string]interface{}{}
	if err := json.Unmarshal(body, &botResult); err != nil {
		return "", err
	}
	bot, _ := botResult["bot"].(map[string]interface{})
	return firstNonEmpty(feishuStringValue(bot["app_name"]), feishuStringValue(bot["bot_name"])), nil
}

func appendFeishuQRTracking(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	query := parsed.Query()
	query.Set("from", "hermes")
	query.Set("tp", "hermes")
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func waitForFeishuPoll(ctx context.Context, interval time.Duration) error {
	timer := time.NewTimer(interval)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func feishuStringValue(value interface{}) string {
	text, _ := value.(string)
	return strings.TrimSpace(text)
}

func intValue(value interface{}, fallback int) int {
	number, ok := value.(float64)
	if !ok || number <= 0 {
		return fallback
	}
	return int(number)
}

func stringListContains(value interface{}, expected string) bool {
	items, _ := value.([]interface{})
	for _, item := range items {
		if feishuStringValue(item) == expected {
			return true
		}
	}
	return false
}
