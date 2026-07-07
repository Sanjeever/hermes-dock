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

func TestAgnesProviderAPIKeyEnv(t *testing.T) {
	key := modelProviderAPIKeyEnv("custom", "https://apihub.agnes-ai.com/v1")
	if key != "AGNES_API_KEY" {
		t.Fatalf("env key = %q, want AGNES_API_KEY", key)
	}
}
