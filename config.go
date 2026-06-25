package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

var auxiliaryNames = []string{
	"vision",
	"web_extract",
	"compression",
	"skills_hub",
	"approval",
	"mcp",
	"title_generation",
	"tts_audio_tags",
	"triage_specifier",
	"kanban_decomposer",
	"profile_describer",
	"curator",
	"monitor",
}

var modelProviderPresets = []ModelProviderPreset{
	{
		Key:          "dashscope-payg",
		Label:        "DashScope 按量计费",
		Provider:     "custom",
		BaseURL:      "https://dashscope.aliyuncs.com/compatible-mode/v1",
		APIMode:      "chat_completions",
		DefaultModel: "qwen3.7-max",
		ModelListURL: "https://dashscope.aliyuncs.com/compatible-mode/v1/models",
	},
	{
		Key:          "opencode-go",
		Label:        "OpenCode Go",
		Provider:     "custom",
		BaseURL:      "https://opencode.ai/zen/go/v1",
		APIMode:      "chat_completions",
		DefaultModel: "deepseek-v4-flash",
		ModelListURL: "https://opencode.ai/zen/go/v1/models",
	},
	{
		Key:          "deepseek",
		Label:        "DeepSeek",
		Provider:     "deepseek",
		BaseURL:      "https://api.deepseek.com",
		APIMode:      "chat_completions",
		DefaultModel: "deepseek-v4-flash",
		ModelListURL: "https://api.deepseek.com/models",
	},
}

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
	modelMap := asMap(cfg["model"])
	auxMap := asMap(cfg["auxiliary"])
	model := ModelConfig{
		Provider:      asString(modelMap["provider"]),
		Default:       asString(modelMap["default"]),
		BaseURL:       asString(modelMap["base_url"]),
		APIMode:       asString(modelMap["api_mode"]),
		APIKey:        asString(modelMap["api_key"]),
		AuxiliaryMode: "auto",
		Auxiliary:     map[string]AuxModel{},
		RawProviders:  asMap(cfg["providers"]),
	}
	state, _ := a.readState()
	if state.ModelAuxiliaryMode != "" {
		model.AuxiliaryMode = state.ModelAuxiliaryMode
	}
	if fallbacks, ok := cfg["fallback_providers"].([]interface{}); ok {
		for _, item := range fallbacks {
			model.Fallbacks = append(model.Fallbacks, asString(item))
		}
	}
	for _, name := range auxiliaryNames {
		entry := asMap(auxMap[name])
		model.Auxiliary[name] = AuxModel{
			Provider:  firstNonEmpty(asString(entry["provider"]), "auto"),
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
		Provider:      "custom",
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
	model = a.normalizeModelConfigForSave(model)
	modelMap := asMap(cfg["model"])
	modelMap["provider"] = model.Provider
	modelMap["default"] = model.Default
	modelMap["base_url"] = model.BaseURL
	modelMap["api_mode"] = model.APIMode
	if model.APIKey != "" {
		modelMap["api_key"] = model.APIKey
	}
	cfg["model"] = modelMap

	auxMap := asMap(cfg["auxiliary"])
	switch model.AuxiliaryMode {
	case "auto":
		for _, name := range auxiliaryNames {
			auxMap[name] = map[string]interface{}{
				"provider":   "auto",
				"model":      "",
				"base_url":   "",
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
				"base_url":   model.BaseURL,
				"api_key":    model.APIKey,
				"timeout":    defaultAuxTimeout(name),
				"extra_body": map[string]interface{}{},
			}
		}
	default:
		for name, aux := range model.Auxiliary {
			auxMap[name] = map[string]interface{}{
				"provider":   firstNonEmpty(aux.Provider, "auto"),
				"model":      aux.Model,
				"base_url":   aux.BaseURL,
				"api_key":    aux.APIKey,
				"timeout":    aux.Timeout,
				"extra_body": aux.ExtraBody,
			}
		}
	}
	cfg["auxiliary"] = auxMap

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	if err := ensureDir(a.dataDir()); err != nil {
		return err
	}
	if err := os.WriteFile(a.configPath(), data, 0644); err != nil {
		return err
	}
	if err := a.syncModelProviderEnv(model); err != nil {
		return err
	}
	state, _ := a.readState()
	state.ModelAuxiliaryMode = firstNonEmpty(model.AuxiliaryMode, "auto")
	state.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	return a.writeState(state)
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

func (a *App) FetchModelList(req ModelListRequest) ([]ModelOption, error) {
	preset := modelProviderPresetByKey(req.ProviderKey)
	if preset == nil {
		return nil, fmt.Errorf("不支持的模型供应商：%s", req.ProviderKey)
	}
	apiKey := strings.TrimSpace(req.APIKey)
	if apiKey == "" {
		return nil, fmt.Errorf("请先在模型页填写 API 密钥")
	}

	client := &http.Client{Timeout: 20 * time.Second}
	httpReq, err := http.NewRequest(http.MethodGet, preset.ModelListURL, nil)
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

func modelProviderPresetByKey(key string) *ModelProviderPreset {
	for i := range modelProviderPresets {
		if modelProviderPresets[i].Key == key {
			return &modelProviderPresets[i]
		}
	}
	return nil
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

	order := []string{"OPENCODE_GO_API_KEY", "DASHSCOPE_API_KEY", "DEEPSEEK_API_KEY"}
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
	case provider == "deepseek" || strings.Contains(baseURL, "api.deepseek.com"):
		return "DEEPSEEK_API_KEY"
	case provider == "opencode" || provider == "opencode-go" || strings.Contains(baseURL, "opencode.ai/zen/go"):
		return "OPENCODE_GO_API_KEY"
	case provider == "custom" && strings.Contains(baseURL, "dashscope.aliyuncs.com"):
		return "DASHSCOPE_API_KEY"
	case provider == "dashscope" || provider == "alibaba" || provider == "alibaba-cloud" || provider == "qwen-dashscope":
		return "DASHSCOPE_API_KEY"
	default:
		return ""
	}
}

func detectModelProviderPreset(model ModelConfig) *ModelProviderPreset {
	provider := strings.ToLower(strings.TrimSpace(model.Provider))
	baseURL := strings.ToLower(strings.TrimSpace(model.BaseURL))
	switch {
	case provider == "deepseek" || strings.Contains(baseURL, "api.deepseek.com"):
		return modelProviderPresetByKey("deepseek")
	case provider == "opencode" || provider == "opencode-go" || strings.Contains(baseURL, "opencode.ai/zen/go"):
		return modelProviderPresetByKey("opencode-go")
	case provider == "custom" && strings.Contains(baseURL, "dashscope.aliyuncs.com"):
		return modelProviderPresetByKey("dashscope-payg")
	case provider == "dashscope" || provider == "alibaba" || provider == "alibaba-cloud" || provider == "qwen-dashscope":
		return modelProviderPresetByKey("dashscope-payg")
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
