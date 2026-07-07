package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

func (a *App) SaveWeComConfig(config WeComConfig) error {
	dmPolicy, err := normalizeOpenClosedPolicy(config.DMPolicy, "closed", "allowlist")
	if err != nil {
		return fmt.Errorf("企业微信私聊策略无效：%s", config.DMPolicy)
	}
	groupPolicy, err := normalizeOpenClosedPolicy(config.GroupPolicy, "closed", "allowlist")
	if err != nil {
		return fmt.Errorf("企业微信群聊策略无效：%s", config.GroupPolicy)
	}
	env, _ := readEnvFile(a.envPath())
	updates := []EnvVar{
		{Key: "WECOM_BOT_ID", Value: config.BotID},
		{Key: "WECOM_SECRET", Value: keepExistingIfMaskedSecret(env, "WECOM_SECRET", config.Secret)},
		{Key: "WECOM_WEBSOCKET_URL", Value: firstNonEmpty(config.WebSocketURL, "wss://openws.work.weixin.qq.com")},
		{Key: "WECOM_DM_POLICY", Value: dmPolicy},
		{Key: "WECOM_ALLOWED_USERS", Value: ""},
		{Key: "WECOM_GROUP_POLICY", Value: groupPolicy},
		{Key: "WECOM_GROUP_ALLOWED_USERS", Value: ""},
	}
	return a.SaveEnvironment(mergeEnv(env, updates))
}

func (a *App) SaveFeishuConfig(config FeishuConfig) error {
	domain := firstNonEmpty(strings.TrimSpace(config.Domain), "feishu")
	if !oneOf(domain, "feishu", "lark") {
		return fmt.Errorf("飞书平台区域无效：%s", domain)
	}
	groupPolicy, err := normalizeOpenClosedPolicy(config.GroupPolicy, "disabled", "allowlist")
	if err != nil {
		return fmt.Errorf("飞书群聊策略无效：%s", config.GroupPolicy)
	}
	env, _ := readEnvFile(a.envPath())
	updates := []EnvVar{
		{Key: "FEISHU_APP_ID", Value: strings.TrimSpace(config.AppID)},
		{Key: "FEISHU_APP_SECRET", Value: keepExistingIfMaskedSecret(env, "FEISHU_APP_SECRET", config.AppSecret)},
		{Key: "FEISHU_DOMAIN", Value: domain},
		{Key: "FEISHU_CONNECTION_MODE", Value: "websocket"},
		{Key: "FEISHU_ALLOW_ALL_USERS", Value: "true"},
		{Key: "FEISHU_ALLOWED_USERS", Value: ""},
		{Key: "FEISHU_GROUP_POLICY", Value: groupPolicy},
	}
	return a.SaveEnvironment(mergeEnv(env, updates))
}

func (a *App) UnbindPlatform(platform string) error {
	platform = strings.TrimSpace(platform)
	env, _ := readEnvFile(a.envPath())
	var updates []EnvVar
	switch platform {
	case "weixin":
		a.CancelWeixinLogin()
		updates = []EnvVar{
			{Key: "WEIXIN_ACCOUNT_ID", Value: ""},
			{Key: "WEIXIN_TOKEN", Value: ""},
			{Key: "WEIXIN_BASE_URL", Value: ""},
			{Key: "WEIXIN_CDN_BASE_URL", Value: "https://novac2c.cdn.weixin.qq.com/c2c"},
			{Key: "WEIXIN_DM_POLICY", Value: "open"},
			{Key: "WEIXIN_ALLOW_ALL_USERS", Value: "true"},
			{Key: "WEIXIN_ALLOWED_USERS", Value: ""},
			{Key: "WEIXIN_GROUP_POLICY", Value: "open"},
			{Key: "WEIXIN_GROUP_ALLOWED_USERS", Value: ""},
			{Key: "WEIXIN_HOME_CHANNEL", Value: ""},
		}
	case "wecom":
		updates = []EnvVar{
			{Key: "WECOM_BOT_ID", Value: ""},
			{Key: "WECOM_SECRET", Value: ""},
			{Key: "WECOM_WEBSOCKET_URL", Value: "wss://openws.work.weixin.qq.com"},
			{Key: "WECOM_DM_POLICY", Value: "open"},
			{Key: "WECOM_ALLOWED_USERS", Value: ""},
			{Key: "WECOM_GROUP_POLICY", Value: "open"},
			{Key: "WECOM_GROUP_ALLOWED_USERS", Value: ""},
		}
	case "feishu":
		updates = []EnvVar{
			{Key: "FEISHU_APP_ID", Value: ""},
			{Key: "FEISHU_APP_SECRET", Value: ""},
			{Key: "FEISHU_DOMAIN", Value: "feishu"},
			{Key: "FEISHU_CONNECTION_MODE", Value: "websocket"},
			{Key: "FEISHU_ALLOW_ALL_USERS", Value: "true"},
			{Key: "FEISHU_ALLOWED_USERS", Value: ""},
			{Key: "FEISHU_GROUP_POLICY", Value: "open"},
		}
	default:
		return fmt.Errorf("不支持的平台：%s", platform)
	}
	return a.SaveEnvironment(mergeEnv(env, updates))
}

func keepExistingIfMaskedSecret(existing []EnvVar, key string, value string) string {
	value = strings.TrimSpace(value)
	if isMaskedSecretPlaceholder(value) {
		return envValue(existing, key)
	}
	return value
}

func isMaskedSecretPlaceholder(value string) bool {
	if value == "" {
		return false
	}
	lower := strings.ToLower(value)
	if lower == "<redacted>" || lower == "redacted" || lower == "[redacted]" {
		return true
	}
	hasMask := false
	for _, ch := range value {
		switch ch {
		case '*', '•', '●', '·':
			hasMask = true
		default:
			if ch != ' ' {
				return false
			}
		}
	}
	return hasMask
}

func normalizeOpenClosedPolicy(value string, closeValue string, legacyCloseValues ...string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || value == "open" {
		return "open", nil
	}
	if value == closeValue {
		return closeValue, nil
	}
	for _, legacy := range legacyCloseValues {
		if value == legacy {
			return closeValue, nil
		}
	}
	return "", fmt.Errorf("invalid policy")
}

func (a *App) GetChannels() (ChannelFile, error) {
	out := ChannelFile{Platforms: map[string][]ChannelSummary{}}
	data, err := os.ReadFile(a.channelDirectoryPath())
	if err != nil {
		return out, nil
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return ChannelFile{Platforms: map[string][]ChannelSummary{}}, err
	}
	if out.Platforms == nil {
		out.Platforms = map[string][]ChannelSummary{}
	}
	return out, nil
}

func (a *App) SetHomeChannel(platform string, channelID string) error {
	env, _ := readEnvFile(a.envPath())
	switch platform {
	case "weixin":
		return a.SaveEnvironment(mergeEnv(env, []EnvVar{{Key: "WEIXIN_HOME_CHANNEL", Value: channelID}}))
	default:
		return nil
	}
}

func (a *App) SendTestMessage(platform string, channelID string, message string) error {
	target := platform
	if channelID != "" {
		target = platform + ":" + channelID
	}
	if message == "" {
		message = "企智盒测试消息"
	}
	args := append([]string{"run", "--rm"}, a.currentProfileComposeEnvArgs()...)
	args = append(args, "hermes")
	args = append(args, a.currentProfileHermesArgs("send", "--to", target, message)...)
	return a.runComposeStreaming(context.Background(), "docker:progress", args...)
}
