package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	containerNamePattern  = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]*$`)
	imageReferencePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._/:@-]*$`)
	memoryLimitPattern    = regexp.MustCompile(`(?i)^[1-9][0-9]*(?:\.[0-9]+)?[bkmg]?(?:b)?$`)
)

const (
	fallbackMemoryLimit = "4G"
	fallbackCPULimit    = "2.0"
	runtimeStatusWait   = 60 * time.Second
)

func defaultComposeSettings() ComposeSettings {
	return ComposeSettings{
		Image:                    defaultImage,
		ContainerName:            "hermes",
		GatewayBusyInputMode:     "steer",
		GatewayBusyAckEnabled:    "false",
		BackgroundNotifications:  "result",
		HostControlEnabled:       "true",
		MemoryLimit:              fallbackMemoryLimit,
		CPULimit:                 fallbackCPULimit,
		ShmSize:                  "1g",
		SharedDirectory:          filepath.Join(detectInstanceRoot(), "shared"),
		DufsEnabled:              true,
		DufsPort:                 defaultDufsPort,
		DufsUsername:             defaultDufsUsername,
		DufsUsingDefaultPassword: true,
	}
}

func withComposeDefaults(settings ComposeSettings) ComposeSettings {
	defaults := defaultComposeSettings()
	settings.Image = defaultImage
	if settings.ContainerName == "" {
		settings.ContainerName = defaults.ContainerName
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
	if settings.SharedDirectory == "" {
		settings.SharedDirectory = defaults.SharedDirectory
	}
	settings.DufsPort = strings.TrimSpace(settings.DufsPort)
	settings.DufsUsername = strings.TrimSpace(settings.DufsUsername)
	if settings.DufsPort == "" {
		settings.DufsEnabled = true
		settings.DufsPort = defaults.DufsPort
		settings.DufsUsername = defaults.DufsUsername
		settings.DufsUsingDefaultPassword = true
	} else if settings.DufsUsername == "" {
		settings.DufsUsername = defaults.DufsUsername
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

func validateSharedDirectory(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("共享文件目录不能为空")
	}
	if !filepath.IsAbs(path) {
		return "", fmt.Errorf("共享文件目录必须使用宿主机绝对路径")
	}
	return filepath.Clean(path), nil
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
	state.NeedsRebuild = true
	state.PendingDufsOnly = false
	state.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	return a.writeState(state)
}

func (a *App) SaveComposeSettings(settings ComposeSettings) error {
	previousSettings := a.readComposeSettings()
	previousState, err := a.readState()
	if err != nil {
		return err
	}
	password := settings.DufsPassword
	settings.DufsPassword = ""
	settings = withComposeDefaults(settings)
	if err := validateComposeSettings(settings); err != nil {
		return err
	}
	if err := a.validateDufsSettings(settings); err != nil {
		return err
	}
	sharedDirectory, err := validateSharedDirectory(settings.SharedDirectory)
	if err != nil {
		return err
	}
	if err := ensureWritableDirectory(sharedDirectory); err != nil {
		return err
	}
	settings.SharedDirectory = sharedDirectory
	settings, err = a.ensureDufsConfig(settings, password, "before-dufs-config-save")
	if err != nil {
		return err
	}
	if err := a.syncComposeEnv(settings); err != nil {
		return err
	}
	if err := a.writeCompose(settings, "before-compose-save"); err != nil {
		return err
	}
	state := previousState
	hermesUnchanged := renderHermesService(previousSettings, a.readProxySettings()) == renderHermesService(settings, a.readProxySettings())
	dufsOnly := hermesUnchanged && (!previousState.NeedsRebuild || previousState.PendingDufsOnly)
	state.PreviousHermesImage = state.HermesImage
	state.HermesImage = settings.Image
	state.ComposeSettings = settings
	state.ComposeHash = fileSHA256(a.composePath())
	state.NeedsRebuild = true
	state.PendingDufsOnly = dufsOnly
	state.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if err := a.writeState(state); err != nil {
		return err
	}
	return a.syncHostBridge(settings.HostControlEnabled == "true")
}

func validateComposeSettings(settings ComposeSettings) error {
	if !imageReferencePattern.MatchString(strings.TrimSpace(settings.Image)) {
		return fmt.Errorf("Hermes 镜像格式无效")
	}
	if !containerNamePattern.MatchString(settings.ContainerName) {
		return fmt.Errorf("容器名称格式无效")
	}
	if !memoryLimitPattern.MatchString(strings.TrimSpace(settings.MemoryLimit)) {
		return fmt.Errorf("内存限制格式无效")
	}
	if !memoryLimitPattern.MatchString(strings.TrimSpace(settings.ShmSize)) {
		return fmt.Errorf("共享内存格式无效")
	}
	cpu, err := strconv.ParseFloat(strings.TrimSpace(settings.CPULimit), 64)
	if err != nil || cpu <= 0 {
		return fmt.Errorf("CPU 限制必须是大于 0 的数字")
	}
	return nil
}

const (
	composeRuntimeMigrationID = "compose-runtime-v4"
	dufsComposeMigrationID    = "compose-dufs-v1"
	fixedImageMigrationID     = "compose-fixed-image-v1"
	privateHermesMigrationID  = "compose-private-hermes-services-v1"
)

func (a *App) syncComposeEnv(settings ComposeSettings) error {
	settings = withComposeDefaults(settings)
	updates := []EnvVar{
		{Key: "HERMES_DASHBOARD", Value: "0"},
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

	cmd := backgroundCommandContext(ctx, "docker", "info", "--format", "{{json .}}")
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
	settings = withComposeDefaults(settings)
	sharedDirectory, err := validateSharedDirectory(settings.SharedDirectory)
	if err != nil {
		return err
	}
	if err := ensureWritableDirectory(sharedDirectory); err != nil {
		return err
	}
	settings.SharedDirectory = sharedDirectory
	if _, err := os.Stat(a.composePath()); err == nil {
		if err := a.backupFile(a.composePath(), reason); err != nil {
			return err
		}
	}
	content := renderCompose(settings, a.readProxySettings())
	return atomicWriteFile(a.composePath(), []byte(content), 0644)
}

func (a *App) migrateComposeIfNeeded(settings ComposeSettings) error {
	if fileExists(a.statePath()) {
		state, err := a.readState()
		if err != nil {
			return err
		}
		if migrationApplied(state.Migrations, composeRuntimeMigrationID) {
			return nil
		}
	}
	data, err := os.ReadFile(a.composePath())
	if err != nil {
		return err
	}
	content := string(data)
	current := strings.Contains(content, "hermes-profile-runner") &&
		!strings.Contains(content, "env_file:") &&
		!strings.Contains(content, "init-permissions") &&
		strings.Contains(content, `HERMES_WRITE_SAFE_ROOT: "/opt/data"`) &&
		strings.Contains(content, ":/opt/data/.dock/shared") &&
		strings.Contains(content, "host-bridge.token") &&
		strings.Contains(content, "/usr/local/bin/hostctl") &&
		strings.Contains(content, "/etc/cont-init.d/017-patch-wecom-filenames") &&
		strings.Contains(content, "/etc/cont-init.d/018-install-feishu-deps") &&
		strings.Contains(content, "/etc/cont-init.d/019-patch-home-channel-prompt") &&
		strings.Contains(content, "/etc/cont-init.d/020-install-dingtalk-deps") &&
		!strings.Contains(content, "/etc/cont-init.d/021-install-paddleocr-deps") &&
		strings.Contains(content, "HERMES_DOCK_SUPPRESS_HOME_CHANNEL_PROMPT")
	if !current {
		if err := a.writeCompose(settings, "before-compose-runtime-v4-migration"); err != nil {
			return err
		}
	}
	if !fileExists(a.statePath()) {
		return nil
	}
	state, err := a.readState()
	if err != nil {
		return err
	}
	state.Migrations = appendIfMissingMigration(state.Migrations, MigrationRecord{
		ID:        composeRuntimeMigrationID,
		AppliedAt: time.Now().UTC().Format(time.RFC3339),
	})
	state.ComposeHash = fileSHA256(a.composePath())
	state.NeedsRebuild = state.NeedsRebuild || !current
	if !current {
		state.PendingDufsOnly = false
	}
	state.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	return a.writeState(state)
}

func (a *App) migrateRuntimeDependencyComposeIfNeeded(settings ComposeSettings) error {
	data, err := os.ReadFile(a.composePath())
	if err != nil {
		return err
	}
	expectedMount := "./launcher/runtime-deps/" + runtimeDependencyBundleVersion + ":/opt/hermes-dock/runtime-deps:ro"
	current := strings.Contains(string(data), expectedMount) &&
		strings.Contains(string(data), "/etc/cont-init.d/016-verify-runtime-deps")
	if !current {
		if err := a.writeCompose(settings, "before-runtime-deps-compose-migration"); err != nil {
			return err
		}
	}
	if !fileExists(a.statePath()) {
		return nil
	}
	state, err := a.readState()
	if err != nil {
		return err
	}
	if current && state.RuntimeDependencyVersion == runtimeDependencyBundleVersion {
		return nil
	}
	composeHash := fileSHA256(a.composePath())
	if state.ComposeHash == composeHash && state.NeedsRebuild && !state.PendingDufsOnly {
		return nil
	}
	state.ComposeHash = composeHash
	state.NeedsRebuild = true
	state.PendingDufsOnly = false
	state.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	return a.writeState(state)
}

func (a *App) migrateFixedHermesImageIfNeeded(settings ComposeSettings) error {
	if !fileExists(a.statePath()) {
		return nil
	}
	state, err := a.readState()
	if err != nil {
		return err
	}
	if migrationApplied(state.Migrations, fixedImageMigrationID) {
		return nil
	}
	data, err := os.ReadFile(a.composePath())
	if err != nil {
		return err
	}
	image, hasImage, err := composeServiceImage(data, "hermes")
	if err != nil {
		return fmt.Errorf("读取标准 Docker Compose 失败：%w", err)
	}
	current := hasImage && image == defaultImage
	if !current {
		if err := a.writeCompose(settings, "before-compose-fixed-image-migration"); err != nil {
			return err
		}
	}
	imageChanged := state.HermesImage != defaultImage || state.ComposeSettings.Image != defaultImage
	if imageChanged {
		state.PreviousHermesImage = state.HermesImage
	}
	state.HermesImage = defaultImage
	state.ComposeSettings = settings
	state.Migrations = appendIfMissingMigration(state.Migrations, MigrationRecord{
		ID:        fixedImageMigrationID,
		AppliedAt: time.Now().UTC().Format(time.RFC3339),
	})
	state.ComposeHash = fileSHA256(a.composePath())
	state.NeedsRebuild = state.NeedsRebuild || !current || imageChanged
	if !current || imageChanged {
		state.PendingDufsOnly = false
	}
	state.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	return a.writeState(state)
}

func (a *App) migratePrivateHermesServicesIfNeeded(settings ComposeSettings) error {
	if !fileExists(a.statePath()) {
		return nil
	}
	state, err := a.readState()
	if err != nil {
		return err
	}
	migrationRecorded := migrationApplied(state.Migrations, privateHermesMigrationID)
	data, err := os.ReadFile(a.composePath())
	if err != nil {
		return err
	}
	current := privateHermesServicesCurrent(data, settings.DufsEnabled)
	if migrationRecorded && current {
		return nil
	}
	if !current {
		if err := a.writeCompose(settings, "before-private-hermes-services-migration"); err != nil {
			return err
		}
	}
	state.ComposeSettings = settings
	state.ComposeSettings.DufsPassword = ""
	state.Migrations = appendIfMissingMigration(state.Migrations, MigrationRecord{
		ID:        privateHermesMigrationID,
		AppliedAt: time.Now().UTC().Format(time.RFC3339),
	})
	state.ComposeHash = fileSHA256(a.composePath())
	state.NeedsRebuild = state.NeedsRebuild || !current
	if !current {
		state.PendingDufsOnly = false
	}
	state.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	return a.writeState(state)
}

func privateHermesServicesCurrent(data []byte, dufsEnabled bool) bool {
	var compose map[string]interface{}
	if err := parseYAML(data, &compose); err != nil {
		return false
	}
	services := asMap(compose["services"])
	hermes := asMap(services["hermes"])
	if len(hermes) == 0 {
		return false
	}
	if _, exists := hermes["ports"]; exists {
		return false
	}
	if _, exists := hermes["expose"]; exists {
		return false
	}
	if asString(asMap(hermes["environment"])["HERMES_DASHBOARD"]) != "0" {
		return false
	}
	if !sameStrings(composeServiceNetworks(hermes["networks"]), []string{"hermes_runtime"}) {
		return false
	}
	networks := asMap(compose["networks"])
	if _, exists := networks["hermes_runtime"]; !exists {
		return false
	}
	if _, exists := networks["file_management"]; !exists {
		return false
	}
	if !dufsEnabled {
		return true
	}
	dufs := asMap(services["dufs"])
	return len(dufs) > 0 && sameStrings(composeServiceNetworks(dufs["networks"]), []string{"file_management"})
}

func composeServiceNetworks(value interface{}) []string {
	items, ok := value.([]interface{})
	if !ok {
		return nil
	}
	networks := make([]string, 0, len(items))
	for _, item := range items {
		name := asString(item)
		if name == "" {
			return nil
		}
		networks = append(networks, name)
	}
	return networks
}

func sameStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for index := range got {
		if got[index] != want[index] {
			return false
		}
	}
	return true
}

func (a *App) migrateDufsComposeIfNeeded(settings ComposeSettings) error {
	if !fileExists(a.statePath()) {
		return nil
	}
	state, err := a.readState()
	if err != nil {
		return err
	}
	if migrationApplied(state.Migrations, dufsComposeMigrationID) {
		if state.ComposeSettings.DufsEnabled == settings.DufsEnabled &&
			state.ComposeSettings.DufsPort == settings.DufsPort &&
			state.ComposeSettings.DufsUsername == settings.DufsUsername &&
			state.ComposeSettings.DufsUsingDefaultPassword == settings.DufsUsingDefaultPassword {
			return nil
		}
		state.ComposeSettings.DufsEnabled = settings.DufsEnabled
		state.ComposeSettings.DufsPort = settings.DufsPort
		state.ComposeSettings.DufsUsername = settings.DufsUsername
		state.ComposeSettings.DufsUsingDefaultPassword = settings.DufsUsingDefaultPassword
		state.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		return a.writeState(state)
	}
	data, err := os.ReadFile(a.composePath())
	if err != nil {
		return err
	}
	content := string(data)
	current := !settings.DufsEnabled || (strings.Contains(content, defaultDufsImage) &&
		strings.Contains(content, "./launcher/dufs/config.yaml:/etc/dufs.yaml:ro") &&
		strings.Contains(content, ":/data\"") &&
		strings.Contains(content, `0.0.0.0:`+settings.DufsPort+`:`))
	if !current {
		if err := a.writeCompose(settings, "before-dufs-compose-migration"); err != nil {
			return err
		}
	}
	wasPending := state.NeedsRebuild
	state.ComposeSettings = settings
	state.ComposeSettings.DufsPassword = ""
	state.Migrations = appendIfMissingMigration(state.Migrations, MigrationRecord{
		ID:        dufsComposeMigrationID,
		AppliedAt: time.Now().UTC().Format(time.RFC3339),
	})
	state.ComposeHash = fileSHA256(a.composePath())
	state.NeedsRebuild = state.NeedsRebuild || !current
	if !current && (!wasPending || state.PendingDufsOnly) {
		state.PendingDufsOnly = true
	}
	state.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	return a.writeState(state)
}

func renderCompose(settings ComposeSettings, proxy ProxySettings) string {
	return "services:\n" + renderHermesService(settings, proxy) + renderDufsService(settings) + `networks:
  hermes_runtime:
  file_management:
`
}

func renderHermesService(settings ComposeSettings, proxy ProxySettings) string {
	settings = withComposeDefaults(settings)
	proxy = withProxyDefaults(proxy)
	proxyEnv := renderComposeProxyEnvironment(proxy)
	return fmt.Sprintf(`  hermes:
    image: "%s"
    container_name: "%s"
    restart: unless-stopped
    init: false
    stop_grace_period: 120s
    command: /opt/hermes-dock/hermes-profile-runner
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
    shm_size: "%s"
    environment:
      HERMES_WRITE_SAFE_ROOT: "/opt/data"
      HERMES_HOME: "/opt/data"
      TMPDIR: "/opt/data/tmp"
      TZ: "Asia/Shanghai"
      HERMES_DASHBOARD: "0"
      HERMES_GATEWAY_BUSY_INPUT_MODE: "%s"
      HERMES_GATEWAY_BUSY_ACK_ENABLED: "%s"
      HERMES_BACKGROUND_NOTIFICATIONS: "%s"
      HERMES_DOCK_SUPPRESS_HOME_CHANNEL_PROMPT: "true"
%s
      UV_DEFAULT_INDEX: "https://mirrors.cloud.tencent.com/pypi/simple/"
      PIP_INDEX_URL: "https://mirrors.cloud.tencent.com/pypi/simple/"
      PIP_DEFAULT_TIMEOUT: "120"
      PIP_DISABLE_PIP_VERSION_CHECK: "1"
      NPM_CONFIG_REGISTRY: "https://registry.npmmirror.com"
    volumes:
      - ./data:/opt/data
      - ./launcher/runtime-deps/%s:/opt/hermes-dock/runtime-deps:ro
      - "%s:/opt/data/.dock/shared"
      - ./launcher/helpers/verify-runtime-deps:/etc/cont-init.d/016-verify-runtime-deps:ro
      - ./launcher/helpers/patch-wecom-filenames:/etc/cont-init.d/017-patch-wecom-filenames:ro
      - ./launcher/helpers/install-feishu-deps:/etc/cont-init.d/018-install-feishu-deps:ro
      - ./launcher/helpers/patch-home-channel-prompt:/etc/cont-init.d/019-patch-home-channel-prompt:ro
      - ./launcher/helpers/install-dingtalk-deps:/etc/cont-init.d/020-install-dingtalk-deps:ro
      - ./launcher/helpers/hermes-profile-runner:/opt/hermes-dock/hermes-profile-runner:ro
      - ./launcher/helpers/hostctl:/usr/local/bin/hostctl:ro
      - ./launcher/host-bridge.token:/opt/hermes-dock/host-bridge.token:ro
    networks:
      - hermes_runtime
    deploy:
      resources:
        limits:
          memory: "%s"
          cpus: "%s"
`, yamlQuote(settings.Image), yamlQuote(settings.ContainerName), yamlQuote(settings.ShmSize), yamlQuote(settings.GatewayBusyInputMode), yamlQuote(settings.GatewayBusyAckEnabled), yamlQuote(settings.BackgroundNotifications), proxyEnv, runtimeDependencyBundleVersion, yamlQuote(settings.SharedDirectory), yamlQuote(settings.MemoryLimit), yamlQuote(settings.CPULimit))
}

func renderDufsService(settings ComposeSettings) string {
	settings = withComposeDefaults(settings)
	if !settings.DufsEnabled {
		return ""
	}
	return fmt.Sprintf(`  dufs:
    image: %s
    container_name: hermes-dufs
    restart: unless-stopped
    user: "%s"
    read_only: true
    security_opt:
      - no-new-privileges:true
    cap_drop:
      - ALL
    command: ["--config", "/etc/dufs.yaml"]
    ports:
      - "0.0.0.0:%s:%d"
    volumes:
      - "%s:/data"
      - ./launcher/dufs/config.yaml:/etc/dufs.yaml:ro
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
    networks:
      - file_management
`, defaultDufsImage, dufsContainerUser(), settings.DufsPort, dufsContainerPort, yamlQuote(settings.SharedDirectory))
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
	release, err := a.beginExclusiveOperation("启动 Hermes")
	if err != nil {
		return err
	}
	defer release()
	if err := ensureWritableDirectory(a.readComposeSettings().SharedDirectory); err != nil {
		return err
	}
	if err := a.ensureRuntimeDependencies(); err != nil {
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
	manifest, err := a.writeRuntimeManifest()
	if err != nil {
		return err
	}
	composeHash, err := a.composeRuntimeHash()
	if err != nil {
		return err
	}
	if err := a.runComposeStreaming(context.Background(), "docker:progress", "up", "-d"); err != nil {
		return err
	}
	if err := a.waitForRuntimeStatus(manifest, runtimeStatusWait); err != nil {
		return fmt.Errorf("容器已启动，但%w", err)
	}
	return a.markRebuildApplied(composeHash)
}

func (a *App) StopHermes() error {
	release, err := a.beginExclusiveOperation("停止 Hermes")
	if err != nil {
		return err
	}
	defer release()
	return a.stopHermesRuntime()
}

func (a *App) stopHermesRuntime() error {
	return a.runComposeStreaming(context.Background(), "docker:progress", "stop")
}

func (a *App) RestartHermes() error {
	release, err := a.beginExclusiveOperation("重启 Hermes")
	if err != nil {
		return err
	}
	defer release()
	return a.runComposeStreaming(context.Background(), "docker:progress", "restart", "hermes")
}

func (a *App) RebuildHermes() error {
	return a.startApplyConfigTask()
}

func (a *App) markRebuildApplied(composeHash string) error {
	dufsHash, err := a.dufsRuntimeHash()
	if err != nil {
		return err
	}
	if err := a.cleanupObsoleteRuntimeDependencies(); err != nil {
		return fmt.Errorf("清理旧运行依赖失败：%w", err)
	}
	a.stateMu.Lock()
	defer a.stateMu.Unlock()
	state, err := a.readState()
	if err != nil {
		return err
	}
	state.LastAppliedComposeHash = composeHash
	state.LastAppliedDufsHash = dufsHash
	state.RuntimeDependencyVersion = runtimeDependencyBundleVersion
	state.LastSuccessfulHermesImage = state.HermesImage
	state.NeedsRebuild = false
	state.PendingDufsOnly = false
	state.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	return a.writeStateUnlocked(state)
}

func (a *App) finalizeRebuildAppliedSnapshot(composeHash string, dufsHash string, inputHash string, hermesImage string, hermesApplied bool) (bool, error) {
	a.stateMu.Lock()
	defer a.stateMu.Unlock()
	currentComposeHash, err := a.composeRuntimeHash()
	if err != nil {
		return false, err
	}
	currentDufsHash, err := a.dufsRuntimeHash()
	if err != nil {
		return false, err
	}
	unchanged := currentComposeHash == composeHash && currentDufsHash == dufsHash
	if unchanged && inputHash != "" {
		currentInputHash, err := a.applyRuntimeInputHash(currentComposeHash, currentDufsHash)
		if err != nil {
			return false, err
		}
		unchanged = currentInputHash == inputHash
	}
	if hermesApplied {
		if err := a.cleanupObsoleteRuntimeDependencies(); err != nil {
			return false, fmt.Errorf("清理旧运行依赖失败：%w", err)
		}
	}
	state, err := a.readState()
	if err != nil {
		return false, err
	}
	state.LastAppliedDufsHash = dufsHash
	if hermesApplied {
		state.LastAppliedComposeHash = composeHash
		state.RuntimeDependencyVersion = runtimeDependencyBundleVersion
		if state.HermesImage == hermesImage {
			state.LastSuccessfulHermesImage = hermesImage
		}
	}
	if unchanged {
		if hermesApplied {
			state.NeedsRebuild = false
			state.PendingDufsOnly = false
		} else if state.PendingDufsOnly {
			state.PendingDufsOnly = false
			if state.RuntimeDependencyVersion == runtimeDependencyBundleVersion {
				state.NeedsRebuild = false
			}
		}
	}
	state.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if err := a.writeStateUnlocked(state); err != nil {
		return false, err
	}
	return unchanged, nil
}

func (a *App) composeRuntimeHash() (string, error) {
	hash := sha256.New()
	hermesService := []byte(renderHermesService(a.readComposeSettings(), a.readProxySettings()))
	_, _ = fmt.Fprintf(hash, "%d:", len(hermesService))
	_, _ = hash.Write(hermesService)
	for _, path := range []string{a.overridePath()} {
		data, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("读取 Compose 配置失败：%w", err)
		}
		_, _ = fmt.Fprintf(hash, "%d:", len(data))
		_, _ = hash.Write(data)
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func shouldRecreateComposeRuntime(lastAppliedHash string, currentHash string, containerStatus string) bool {
	return lastAppliedHash == "" || lastAppliedHash != currentHash || (containerStatus != "running" && containerStatus != "stopped")
}

func (a *App) applyComposeRuntime(recreate bool) error {
	return a.applyComposeRuntimeContext(context.Background(), recreate)
}

func (a *App) applyComposeRuntimeContext(ctx context.Context, recreate bool) error {
	if recreate {
		a.emit("docker:progress", StreamEvent{Line: "检测到容器配置变化，正在重建 Hermes 容器"})
		return a.runComposeStreaming(ctx, "docker:progress", "up", "-d", "--force-recreate", "--remove-orphans", "hermes")
	}
	a.emit("docker:progress", StreamEvent{Line: "容器配置未变化，正在快速重启 Hermes 服务"})
	return a.runComposeStreaming(ctx, "docker:progress", "restart", "hermes")
}

func (a *App) applyDufsRuntime(settings ComposeSettings, recreate bool) error {
	return a.applyDufsRuntimeContext(context.Background(), settings, recreate)
}

func (a *App) applyDufsRuntimeContext(ctx context.Context, settings ComposeSettings, recreate bool) error {
	if !settings.DufsEnabled {
		a.emit("docker:progress", StreamEvent{Line: "正在关闭 Dufs 文件管理并移除旧容器"})
		return a.runComposeStreaming(ctx, "docker:progress", "up", "-d", "--remove-orphans", "hermes")
	}
	if recreate {
		a.emit("docker:progress", StreamEvent{Line: "检测到文件管理配置变化，正在重建 Dufs 容器"})
		return a.runComposeStreaming(ctx, "docker:progress", "up", "-d", "--force-recreate", "dufs")
	}
	return a.runComposeStreaming(ctx, "docker:progress", "up", "-d", "dufs")
}

func (a *App) waitForRuntimeStatus(manifest RuntimeManifest, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		data, err := os.ReadFile(a.runtimeStatusPath())
		if err == nil {
			var status RuntimeStatus
			if err := json.Unmarshal(data, &status); err != nil {
				return fmt.Errorf("助手运行状态无效：%w", err)
			}
			ready, err := runtimeStatusReady(manifest, status)
			if err != nil {
				return err
			}
			if ready {
				return nil
			}
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("无法读取助手运行状态：%w", err)
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("助手未在 %d 秒内上报运行状态，请查看运行日志", int(timeout/time.Second))
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func runtimeStatusReady(manifest RuntimeManifest, status RuntimeStatus) (bool, error) {
	if manifest.Generation == "" || status.Generation != manifest.Generation {
		return false, nil
	}
	for _, profile := range manifest.Profiles {
		if !profile.Runnable {
			continue
		}
		profileStatus, ok := status.Profiles[profile.ID]
		if !ok {
			return false, nil
		}
		if profileStatus.State == "failed" {
			message := strings.TrimSpace(redact(profileStatus.Message))
			if message == "" {
				message = "启动失败"
			}
			return false, fmt.Errorf("助手 %s %s", firstNonEmpty(profile.Name, profile.ID), message)
		}
		if profileStatus.State != "running" {
			return false, nil
		}
	}
	return true, nil
}

func (a *App) TestModel() error {
	return a.TestModelForProfile(a.currentProfileID())
}

func (a *App) TestModelForProfile(profileID string) error {
	profileID, err := a.resolveProfileID(profileID)
	if err != nil {
		return err
	}
	if err := a.syncSavedModelProviderEnvForProfile(profileID); err != nil {
		return err
	}
	if err := a.ensureRuntimeDependencies(); err != nil {
		return err
	}
	args := append([]string{"run", "--rm"}, a.profileComposeEnvArgs(profileID)...)
	args = append(args, "hermes")
	args = append(args, a.profileHermesArgs(profileID, "-z", "请只回复 OK。")...)
	return a.runComposeStreaming(context.Background(), "docker:progress", args...)
}

func (a *App) currentProfileHermesArgs(args ...string) []string {
	return a.profileHermesArgs(a.currentProfileID(), args...)
}

func (a *App) profileHermesArgs(profileID string, args ...string) []string {
	out := []string{"hermes"}
	if profileID != defaultProfileID {
		out = append(out, "-p", profileID)
	}
	return append(out, args...)
}

func (a *App) currentProfileComposeEnvArgs() []string {
	return a.profileComposeEnvArgs(a.currentProfileID())
}

func (a *App) profileComposeEnvArgs(profileID string) []string {
	out := []string{"-e", "HERMES_HOME=/opt/data"}
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
	return a.syncSavedModelProviderEnvForProfile(a.currentProfileID())
}

func (a *App) syncSavedModelProviderEnvForProfile(profileID string) error {
	cfg, err := a.readConfigMapForProfile(profileID)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return a.syncReferencedProviderEnvForProfile(profileID, normalizeProviderConfig(readProviderConfigFromMap(cfg)))
}

func (a *App) composeAvailable(ctx context.Context) bool {
	cmd := backgroundCommandContext(ctx, "docker", "compose", "version")
	cmd.Dir = a.instanceRoot
	return cmd.Run() == nil
}

func (a *App) containerStatus(ctx context.Context) string {
	cmd := backgroundCommandContext(ctx, "docker", "compose", "ps", "--status", "running", "--services")
	cmd.Dir = a.instanceRoot
	out, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	if strings.Contains(string(out), "hermes") {
		return "running"
	}
	cmd = backgroundCommandContext(ctx, "docker", "compose", "ps", "--services")
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
