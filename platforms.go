package main

import (
	"context"
	"encoding/json"
	"os"
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
	return a.runComposeStreaming(context.Background(), "docker:progress", "run", "--rm", "hermes", "send", "--to", target, message)
}
