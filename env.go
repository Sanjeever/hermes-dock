package main

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var secretFragments = []string{"KEY", "TOKEN", "SECRET", "PASSWORD", "PASS", "AUTH"}

func readEnvFile(path string) ([]EnvVar, error) {
	file, err := os.Open(path)
	if err != nil {
		return defaultEnvVars(), err
	}
	defer file.Close()

	var vars []EnvVar
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || !strings.Contains(line, "=") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if key == "" {
			continue
		}
		vars = append(vars, EnvVar{Key: key, Value: unquoteEnv(value), Secret: isSecretKey(key)})
	}
	return mergeDefaultEnvVars(vars), scanner.Err()
}

func (a *App) SaveEnvironment(vars []EnvVar) error {
	return a.saveEnvironmentTo(a.envPath(), vars)
}

func (a *App) saveEnvironmentTo(path string, vars []EnvVar) error {
	if err := ensureDir(filepath.Dir(path)); err != nil {
		return err
	}
	if _, err := os.Stat(path); err == nil {
		if err := a.backupFile(path, "before-env-save"); err != nil {
			return err
		}
	}
	existing, _ := readEnvFile(path)
	merged := mergeEnv(existing, vars)
	return writeEnvFile(path, merged)
}

func writeEnvFile(path string, vars []EnvVar) error {
	sort.SliceStable(vars, func(i, j int) bool {
		return envOrder(vars[i].Key) < envOrder(vars[j].Key)
	})
	var b strings.Builder
	b.WriteString("# 由企智盒管理。通过界面编辑时，未知变量会保留。\n")
	for _, item := range vars {
		if strings.TrimSpace(item.Key) == "" {
			continue
		}
		b.WriteString(item.Key)
		b.WriteString("=")
		b.WriteString(quoteEnv(item.Value))
		b.WriteString("\n")
	}
	return os.WriteFile(path, []byte(b.String()), 0600)
}

func mergeEnv(existing []EnvVar, updates []EnvVar) []EnvVar {
	seen := map[string]int{}
	result := make([]EnvVar, 0, len(existing)+len(updates))
	for _, item := range existing {
		item.Secret = isSecretKey(item.Key)
		seen[item.Key] = len(result)
		result = append(result, item)
	}
	for _, item := range updates {
		item.Key = strings.TrimSpace(item.Key)
		item.Secret = isSecretKey(item.Key)
		if item.Key == "" {
			continue
		}
		if idx, ok := seen[item.Key]; ok {
			result[idx] = item
		} else {
			seen[item.Key] = len(result)
			result = append(result, item)
		}
	}
	return mergeDefaultEnvVars(result)
}

func defaultEnvVars() []EnvVar {
	defaults := map[string]string{
		"OPENCODE_GO_API_KEY":                  "",
		"DASHSCOPE_API_KEY":                    "",
		"DEEPSEEK_API_KEY":                     "",
		"WEIXIN_ACCOUNT_ID":                    "",
		"WEIXIN_TOKEN":                         "",
		"WEIXIN_BASE_URL":                      "",
		"WEIXIN_CDN_BASE_URL":                  "https://novac2c.cdn.weixin.qq.com/c2c",
		"WEIXIN_DM_POLICY":                     "open",
		"WEIXIN_ALLOW_ALL_USERS":               "true",
		"WEIXIN_ALLOWED_USERS":                 "",
		"WEIXIN_GROUP_POLICY":                  "open",
		"WEIXIN_GROUP_ALLOWED_USERS":           "",
		"WEIXIN_HOME_CHANNEL":                  "",
		"WECOM_BOT_ID":                         "",
		"WECOM_SECRET":                         "",
		"WECOM_WEBSOCKET_URL":                  "wss://openws.work.weixin.qq.com",
		"WECOM_DM_POLICY":                      "open",
		"WECOM_ALLOWED_USERS":                  "",
		"WECOM_GROUP_POLICY":                   "open",
		"WECOM_GROUP_ALLOWED_USERS":            "",
		"FEISHU_APP_ID":                        "",
		"FEISHU_APP_SECRET":                    "",
		"FEISHU_DOMAIN":                        "feishu",
		"FEISHU_CONNECTION_MODE":               "websocket",
		"FEISHU_ALLOW_ALL_USERS":               "true",
		"FEISHU_ALLOWED_USERS":                 "",
		"FEISHU_GROUP_POLICY":                  "open",
		"TERMINAL_LIFETIME_SECONDS":            "86400",
		"HERMES_DASHBOARD_BASIC_AUTH_USERNAME": "admin",
		"HERMES_DASHBOARD_BASIC_AUTH_PASSWORD": "123456",
		"HERMES_DASHBOARD":                     "1",
	}
	var vars []EnvVar
	for key, value := range defaults {
		vars = append(vars, EnvVar{Key: key, Value: value, Secret: isSecretKey(key)})
	}
	return vars
}

func mergeDefaultEnvVars(vars []EnvVar) []EnvVar {
	return mergeEnvNoDefaults(vars, defaultEnvVars())
}

func mergeEnvNoDefaults(existing []EnvVar, defaults []EnvVar) []EnvVar {
	seen := map[string]bool{}
	for i := range existing {
		existing[i].Secret = isSecretKey(existing[i].Key)
		seen[existing[i].Key] = true
	}
	for _, item := range defaults {
		if !seen[item.Key] {
			existing = append(existing, item)
		}
	}
	return existing
}

func envValue(vars []EnvVar, key string) string {
	for _, item := range vars {
		if item.Key == key {
			return item.Value
		}
	}
	return ""
}

func isSecretKey(key string) bool {
	upper := strings.ToUpper(key)
	for _, fragment := range secretFragments {
		if strings.Contains(upper, fragment) {
			return true
		}
	}
	return false
}

func quoteEnv(value string) string {
	if value == "" {
		return ""
	}
	if strings.ContainsAny(value, " #\t\n\"'") {
		escaped := strings.ReplaceAll(value, "\\", "\\\\")
		escaped = strings.ReplaceAll(escaped, "\"", "\\\"")
		return "\"" + escaped + "\""
	}
	return value
}

func unquoteEnv(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
		value = strings.TrimPrefix(strings.TrimSuffix(value, "\""), "\"")
		value = strings.ReplaceAll(value, "\\\"", "\"")
		value = strings.ReplaceAll(value, "\\\\", "\\")
	}
	return value
}

func envOrder(key string) int {
	order := []string{
		"OPENCODE_GO_API_KEY",
		"DASHSCOPE_API_KEY", "DEEPSEEK_API_KEY",
		"WEIXIN_ACCOUNT_ID", "WEIXIN_TOKEN", "WEIXIN_BASE_URL", "WEIXIN_CDN_BASE_URL",
		"WEIXIN_DM_POLICY", "WEIXIN_ALLOW_ALL_USERS", "WEIXIN_ALLOWED_USERS",
		"WEIXIN_GROUP_POLICY", "WEIXIN_GROUP_ALLOWED_USERS", "WEIXIN_HOME_CHANNEL",
		"WECOM_BOT_ID", "WECOM_SECRET", "WECOM_WEBSOCKET_URL",
		"WECOM_DM_POLICY", "WECOM_ALLOWED_USERS", "WECOM_GROUP_POLICY", "WECOM_GROUP_ALLOWED_USERS",
		"FEISHU_APP_ID", "FEISHU_APP_SECRET", "FEISHU_DOMAIN", "FEISHU_CONNECTION_MODE",
		"FEISHU_ALLOW_ALL_USERS", "FEISHU_ALLOWED_USERS", "FEISHU_GROUP_POLICY",
		"TERMINAL_LIFETIME_SECONDS",
		"HERMES_DASHBOARD", "HERMES_DASHBOARD_BASIC_AUTH_USERNAME", "HERMES_DASHBOARD_BASIC_AUTH_PASSWORD",
		"HERMES_GATEWAY_BUSY_INPUT_MODE", "HERMES_GATEWAY_BUSY_ACK_ENABLED", "HERMES_BACKGROUND_NOTIFICATIONS",
	}
	for idx, item := range order {
		if item == key {
			return idx
		}
	}
	return 1000
}
