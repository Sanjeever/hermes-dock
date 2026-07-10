package main

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
	{
		Key:          "agnes",
		Label:        "Agnes AI",
		Provider:     "custom",
		BaseURL:      "https://apihub.agnes-ai.com/v1",
		APIMode:      "chat_completions",
		DefaultModel: "agnes-2.0-flash",
		ModelListURL: "https://apihub.agnes-ai.com/v1/models",
	},
}
