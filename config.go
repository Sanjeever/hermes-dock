package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

func parseYAMLFile(path string, out interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return parseYAML(data, out)
}

func parseYAML(data []byte, out interface{}) error {
	if out == nil {
		var tmp interface{}
		return yaml.Unmarshal(data, &tmp)
	}
	return yaml.Unmarshal(data, out)
}

func (a *App) readModelConfig() (ModelConfig, error) {
	cfg, err := a.readConfigMap()
	if err != nil {
		return defaultModelConfig(), err
	}
	providers := normalizeProviderConfig(readProviderConfigFromMap(cfg))
	modelMap := asMap(cfg["model"])
	auxMap := asMap(cfg["auxiliary"])
	providerID := asString(modelMap["provider"])
	if _, ok := providers.Providers[providerID]; !ok {
		providerID = "dashscope-payg"
	}
	model := ModelConfig{
		Provider:      providerID,
		Default:       asString(modelMap["default"]),
		BaseURL:       asString(modelMap["base_url"]),
		APIMode:       asString(modelMap["api_mode"]),
		APIKey:        asString(modelMap["api_key"]),
		AuxiliaryMode: "auto",
		Auxiliary:     map[string]AuxModel{},
		RawProviders:  asMap(cfg["providers"]),
	}
	if mode := a.currentProfileAuxiliaryMode(); mode != "" {
		model.AuxiliaryMode = mode
	}
	if fallbacks, ok := cfg["fallback_providers"].([]interface{}); ok {
		for _, item := range fallbacks {
			model.Fallbacks = append(model.Fallbacks, asString(item))
		}
	}
	for _, name := range auxiliaryNames {
		entry := asMap(auxMap[name])
		auxProvider := firstNonEmpty(asString(entry["provider"]), "auto")
		if auxProvider != "auto" {
			if _, ok := providers.Providers[auxProvider]; !ok {
				auxProvider = providerID
			}
		}
		model.Auxiliary[name] = AuxModel{
			Provider:  auxProvider,
			Model:     asString(entry["model"]),
			BaseURL:   asString(entry["base_url"]),
			APIKey:    asString(entry["api_key"]),
			Timeout:   asInt(entry["timeout"]),
			ExtraBody: asMap(entry["extra_body"]),
		}
	}
	return model, nil
}

func defaultModelConfig() ModelConfig {
	aux := map[string]AuxModel{}
	for _, name := range auxiliaryNames {
		aux[name] = AuxModel{Provider: "auto", ExtraBody: map[string]interface{}{}}
	}
	return ModelConfig{
		Provider:      "dashscope-payg",
		Default:       "qwen3.7-max",
		BaseURL:       "https://dashscope.aliyuncs.com/compatible-mode/v1",
		APIMode:       "chat_completions",
		AuxiliaryMode: "auto",
		Auxiliary:     aux,
		RawProviders:  map[string]interface{}{},
	}
}

func (a *App) SaveModelConfig(model ModelConfig) error {
	cfg, err := a.readConfigMap()
	if err != nil {
		cfg = map[string]interface{}{}
	}
	if _, err := os.Stat(a.configPath()); err == nil {
		if err := a.backupFile(a.configPath(), "before-model-save"); err != nil {
			return err
		}
	}
	providers := normalizeProviderConfig(readProviderConfigFromMap(cfg))
	model = normalizeModelConfigReferences(model, providers)
	modelMap := asMap(cfg["model"])
	modelMap["provider"] = model.Provider
	modelMap["default"] = model.Default
	cfg["model"] = modelMap

	auxMap := asMap(cfg["auxiliary"])
	switch model.AuxiliaryMode {
	case "auto":
		for _, name := range auxiliaryNames {
			auxMap[name] = map[string]interface{}{
				"provider":   "auto",
				"model":      "",
				"base_url":   "",
				"api_mode":   "",
				"api_key":    "",
				"timeout":    defaultAuxTimeout(name),
				"extra_body": map[string]interface{}{},
			}
		}
	case "follow-main":
		for _, name := range auxiliaryNames {
			auxMap[name] = map[string]interface{}{
				"provider":   model.Provider,
				"model":      model.Default,
				"timeout":    defaultAuxTimeout(name),
				"extra_body": map[string]interface{}{},
			}
		}
	default:
		for name, aux := range model.Auxiliary {
			auxMap[name] = map[string]interface{}{
				"provider":   firstNonEmpty(aux.Provider, "auto"),
				"model":      aux.Model,
				"timeout":    aux.Timeout,
				"extra_body": aux.ExtraBody,
			}
		}
	}
	cfg["auxiliary"] = auxMap
	cfg["providers"] = providerConfigToYAMLMap(providers)
	applyProviderCompatibility(cfg, providers)

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	if err := ensureDir(a.currentProfileDataDir()); err != nil {
		return err
	}
	if err := atomicWriteFile(a.configPath(), data, 0644); err != nil {
		return err
	}
	if err := a.syncReferencedProviderEnv(providers); err != nil {
		return err
	}
	return a.updateCurrentProfileAuxiliaryMode(firstNonEmpty(model.AuxiliaryMode, "auto"))
}

func (a *App) normalizeModelConfigForSave(model ModelConfig) ModelConfig {
	preset := detectModelProviderPreset(model)
	if preset == nil {
		preset = &modelProviderPresets[0]
	}
	if model.Provider == "" {
		model.Provider = preset.Provider
	}
	if model.BaseURL == "" {
		model.BaseURL = preset.BaseURL
	}
	if model.APIMode == "" {
		model.APIMode = preset.APIMode
	}
	provider := strings.ToLower(strings.TrimSpace(model.Provider))
	baseURL := strings.ToLower(strings.TrimSpace(model.BaseURL))
	if preset.Key == "dashscope-payg" {
		if strings.Contains(baseURL, "dashscope.aliyuncs.com/apps/anthropic") {
			model.BaseURL = preset.BaseURL
		}
		if strings.EqualFold(strings.TrimSpace(model.APIMode), "anthropic_messages") {
			model.APIMode = preset.APIMode
		}
	}
	if provider == "dashscope" || provider == "alibaba" || provider == "alibaba-cloud" || provider == "qwen-dashscope" {
		if strings.Contains(model.BaseURL, "dashscope.aliyuncs.com") {
			model.Provider = "custom"
		}
	}
	if provider == "opencode" || provider == "opencode-go" {
		model.Provider = "custom"
	}
	return model
}

func (a *App) GetModelProviderPresets() []ModelProviderPreset {
	return modelProviderPresets
}

func (a *App) readProviderConfig() (ProviderConfig, error) {
	cfg, err := a.readConfigMap()
	if err != nil {
		return normalizeProviderConfig(ProviderConfig{}), err
	}
	return normalizeProviderConfig(readProviderConfigFromMap(cfg)), nil
}

func (a *App) GetProviderConfig() (ProviderConfig, error) {
	return a.readProviderConfig()
}

func (a *App) SaveProviderConfig(providers ProviderConfig) error {
	cfg, err := a.readConfigMap()
	if err != nil {
		cfg = map[string]interface{}{}
	}
	if _, err := os.Stat(a.configPath()); err == nil {
		if err := a.backupFile(a.configPath(), "before-provider-save"); err != nil {
			return err
		}
	}
	normalized := normalizeProviderConfig(providers)
	if err := validateProviderConfig(normalized); err != nil {
		return err
	}
	existing := normalizeProviderConfig(readProviderConfigFromMap(cfg))
	normalized = preserveExistingMaskedProviderSecrets(existing, normalized)
	for id := range existing.Providers {
		if _, ok := normalized.Providers[id]; ok {
			continue
		}
		if refs := providerReferences(cfg, id); len(refs) > 0 {
			return fmt.Errorf("供应商正在被使用，不能删除：%s", strings.Join(refs, "、"))
		}
	}
	cfg["providers"] = providerConfigToYAMLMap(normalized)
	applyProviderCompatibility(cfg, normalized)
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	if err := ensureDir(a.currentProfileDataDir()); err != nil {
		return err
	}
	if err := atomicWriteFile(a.configPath(), data, 0644); err != nil {
		return err
	}
	if err := a.syncReferencedProviderEnv(normalized); err != nil {
		return err
	}
	state, _ := a.readState()
	state.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	return a.writeState(state)
}

func preserveExistingMaskedProviderSecrets(existing ProviderConfig, next ProviderConfig) ProviderConfig {
	for id, entry := range next.Providers {
		if !isMaskedSecretPlaceholder(entry.APIKey) {
			continue
		}
		if existingEntry, ok := existing.Providers[id]; ok {
			entry.APIKey = existingEntry.APIKey
			next.Providers[id] = entry
		}
	}
	return next
}

func (a *App) DeleteProvider(id string) error {
	cfg, err := a.readConfigMap()
	if err != nil {
		return err
	}
	providers := normalizeProviderConfig(readProviderConfigFromMap(cfg))
	entry, ok := providers.Providers[id]
	if !ok {
		return fmt.Errorf("供应商不存在：%s", id)
	}
	if entry.Builtin {
		return fmt.Errorf("内置供应商不能删除")
	}
	if refs := providerReferences(cfg, id); len(refs) > 0 {
		return fmt.Errorf("供应商正在被使用：%s", strings.Join(refs, "、"))
	}
	delete(providers.Providers, id)
	return a.SaveProviderConfig(providers)
}

func (a *App) FetchProviderModelList(providerID string) ([]ModelOption, error) {
	cfg, err := a.readConfigMap()
	if err != nil {
		return nil, err
	}
	providers := normalizeProviderConfig(readProviderConfigFromMap(cfg))
	entry, ok := providers.Providers[providerID]
	if !ok {
		return nil, fmt.Errorf("供应商不存在：%s", providerID)
	}
	if entry.Disabled {
		return nil, fmt.Errorf("供应商已禁用：%s", entry.Label)
	}
	return fetchModelListFromProvider(entry)
}

func (a *App) FetchProviderConfigModelList(provider ProviderConfigEntry) ([]ModelOption, error) {
	if provider.Provider == "" {
		provider.Provider = "custom"
	}
	if provider.APIMode == "" {
		provider.APIMode = "chat_completions"
	}
	return fetchModelListFromProvider(provider)
}

func (a *App) FetchModelList(req ModelListRequest) ([]ModelOption, error) {
	providerID := firstNonEmpty(req.ProviderID, req.ProviderKey)
	if providerID != "" && req.APIKey == "" && req.BaseURL == "" {
		return a.FetchProviderModelList(providerID)
	}
	entry, err := providerEntryFromLegacyRequest(req)
	if err != nil {
		return nil, err
	}
	return fetchModelListFromProvider(entry)
}

func fetchModelListFromProvider(entry ProviderConfigEntry) ([]ModelOption, error) {
	apiKey := strings.TrimSpace(entry.APIKey)
	if apiKey == "" {
		return nil, fmt.Errorf("请先在供应商页填写 API 密钥")
	}
	modelListURL, err := resolveProviderModelListURL(entry)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 20 * time.Second}
	httpReq, err := http.NewRequest(http.MethodGet, modelListURL, nil)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Accept", "application/json")

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("拉取模型列表失败：HTTP %d %s", resp.StatusCode, compactBody(body))
	}

	var payload struct {
		Data []struct {
			ID      string `json:"id"`
			OwnedBy string `json:"owned_by"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("模型列表响应不是有效 JSON：%w", err)
	}

	seen := map[string]bool{}
	var models []ModelOption
	for _, item := range payload.Data {
		id := strings.TrimSpace(item.ID)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		models = append(models, ModelOption{ID: id, OwnedBy: item.OwnedBy})
	}
	sort.Slice(models, func(i, j int) bool {
		return models[i].ID < models[j].ID
	})
	if len(models) == 0 {
		return nil, fmt.Errorf("供应商返回的模型列表为空")
	}
	return models, nil
}

func providerEntryFromLegacyRequest(req ModelListRequest) (ProviderConfigEntry, error) {
	preset := modelProviderPresetByKey(req.ProviderKey)
	if preset != nil {
		return ProviderConfigEntry{
			Label:        preset.Label,
			Provider:     preset.Provider,
			BaseURL:      preset.BaseURL,
			APIMode:      preset.APIMode,
			APIKey:       req.APIKey,
			ModelListURL: preset.ModelListURL,
			DefaultModel: preset.DefaultModel,
			Builtin:      true,
		}, nil
	}
	if req.ProviderKey != "custom" {
		return ProviderConfigEntry{}, fmt.Errorf("不支持的模型供应商：%s", req.ProviderKey)
	}
	return ProviderConfigEntry{
		Label:        "自定义供应商",
		Provider:     "custom",
		BaseURL:      req.BaseURL,
		APIMode:      "chat_completions",
		APIKey:       req.APIKey,
		ModelListURL: "",
	}, nil
}

func resolveProviderModelListURL(provider ProviderConfigEntry) (string, error) {
	if strings.TrimSpace(provider.ModelListURL) != "" {
		if err := validateOptionalURL(provider.ModelListURL); err != nil {
			return "", fmt.Errorf("模型列表地址不是有效 URL")
		}
		return strings.TrimSpace(provider.ModelListURL), nil
	}
	baseURL := strings.TrimSpace(provider.BaseURL)
	if baseURL == "" {
		return "", fmt.Errorf("请先填写接口地址")
	}
	if err := validateRequiredURL(baseURL); err != nil {
		return "", fmt.Errorf("接口地址不是有效 URL")
	}
	baseURL = strings.TrimRight(baseURL, "/")
	for _, suffix := range []string{"/chat/completions", "/messages", "/responses"} {
		if strings.HasSuffix(baseURL, suffix) {
			baseURL = strings.TrimSuffix(baseURL, suffix)
			break
		}
	}
	return baseURL + "/models", nil
}

func readProviderConfigFromMap(cfg map[string]interface{}) ProviderConfig {
	out := ProviderConfig{Providers: map[string]ProviderConfigEntry{}}
	for id, value := range asMap(cfg["providers"]) {
		entryMap := asMap(value)
		out.Providers[id] = ProviderConfigEntry{
			Label:        asString(entryMap["label"]),
			Provider:     asString(entryMap["provider"]),
			BaseURL:      asString(entryMap["base_url"]),
			APIMode:      asString(entryMap["api_mode"]),
			APIKey:       asString(entryMap["api_key"]),
			ModelListURL: asString(entryMap["model_list_url"]),
			DefaultModel: asString(entryMap["default_model"]),
			Builtin:      asBool(entryMap["builtin"]),
			Disabled:     asBool(entryMap["disabled"]),
		}
	}
	return out
}

func normalizeProviderConfig(config ProviderConfig) ProviderConfig {
	if config.Providers == nil {
		config.Providers = map[string]ProviderConfigEntry{}
	}
	for _, preset := range modelProviderPresets {
		existing, ok := config.Providers[preset.Key]
		config.Providers[preset.Key] = normalizeProviderEntry(existing, preset, ok)
	}
	for id, entry := range config.Providers {
		if modelProviderPresetByKey(id) != nil {
			continue
		}
		if entry.Label == "" {
			entry.Label = id
		}
		if entry.Provider == "" {
			entry.Provider = "custom"
		}
		if entry.APIMode == "" {
			entry.APIMode = "chat_completions"
		}
		entry.Builtin = false
		config.Providers[id] = entry
	}
	return config
}

func normalizeProviderEntry(entry ProviderConfigEntry, preset ModelProviderPreset, existed bool) ProviderConfigEntry {
	if !existed {
		return ProviderConfigEntry{
			Label:        preset.Label,
			Provider:     preset.Provider,
			BaseURL:      preset.BaseURL,
			APIMode:      preset.APIMode,
			APIKey:       "",
			ModelListURL: preset.ModelListURL,
			DefaultModel: preset.DefaultModel,
			Builtin:      true,
			Disabled:     false,
		}
	}
	if preset.Key == "dashscope-payg" && entry.Label == "DashScope 按量计费" {
		entry.Label = preset.Label
	}
	entry.Label = firstNonEmpty(entry.Label, preset.Label)
	entry.Provider = firstNonEmpty(entry.Provider, preset.Provider)
	entry.BaseURL = firstNonEmpty(entry.BaseURL, preset.BaseURL)
	entry.APIMode = firstNonEmpty(entry.APIMode, preset.APIMode)
	entry.ModelListURL = firstNonEmpty(entry.ModelListURL, preset.ModelListURL)
	entry.DefaultModel = firstNonEmpty(entry.DefaultModel, preset.DefaultModel)
	entry.Builtin = true
	return entry
}

func providerConfigToYAMLMap(config ProviderConfig) map[string]interface{} {
	out := map[string]interface{}{}
	keys := make([]string, 0, len(config.Providers))
	for key := range config.Providers {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		entry := config.Providers[key]
		out[key] = map[string]interface{}{
			"label":          entry.Label,
			"provider":       entry.Provider,
			"base_url":       entry.BaseURL,
			"api_mode":       entry.APIMode,
			"api_key":        entry.APIKey,
			"model_list_url": entry.ModelListURL,
			"default_model":  entry.DefaultModel,
			"builtin":        entry.Builtin,
			"disabled":       entry.Disabled,
		}
	}
	return out
}

func validateProviderConfig(config ProviderConfig) error {
	if len(config.Providers) == 0 {
		return fmt.Errorf("至少需要一个供应商")
	}
	for id, entry := range config.Providers {
		if !validProviderID(id) {
			return fmt.Errorf("供应商 ID 无效：%s", id)
		}
		if strings.TrimSpace(entry.Label) == "" {
			return fmt.Errorf("供应商名称不能为空：%s", id)
		}
		if entry.Provider == "" {
			return fmt.Errorf("供应商类型不能为空：%s", id)
		}
		if err := validateRequiredURL(entry.BaseURL); err != nil {
			return fmt.Errorf("%s 的接口地址无效：%w", entry.Label, err)
		}
		if entry.APIMode != "chat_completions" && entry.APIMode != "anthropic_messages" {
			return fmt.Errorf("%s 的 API 模式无效：%s", entry.Label, entry.APIMode)
		}
		if err := validateOptionalURL(entry.ModelListURL); err != nil {
			return fmt.Errorf("%s 的模型列表地址无效：%w", entry.Label, err)
		}
		if entry.Builtin && modelProviderPresetByKey(id) == nil {
			return fmt.Errorf("非内置供应商不能标记为内置：%s", id)
		}
	}
	return nil
}

func validProviderID(id string) bool {
	if id == "" {
		return false
	}
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			continue
		}
		return false
	}
	return true
}

func validateRequiredURL(value string) error {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("不是有效 URL")
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return fmt.Errorf("必须以 http:// 或 https:// 开头")
	}
	return nil
}

func validateOptionalURL(value string) error {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return validateRequiredURL(value)
}

func modelProviderPresetByKey(key string) *ModelProviderPreset {
	for i := range modelProviderPresets {
		if modelProviderPresets[i].Key == key {
			return &modelProviderPresets[i]
		}
	}
	return nil
}

func normalizeModelConfigReferences(model ModelConfig, providers ProviderConfig) ModelConfig {
	if _, ok := providers.Providers[model.Provider]; !ok {
		model.Provider = "dashscope-payg"
	}
	if strings.TrimSpace(model.Default) == "" {
		model.Default = providers.Providers[model.Provider].DefaultModel
	}
	for name, aux := range model.Auxiliary {
		if aux.Provider == "" {
			aux.Provider = "auto"
		}
		if aux.Provider != "auto" {
			if _, ok := providers.Providers[aux.Provider]; !ok {
				aux.Provider = model.Provider
			}
		}
		if aux.Timeout == 0 {
			aux.Timeout = defaultAuxTimeout(name)
		}
		if aux.ExtraBody == nil {
			aux.ExtraBody = map[string]interface{}{}
		}
		model.Auxiliary[name] = aux
	}
	return model
}

func applyProviderCompatibility(cfg map[string]interface{}, providers ProviderConfig) {
	modelMap := asMap(cfg["model"])
	modelProviderID := asString(modelMap["provider"])
	if _, ok := providers.Providers[modelProviderID]; !ok {
		modelProviderID = "dashscope-payg"
		modelMap["provider"] = modelProviderID
	}
	if entry, ok := providers.Providers[modelProviderID]; ok {
		applyProviderFields(modelMap, entry)
		if strings.TrimSpace(asString(modelMap["default"])) == "" {
			modelMap["default"] = entry.DefaultModel
		}
	}
	cfg["model"] = modelMap

	auxMap := asMap(cfg["auxiliary"])
	for _, name := range auxiliaryNames {
		entry := asMap(auxMap[name])
		providerID := firstNonEmpty(asString(entry["provider"]), "auto")
		if providerID == "auto" {
			entry["provider"] = "auto"
			entry["base_url"] = ""
			entry["api_mode"] = ""
			entry["api_key"] = ""
		} else if provider, ok := providers.Providers[providerID]; ok {
			applyProviderFields(entry, provider)
		} else if provider, ok := providers.Providers[modelProviderID]; ok {
			entry["provider"] = modelProviderID
			applyProviderFields(entry, provider)
		}
		if _, ok := entry["timeout"]; !ok || asInt(entry["timeout"]) == 0 {
			entry["timeout"] = defaultAuxTimeout(name)
		}
		if _, ok := entry["extra_body"]; !ok {
			entry["extra_body"] = map[string]interface{}{}
		}
		auxMap[name] = entry
	}
	cfg["auxiliary"] = auxMap
}

func applyProviderFields(target map[string]interface{}, provider ProviderConfigEntry) {
	target["base_url"] = provider.BaseURL
	target["api_mode"] = provider.APIMode
	target["api_key"] = provider.APIKey
}

func providerReferences(cfg map[string]interface{}, providerID string) []string {
	var refs []string
	modelMap := asMap(cfg["model"])
	if asString(modelMap["provider"]) == providerID {
		refs = append(refs, "主模型")
	}
	auxMap := asMap(cfg["auxiliary"])
	for _, name := range auxiliaryNames {
		if asString(asMap(auxMap[name])["provider"]) == providerID {
			refs = append(refs, "辅助模型："+auxLabel(name))
		}
	}
	return refs
}

func auxLabel(name string) string {
	switch name {
	case "vision":
		return "视觉理解"
	case "web_extract":
		return "网页提取"
	case "compression":
		return "上下文压缩"
	default:
		return name
	}
}

func (a *App) syncModelProviderEnv(model ModelConfig) error {
	updates := modelProviderEnvUpdates(model)
	if len(updates) == 0 {
		return nil
	}
	existing, _ := readEnvFile(a.envPath())
	changed := false
	for _, item := range updates {
		if envValue(existing, item.Key) != item.Value {
			changed = true
			break
		}
	}
	if !changed {
		return nil
	}
	return a.SaveEnvironment(updates)
}

func (a *App) syncReferencedProviderEnv(providers ProviderConfig) error {
	cfg, err := a.readConfigMap()
	if err != nil {
		return err
	}
	updates := referencedProviderEnvUpdates(cfg, providers)
	if len(updates) == 0 {
		return nil
	}
	existing, _ := readEnvFile(a.envPath())
	changed := false
	for _, item := range updates {
		if envValue(existing, item.Key) != item.Value {
			changed = true
			break
		}
	}
	if !changed {
		return nil
	}
	return a.SaveEnvironment(updates)
}

func referencedProviderEnvUpdates(cfg map[string]interface{}, providers ProviderConfig) []EnvVar {
	ids := map[string]bool{}
	modelMap := asMap(cfg["model"])
	if id := asString(modelMap["provider"]); id != "" {
		ids[id] = true
	}
	auxMap := asMap(cfg["auxiliary"])
	for _, name := range auxiliaryNames {
		id := asString(asMap(auxMap[name])["provider"])
		if id != "" && id != "auto" {
			ids[id] = true
		}
	}
	byKey := map[string]EnvVar{}
	for id := range ids {
		provider, ok := providers.Providers[id]
		if !ok {
			continue
		}
		apiKey := strings.TrimSpace(provider.APIKey)
		if apiKey == "" {
			continue
		}
		key := modelProviderAPIKeyEnv(provider.Provider, provider.BaseURL)
		if key == "" {
			continue
		}
		byKey[key] = EnvVar{Key: key, Value: apiKey, Secret: true}
	}
	return orderedEnvUpdates(byKey)
}

func modelProviderEnvUpdates(model ModelConfig) []EnvVar {
	byKey := map[string]EnvVar{}
	add := func(provider string, baseURL string, apiKey string) {
		apiKey = strings.TrimSpace(apiKey)
		if apiKey == "" {
			return
		}
		key := modelProviderAPIKeyEnv(provider, baseURL)
		if key == "" {
			return
		}
		byKey[key] = EnvVar{Key: key, Value: apiKey, Secret: true}
	}

	add(model.Provider, model.BaseURL, model.APIKey)
	if model.AuxiliaryMode == "custom" {
		for _, aux := range model.Auxiliary {
			add(aux.Provider, aux.BaseURL, aux.APIKey)
		}
	}

	return orderedEnvUpdates(byKey)
}

func orderedEnvUpdates(byKey map[string]EnvVar) []EnvVar {
	order := []string{"OPENCODE_GO_API_KEY", "DASHSCOPE_API_KEY", "DEEPSEEK_API_KEY", "AGNES_API_KEY"}
	updates := make([]EnvVar, 0, len(byKey))
	for _, key := range order {
		if item, ok := byKey[key]; ok {
			updates = append(updates, item)
			delete(byKey, key)
		}
	}
	for _, item := range byKey {
		updates = append(updates, item)
	}
	return updates
}

func modelProviderAPIKeyEnv(provider string, baseURL string) string {
	provider = strings.ToLower(strings.TrimSpace(provider))
	baseURL = strings.ToLower(strings.TrimSpace(baseURL))
	switch {
	case provider == "custom" && strings.Contains(baseURL, "apihub.agnes-ai.com"):
		return "AGNES_API_KEY"
	case provider == "deepseek" || strings.Contains(baseURL, "api.deepseek.com"):
		return "DEEPSEEK_API_KEY"
	case provider == "opencode" || provider == "opencode-go" || strings.Contains(baseURL, "opencode.ai/zen/go"):
		return "OPENCODE_GO_API_KEY"
	case provider == "custom" && (strings.Contains(baseURL, "dashscope.aliyuncs.com") || strings.Contains(baseURL, "maas.aliyuncs.com")):
		return "DASHSCOPE_API_KEY"
	case provider == "dashscope" || provider == "alibaba" || provider == "alibaba-cloud" || provider == "qwen-dashscope":
		return "DASHSCOPE_API_KEY"
	case provider == "custom" && strings.Contains(baseURL, "bigmodel.cn"):
		return "ZHIPU_API_KEY"
	default:
		return ""
	}
}

func detectModelProviderPreset(model ModelConfig) *ModelProviderPreset {
	provider := strings.ToLower(strings.TrimSpace(model.Provider))
	baseURL := strings.ToLower(strings.TrimSpace(model.BaseURL))
	switch {
	case provider == "custom" && strings.Contains(baseURL, "apihub.agnes-ai.com"):
		return modelProviderPresetByKey("agnes")
	case provider == "deepseek" || strings.Contains(baseURL, "api.deepseek.com"):
		return modelProviderPresetByKey("deepseek")
	case provider == "opencode" || provider == "opencode-go" || strings.Contains(baseURL, "opencode.ai/zen/go"):
		return modelProviderPresetByKey("opencode-go")
	case provider == "custom" && strings.Contains(baseURL, "dashscope.aliyuncs.com"):
		return modelProviderPresetByKey("dashscope-payg")
	case provider == "dashscope" || provider == "alibaba" || provider == "alibaba-cloud" || provider == "qwen-dashscope":
		return modelProviderPresetByKey("dashscope-payg")
	case provider == "custom" && strings.Contains(baseURL, "bigmodel.cn"):
		return modelProviderPresetByKey("zhipu-payg")
	default:
		return nil
	}
}

func compactBody(body []byte) string {
	text := strings.TrimSpace(string(body))
	if text == "" {
		return ""
	}
	text = strings.Join(strings.Fields(text), " ")
	if len(text) > 300 {
		return text[:300] + "..."
	}
	return text
}

func (a *App) readConfigMap() (map[string]interface{}, error) {
	cfg := map[string]interface{}{}
	err := parseYAMLFile(a.configPath(), &cfg)
	if err != nil {
		return cfg, err
	}
	return cfg, nil
}

func defaultAuxTimeout(name string) int {
	switch name {
	case "web_extract":
		return 360
	case "curator":
		return 600
	case "triage_specifier":
		return 120
	case "kanban_decomposer":
		return 180
	default:
		return 30
	}
}

func asMap(value interface{}) map[string]interface{} {
	if value == nil {
		return map[string]interface{}{}
	}
	if typed, ok := value.(map[string]interface{}); ok {
		return typed
	}
	if typed, ok := value.(map[interface{}]interface{}); ok {
		out := map[string]interface{}{}
		for key, item := range typed {
			out[asString(key)] = item
		}
		return out
	}
	return map[string]interface{}{}
}

func asString(value interface{}) string {
	if value == nil {
		return ""
	}
	if typed, ok := value.(string); ok {
		return typed
	}
	return ""
}

func asInt(value interface{}) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	default:
		return 0
	}
}

func asBool(value interface{}) bool {
	if typed, ok := value.(bool); ok {
		return typed
	}
	return false
}
