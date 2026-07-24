package main

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	defaultProfileID   = "default"
	profileCopyClean   = "clean"
	profileCopyPersona = "personality-skills"
)

func defaultProfileRegistry(createdAt string, auxiliaryMode string) ProfileRegistry {
	if createdAt == "" {
		createdAt = time.Now().UTC().Format(time.RFC3339)
	}
	return ProfileRegistry{
		SchemaVersion: 1,
		Profiles: []ProfileEntry{{
			ID:                 defaultProfileID,
			Name:               "默认助手",
			Enabled:            true,
			CreatedAt:          createdAt,
			UpdatedAt:          createdAt,
			ModelAuxiliaryMode: firstNonEmpty(auxiliaryMode, "auto"),
		}},
	}
}

func (a *App) ensureProfileRegistry() error {
	needsWrite := !fileExists(a.profilesPath())
	registry, err := a.readProfileRegistry()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if len(registry.Profiles) == 0 {
		state, err := a.readStateAllowMissing()
		if err != nil {
			return err
		}
		registry = defaultProfileRegistry(firstNonEmpty(state.InitializedAt, time.Now().UTC().Format(time.RFC3339)), state.ModelAuxiliaryMode)
		needsWrite = true
	}
	if !profileExists(registry, defaultProfileID) {
		state, err := a.readStateAllowMissing()
		if err != nil {
			return err
		}
		registry.Profiles = append([]ProfileEntry{defaultProfileRegistry(time.Now().UTC().Format(time.RFC3339), state.ModelAuxiliaryMode).Profiles[0]}, registry.Profiles...)
		needsWrite = true
	}
	if !needsWrite {
		return nil
	}
	return a.writeProfileRegistry(registry)
}

func (a *App) readProfileRegistry() (ProfileRegistry, error) {
	var registry ProfileRegistry
	data, err := os.ReadFile(a.profilesPath())
	if err != nil {
		state, stateErr := a.readStateAllowMissing()
		if stateErr != nil {
			return ProfileRegistry{}, fmt.Errorf("读取 profile 默认状态失败：%w", stateErr)
		}
		return defaultProfileRegistry(firstNonEmpty(state.InitializedAt, time.Now().UTC().Format(time.RFC3339)), state.ModelAuxiliaryMode), err
	}
	if err := json.Unmarshal(data, &registry); err != nil {
		return ProfileRegistry{}, err
	}
	if registry.SchemaVersion == 0 {
		registry.SchemaVersion = 1
	}
	if len(registry.Profiles) == 0 {
		state, err := a.readStateAllowMissing()
		if err != nil {
			return ProfileRegistry{}, err
		}
		registry = defaultProfileRegistry(firstNonEmpty(state.InitializedAt, time.Now().UTC().Format(time.RFC3339)), state.ModelAuxiliaryMode)
	}
	for i := range registry.Profiles {
		if registry.Profiles[i].ID == defaultProfileID && registry.Profiles[i].Name == "" {
			registry.Profiles[i].Name = "默认助手"
		}
		if registry.Profiles[i].ModelAuxiliaryMode == "" {
			registry.Profiles[i].ModelAuxiliaryMode = "auto"
		}
	}
	return registry, nil
}

func (a *App) writeProfileRegistry(registry ProfileRegistry) error {
	if registry.SchemaVersion == 0 {
		registry.SchemaVersion = 1
	}
	if err := validateProfileRegistry(registry); err != nil {
		return err
	}
	if err := ensureDir(filepath.Dir(a.profilesPath())); err != nil {
		return err
	}
	if fileExists(a.profilesPath()) {
		if err := a.backupFile(a.profilesPath(), "before-profiles-save"); err != nil {
			return err
		}
	}
	data, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		return err
	}
	return atomicWriteFile(a.profilesPath(), append(data, '\n'), 0644)
}

func validateProfileRegistry(registry ProfileRegistry) error {
	seen := map[string]bool{}
	for _, profile := range registry.Profiles {
		if err := validateProfileID(profile.ID, true); err != nil {
			return err
		}
		if seen[profile.ID] {
			return fmt.Errorf("profile ID 重复：%s", profile.ID)
		}
		seen[profile.ID] = true
	}
	if !seen[defaultProfileID] {
		return fmt.Errorf("缺少 default profile")
	}
	return nil
}

func validateProfileID(id string, allowDefault bool) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("Profile ID 不能为空")
	}
	if id == defaultProfileID && !allowDefault {
		return fmt.Errorf("default 是保留 Profile ID")
	}
	if len(id) < 2 || len(id) > 40 {
		return fmt.Errorf("Profile ID 长度必须是 2-40 个字符")
	}
	for i, r := range id {
		valid := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-'
		if !valid {
			return fmt.Errorf("Profile ID 只能包含小写字母、数字和连字符")
		}
		if (i == 0 || i == len(id)-1) && r == '-' {
			return fmt.Errorf("Profile ID 不能以连字符开头或结尾")
		}
	}
	return nil
}

func profileExists(registry ProfileRegistry, id string) bool {
	for _, profile := range registry.Profiles {
		if profile.ID == id {
			return true
		}
	}
	return false
}

func profileIndex(registry ProfileRegistry, id string) int {
	for i, profile := range registry.Profiles {
		if profile.ID == id {
			return i
		}
	}
	return -1
}

func (a *App) currentProfileAuxiliaryMode() string {
	return a.profileAuxiliaryMode(a.currentProfileID())
}

func (a *App) profileAuxiliaryMode(id string) string {
	registry, err := a.readProfileRegistry()
	if err != nil {
		state, _ := a.readState()
		return firstNonEmpty(state.ModelAuxiliaryMode, "auto")
	}
	idx := profileIndex(registry, id)
	if idx < 0 {
		return "auto"
	}
	return firstNonEmpty(registry.Profiles[idx].ModelAuxiliaryMode, "auto")
}

func (a *App) updateCurrentProfileAuxiliaryMode(mode string) error {
	return a.updateProfileAuxiliaryMode(a.currentProfileID(), mode)
}

func (a *App) updateProfileAuxiliaryMode(id string, mode string) error {
	registry, err := a.readProfileRegistry()
	if err != nil {
		return err
	}
	idx := profileIndex(registry, id)
	if idx < 0 {
		return fmt.Errorf("profile 不存在：%s", id)
	}
	registry.Profiles[idx].ModelAuxiliaryMode = firstNonEmpty(mode, "auto")
	registry.Profiles[idx].UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	return a.writeProfileRegistry(registry)
}

func (a *App) ListProfiles() (ProfileRegistry, error) {
	return a.readProfileRegistry()
}

func (a *App) SelectProfile(id string) error {
	registry, err := a.readProfileRegistry()
	if err != nil {
		return err
	}
	if !profileExists(registry, id) {
		return fmt.Errorf("profile 不存在：%s", id)
	}
	return a.updateState(func(state *LauncherState) error {
		state.UI.LastProfile = id
		state.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		return nil
	})
}

func (a *App) CreateProfile(req CreateProfileRequest) (err error) {
	release, err := a.beginExclusiveOperation("创建助手")
	if err != nil {
		return err
	}
	defer release()
	id := strings.TrimSpace(req.ID)
	if err := validateProfileID(id, false); err != nil {
		return err
	}
	registry, err := a.readProfileRegistry()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if profileExists(registry, id) {
		return fmt.Errorf("profile 已存在：%s", id)
	}
	copyMode := firstNonEmpty(req.CopyMode, profileCopyClean)
	sourceID := firstNonEmpty(req.CopyFrom, a.currentProfileID())
	if copyMode == profileCopyPersona {
		if !profileExists(registry, sourceID) {
			return fmt.Errorf("源 profile 不存在：%s", sourceID)
		}
	} else if copyMode != profileCopyClean {
		return fmt.Errorf("不支持的创建方式：%s", copyMode)
	}
	target := a.profileDataDir(id)
	if fileExists(target) {
		return fmt.Errorf("profile 目录已存在，请先手动处理或恢复：%s", filepath.Join("data", "profiles", id))
	}
	committed := false
	defer func() {
		if !committed {
			if cleanupErr := os.RemoveAll(target); cleanupErr != nil {
				err = errors.Join(err, fmt.Errorf("清理未完成的 profile 目录失败：%w", cleanupErr))
			}
		}
	}()
	if err := a.releaseSeedDataTo(target, id); err != nil {
		return err
	}
	if err := a.ensureProfileEnvMarkers(id); err != nil {
		return err
	}
	if copyMode == profileCopyPersona {
		if err := a.copyProfilePersonality(sourceID, id); err != nil {
			return err
		}
	}
	originalRegistry := registry
	originalRegistry.Profiles = append([]ProfileEntry(nil), registry.Profiles...)
	now := time.Now().UTC().Format(time.RFC3339)
	registry.Profiles = append(append([]ProfileEntry(nil), registry.Profiles...), ProfileEntry{
		ID:                 id,
		Name:               firstNonEmpty(strings.TrimSpace(req.Name), id),
		Enabled:            req.Enabled,
		CreatedAt:          now,
		UpdatedAt:          now,
		ModelAuxiliaryMode: "auto",
	})
	if err := a.writeProfileRegistry(registry); err != nil {
		return err
	}
	if err := a.SelectProfile(id); err != nil {
		if rollbackErr := a.writeProfileRegistry(originalRegistry); rollbackErr != nil {
			committed = true
			return errors.Join(err, fmt.Errorf("回滚 profile registry 失败：%w", rollbackErr))
		}
		return err
	}
	committed = true
	return nil
}

func (a *App) CompleteProfileSetup(id string) error {
	registry, err := a.readProfileRegistry()
	if err != nil {
		return err
	}
	idx := profileIndex(registry, id)
	if idx < 0 {
		return fmt.Errorf("profile 不存在：%s", id)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	registry.Profiles[idx].SetupCompletedAt = now
	registry.Profiles[idx].UpdatedAt = now
	if err := a.writeProfileRegistry(registry); err != nil {
		return err
	}
	return a.markRebuildRequired()
}

func (a *App) UpdateProfileName(id string, name string) error {
	registry, err := a.readProfileRegistry()
	if err != nil {
		return err
	}
	idx := profileIndex(registry, id)
	if idx < 0 {
		return fmt.Errorf("profile 不存在：%s", id)
	}
	registry.Profiles[idx].Name = firstNonEmpty(strings.TrimSpace(name), id)
	registry.Profiles[idx].UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	return a.writeProfileRegistry(registry)
}

func (a *App) SetProfileEnabled(id string, enabled bool) error {
	registry, err := a.readProfileRegistry()
	if err != nil {
		return err
	}
	idx := profileIndex(registry, id)
	if idx < 0 {
		return fmt.Errorf("profile 不存在：%s", id)
	}
	registry.Profiles[idx].Enabled = enabled
	registry.Profiles[idx].UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if enabled {
		if err := a.validateRuntimeProfiles(registry); err != nil {
			return err
		}
	}
	if err := a.writeProfileRegistry(registry); err != nil {
		return err
	}
	return a.markRebuildRequired()
}

func (a *App) MoveProfile(id string, direction string) error {
	registry, err := a.readProfileRegistry()
	if err != nil {
		return err
	}
	idx := profileIndex(registry, id)
	if idx < 0 {
		return fmt.Errorf("profile 不存在：%s", id)
	}
	swapWith := idx
	switch direction {
	case "up":
		swapWith = idx - 1
	case "down":
		swapWith = idx + 1
	default:
		return fmt.Errorf("不支持的移动方向：%s", direction)
	}
	if swapWith < 0 || swapWith >= len(registry.Profiles) {
		return nil
	}
	registry.Profiles[idx], registry.Profiles[swapWith] = registry.Profiles[swapWith], registry.Profiles[idx]
	return a.writeProfileRegistry(registry)
}

func (a *App) DeleteProfile(id string) (err error) {
	release, err := a.beginExclusiveOperation("删除助手")
	if err != nil {
		return err
	}
	defer release()
	if id == defaultProfileID {
		return fmt.Errorf("default profile 不能删除，只能停用")
	}
	registry, err := a.readProfileRegistry()
	if err != nil {
		return err
	}
	idx := profileIndex(registry, id)
	if idx < 0 {
		return fmt.Errorf("profile 不存在：%s", id)
	}
	if err := a.cancelProfileLoginSessionAndWait(id); err != nil {
		return fmt.Errorf("停止 profile 扫码绑定失败：%w", err)
	}
	if fileExists(a.composePath()) && a.containerStatus(context.Background()) == "running" {
		if err := a.stopHermesRuntime(); err != nil {
			return fmt.Errorf("停止 Hermes 容器失败：%w", err)
		}
	}
	dir := a.profileDataDir(id)
	if fileExists(dir) {
		if err := a.backupDirectory(dir, "before-profile-delete-"+id); err != nil {
			return fmt.Errorf("备份 profile 失败：%w", err)
		}
	}
	if _, err := a.readState(); err != nil {
		return err
	}
	quarantine := ""
	if fileExists(dir) {
		quarantine = filepath.Join(a.hermesDockDir(), "backups", ".profile-delete-"+id+"-"+uuid.NewString())
		if err := ensureDir(filepath.Dir(quarantine)); err != nil {
			return err
		}
		if err := os.Rename(dir, quarantine); err != nil {
			return fmt.Errorf("暂存待删除 profile 失败：%w", err)
		}
	}
	restoreDirectory := func() error {
		if quarantine == "" || !fileExists(quarantine) {
			return nil
		}
		return os.Rename(quarantine, dir)
	}
	originalRegistry := registry
	originalRegistry.Profiles = append([]ProfileEntry(nil), registry.Profiles...)
	next := append([]ProfileEntry{}, registry.Profiles[:idx]...)
	next = append(next, registry.Profiles[idx+1:]...)
	registry.Profiles = next
	if err := a.writeProfileRegistry(registry); err != nil {
		return errors.Join(err, restoreDirectory())
	}
	if err := a.updateState(func(state *LauncherState) error {
		if state.UI.LastProfile == id {
			state.UI.LastProfile = defaultProfileID
		}
		state.NeedsRebuild = true
		state.PendingDufsOnly = false
		state.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		return nil
	}); err != nil {
		rollbackErr := a.writeProfileRegistry(originalRegistry)
		return errors.Join(err, rollbackErr, restoreDirectory())
	}
	if quarantine != "" {
		if err := os.RemoveAll(quarantine); err != nil {
			a.emit("docker:progress", StreamEvent{Line: redact(fmt.Sprintf("profile 已删除，但暂存目录清理失败并保留在备份目录：%v", err))})
		}
	}
	if err := os.Remove(a.bundledContentStatePath(id)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("删除 profile 内置内容状态失败：%w", err)
	}
	return nil
}

func (a *App) ensureProfileEnvMarkers(profileID string) error {
	path := filepath.Join(a.profileDataDir(profileID), ".env")
	existing, err := readEnvFileAllowMissing(path)
	if err != nil {
		return err
	}
	home := "/opt/data"
	if profileID != defaultProfileID {
		home = "/opt/data/profiles/" + profileID
	}
	merged := mergeEnv(existing, []EnvVar{
		{Key: "HERMES_DOCK_PROFILE", Value: profileID},
		{Key: "HERMES_DOCK_PROFILE_HOME", Value: home},
	})
	return writeEnvFile(path, merged)
}

func (a *App) copyProfilePersonality(sourceID string, targetID string) error {
	sourceDir := a.profileDataDir(sourceID)
	targetDir := a.profileDataDir(targetID)
	sourceSoul := filepath.Join(sourceDir, "SOUL.md")
	targetSoul := filepath.Join(targetDir, "SOUL.md")
	if fileExists(sourceSoul) {
		data, err := os.ReadFile(sourceSoul)
		if err != nil {
			return err
		}
		text := rewriteProfileContainerHome(string(data), sourceID, targetID)
		if err := atomicWriteFile(targetSoul, []byte(text), 0644); err != nil {
			return err
		}
	}
	sourceSkills := filepath.Join(sourceDir, "skills")
	targetSkills := filepath.Join(targetDir, "skills")
	if fileExists(sourceSkills) {
		if err := os.RemoveAll(targetSkills); err != nil {
			return err
		}
		if err := copyDir(sourceSkills, targetSkills); err != nil {
			return err
		}
	}
	return nil
}

func profileContainerHome(profileID string) string {
	if profileID == "" || profileID == defaultProfileID {
		return "/opt/data"
	}
	return "/opt/data/profiles/" + profileID
}

func rewriteProfileContainerHome(text string, sourceID string, targetID string) string {
	const sharedDirectory = "/opt/data/.dock/shared"
	const sharedDirectoryPlaceholder = "__HERMES_DOCK_SHARED_DIRECTORY__"
	text = strings.ReplaceAll(text, sharedDirectory, sharedDirectoryPlaceholder)
	text = strings.ReplaceAll(text, profileContainerHome(sourceID), profileContainerHome(targetID))
	return strings.ReplaceAll(text, sharedDirectoryPlaceholder, sharedDirectory)
}

func (a *App) writeRuntimeManifest() (RuntimeManifest, error) {
	registry, err := a.readProfileRegistry()
	if err != nil {
		return RuntimeManifest{}, err
	}
	if err := a.syncAllProfileProviderEnv(registry); err != nil {
		return RuntimeManifest{}, err
	}
	if err := a.validateRuntimeProfiles(registry); err != nil {
		return RuntimeManifest{}, err
	}
	manifest, err := a.buildRuntimeManifest(registry)
	if err != nil {
		return RuntimeManifest{}, err
	}
	if err := ensureDir(filepath.Dir(a.runtimeManifestPath())); err != nil {
		return RuntimeManifest{}, err
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return RuntimeManifest{}, err
	}
	if err := atomicWriteFile(a.runtimeManifestPath(), append(data, '\n'), 0644); err != nil {
		return RuntimeManifest{}, err
	}
	return manifest, nil
}

func (a *App) syncAllProfileProviderEnv(registry ProfileRegistry) error {
	for _, profile := range registry.Profiles {
		cfg := map[string]interface{}{}
		configPath := filepath.Join(a.profileDataDir(profile.ID), "config.yaml")
		if err := parseYAMLFile(configPath, &cfg); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("%s 的 config.yaml 无法解析：%w", profile.Name, err)
		}
		providers := normalizeProviderConfig(readProviderConfigFromMap(cfg))
		updates := referencedProviderEnvUpdates(cfg, providers)
		if len(updates) == 0 {
			continue
		}
		envPath := filepath.Join(a.profileDataDir(profile.ID), ".env")
		existing, err := readEnvFileAllowMissing(envPath)
		if err != nil {
			return fmt.Errorf("读取 %s 的 .env 失败：%w", profile.Name, err)
		}
		changed := false
		for _, item := range updates {
			if envValue(existing, item.Key) != item.Value {
				changed = true
				break
			}
		}
		if changed {
			if err := a.saveEnvironmentTo(envPath, updates); err != nil {
				return err
			}
		}
	}
	return nil
}

func (a *App) buildRuntimeManifest(registry ProfileRegistry) (RuntimeManifest, error) {
	manifest := RuntimeManifest{
		SchemaVersion: 1,
		Generation:    uuid.NewString(),
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
	}
	for _, profile := range registry.Profiles {
		item := RuntimeManifestProfile{
			ID:        profile.ID,
			Name:      firstNonEmpty(profile.Name, profile.ID),
			Enabled:   profile.Enabled,
			Home:      "/opt/data",
			IsDefault: profile.ID == defaultProfileID,
		}
		if profile.ID != defaultProfileID {
			item.Home = "/opt/data/profiles/" + profile.ID
		}
		if !profile.Enabled {
			item.Reason = "disabled"
		} else {
			binding, err := a.profilePlatformBinding(profile.ID)
			if err != nil {
				return RuntimeManifest{}, err
			}
			if binding.runnable {
				item.Runnable = true
			} else {
				item.Reason = "not_configured"
			}
		}
		manifest.Profiles = append(manifest.Profiles, item)
	}
	return manifest, nil
}

func (a *App) validateRuntimeProfiles(registry ProfileRegistry) error {
	wecomOwners := map[string]string{}
	weixinOwners := map[string]string{}
	feishuOwners := map[string]string{}
	dingtalkOwners := map[string]string{}
	for _, profile := range registry.Profiles {
		if err := validateProfileID(profile.ID, true); err != nil {
			return err
		}
		if !profile.Enabled {
			continue
		}
		if err := parseYAMLFile(filepath.Join(a.profileDataDir(profile.ID), "config.yaml"), nil); err != nil {
			return fmt.Errorf("%s 的 config.yaml 无法解析：%w", profile.Name, err)
		}
		binding, err := a.profilePlatformBinding(profile.ID)
		if err != nil {
			return fmt.Errorf("%s 的平台配置无效：%w", profile.Name, err)
		}
		if binding.wecomID != "" {
			if owner := wecomOwners[binding.wecomID]; owner != "" {
				return fmt.Errorf("企业微信 Bot 被多个启用 profile 使用：%s 和 %s", owner, profile.Name)
			}
			wecomOwners[binding.wecomID] = profile.Name
		}
		if binding.weixinID != "" {
			if owner := weixinOwners[binding.weixinID]; owner != "" {
				return fmt.Errorf("个人微信账号被多个启用 profile 使用：%s 和 %s", owner, profile.Name)
			}
			weixinOwners[binding.weixinID] = profile.Name
		}
		if binding.feishuAppID != "" {
			if owner := feishuOwners[binding.feishuAppID]; owner != "" {
				return fmt.Errorf("飞书 App 被多个启用 profile 使用：%s 和 %s", owner, profile.Name)
			}
			feishuOwners[binding.feishuAppID] = profile.Name
		}
		if binding.dingtalkClientID != "" {
			if owner := dingtalkOwners[binding.dingtalkClientID]; owner != "" {
				return fmt.Errorf("钉钉 AppKey 被多个启用 profile 使用：%s 和 %s", owner, profile.Name)
			}
			dingtalkOwners[binding.dingtalkClientID] = profile.Name
		}
	}
	return nil
}

type platformBinding struct {
	runnable         bool
	wecomID          string
	weixinID         string
	feishuAppID      string
	dingtalkClientID string
}

func (a *App) profilePlatformBinding(profileID string) (platformBinding, error) {
	env, err := readEnvFile(filepath.Join(a.profileDataDir(profileID), ".env"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return platformBinding{}, nil
		}
		return platformBinding{}, fmt.Errorf("读取 .env 失败：%w", err)
	}
	wecomID := strings.TrimSpace(envValue(env, "WECOM_BOT_ID"))
	wecomSecret := strings.TrimSpace(envValue(env, "WECOM_SECRET"))
	weixinID := strings.TrimSpace(envValue(env, "WEIXIN_ACCOUNT_ID"))
	weixinToken := strings.TrimSpace(envValue(env, "WEIXIN_TOKEN"))
	feishuAppID := strings.TrimSpace(envValue(env, "FEISHU_APP_ID"))
	feishuAppSecret := strings.TrimSpace(envValue(env, "FEISHU_APP_SECRET"))
	feishuConnectionMode := firstNonEmpty(strings.TrimSpace(envValue(env, "FEISHU_CONNECTION_MODE")), "websocket")
	dingtalkClientID := strings.TrimSpace(envValue(env, "DINGTALK_CLIENT_ID"))
	dingtalkClientSecret := strings.TrimSpace(envValue(env, "DINGTALK_CLIENT_SECRET"))
	if (wecomID == "") != (wecomSecret == "") {
		return platformBinding{}, fmt.Errorf("企业微信 Bot ID 和密钥必须同时填写")
	}
	if (weixinID == "") != (weixinToken == "") {
		return platformBinding{}, fmt.Errorf("个人微信账号和 token 必须同时存在")
	}
	if (feishuAppID == "") != (feishuAppSecret == "") {
		return platformBinding{}, fmt.Errorf("飞书 App ID 和 App Secret 必须同时填写")
	}
	if feishuAppID != "" && feishuConnectionMode != "websocket" {
		return platformBinding{}, fmt.Errorf("飞书第一版只支持 WebSocket 模式")
	}
	if (dingtalkClientID == "") != (dingtalkClientSecret == "") {
		return platformBinding{}, fmt.Errorf("钉钉 AppKey 和 AppSecret 必须同时填写")
	}
	return platformBinding{
		runnable:         (wecomID != "" && wecomSecret != "") || (weixinID != "" && weixinToken != "") || (feishuAppID != "" && feishuAppSecret != "") || (dingtalkClientID != "" && dingtalkClientSecret != ""),
		wecomID:          wecomID,
		weixinID:         weixinID,
		feishuAppID:      feishuAppID,
		dingtalkClientID: dingtalkClientID,
	}, nil
}

func (a *App) readRuntimeStatus(containerStatus string) RuntimeStatus {
	out := RuntimeStatus{Profiles: map[string]RuntimeProfileStatus{}}
	data, err := os.ReadFile(a.runtimeStatusPath())
	if err == nil {
		if err := json.Unmarshal(data, &out); err != nil {
			out = RuntimeStatus{Profiles: map[string]RuntimeProfileStatus{}}
		}
	}
	if out.Profiles == nil {
		out.Profiles = map[string]RuntimeProfileStatus{}
	}
	statusOutdated := a.runtimeStatusOutdated(out)
	registry, err := a.readProfileRegistry()
	if err != nil {
		return out
	}
	for _, profile := range registry.Profiles {
		status, ok := out.Profiles[profile.ID]
		if !ok || statusOutdated {
			out.Profiles[profile.ID] = a.derivedRuntimeProfileStatus(profile, containerStatus)
			continue
		}
		status.Enabled = profile.Enabled
		if profile.Enabled && (status.State == "" || status.State == "disabled") {
			out.Profiles[profile.ID] = a.derivedRuntimeProfileStatus(profile, containerStatus)
			continue
		}
		if !profile.Enabled {
			status.State = "disabled"
			status.PID = 0
		} else if containerStatus != "running" && (status.State == "running" || status.State == "starting") {
			status.State = "stopped"
			status.PID = 0
			status.Message = "容器未运行"
		}
		out.Profiles[profile.ID] = status
	}
	return out
}

func (a *App) runtimeStatusOutdated(status RuntimeStatus) bool {
	if status.UpdatedAt == "" {
		return true
	}
	var manifest RuntimeManifest
	data, err := os.ReadFile(a.runtimeManifestPath())
	if err != nil {
		return false
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return false
	}
	if manifest.GeneratedAt == "" {
		return false
	}
	if manifest.Generation != "" {
		return status.Generation != manifest.Generation
	}
	statusUpdatedAt, err := time.Parse(time.RFC3339, status.UpdatedAt)
	if err != nil {
		return true
	}
	manifestGeneratedAt, err := time.Parse(time.RFC3339, manifest.GeneratedAt)
	if err != nil {
		return false
	}
	return statusUpdatedAt.Before(manifestGeneratedAt)
}

func (a *App) derivedRuntimeProfileStatus(profile ProfileEntry, containerStatus string) RuntimeProfileStatus {
	status := RuntimeProfileStatus{Enabled: profile.Enabled}
	if !profile.Enabled {
		status.State = "disabled"
		status.Message = "profile disabled"
		return status
	}
	binding, err := a.profilePlatformBinding(profile.ID)
	if err != nil {
		status.State = "failed"
		status.Message = err.Error()
		return status
	}
	if !binding.runnable {
		status.State = "not_configured"
		status.Message = "not_configured"
		return status
	}
	if containerStatus == "running" {
		status.State = "starting"
		status.Message = "等待 runner 上报状态"
		return status
	}
	status.State = "stopped"
	status.Message = "容器未运行"
	return status
}

func (a *App) backupDirectory(path string, reason string) (err error) {
	if !fileExists(path) {
		return nil
	}
	id := newBackupID()
	reason = sanitizeName(firstNonEmpty(reason, "backup"))
	rel, err := filepath.Rel(a.instanceRoot, path)
	if err != nil {
		rel = filepath.Base(path)
	}
	backupRoot, err := a.createBackupRoot(id, reason)
	if err != nil {
		return err
	}
	target := filepath.Join(backupRoot, rel+".tar.gz")
	if err := ensureDir(filepath.Dir(target)); err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(target), ".directory-backup-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := file.Name()
	committed := false
	defer func() {
		_ = file.Close()
		if !committed {
			_ = os.Remove(tmpPath)
			_ = os.RemoveAll(backupRoot)
		}
	}()
	gz := gzip.NewWriter(file)
	tw := tar.NewWriter(gz)
	if err := filepath.Walk(path, func(item string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		linkTarget := ""
		if info.Mode()&os.ModeSymlink != 0 {
			linkTarget, err = os.Readlink(item)
			if err != nil {
				return err
			}
		}
		header, err := tar.FileInfoHeader(info, linkTarget)
		if err != nil {
			return err
		}
		header.Name, err = filepath.Rel(filepath.Dir(path), item)
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(header.Name)
		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		src, err := os.Open(item)
		if err != nil {
			return err
		}
		_, copyErr := io.CopyN(tw, src, info.Size())
		closeErr := src.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	}); err != nil {
		_ = tw.Close()
		_ = gz.Close()
		return err
	}
	if err := tw.Close(); err != nil {
		_ = gz.Close()
		return err
	}
	if err := gz.Close(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, target); err != nil {
		return err
	}
	if err := a.updateStateAllowMissing(func(state *LauncherState) error {
		state.Backups = append(state.Backups, BackupRecord{
			ID:     id,
			Reason: reason,
			Path:   strings.TrimPrefix(target, a.instanceRoot+string(os.PathSeparator)),
		})
		state.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		return nil
	}); err != nil {
		return err
	}
	committed = true
	return nil
}

func copyDir(source string, target string) error {
	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		dst := filepath.Join(target, rel)
		if info.IsDir() {
			return ensureDir(dst)
		}
		return copyFile(path, dst, info.Mode())
	})
}

func copyFile(source string, target string, mode os.FileMode) error {
	if err := ensureDir(filepath.Dir(target)); err != nil {
		return err
	}
	src, err := os.Open(source)
	if err != nil {
		return err
	}
	defer src.Close()
	dst, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer dst.Close()
	_, err = io.Copy(dst, src)
	return err
}
