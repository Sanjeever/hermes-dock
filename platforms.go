package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

func (a *App) SaveWeComConfig(config WeComConfig) error {
	env, _ := readEnvFile(a.envPath())
	updates := []EnvVar{
		{Key: "WECOM_BOT_ID", Value: config.BotID},
		{Key: "WECOM_SECRET", Value: config.Secret},
		{Key: "WECOM_WEBSOCKET_URL", Value: firstNonEmpty(config.WebSocketURL, "wss://openws.work.weixin.qq.com")},
		{Key: "WECOM_DM_POLICY", Value: firstNonEmpty(config.DMPolicy, "open")},
		{Key: "WECOM_ALLOWED_USERS", Value: config.AllowedUsers},
		{Key: "WECOM_GROUP_POLICY", Value: firstNonEmpty(config.GroupPolicy, "open")},
		{Key: "WECOM_GROUP_ALLOWED_USERS", Value: config.GroupAllowUsers},
	}
	return a.SaveEnvironment(mergeEnv(env, updates))
}

func (a *App) SaveFeishuConfig(config FeishuConfig) error {
	domain := firstNonEmpty(strings.TrimSpace(config.Domain), "feishu")
	if !oneOf(domain, "feishu", "lark") {
		return fmt.Errorf("飞书平台区域无效：%s", domain)
	}
	groupPolicy := firstNonEmpty(strings.TrimSpace(config.GroupPolicy), "allowlist")
	if !oneOf(groupPolicy, "open", "allowlist", "disabled") {
		return fmt.Errorf("飞书群聊策略无效：%s", groupPolicy)
	}
	env, _ := readEnvFile(a.envPath())
	updates := []EnvVar{
		{Key: "FEISHU_APP_ID", Value: strings.TrimSpace(config.AppID)},
		{Key: "FEISHU_APP_SECRET", Value: strings.TrimSpace(config.AppSecret)},
		{Key: "FEISHU_DOMAIN", Value: domain},
		{Key: "FEISHU_CONNECTION_MODE", Value: "websocket"},
		{Key: "FEISHU_ALLOWED_USERS", Value: config.AllowedUsers},
		{Key: "FEISHU_GROUP_POLICY", Value: groupPolicy},
	}
	return a.SaveEnvironment(mergeEnv(env, updates))
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
		message = "Hermes Dock 测试消息"
	}
	args := append([]string{"run", "--rm"}, a.currentProfileComposeEnvArgs()...)
	args = append(args, "hermes")
	args = append(args, a.currentProfileHermesArgs("send", "--to", target, message)...)
	return a.runComposeStreaming(context.Background(), "docker:progress", args...)
}
