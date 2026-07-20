package main

import (
	"os"
	"testing"
)

func TestSaveProviderConfigPreservesExistingAPIKeyWhenRequestIsMasked(t *testing.T) {
	app := NewApp()
	app.instanceRoot = t.TempDir()
	if err := ensureDir(app.defaultDataDir()); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(app.configPath(), []byte(`model:
  provider: dashscope-payg
  default: qwen3.7-max
providers:
  dashscope-payg:
    label: DashScope 按量计费
    provider: custom
    base_url: https://dashscope.aliyuncs.com/compatible-mode/v1
    api_mode: chat_completions
    api_key: real-secret
    model_list_url: https://dashscope.aliyuncs.com/compatible-mode/v1/models
    default_model: qwen3.7-max
    builtin: true
    disabled: false
`), 0644); err != nil {
		t.Fatal(err)
	}

	providers, err := app.readProviderConfig()
	if err != nil {
		t.Fatal(err)
	}
	entry := providers.Providers["dashscope-payg"]
	entry.APIKey = "******"
	providers.Providers["dashscope-payg"] = entry

	if err := app.SaveProviderConfig(providers); err != nil {
		t.Fatal(err)
	}

	saved, err := app.readProviderConfig()
	if err != nil {
		t.Fatal(err)
	}
	if got := saved.Providers["dashscope-payg"].APIKey; got != "real-secret" {
		t.Fatalf("provider api_key = %q, want real-secret", got)
	}
	env, err := readEnvFile(app.envPath())
	if err != nil {
		t.Fatal(err)
	}
	if got := envValue(env, "DASHSCOPE_API_KEY"); got != "real-secret" {
		t.Fatalf("DASHSCOPE_API_KEY = %q, want real-secret", got)
	}
}

func TestSaveModelConfigRejectsInvalidExistingConfig(t *testing.T) {
	app := NewApp()
	app.instanceRoot = t.TempDir()
	if err := ensureDir(app.defaultDataDir()); err != nil {
		t.Fatal(err)
	}
	original := []byte("model: [\nterminal:\n  cwd: /opt/data\n")
	if err := os.WriteFile(app.configPath(), original, 0644); err != nil {
		t.Fatal(err)
	}

	if err := app.SaveModelConfig(defaultModelConfig()); err == nil {
		t.Fatal("invalid config.yaml should block model save")
	}
	saved, err := os.ReadFile(app.configPath())
	if err != nil {
		t.Fatal(err)
	}
	if string(saved) != string(original) {
		t.Fatalf("invalid config.yaml was overwritten: %q", saved)
	}
}

func TestSaveProviderConfigRejectsInvalidExistingConfig(t *testing.T) {
	app := NewApp()
	app.instanceRoot = t.TempDir()
	if err := ensureDir(app.defaultDataDir()); err != nil {
		t.Fatal(err)
	}
	original := []byte("providers: {\nterminal:\n  cwd: /opt/data\n")
	if err := os.WriteFile(app.configPath(), original, 0644); err != nil {
		t.Fatal(err)
	}

	if err := app.SaveProviderConfig(ProviderConfig{}); err == nil {
		t.Fatal("invalid config.yaml should block provider save")
	}
	saved, err := os.ReadFile(app.configPath())
	if err != nil {
		t.Fatal(err)
	}
	if string(saved) != string(original) {
		t.Fatalf("invalid config.yaml was overwritten: %q", saved)
	}
}

func TestProfileScopedModelSaveDoesNotFollowGlobalSelection(t *testing.T) {
	app := newTestApp(t)
	if err := app.CreateProfile(CreateProfileRequest{ID: "sales", CopyMode: profileCopyClean}); err != nil {
		t.Fatal(err)
	}
	if err := app.SelectProfile("sales"); err != nil {
		t.Fatal(err)
	}
	model, err := app.readModelConfigForProfile("default")
	if err != nil {
		t.Fatal(err)
	}
	model.Default = "default-only-model"
	if err := app.SaveModelConfigForProfile("default", model); err != nil {
		t.Fatal(err)
	}
	defaultModel, err := app.readModelConfigForProfile("default")
	if err != nil {
		t.Fatal(err)
	}
	salesModel, err := app.readModelConfigForProfile("sales")
	if err != nil {
		t.Fatal(err)
	}
	if defaultModel.Default != "default-only-model" {
		t.Fatalf("default model = %q", defaultModel.Default)
	}
	if salesModel.Default == "default-only-model" {
		t.Fatal("profile-scoped save wrote to globally selected sales profile")
	}
	state, err := app.GetAppStateForProfile("default")
	if err != nil {
		t.Fatal(err)
	}
	if state.ActiveProfile != "default" || state.Model.Default != "default-only-model" {
		t.Fatalf("scoped app state = %+v", state)
	}
}

func TestNormalizeAgnesDefaults(t *testing.T) {
	app := NewApp()
	model := app.normalizeModelConfigForSave(ModelConfig{
		Provider: "custom",
		BaseURL:  "https://apihub.agnes-ai.com/v1",
	})
	if model.Provider != "custom" {
		t.Fatalf("provider = %q, want custom", model.Provider)
	}
	if model.BaseURL != "https://apihub.agnes-ai.com/v1" {
		t.Fatalf("base URL = %q", model.BaseURL)
	}
	if model.APIMode != "chat_completions" {
		t.Fatalf("api mode = %q", model.APIMode)
	}
}

func TestNormalizeProviderConfigAddsBailianPlansAndRenamesPayg(t *testing.T) {
	providers := normalizeProviderConfig(ProviderConfig{Providers: map[string]ProviderConfigEntry{
		"dashscope-payg": {Label: "DashScope 按量计费"},
	}})

	if got := providers.Providers["dashscope-payg"].Label; got != "百炼按量计费" {
		t.Fatalf("dashscope-payg label = %q", got)
	}
	if got := providers.Providers["bailian-coding-plan"].BaseURL; got != "https://coding.dashscope.aliyuncs.com/v1" {
		t.Fatalf("bailian-coding-plan base URL = %q", got)
	}
	if got := providers.Providers["bailian-token-plan-team"].BaseURL; got != "https://token-plan.cn-beijing.maas.aliyuncs.com/compatible-mode/v1" {
		t.Fatalf("bailian-token-plan-team base URL = %q", got)
	}
}

func TestBailianTokenPlanAPIKeyEnv(t *testing.T) {
	key := modelProviderAPIKeyEnv("custom", "https://token-plan.cn-beijing.maas.aliyuncs.com/compatible-mode/v1")
	if key != "DASHSCOPE_API_KEY" {
		t.Fatalf("env key = %q, want DASHSCOPE_API_KEY", key)
	}
}

func TestZhipuProviderAPIKeyEnv(t *testing.T) {
	key := modelProviderAPIKeyEnv("custom", "https://open.bigmodel.cn/api/paas/v4")
	if key != "ZHIPU_API_KEY" {
		t.Fatalf("env key = %q, want ZHIPU_API_KEY", key)
	}
}

func TestNormalizeProviderConfigAddsZhipuPresets(t *testing.T) {
	providers := normalizeProviderConfig(ProviderConfig{Providers: map[string]ProviderConfigEntry{}})

	if got := providers.Providers["zhipu-payg"].BaseURL; got != "https://open.bigmodel.cn/api/paas/v4" {
		t.Fatalf("zhipu-payg base URL = %q", got)
	}
	if got := providers.Providers["zhipu-coding-plan"].BaseURL; got != "https://open.bigmodel.cn/api/coding/paas/v4" {
		t.Fatalf("zhipu-coding-plan base URL = %q", got)
	}
}

func TestNormalizeProviderConfigAddsVolcengineArkPlans(t *testing.T) {
	providers := normalizeProviderConfig(ProviderConfig{Providers: map[string]ProviderConfigEntry{}})
	provider := providers.Providers["volcengine-ark-coding-plan"]

	if provider.Label != "火山方舟 Coding Plan" {
		t.Fatalf("label = %q", provider.Label)
	}
	if provider.BaseURL != "https://ark.cn-beijing.volces.com/api/coding/v3" {
		t.Fatalf("base URL = %q", provider.BaseURL)
	}
	if provider.DefaultModel != "doubao-seed-2.0-code" {
		t.Fatalf("default model = %q", provider.DefaultModel)
	}
	if provider.APIMode != "chat_completions" {
		t.Fatalf("api mode = %q", provider.APIMode)
	}
	if provider.ModelListURL != "https://ark.cn-beijing.volces.com/api/coding/v3/models" {
		t.Fatalf("model list URL = %q", provider.ModelListURL)
	}
	if key := modelProviderAPIKeyEnv(provider.Provider, provider.BaseURL); key != "ARK_API_KEY" {
		t.Fatalf("env key = %q, want ARK_API_KEY", key)
	}

	agentPlan := providers.Providers["volcengine-ark-agent-plan"]
	if agentPlan.Label != "火山方舟 Agent Plan" {
		t.Fatalf("label = %q", agentPlan.Label)
	}
	if agentPlan.BaseURL != "https://ark.cn-beijing.volces.com/api/plan/v3" {
		t.Fatalf("base URL = %q", agentPlan.BaseURL)
	}
	if agentPlan.DefaultModel != "doubao-seed-2.0-code" {
		t.Fatalf("default model = %q", agentPlan.DefaultModel)
	}
	if agentPlan.APIMode != "chat_completions" {
		t.Fatalf("api mode = %q", agentPlan.APIMode)
	}
	if agentPlan.ModelListURL != "https://ark.cn-beijing.volces.com/api/plan/v3/models" {
		t.Fatalf("model list URL = %q", agentPlan.ModelListURL)
	}
	if key := modelProviderAPIKeyEnv(agentPlan.Provider, agentPlan.BaseURL); key != "ARK_AGENT_PLAN_API_KEY" {
		t.Fatalf("env key = %q, want ARK_AGENT_PLAN_API_KEY", key)
	}
}

func TestVolcengineArkPlansUseSeparateAPIKeyEnv(t *testing.T) {
	providers := normalizeProviderConfig(ProviderConfig{Providers: map[string]ProviderConfigEntry{}})
	codingPlan := providers.Providers["volcengine-ark-coding-plan"]
	codingPlan.APIKey = "coding-secret"
	providers.Providers["volcengine-ark-coding-plan"] = codingPlan
	agentPlan := providers.Providers["volcengine-ark-agent-plan"]
	agentPlan.APIKey = "agent-secret"
	providers.Providers["volcengine-ark-agent-plan"] = agentPlan

	codingKey := modelProviderAPIKeyEnv(codingPlan.Provider, codingPlan.BaseURL)
	agentKey := modelProviderAPIKeyEnv(agentPlan.Provider, agentPlan.BaseURL)
	if codingKey == agentKey {
		t.Fatalf("coding and agent plans share env key %q", codingKey)
	}

	updates := referencedProviderEnvUpdates(map[string]interface{}{
		"model": map[string]interface{}{"provider": "volcengine-ark-coding-plan"},
		"auxiliary": map[string]interface{}{
			"vision": map[string]interface{}{"provider": "volcengine-ark-agent-plan"},
		},
	}, providers)
	if len(updates) != 2 {
		t.Fatalf("env updates = %#v, want two separate keys", updates)
	}
	if updates[0].Key != "ARK_API_KEY" || updates[0].Value != "coding-secret" {
		t.Fatalf("coding env update = %#v", updates[0])
	}
	if updates[1].Key != "ARK_AGENT_PLAN_API_KEY" || updates[1].Value != "agent-secret" {
		t.Fatalf("agent env update = %#v", updates[1])
	}
}

func TestDetectVolcengineArkPlanPresets(t *testing.T) {
	agentPlan := detectModelProviderPreset(ModelConfig{
		Provider: "custom",
		BaseURL:  "https://ark.cn-beijing.volces.com/api/plan/v3",
	})
	if agentPlan == nil || agentPlan.Key != "volcengine-ark-agent-plan" {
		t.Fatalf("agent plan preset = %#v", agentPlan)
	}

	codingPlan := detectModelProviderPreset(ModelConfig{
		Provider: "custom",
		BaseURL:  "https://ark.cn-beijing.volces.com/api/coding/v3",
	})
	if codingPlan == nil || codingPlan.Key != "volcengine-ark-coding-plan" {
		t.Fatalf("coding plan preset = %#v", codingPlan)
	}

	for _, baseURL := range []string{
		"https://ark.cn-beijing.volces.com/api/plan/v3-preview",
		"https://ark.cn-beijing.volces.com/api/other/v3?next=/api/plan/v3",
		"https://ark.cn-beijing.volces.com.evil.example/api/plan/v3",
	} {
		if preset := detectModelProviderPreset(ModelConfig{Provider: "custom", BaseURL: baseURL}); preset != nil {
			t.Fatalf("unexpected preset for %q: %#v", baseURL, preset)
		}
		if key := modelProviderAPIKeyEnv("custom", baseURL); key != "" {
			t.Fatalf("unexpected env key for %q: %q", baseURL, key)
		}
	}
}

func TestAgnesProviderAPIKeyEnv(t *testing.T) {
	key := modelProviderAPIKeyEnv("custom", "https://apihub.agnes-ai.com/v1")
	if key != "AGNES_API_KEY" {
		t.Fatalf("env key = %q, want AGNES_API_KEY", key)
	}
}
