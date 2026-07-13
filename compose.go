package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	fallbackMemoryLimit = "4G"
	fallbackCPULimit    = "2.0"
)

func defaultComposeSettings() ComposeSettings {
	return ComposeSettings{
		Image:                   defaultImage,
		ContainerName:           "hermes",
		GatewayHost:             "127.0.0.1",
		GatewayPort:             "8642",
		DashboardHost:           "127.0.0.1",
		DashboardPort:           "9119",
		DashboardEnabled:        true,
		DashboardUsername:       "admin",
		DashboardPassword:       "123456",
		GatewayBusyInputMode:    "steer",
		GatewayBusyAckEnabled:   "false",
		BackgroundNotifications: "result",
		HostControlEnabled:      "true",
		MemoryLimit:             fallbackMemoryLimit,
		CPULimit:                fallbackCPULimit,
		ShmSize:                 "1g",
	}
}

func withComposeDefaults(settings ComposeSettings) ComposeSettings {
	defaults := defaultComposeSettings()
	if settings.Image == "" {
		settings.Image = defaults.Image
	}
	if settings.ContainerName == "" {
		settings.ContainerName = defaults.ContainerName
	}
	if settings.GatewayHost == "" {
		settings.GatewayHost = defaults.GatewayHost
	}
	if settings.GatewayPort == "" {
		settings.GatewayPort = defaults.GatewayPort
	}
	if settings.DashboardHost == "" {
		settings.DashboardHost = defaults.DashboardHost
	}
	if settings.DashboardPort == "" {
		settings.DashboardPort = defaults.DashboardPort
	}
	settings.DashboardEnabled = true
	if settings.DashboardUsername == "" {
		settings.DashboardUsername = defaults.DashboardUsername
	}
	if settings.DashboardPassword == "" {
		settings.DashboardPassword = defaults.DashboardPassword
	}
	if settings.GatewayBusyInputMode == "" {
		settings.GatewayBusyInputMode = defaults.GatewayBusyInputMode
	}
	if !oneOf(settings.GatewayBusyInputMode, "queue", "steer", "interrupt") {
		settings.GatewayBusyInputMode = defaults.GatewayBusyInputMode
	}
	if settings.GatewayBusyAckEnabled == "" {
		settings.GatewayBusyAckEnabled = defaults.GatewayBusyAckEnabled
	}
	if !oneOf(settings.GatewayBusyAckEnabled, "true", "false") {
		settings.GatewayBusyAckEnabled = defaults.GatewayBusyAckEnabled
	}
	if settings.BackgroundNotifications == "" {
		settings.BackgroundNotifications = defaults.BackgroundNotifications
	}
	if settings.HostControlEnabled == "" {
		settings.HostControlEnabled = defaults.HostControlEnabled
	}
	if !oneOf(settings.HostControlEnabled, "true", "false") {
		settings.HostControlEnabled = defaults.HostControlEnabled
	}
	if !oneOf(settings.BackgroundNotifications, "all", "result", "error", "off") {
		settings.BackgroundNotifications = defaults.BackgroundNotifications
	}
	if settings.MemoryLimit == "" {
		settings.MemoryLimit = defaults.MemoryLimit
	}
	if settings.CPULimit == "" {
		settings.CPULimit = defaults.CPULimit
	}
	if settings.ShmSize == "" {
		settings.ShmSize = defaults.ShmSize
	}
	return settings
}

func (a *App) withRecommendedResourceDefaults(settings ComposeSettings) ComposeSettings {
	recommendation, err := a.recommendedResourceLimits(context.Background())
	if err != nil {
		if settings.MemoryLimit == "" {
			settings.MemoryLimit = fallbackMemoryLimit
		}
		if settings.CPULimit == "" {
			settings.CPULimit = fallbackCPULimit
		}
		return settings
	}
	if settings.MemoryLimit == "" {
		settings.MemoryLimit = recommendation.MemoryLimit
	}
	if settings.CPULimit == "" {
		settings.CPULimit = recommendation.CPULimit
	}
	return settings
}

func (a *App) withInitialResourceDefaults(settings ComposeSettings) ComposeSettings {
	settings.MemoryLimit = ""
	settings.CPULimit = ""
	return a.withRecommendedResourceDefaults(settings)
}

func oneOf(value string, allowed ...string) bool {
	for _, item := range allowed {
		if value == item {
			return true
		}
	}
	return false
}

func (a *App) readComposeSettings() ComposeSettings {
	state, _ := a.readState()
	settings := defaultComposeSettings()
	if state.ComposeSettings.Image != "" {
		settings = state.ComposeSettings
		if settings.MemoryLimit == "" || settings.CPULimit == "" {
			settings = a.withRecommendedResourceDefaults(settings)
		}
		settings = withComposeDefaults(settings)
	}
	if state.HermesImage != "" {
		settings.Image = state.HermesImage
	}
	env, _ := readEnvFile(a.defaultEnvPath())
	if value := envValue(env, "HERMES_DASHBOARD_BASIC_AUTH_USERNAME"); value != "" {
		settings.DashboardUsername = value
	}
	if value := envValue(env, "HERMES_DASHBOARD_BASIC_AUTH_PASSWORD"); value != "" {
		settings.DashboardPassword = value
	}
	if value := envValue(env, "HERMES_GATEWAY_BUSY_INPUT_MODE"); value != "" {
		settings.GatewayBusyInputMode = value
	}
	if value := envValue(env, "HERMES_GATEWAY_BUSY_ACK_ENABLED"); value != "" {
		settings.GatewayBusyAckEnabled = value
	}
	if value := envValue(env, "HERMES_BACKGROUND_NOTIFICATIONS"); value != "" {
		settings.BackgroundNotifications = value
	}
	return withComposeDefaults(settings)
}

func (a *App) ensureComposeResourceDefaults() error {
	state, err := a.readState()
	if err != nil || state.ComposeSettings.Image == "" {
		return nil
	}
	if state.ComposeSettings.MemoryLimit != "" && state.ComposeSettings.CPULimit != "" {
		return nil
	}
	settings := a.withRecommendedResourceDefaults(state.ComposeSettings)
	settings = withComposeDefaults(settings)
	if err := a.writeCompose(settings, "before-compose-resource-defaults"); err != nil {
		return err
	}
	state.ComposeSettings = settings
	state.ComposeHash = fileSHA256(a.composePath())
	state.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	return a.writeState(state)
}

func (a *App) SaveComposeSettings(settings ComposeSettings) error {
	settings = withComposeDefaults(settings)
	if err := a.syncComposeEnv(settings); err != nil {
		return err
	}
	if err := a.writeCompose(settings, "before-compose-save"); err != nil {
		return err
	}
	state, _ := a.readState()
	state.PreviousHermesImage = state.HermesImage
	state.HermesImage = settings.Image
	state.ComposeSettings = settings
	state.ComposeHash = fileSHA256(a.composePath())
	state.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if err := a.writeState(state); err != nil {
		return err
	}
	return a.syncHostBridge(settings.HostControlEnabled == "true")
}

func (a *App) syncComposeEnv(settings ComposeSettings) error {
	settings = withComposeDefaults(settings)
	updates := []EnvVar{
		{Key: "HERMES_DASHBOARD", Value: "1"},
		{Key: "HERMES_DASHBOARD_BASIC_AUTH_USERNAME", Value: settings.DashboardUsername},
		{Key: "HERMES_DASHBOARD_BASIC_AUTH_PASSWORD", Value: settings.DashboardPassword},
		{Key: "HERMES_GATEWAY_BUSY_INPUT_MODE", Value: settings.GatewayBusyInputMode},
		{Key: "HERMES_GATEWAY_BUSY_ACK_ENABLED", Value: settings.GatewayBusyAckEnabled},
		{Key: "HERMES_BACKGROUND_NOTIFICATIONS", Value: settings.BackgroundNotifications},
	}
	existing, _ := readEnvFile(a.defaultEnvPath())
	for _, item := range updates {
		if envValue(existing, item.Key) != item.Value {
			return a.saveEnvironmentTo(a.defaultEnvPath(), updates)
		}
	}
	return nil
}

func (a *App) GetRecommendedResourceLimits() (ResourceLimitsRecommendation, error) {
	return a.recommendedResourceLimits(context.Background())
}

func (a *App) recommendedResourceLimits(ctx context.Context) (ResourceLimitsRecommendation, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "info", "--format", "{{json .}}")
	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() != nil {
			return ResourceLimitsRecommendation{}, fmt.Errorf("读取 Docker 可用资源超时")
		}
		detail := strings.TrimSpace(string(output))
		if detail != "" {
			return ResourceLimitsRecommendation{}, fmt.Errorf("无法读取 Docker 可用资源：%w：%s", err, detail)
		}
		return ResourceLimitsRecommendation{}, fmt.Errorf("无法读取 Docker 可用资源：%w", err)
	}

	var info struct {
		MemTotal int64 `json:"MemTotal"`
		NCPU     int   `json:"NCPU"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(string(output))), &info); err != nil {
		return ResourceLimitsRecommendation{}, fmt.Errorf("无法解析 Docker 可用资源：%w", err)
	}
	return calculateRecommendedResourceLimits(info.MemTotal, info.NCPU)
}

func calculateRecommendedResourceLimits(memTotalBytes int64, ncpu int) (ResourceLimitsRecommendation, error) {
	if memTotalBytes <= 0 || ncpu <= 0 {
		return ResourceLimitsRecommendation{}, fmt.Errorf("Docker 可用资源无效")
	}
	const gib = int64(1024 * 1024 * 1024)
	memoryGB := int(memTotalBytes / gib)
	recommendedMemoryGB := memoryGB - 2
	if recommendedMemoryGB < 1 {
		recommendedMemoryGB = 1
	}
	return ResourceLimitsRecommendation{
		MemoryLimit:    fmt.Sprintf("%dG", recommendedMemoryGB),
		CPULimit:       fmt.Sprintf("%.1f", float64(ncpu)),
		DockerMemoryGB: memoryGB,
		DockerCPU:      ncpu,
	}, nil
}

func (a *App) writeCompose(settings ComposeSettings, reason string) error {
	if _, err := os.Stat(a.composePath()); err == nil {
		if err := a.backupFile(a.composePath(), reason); err != nil {
			return err
		}
	}
	content := renderCompose(settings, a.readProxySettings())
	return atomicWriteFile(a.composePath(), []byte(content), 0644)
}

func (a *App) migrateComposeIfNeeded(settings ComposeSettings) error {
	data, err := os.ReadFile(a.composePath())
	if err != nil {
		return err
	}
	content := string(data)
	if strings.Contains(content, "hermes-profile-runner") &&
		!strings.Contains(content, "env_file:") &&
		strings.Contains(content, "host-bridge.token") &&
		strings.Contains(content, "/usr/local/bin/hostctl") &&
		strings.Contains(content, "/etc/cont-init.d/017-patch-wecom-filenames") &&
		strings.Contains(content, "/etc/cont-init.d/018-install-feishu-deps") {
		return nil
	}
	return a.writeCompose(settings, "before-compose-runtime-helper-migration")
}

func renderCompose(settings ComposeSettings, proxy ProxySettings) string {
	settings = withComposeDefaults(settings)
	proxy = withProxyDefaults(proxy)
	dashboard := "1"
	proxyEnv := renderComposeProxyEnvironment(proxy)
	return fmt.Sprintf(`services:
  init-permissions:
    image: alpine:3.22
    user: "0:0"
    command: chown -R 10000:10000 /opt/data
    volumes:
      - ./data:/opt/data
    restart: "no"

  hermes:
    image: %s
    container_name: %s
    restart: unless-stopped
    init: false
    stop_grace_period: 120s
    command: /opt/hermes-dock/hermes-profile-runner
    depends_on:
      init-permissions:
        condition: service_completed_successfully
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "5"
    dns:
      - 223.5.5.5
      - 119.29.29.29
    extra_hosts:
      - "host.docker.internal:host-gateway"
    shm_size: %s
    ports:
      - "%s:%s:8642"
      - "%s:%s:9119"
    environment:
      HERMES_WRITE_SAFE_ROOT: "/opt/data"
      HERMES_HOME: "/opt/data"
      TMPDIR: "/opt/data/tmp"
      TZ: "Asia/Shanghai"
      HERMES_DASHBOARD: "%s"
      HERMES_DASHBOARD_BASIC_AUTH_USERNAME: "%s"
      HERMES_DASHBOARD_BASIC_AUTH_PASSWORD: "%s"
      HERMES_GATEWAY_BUSY_INPUT_MODE: "%s"
      HERMES_GATEWAY_BUSY_ACK_ENABLED: "%s"
      HERMES_BACKGROUND_NOTIFICATIONS: "%s"
%s
      UV_DEFAULT_INDEX: "https://mirrors.cloud.tencent.com/pypi/simple/"
      PIP_INDEX_URL: "https://mirrors.cloud.tencent.com/pypi/simple/"
      PIP_DEFAULT_TIMEOUT: "120"
      PIP_DISABLE_PIP_VERSION_CHECK: "1"
      NPM_CONFIG_REGISTRY: "https://registry.npmmirror.com"
    volumes:
      - ./data:/opt/data
      - ./launcher/helpers/patch-wecom-filenames:/etc/cont-init.d/017-patch-wecom-filenames:ro
      - ./launcher/helpers/install-feishu-deps:/etc/cont-init.d/018-install-feishu-deps:ro
      - ./launcher/helpers/hermes-profile-runner:/opt/hermes-dock/hermes-profile-runner:ro
      - ./launcher/helpers/hostctl:/usr/local/bin/hostctl:ro
      - ./launcher/host-bridge.token:/opt/hermes-dock/host-bridge.token:ro
    deploy:
      resources:
        limits:
          memory: %s
          cpus: "%s"
`, settings.Image, settings.ContainerName, settings.ShmSize, settings.GatewayHost, settings.GatewayPort, settings.DashboardHost, settings.DashboardPort, dashboard, yamlQuote(settings.DashboardUsername), yamlQuote(settings.DashboardPassword), yamlQuote(settings.GatewayBusyInputMode), yamlQuote(settings.GatewayBusyAckEnabled), yamlQuote(settings.BackgroundNotifications), proxyEnv, settings.MemoryLimit, settings.CPULimit)
}

func renderComposeProxyEnvironment(proxy ProxySettings) string {
	proxy = withProxyDefaults(proxy)
	if !proxy.Enabled {
		return ""
	}
	return fmt.Sprintf(`      HTTP_PROXY: "%s"
      HTTPS_PROXY: "%s"
      ALL_PROXY: "%s"
      NO_PROXY: "%s"
      http_proxy: "%s"
      https_proxy: "%s"
      all_proxy: "%s"
      no_proxy: "%s"
`, yamlQuote(proxy.HTTPProxy), yamlQuote(proxy.HTTPSProxy), yamlQuote(proxy.ALLProxy), yamlQuote(proxy.NoProxy), yamlQuote(proxy.HTTPProxy), yamlQuote(proxy.HTTPSProxy), yamlQuote(proxy.ALLProxy), yamlQuote(proxy.NoProxy))
}

func (a *App) StartHermes() error {
	if err := a.writeRuntimeManifest(); err != nil {
		return err
	}
	if err := a.ensureContainerInitHelpers(); err != nil {
		return err
	}
	if err := a.ensureProfileRunnerHelper(); err != nil {
		return err
	}
	if err := a.syncSavedModelProviderEnv(); err != nil {
		return err
	}
	return a.runComposeStreaming(context.Background(), "docker:progress", "up", "-d")
}

func (a *App) StopHermes() error {
	return a.runComposeStreaming(context.Background(), "docker:progress", "stop")
}

func (a *App) RestartHermes() error {
	return a.runComposeStreaming(context.Background(), "docker:progress", "restart")
}

func (a *App) RebuildHermes() error {
	err := a.forceRecreateComposeRuntime()
	if err == nil {
		state, _ := a.readState()
		state.LastSuccessfulHermesImage = state.HermesImage
		state.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		_ = a.writeState(state)
	}
	return err
}

func (a *App) forceRecreateComposeRuntime() error {
	if err := a.writeRuntimeManifest(); err != nil {
		return err
	}
	if err := a.ensureContainerInitHelpers(); err != nil {
		return err
	}
	if err := a.ensureProfileRunnerHelper(); err != nil {
		return err
	}
	if err := a.syncSavedModelProviderEnv(); err != nil {
		return err
	}
	return a.runComposeStreaming(context.Background(), "docker:progress", "up", "-d", "--force-recreate")
}

func (a *App) TestModel() error {
	if err := a.syncSavedModelProviderEnv(); err != nil {
		return err
	}
	args := append([]string{"run", "--rm"}, a.currentProfileComposeEnvArgs()...)
	args = append(args, "hermes")
	args = append(args, a.currentProfileHermesArgs("-z", "请只回复 OK。")...)
	return a.runComposeStreaming(context.Background(), "docker:progress", args...)
}

func (a *App) currentProfileHermesArgs(args ...string) []string {
	profileID := a.currentProfileID()
	out := []string{"hermes"}
	if profileID != defaultProfileID {
		out = append(out, "-p", profileID)
	}
	return append(out, args...)
}

func (a *App) currentProfileComposeEnvArgs() []string {
	out := []string{"-e", "HERMES_HOME=/opt/data"}
	profileID := a.currentProfileID()
	profileHome := "/opt/data"
	envFile := "data/.env"
	if profileID != defaultProfileID {
		profileHome = "/opt/data/profiles/" + profileID
		envFile = filepath.ToSlash(filepath.Join("data", "profiles", profileID, ".env"))
	}
	out = append(out, "-e", "HERMES_DOCK_PROFILE="+profileID, "-e", "HERMES_DOCK_PROFILE_HOME="+profileHome)
	return append(out, "--env-from-file", envFile)
}

func (a *App) syncSavedModelProviderEnv() error {
	cfg, err := a.readConfigMap()
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return a.syncReferencedProviderEnv(normalizeProviderConfig(readProviderConfigFromMap(cfg)))
}

func (a *App) composeAvailable(ctx context.Context) bool {
	cmd := exec.CommandContext(ctx, "docker", "compose", "version")
	cmd.Dir = a.instanceRoot
	return cmd.Run() == nil
}

func (a *App) containerStatus(ctx context.Context) string {
	cmd := exec.CommandContext(ctx, "docker", "compose", "ps", "--status", "running", "--services")
	cmd.Dir = a.instanceRoot
	out, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	if strings.Contains(string(out), "hermes") {
		return "running"
	}
	cmd = exec.CommandContext(ctx, "docker", "compose", "ps", "--services")
	cmd.Dir = a.instanceRoot
	out, err = cmd.Output()
	if err != nil {
		return "missing"
	}
	if strings.Contains(string(out), "hermes") {
		return "stopped"
	}
	return "missing"
}

func localizedContainerStatus(status string) string {
	switch status {
	case "running":
		return "运行中"
	case "stopped":
		return "已停止"
	case "missing":
		return "未创建"
	case "unknown":
		return "未知"
	default:
		return status
	}
}
