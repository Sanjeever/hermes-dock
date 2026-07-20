package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

const (
	dingtalkRegistrationInitPath  = "/app/registration/init"
	dingtalkRegistrationBeginPath = "/app/registration/begin"
	dingtalkRegistrationPollPath  = "/app/registration/poll"
	dingtalkLoginTimeout          = 2 * time.Hour
	dingtalkDefaultExpiry         = 2 * time.Hour
	dingtalkTransientRetryWindow  = 2 * time.Minute
	dingtalkResponseLimit         = 1 << 20
)

var dingtalkRegistrationBaseURL = "https://oapi.dingtalk.com"

type dingtalkLoginEvent struct {
	Type      string `json:"type"`
	ProfileID string `json:"profile_id,omitempty"`
	ScanData  string `json:"scan_data,omitempty"`
	Status    string `json:"status,omitempty"`
	Message   string `json:"message,omitempty"`
}

type dingtalkCredentials struct {
	ClientID     string
	ClientSecret string
}

type dingtalkTransientError struct {
	err error
}

func (e *dingtalkTransientError) Error() string {
	return e.err.Error()
}

func (e *dingtalkTransientError) Unwrap() error {
	return e.err
}

func (a *App) StartDingTalkLogin() error {
	return a.StartDingTalkLoginForProfile(a.currentProfileID())
}

func (a *App) StartDingTalkLoginForProfile(profileID string) error {
	profileID, err := a.resolveProfileID(profileID)
	if err != nil {
		return err
	}
	if err := ensureDir(a.profileDataDir(profileID)); err != nil {
		return err
	}
	ctx, err := a.startLoginSession("dingtalk", profileID, dingtalkLoginTimeout)
	if err != nil {
		return err
	}
	go a.runDingTalkLogin(ctx, profileID)
	return nil
}

func (a *App) CancelDingTalkLogin() {
	a.cancelLoginSession("dingtalk")
}

func (a *App) runDingTalkLogin(ctx context.Context, profileID string) {
	defer a.finishLoginSession("dingtalk", nil)
	credentials, err := a.registerDingTalkBot(ctx, profileID)
	if err != nil {
		message := "钉钉扫码绑定失败"
		switch ctx.Err() {
		case context.Canceled:
			message = "已取消钉钉扫码绑定"
		case context.DeadlineExceeded:
			message = "钉钉扫码绑定超时，请重新开始"
		default:
			message = redact(err.Error())
		}
		a.emit("dingtalk-login:error", dingtalkLoginEvent{Type: "error", ProfileID: profileID, Message: message})
		return
	}
	if err := ctx.Err(); err != nil {
		a.emit("dingtalk-login:error", dingtalkLoginEvent{Type: "error", ProfileID: profileID, Message: "已取消钉钉扫码绑定"})
		return
	}
	if err := a.persistDingTalkCredentials(ctx, profileID, credentials); err != nil {
		message := redact(err.Error())
		if errors.Is(err, context.Canceled) {
			message = "已取消钉钉扫码绑定"
		}
		a.emit("dingtalk-login:error", dingtalkLoginEvent{Type: "error", ProfileID: profileID, Message: message})
		return
	}
	a.emit("dingtalk-login:confirmed", dingtalkLoginEvent{
		Type:      "confirmed",
		ProfileID: profileID,
		Status:    "钉钉配置已保存，请应用配置后生效",
	})
}

func (a *App) registerDingTalkBot(ctx context.Context, profileID string) (dingtalkCredentials, error) {
	init, err := postDingTalkRegistrationWithRetry(ctx, dingtalkRegistrationInitPath, map[string]string{"source": "openClaw"})
	if err != nil {
		return dingtalkCredentials{}, err
	}
	nonce := dingtalkStringValue(init["nonce"])
	if nonce == "" {
		return dingtalkCredentials{}, fmt.Errorf("钉钉扫码初始化响应缺少 nonce")
	}
	begin, err := postDingTalkRegistrationWithRetry(ctx, dingtalkRegistrationBeginPath, map[string]string{"nonce": nonce})
	if err != nil {
		return dingtalkCredentials{}, err
	}
	deviceCode := dingtalkStringValue(begin["device_code"])
	qrURL := dingtalkStringValue(begin["verification_uri_complete"])
	if deviceCode == "" || qrURL == "" {
		return dingtalkCredentials{}, fmt.Errorf("钉钉二维码响应缺少必要信息")
	}
	a.emit("dingtalk-login:qr", dingtalkLoginEvent{Type: "qr_ready", ProfileID: profileID, ScanData: qrURL, Status: "等待钉钉扫码"})

	pollCtx, cancel := context.WithTimeout(ctx, dingtalkExpiry(begin["expires_in"]))
	defer cancel()
	interval := dingtalkInterval(begin["interval"])
	for {
		if err := waitForDingTalkPoll(pollCtx, interval); err != nil {
			return dingtalkCredentials{}, err
		}
		result, err := postDingTalkRegistrationWithRetry(pollCtx, dingtalkRegistrationPollPath, map[string]string{"device_code": deviceCode})
		if err != nil {
			return dingtalkCredentials{}, err
		}
		switch strings.ToUpper(dingtalkStringValue(result["status"])) {
		case "WAITING":
			continue
		case "SUCCESS":
			credentials := dingtalkCredentials{
				ClientID:     dingtalkStringValue(result["client_id"]),
				ClientSecret: dingtalkStringValue(result["client_secret"]),
			}
			if credentials.ClientID == "" || credentials.ClientSecret == "" {
				return dingtalkCredentials{}, fmt.Errorf("钉钉扫码成功但返回凭据不完整")
			}
			if err := pollCtx.Err(); err != nil {
				return dingtalkCredentials{}, err
			}
			return credentials, nil
		case "EXPIRED":
			return dingtalkCredentials{}, fmt.Errorf("钉钉二维码已过期，请重新开始")
		case "FAIL":
			return dingtalkCredentials{}, fmt.Errorf("钉钉扫码授权失败：%s", firstNonEmpty(dingtalkStringValue(result["fail_reason"]), "未知原因"))
		default:
			return dingtalkCredentials{}, fmt.Errorf("钉钉扫码返回未知状态：%s", firstNonEmpty(dingtalkStringValue(result["status"]), "空"))
		}
	}
}

func (a *App) persistDingTalkCredentials(ctx context.Context, profileID string, credentials dingtalkCredentials) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	// Serialize the commit with cancellation so a cancel that wins the race cannot
	// be followed by a credentials write.
	a.loginMu.Lock()
	defer a.loginMu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, err := a.resolveProfileID(profileID); err != nil {
		return err
	}
	path := filepath.Join(a.profileDataDir(profileID), ".env")
	env, err := readEnvFile(path)
	if err != nil {
		return err
	}
	updates := []EnvVar{
		{Key: "DINGTALK_CLIENT_ID", Value: credentials.ClientID},
		{Key: "DINGTALK_CLIENT_SECRET", Value: credentials.ClientSecret},
		{Key: "DINGTALK_ALLOW_ALL_USERS", Value: "true"},
		{Key: "DINGTALK_ALLOWED_USERS", Value: ""},
		{Key: "DINGTALK_REQUIRE_MENTION", Value: "true"},
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := a.saveEnvironmentTo(path, mergeEnv(env, updates)); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return a.markRebuildRequired()
}

func postDingTalkRegistration(ctx context.Context, path string, payload map[string]string) (map[string]interface{}, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(dingtalkRegistrationBaseURL, "/")+path, strings.NewReader(string(data)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	response, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, &dingtalkTransientError{err: err}
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, dingtalkResponseLimit+1))
	if err != nil {
		return nil, err
	}
	if len(body) > dingtalkResponseLimit {
		return nil, fmt.Errorf("钉钉扫码服务响应过大")
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		err := fmt.Errorf("钉钉扫码服务返回 HTTP %d", response.StatusCode)
		if response.StatusCode >= http.StatusInternalServerError || response.StatusCode == http.StatusTooManyRequests {
			return nil, &dingtalkTransientError{err: err}
		}
		return nil, err
	}
	result := map[string]interface{}{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("钉钉服务返回了无效响应（HTTP %d）", response.StatusCode)
	}
	if dingtalkErrorCode(result) != 0 {
		return nil, fmt.Errorf("钉钉扫码服务错误：%s", firstNonEmpty(dingtalkStringValue(result["errmsg"]), "未知错误"))
	}
	return result, nil
}

func postDingTalkRegistrationWithRetry(ctx context.Context, path string, payload map[string]string) (map[string]interface{}, error) {
	deadline := time.Now().Add(dingtalkTransientRetryWindow)
	for {
		result, err := postDingTalkRegistration(ctx, path, payload)
		if err == nil {
			return result, nil
		}
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		var transient *dingtalkTransientError
		if !errors.As(err, &transient) || !time.Now().Before(deadline) {
			return nil, err
		}
		if err := waitForDingTalkPoll(ctx, 2*time.Second); err != nil {
			return nil, err
		}
	}
}

func dingtalkErrorCode(result map[string]interface{}) int {
	value, ok := result["errcode"].(float64)
	if !ok {
		return -1
	}
	return int(value)
}

func dingtalkStringValue(value interface{}) string {
	text, _ := value.(string)
	return strings.TrimSpace(text)
}

func dingtalkInterval(value interface{}) time.Duration {
	seconds, ok := value.(float64)
	if !ok || seconds < 2 {
		return 2 * time.Second
	}
	return time.Duration(int(seconds)) * time.Second
}

func dingtalkExpiry(value interface{}) time.Duration {
	seconds, ok := value.(float64)
	if !ok || seconds <= 0 {
		return dingtalkDefaultExpiry
	}
	expiresIn := time.Duration(int(seconds)) * time.Second
	if expiresIn > dingtalkLoginTimeout {
		return dingtalkLoginTimeout
	}
	return expiresIn
}

func waitForDingTalkPoll(ctx context.Context, interval time.Duration) error {
	timer := time.NewTimer(interval)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
