package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type batchSkillSource struct {
	path string
	dir  string
}

func (a *App) BatchCopyProfileConfig(req BatchProfileConfigRequest) (BatchProfileConfigResult, error) {
	release, err := a.beginExclusiveOperation("批量修改")
	if err != nil {
		return BatchProfileConfigResult{}, err
	}
	defer release()
	registry, err := a.readProfileRegistry()
	if err != nil {
		return BatchProfileConfigResult{}, err
	}
	sourceID := strings.TrimSpace(req.SourceProfileID)
	if !profileExists(registry, sourceID) {
		return BatchProfileConfigResult{}, fmt.Errorf("源 profile 不存在：%s", sourceID)
	}
	if len(req.TargetProfileIDs) == 0 {
		return BatchProfileConfigResult{}, fmt.Errorf("请至少选择一个目标 profile")
	}
	if !req.CopyMainModel && !req.CopyAuxiliary && !req.CopySoul && !req.CopyProviders && len(req.SkillPaths) == 0 {
		return BatchProfileConfigResult{}, fmt.Errorf("请至少选择一项要复制的内容")
	}
	if req.IncludeAPIKeys && !req.CopyProviders {
		return BatchProfileConfigResult{}, fmt.Errorf("复制 API Key 时必须同时复制供应商定义")
	}

	sourceConfig, err := a.readConfigMapForProfile(sourceID)
	if err != nil {
		return BatchProfileConfigResult{}, fmt.Errorf("读取源 profile 配置失败：%w", err)
	}
	sourceProviders := normalizeProviderConfig(readProviderConfigFromMap(sourceConfig))
	sourceModel, err := a.readModelConfigForProfile(sourceID)
	if err != nil {
		return BatchProfileConfigResult{}, err
	}
	skills, err := a.resolveBatchSkillSources(sourceID, req.SkillPaths)
	if err != nil {
		return BatchProfileConfigResult{}, err
	}

	seen := map[string]bool{}
	result := BatchProfileConfigResult{Results: []ProfileOperationResult{}}
	changedAny := false
	for _, rawID := range req.TargetProfileIDs {
		targetID := strings.TrimSpace(rawID)
		item := ProfileOperationResult{ProfileID: targetID}
		switch {
		case seen[targetID]:
			item.Error = "目标 profile 重复"
		case targetID == sourceID:
			item.Error = "源 profile 不能同时作为目标"
		case !profileExists(registry, targetID):
			item.Error = "目标 profile 不存在"
		default:
			seen[targetID] = true
			changed, err := a.copyProfileConfigSelection(sourceID, targetID, sourceConfig, sourceProviders, sourceModel, skills, req)
			item.Changed = changed
			changedAny = changedAny || changed
			if err != nil {
				item.Error = redact(err.Error())
				if changed {
					item.Error = "已完成部分写入，后续步骤失败：" + item.Error
				}
			} else {
				item.Success = true
				item.Changed = changed
			}
		}
		if item.Success {
			result.Succeeded++
		} else {
			result.Failed++
		}
		result.Results = append(result.Results, item)
	}
	if changedAny {
		if err := a.markRebuildRequired(); err != nil {
			return result, err
		}
	}
	return result, nil
}

func (a *App) resolveBatchSkillSources(profileID string, paths []string) ([]batchSkillSource, error) {
	seen := map[string]bool{}
	items := make([]batchSkillSource, 0, len(paths))
	for _, path := range paths {
		path = filepath.ToSlash(strings.TrimSpace(path))
		if path == "" || seen[path] {
			continue
		}
		_, dir, err := a.resolveSkillDirForProfile(profileID, path)
		if err != nil {
			return nil, fmt.Errorf("读取源技能失败：%s：%w", path, err)
		}
		if err := ensureNoSymlinkComponents(a.profileDataDir(profileID), dir); err != nil {
			return nil, fmt.Errorf("源技能路径不安全：%s：%w", path, err)
		}
		if err := ensureTreeHasNoSymlinks(dir); err != nil {
			return nil, fmt.Errorf("源技能包含不安全文件：%s：%w", path, err)
		}
		summary, err := a.skillSummaryFromDirForProfile(profileID, dir, a.bundledSkillNames())
		if err != nil {
			return nil, fmt.Errorf("读取源技能失败：%s：%w", path, err)
		}
		seen[path] = true
		items = append(items, batchSkillSource{path: summary.Path, dir: dir})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].path < items[j].path })
	return items, nil
}

func (a *App) copyProfileConfigSelection(
	sourceID string,
	targetID string,
	sourceConfig map[string]interface{},
	sourceProviders ProviderConfig,
	sourceModel ModelConfig,
	skills []batchSkillSource,
	req BatchProfileConfigRequest,
) (bool, error) {
	changed := false
	if req.CopyMainModel || req.CopyAuxiliary || req.CopyProviders {
		targetConfig, err := a.readConfigMapForProfile(targetID)
		if err != nil {
			return false, fmt.Errorf("读取 config.yaml 失败：%w", err)
		}
		targetProviders := normalizeProviderConfig(readProviderConfigFromMap(targetConfig))
		if req.CopyProviders {
			for id, sourceEntry := range sourceProviders.Providers {
				targetEntry, exists := targetProviders.Providers[id]
				if !req.IncludeAPIKeys {
					if exists && strings.TrimSpace(targetEntry.APIKey) != "" && !providerSecretIdentityMatches(sourceEntry, targetEntry) {
						return false, fmt.Errorf("供应商 %s 的接口地址或协议不同；为避免把目标 API Key 用于新的接口，请显式选择复制 API Key 或先清空目标密钥", id)
					}
					if exists && providerSecretIdentityMatches(sourceEntry, targetEntry) {
						sourceEntry.APIKey = targetEntry.APIKey
					} else {
						sourceEntry.APIKey = ""
					}
				}
				targetProviders.Providers[id] = sourceEntry
			}
		}
		neededProviders := selectedProviderIDs(sourceModel, req.CopyMainModel, req.CopyAuxiliary)
		for id := range neededProviders {
			if _, ok := targetProviders.Providers[id]; !ok {
				return false, fmt.Errorf("目标缺少供应商 %s，请选择复制供应商定义", id)
			}
		}
		if req.CopyMainModel {
			source := asMap(sourceConfig["model"])
			target := asMap(targetConfig["model"])
			target["provider"] = asString(source["provider"])
			target["default"] = asString(source["default"])
			targetConfig["model"] = target
			fallbacks := make([]interface{}, 0, len(sourceModel.Fallbacks))
			for _, fallback := range sourceModel.Fallbacks {
				fallbacks = append(fallbacks, fallback)
			}
			targetConfig["fallback_providers"] = fallbacks
		}
		if req.CopyAuxiliary {
			auxiliary, err := cloneYAMLMap(asMap(sourceConfig["auxiliary"]))
			if err != nil {
				return false, err
			}
			targetConfig["auxiliary"] = auxiliary
		}
		targetConfig["providers"] = providerConfigToYAMLMap(targetProviders)
		applyProviderCompatibility(targetConfig, targetProviders)
		data, err := yaml.Marshal(targetConfig)
		if err != nil {
			return false, err
		}
		configPath := a.profileConfigPath(targetID)
		existing, err := os.ReadFile(configPath)
		if err != nil {
			return false, err
		}
		if string(existing) != string(data) {
			if err := a.backupFile(configPath, "before-batch-profile-config-"+targetID); err != nil {
				return false, err
			}
			if err := atomicWriteFile(configPath, data, 0644); err != nil {
				return false, err
			}
			changed = true
			if err := a.syncReferencedProviderEnvForProfile(targetID, targetProviders); err != nil {
				return changed, err
			}
		}
		if req.CopyAuxiliary && a.profileAuxiliaryMode(targetID) != sourceModel.AuxiliaryMode {
			if err := a.updateProfileAuxiliaryMode(targetID, sourceModel.AuxiliaryMode); err != nil {
				return changed, err
			}
			changed = true
		}
	}

	if req.CopySoul {
		source, err := os.ReadFile(a.profileSoulPath(sourceID))
		if err != nil {
			return changed, fmt.Errorf("读取源 SOUL.md 失败：%w", err)
		}
		next := []byte(rewriteProfileContainerHome(string(source), sourceID, targetID))
		targetPath := a.profileSoulPath(targetID)
		current, err := os.ReadFile(targetPath)
		if err != nil && !os.IsNotExist(err) {
			return changed, err
		}
		if string(current) != string(next) {
			if err := a.backupFile(targetPath, "before-batch-profile-soul-"+targetID); err != nil {
				return changed, err
			}
			if err := atomicWriteFile(targetPath, next, 0644); err != nil {
				return changed, err
			}
			changed = true
		}
	}

	for _, skill := range skills {
		targetDir := filepath.Join(a.profileDataDir(targetID), filepath.FromSlash(skill.path))
		if err := ensureNoSymlinkComponents(a.profileDataDir(targetID), targetDir); err != nil {
			return changed, fmt.Errorf("目标技能路径不安全：%s：%w", skill.path, err)
		}
		if fileExists(targetDir) {
			if err := a.backupDirectory(targetDir, "before-batch-profile-skill-"+targetID); err != nil {
				return changed, err
			}
		}
		committed, err := replaceSkillDirectory(skill.dir, targetDir)
		if committed {
			changed = true
		}
		if err != nil {
			return changed, err
		}
	}
	return changed, nil
}

func ensureTreeHasNoSymlinks(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("不支持符号链接：%s", path)
		}
		if !info.IsDir() && !info.Mode().IsRegular() {
			return fmt.Errorf("不支持特殊文件：%s", path)
		}
		return nil
	})
}

func replaceSkillDirectory(source string, target string) (bool, error) {
	if err := ensureTreeHasNoSymlinks(source); err != nil {
		return false, err
	}
	parent := filepath.Dir(target)
	if err := ensureDir(parent); err != nil {
		return false, err
	}
	staged, err := os.MkdirTemp(parent, ".batch-skill-*")
	if err != nil {
		return false, err
	}
	stagedCommitted := false
	defer func() {
		if !stagedCommitted {
			_ = os.RemoveAll(staged)
		}
	}()
	if err := copyDirectoryFiles(source, staged); err != nil {
		return false, err
	}

	old := ""
	if fileExists(target) {
		old, err = os.MkdirTemp(parent, ".previous-skill-*")
		if err != nil {
			return false, err
		}
		if err := os.Remove(old); err != nil {
			return false, err
		}
		if err := os.Rename(target, old); err != nil {
			return false, err
		}
	}
	if err := os.Rename(staged, target); err != nil {
		if old != "" {
			if rollbackErr := os.Rename(old, target); rollbackErr != nil {
				return false, errors.Join(err, fmt.Errorf("回滚目标技能目录失败：%w", rollbackErr))
			}
		}
		return false, err
	}
	stagedCommitted = true
	if old != "" {
		if err := os.RemoveAll(old); err != nil {
			return true, fmt.Errorf("技能已复制，但无法清理旧目录：%w", err)
		}
	}
	return true, nil
}

func copyDirectoryFiles(source string, target string) error {
	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		destination := filepath.Join(target, rel)
		if info.IsDir() {
			if err := ensureDir(destination); err != nil {
				return err
			}
			return os.Chmod(destination, info.Mode().Perm())
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("不支持特殊文件：%s", path)
		}
		return copyFile(path, destination, info.Mode().Perm())
	})
}

func providerSecretIdentityMatches(source ProviderConfigEntry, target ProviderConfigEntry) bool {
	normalizeEndpoint := func(value string) string {
		return strings.TrimRight(strings.TrimSpace(value), "/")
	}
	return strings.TrimSpace(source.Provider) == strings.TrimSpace(target.Provider) &&
		normalizeEndpoint(source.BaseURL) == normalizeEndpoint(target.BaseURL) &&
		strings.TrimSpace(source.APIMode) == strings.TrimSpace(target.APIMode)
}

func selectedProviderIDs(model ModelConfig, main bool, auxiliary bool) map[string]bool {
	ids := map[string]bool{}
	if main && model.Provider != "" {
		ids[model.Provider] = true
		for _, fallback := range model.Fallbacks {
			if fallback != "" && fallback != "auto" {
				ids[fallback] = true
			}
		}
	}
	if auxiliary {
		for _, item := range model.Auxiliary {
			if item.Provider != "" && item.Provider != "auto" {
				ids[item.Provider] = true
			}
		}
	}
	return ids
}

func cloneYAMLMap(source map[string]interface{}) (map[string]interface{}, error) {
	data, err := yaml.Marshal(source)
	if err != nil {
		return nil, err
	}
	var target map[string]interface{}
	if err := yaml.Unmarshal(data, &target); err != nil {
		return nil, err
	}
	return target, nil
}
