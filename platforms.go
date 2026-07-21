package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

func (a *App) SaveWeComConfig(config WeComConfig) error {
	return a.SaveWeComConfigForProfile(a.currentProfileID(), config)
}

func (a *App) SaveWeComConfigForProfile(profileID string, config WeComConfig) error {
	profileID, err := a.resolveProfileID(profileID)
	if err != nil {
		return err
	}
	dmPolicy, err := normalizeOpenClosedPolicy(config.DMPolicy, "closed", "allowlist")
	if err != nil {
		return fmt.Errorf("企业微信私聊策略无效：%s", config.DMPolicy)
	}
	groupPolicy, err := normalizeOpenClosedPolicy(config.GroupPolicy, "closed", "allowlist")
	if err != nil {
		return fmt.Errorf("企业微信群聊策略无效：%s", config.GroupPolicy)
	}
	env, err := readEnvFile(a.profileEnvPath(profileID))
	if err != nil {
		return err
	}
	updates := []EnvVar{
		{Key: "WECOM_BOT_ID", Value: config.BotID},
		{Key: "WECOM_SECRET", Value: keepExistingIfMaskedSecret(env, "WECOM_SECRET", config.Secret)},
		{Key: "WECOM_WEBSOCKET_URL", Value: firstNonEmpty(config.WebSocketURL, "wss://openws.work.weixin.qq.com")},
		{Key: "WECOM_DM_POLICY", Value: dmPolicy},
		{Key: "WECOM_ALLOWED_USERS", Value: ""},
		{Key: "WECOM_GROUP_POLICY", Value: groupPolicy},
		{Key: "WECOM_GROUP_ALLOWED_USERS", Value: ""},
	}
	return a.SaveEnvironmentForProfile(profileID, mergeEnv(env, updates))
}

func (a *App) SaveFeishuConfig(config FeishuConfig) error {
	return a.SaveFeishuConfigForProfile(a.currentProfileID(), config)
}

func (a *App) SaveFeishuConfigForProfile(profileID string, config FeishuConfig) error {
	profileID, err := a.resolveProfileID(profileID)
	if err != nil {
		return err
	}
	domain := firstNonEmpty(strings.TrimSpace(config.Domain), "feishu")
	if !oneOf(domain, "feishu", "lark") {
		return fmt.Errorf("飞书平台区域无效：%s", domain)
	}
	groupPolicy, err := normalizeOpenClosedPolicy(config.GroupPolicy, "disabled", "allowlist")
	if err != nil {
		return fmt.Errorf("飞书群聊策略无效：%s", config.GroupPolicy)
	}
	env, err := readEnvFile(a.profileEnvPath(profileID))
	if err != nil {
		return err
	}
	updates := []EnvVar{
		{Key: "FEISHU_APP_ID", Value: strings.TrimSpace(config.AppID)},
		{Key: "FEISHU_APP_SECRET", Value: keepExistingIfMaskedSecret(env, "FEISHU_APP_SECRET", config.AppSecret)},
		{Key: "FEISHU_DOMAIN", Value: domain},
		{Key: "FEISHU_CONNECTION_MODE", Value: "websocket"},
		{Key: "FEISHU_ALLOW_ALL_USERS", Value: "true"},
		{Key: "FEISHU_ALLOWED_USERS", Value: ""},
		{Key: "FEISHU_GROUP_POLICY", Value: groupPolicy},
	}
	return a.SaveEnvironmentForProfile(profileID, mergeEnv(env, updates))
}

func (a *App) SaveDingTalkConfig(config DingTalkConfig) error {
	return a.SaveDingTalkConfigForProfile(a.currentProfileID(), config)
}

func (a *App) SaveDingTalkConfigForProfile(profileID string, config DingTalkConfig) error {
	profileID, err := a.resolveProfileID(profileID)
	if err != nil {
		return err
	}
	env, err := readEnvFile(a.profileEnvPath(profileID))
	if err != nil {
		return err
	}
	updates := []EnvVar{
		{Key: "DINGTALK_CLIENT_ID", Value: strings.TrimSpace(config.ClientID)},
		{Key: "DINGTALK_CLIENT_SECRET", Value: keepExistingIfMaskedSecret(env, "DINGTALK_CLIENT_SECRET", config.ClientSecret)},
		{Key: "DINGTALK_ALLOW_ALL_USERS", Value: "true"},
		{Key: "DINGTALK_ALLOWED_USERS", Value: ""},
		{Key: "DINGTALK_REQUIRE_MENTION", Value: fmt.Sprintf("%t", config.RequireMention)},
	}
	return a.SaveEnvironmentForProfile(profileID, mergeEnv(env, updates))
}

func (a *App) UnbindPlatform(platform string) error {
	return a.UnbindPlatformForProfile(a.currentProfileID(), platform)
}

func (a *App) UnbindPlatformForProfile(profileID string, platform string) error {
	profileID, err := a.resolveProfileID(profileID)
	if err != nil {
		return err
	}
	platform = strings.TrimSpace(platform)
	env, err := readEnvFile(a.profileEnvPath(profileID))
	if err != nil {
		return err
	}
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
	case "dingtalk":
		if err := a.cancelProfileLoginSessionAndWait(profileID); err != nil {
			return err
		}
		updates = []EnvVar{
			{Key: "DINGTALK_CLIENT_ID", Value: ""},
			{Key: "DINGTALK_CLIENT_SECRET", Value: ""},
			{Key: "DINGTALK_ALLOW_ALL_USERS", Value: "true"},
			{Key: "DINGTALK_ALLOWED_USERS", Value: ""},
			{Key: "DINGTALK_REQUIRE_MENTION", Value: "true"},
			{Key: "DINGTALK_HOME_CHANNEL", Value: ""},
		}
	default:
		return fmt.Errorf("不支持的平台：%s", platform)
	}
	return a.SaveEnvironmentForProfile(profileID, mergeEnv(env, updates))
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
	return a.GetChannelsForProfile(a.currentProfileID())
}

func (a *App) GetChannelsForProfile(profileID string) (ChannelFile, error) {
	out := ChannelFile{Platforms: map[string][]ChannelSummary{}}
	profileID, err := a.resolveProfileID(profileID)
	if err != nil {
		return out, err
	}
	data, err := os.ReadFile(a.profileChannelDirectoryPath(profileID))
	if err != nil {
		if os.IsNotExist(err) {
			return out, nil
		}
		return out, err
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
	return a.SetHomeChannelForProfile(a.currentProfileID(), platform, channelID)
}

func (a *App) SetHomeChannelForProfile(profileID string, platform string, channelID string) error {
	profileID, err := a.resolveProfileID(profileID)
	if err != nil {
		return err
	}
	env, err := readEnvFile(a.profileEnvPath(profileID))
	if err != nil {
		return err
	}
	switch platform {
	case "weixin":
		return a.SaveEnvironmentForProfile(profileID, mergeEnv(env, []EnvVar{{Key: "WEIXIN_HOME_CHANNEL", Value: channelID}}))
	case "dingtalk":
		return a.SaveEnvironmentForProfile(profileID, mergeEnv(env, []EnvVar{{Key: "DINGTALK_HOME_CHANNEL", Value: channelID}}))
	default:
		return fmt.Errorf("不支持的平台：%s", platform)
	}
}

func (a *App) SendTestMessage(platform string, channelID string, message string) error {
	return a.SendTestMessageForProfile(a.currentProfileID(), platform, channelID, message)
}

func (a *App) SendTestMessageForProfile(profileID string, platform string, channelID string, message string) error {
	profileID, err := a.resolveProfileID(profileID)
	if err != nil {
		return err
	}
	target := platform
	if channelID != "" {
		target = platform + ":" + channelID
	}
	if message == "" {
		message = "企智盒测试消息"
	}
	if err := a.ensureRuntimeDependencies(); err != nil {
		return err
	}
	args := append([]string{"run", "--rm"}, a.profileComposeEnvArgs(profileID)...)
	args = append(args, "hermes")
	args = append(args, a.profileHermesArgs(profileID, "send", "--to", target, message)...)
	return a.runComposeStreaming(context.Background(), "docker:progress", args...)
}
