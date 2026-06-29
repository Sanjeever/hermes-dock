package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

func defaultComposeSettings() ComposeSettings {
	return ComposeSettings{
		Image:             defaultImage,
		ContainerName:     "hermes",
		GatewayHost:       "127.0.0.1",
		GatewayPort:       "8642",
		DashboardHost:     "127.0.0.1",
		DashboardPort:     "9119",
		DashboardEnabled:  true,
		DashboardUsername: "admin",
		DashboardPassword: "123456",
		MemoryLimit:       "4G",
		CPULimit:          "2.0",
		ShmSize:           "1g",
	}
}

func (a *App) readComposeSettings() ComposeSettings {
	state, _ := a.readState()
	settings := defaultComposeSettings()
	if state.ComposeSettings.Image != "" {
		settings = state.ComposeSettings
	}
	if state.HermesImage != "" {
		settings.Image = state.HermesImage
	}
	settings.DashboardEnabled = true
	env, _ := readEnvFile(a.envPath())
	if value := envValue(env, "HERMES_DASHBOARD_BASIC_AUTH_USERNAME"); value != "" {
		settings.DashboardUsername = value
	}
	if value := envValue(env, "HERMES_DASHBOARD_BASIC_AUTH_PASSWORD"); value != "" {
		settings.DashboardPassword = value
	}
	return settings
}

func (a *App) SaveComposeSettings(settings ComposeSettings) error {
	if settings.Image == "" {
		settings.Image = defaultImage
	}
	settings.DashboardEnabled = true
	if err := a.syncComposeDashboardEnv(settings); err != nil {
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
	return a.writeState(state)
}

func (a *App) syncComposeDashboardEnv(settings ComposeSettings) error {
	updates := []EnvVar{
		{Key: "HERMES_DASHBOARD", Value: "1"},
		{Key: "HERMES_DASHBOARD_BASIC_AUTH_USERNAME", Value: firstNonEmpty(settings.DashboardUsername, "admin")},
		{Key: "HERMES_DASHBOARD_BASIC_AUTH_PASSWORD", Value: firstNonEmpty(settings.DashboardPassword, "123456")},
	}
	existing, _ := readEnvFile(a.envPath())
	for _, item := range updates {
		if envValue(existing, item.Key) != item.Value {
			return a.SaveEnvironment(updates)
		}
	}
	return nil
}

func (a *App) writeCompose(settings ComposeSettings, reason string) error {
	if _, err := os.Stat(a.composePath()); err == nil {
		if err := a.backupFile(a.composePath(), reason); err != nil {
			return err
		}
	}
	content := renderCompose(settings)
	return os.WriteFile(a.composePath(), []byte(content), 0644)
}

func (a *App) migrateComposeIfNeeded(settings ComposeSettings) error {
	data, err := os.ReadFile(a.composePath())
	if err != nil {
		return err
	}
	if strings.Contains(string(data), "env_file:") {
		return nil
	}
	return a.writeCompose(settings, "before-compose-env-file-migration")
}

func renderCompose(settings ComposeSettings) string {
	if settings.ContainerName == "" {
		settings.ContainerName = "hermes"
	}
	if settings.GatewayHost == "" {
		settings.GatewayHost = "127.0.0.1"
	}
	if settings.GatewayPort == "" {
		settings.GatewayPort = "8642"
	}
	if settings.DashboardHost == "" {
		settings.DashboardHost = "127.0.0.1"
	}
	if settings.DashboardPort == "" {
		settings.DashboardPort = "9119"
	}
	if settings.MemoryLimit == "" {
		settings.MemoryLimit = "4G"
	}
	if settings.CPULimit == "" {
		settings.CPULimit = "2.0"
	}
	if settings.ShmSize == "" {
		settings.ShmSize = "1g"
	}
	if settings.DashboardUsername == "" {
		settings.DashboardUsername = "admin"
	}
	if settings.DashboardPassword == "" {
		settings.DashboardPassword = "123456"
	}
	dashboard := "1"
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
    command: gateway run
    depends_on:
      init-permissions:
        condition: service_completed_successfully
    shm_size: %s
    ports:
      - "%s:%s:8642"
      - "%s:%s:9119"
    environment:
      HERMES_WRITE_SAFE_ROOT: "/opt/data"
      TMPDIR: "/opt/data/tmp"
      HERMES_DASHBOARD: "%s"
      HERMES_DASHBOARD_BASIC_AUTH_USERNAME: "%s"
      HERMES_DASHBOARD_BASIC_AUTH_PASSWORD: "%s"
      UV_DEFAULT_INDEX: "https://mirrors.cloud.tencent.com/pypi/simple/"
      PIP_INDEX_URL: "https://mirrors.cloud.tencent.com/pypi/simple/"
      PIP_DEFAULT_TIMEOUT: "120"
      PIP_DISABLE_PIP_VERSION_CHECK: "1"
      NPM_CONFIG_REGISTRY: "https://registry.npmmirror.com"
    env_file:
      - ./data/.env
    volumes:
      - ./data:/opt/data
    deploy:
      resources:
        limits:
          memory: %s
          cpus: "%s"
`, settings.Image, settings.ContainerName, settings.ShmSize, settings.GatewayHost, settings.GatewayPort, settings.DashboardHost, settings.DashboardPort, dashboard, yamlQuote(settings.DashboardUsername), yamlQuote(settings.DashboardPassword), settings.MemoryLimit, settings.CPULimit)
}

func (a *App) StartHermes() error {
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
	if err := a.syncSavedModelProviderEnv(); err != nil {
		return err
	}
	return a.runComposeStreaming(context.Background(), "docker:progress", "up", "-d", "--force-recreate")
}

func (a *App) TestModel() error {
	if err := a.syncSavedModelProviderEnv(); err != nil {
		return err
	}
	return a.runComposeStreaming(context.Background(), "docker:progress", "run", "--rm", "hermes", "hermes", "-z", "请只回复 OK。")
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
