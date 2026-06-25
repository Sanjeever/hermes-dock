package main

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	appVersion      = "0.1.0"
	templateVersion = "2026.06.25"
	defaultImage    = "nousresearch/hermes-agent:v2026.6.19"
)

type App struct {
	ctx          context.Context
	instanceRoot string
	mu           sync.Mutex
	logCancel    context.CancelFunc
	loginCancel  context.CancelFunc
	startupErr   error
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.instanceRoot = detectInstanceRoot()
	a.startupErr = a.ensureInstanceReady()
}

func (a *App) GetAppState() (AppState, error) {
	if a.startupErr != nil {
		if err := a.ensureInstanceReady(); err != nil {
			return AppState{}, err
		}
		a.startupErr = nil
	} else if err := a.ensureInstanceReady(); err != nil {
		return AppState{}, err
	}
	state, _ := a.readState()
	compose := a.readComposeSettings()
	env, _ := readEnvFile(a.envPath())
	model, _ := a.readModelConfig()
	channels, _ := a.GetChannels()
	diagnostics, _ := a.RunDiagnostics()
	containerStatus := a.containerStatus(context.Background())

	return AppState{
		AppVersion:       appVersion,
		InstanceRoot:     a.instanceRoot,
		State:            state,
		Compose:          compose,
		Environment:      env,
		Model:            model,
		Channels:         channels,
		Diagnostics:      diagnostics,
		DockerAvailable:  commandExists("docker"),
		ComposeAvailable: a.composeAvailable(context.Background()),
		ContainerStatus:  containerStatus,
	}, nil
}

func (a *App) ensureInstanceReady() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if fileExists(a.statePath()) && fileExists(a.composePath()) && fileExists(a.envPath()) && fileExists(a.configPath()) {
		return nil
	}
	if err := ensureDir(a.instanceRoot); err != nil {
		return err
	}
	if err := ensureDir(a.hermesDockDir()); err != nil {
		return err
	}
	if err := a.releaseSeedData(); err != nil {
		return err
	}
	settings := a.readComposeSettings()
	if settings.Image == "" {
		settings = defaultComposeSettings()
	}
	if !fileExists(a.composePath()) {
		if err := os.WriteFile(a.composePath(), []byte(renderCompose(settings)), 0644); err != nil {
			return err
		}
	} else if err := a.migrateComposeIfNeeded(settings); err != nil {
		return err
	}
	if err := a.writeOverrideIfMissing(); err != nil {
		return err
	}
	if !fileExists(a.statePath()) {
		now := time.Now().UTC().Format(time.RFC3339)
		state := defaultState()
		state.InstanceID = uuid.NewString()
		state.ComposeHash = fileSHA256(a.composePath())
		state.InitializedAt = now
		state.UpdatedAt = now
		state.Migrations = appendIfMissingMigration(state.Migrations, MigrationRecord{ID: "seed-data-v1", AppliedAt: now})
		return a.writeState(state)
	}
	return nil
}

func (a *App) InitializeInstance(settings ComposeSettings) (LauncherState, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	return a.initializeInstanceLocked(settings)
}

func (a *App) initializeInstanceLocked(settings ComposeSettings) (LauncherState, error) {
	if settings.Image == "" {
		settings = defaultComposeSettings()
	}
	if err := ensureDir(a.instanceRoot); err != nil {
		return LauncherState{}, err
	}
	if err := ensureDir(a.hermesDockDir()); err != nil {
		return LauncherState{}, err
	}
	if err := a.releaseSeedData(); err != nil {
		return LauncherState{}, err
	}
	if err := a.writeCompose(settings, "initialize-compose"); err != nil {
		return LauncherState{}, err
	}
	if err := a.writeOverrideIfMissing(); err != nil {
		return LauncherState{}, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	existing, _ := a.readState()
	instanceID := existing.InstanceID
	if instanceID == "" {
		instanceID = uuid.NewString()
	}
	state := LauncherState{
		SchemaVersion:             1,
		AppVersion:                appVersion,
		InstanceID:                instanceID,
		ManagedCompose:            true,
		ComposeHash:               fileSHA256(a.composePath()),
		TemplateVersion:           templateVersion,
		SkillsSnapshotImage:       defaultImage,
		HermesImage:               settings.Image,
		ComposeSettings:           settings,
		PreviousHermesImage:       existing.PreviousHermesImage,
		LastSuccessfulHermesImage: settings.Image,
		InitializedAt:             firstNonEmpty(existing.InitializedAt, now),
		UpdatedAt:                 now,
		Migrations: appendIfMissingMigration(existing.Migrations, MigrationRecord{
			ID:        "seed-data-v1",
			AppliedAt: now,
		}),
		Backups:            existing.Backups,
		UI:                 UIState{LastPage: "dashboard"},
		ModelAuxiliaryMode: firstNonEmpty(existing.ModelAuxiliaryMode, "auto"),
	}
	if err := a.writeState(state); err != nil {
		return LauncherState{}, err
	}
	return state, nil
}

func (a *App) writeOverrideIfMissing() error {
	if _, err := os.Stat(a.overridePath()); errors.Is(err, os.ErrNotExist) {
		return os.WriteFile(a.overridePath(), []byte("# 在这里添加高级 Docker Compose 覆盖配置。\n"), 0644)
	}
	return nil
}

func (a *App) RunDiagnostics() ([]Diagnostic, error) {
	var items []Diagnostic
	add := func(id, label, status, message, severity string, fixable bool) {
		items = append(items, Diagnostic{ID: id, Label: label, Status: status, Message: redact(message), Severity: severity, Fixable: fixable})
	}

	if commandExists("docker") {
		add("docker-cli", "Docker CLI", "ok", "docker 命令可用", "info", false)
	} else {
		add("docker-cli", "Docker CLI", "error", "未找到 docker 命令", "error", false)
	}
	if a.composeAvailable(context.Background()) {
		add("docker-compose", "Docker Compose", "ok", "docker compose 可用", "info", false)
	} else {
		add("docker-compose", "Docker Compose", "error", "docker compose 不可用", "error", false)
	}
	if err := ensureDir(a.hermesDockDir()); err == nil {
		add("instance-root", "实例目录", "ok", a.instanceRoot, "info", false)
	} else {
		add("instance-root", "实例目录", "error", err.Error(), "error", false)
	}
	if _, err := os.Stat(a.composePath()); err == nil {
		add("compose", "docker-compose.yaml", "ok", "compose 文件已存在", "info", false)
	} else {
		add("compose", "docker-compose.yaml", "warn", "compose 文件缺失", "warning", true)
	}
	if _, err := os.Stat(a.configPath()); err == nil {
		if err := parseYAMLFile(a.configPath(), nil); err != nil {
			add("config-yaml", "config.yaml", "error", err.Error(), "error", false)
		} else {
			add("config-yaml", "config.yaml", "ok", "YAML 解析正常", "info", false)
		}
	} else {
		add("config-yaml", "config.yaml", "warn", "config.yaml 缺失", "warning", true)
	}
	env, err := readEnvFile(a.envPath())
	if err != nil {
		add("env", ".env", "warn", ".env 缺失或不可读取", "warning", true)
	} else {
		add("env", ".env", "ok", "环境变量文件可读取", "info", false)
		if envValue(env, "WEIXIN_ACCOUNT_ID") == "" || envValue(env, "WEIXIN_TOKEN") == "" {
			add("weixin", "个人微信", "warn", "个人微信尚未绑定", "warning", false)
		} else {
			add("weixin", "个人微信", "ok", "个人微信凭据已配置", "info", false)
		}
		if envValue(env, "WECOM_BOT_ID") == "" || envValue(env, "WECOM_SECRET") == "" {
			add("wecom", "企业微信 AI Bot", "warn", "企业微信 AI Bot 凭据不完整", "warning", false)
		} else {
			add("wecom", "企业微信 AI Bot", "ok", "企业微信 AI Bot 凭据已配置", "info", false)
		}
		if envValue(env, "HERMES_DASHBOARD_BASIC_AUTH_PASSWORD") == "123456" {
			add("dashboard-password", "控制台密码", "warn", "控制台密码仍为 123456", "warning", false)
		}
	}
	for _, port := range []string{"8642", "9119"} {
		if portAvailable(port) {
			add("port-"+port, "端口 "+port, "ok", "端口可用", "info", false)
		} else {
			add("port-"+port, "端口 "+port, "warn", "端口可能已被占用", "warning", false)
		}
	}
	status := a.containerStatus(context.Background())
	if status == "running" {
		add("container", "Hermes 容器", "ok", "容器正在运行", "info", false)
	} else {
		add("container", "Hermes 容器", "warn", "容器状态："+localizedContainerStatus(status), "warning", false)
	}
	return items, nil
}

func (a *App) ReadTextFile(path string) (string, error) {
	resolved, err := a.safePath(path)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(resolved)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (a *App) SaveTextFile(req TextFileRequest) error {
	resolved, err := a.safePath(req.Path)
	if err != nil {
		return err
	}
	if strings.HasSuffix(resolved, ".yaml") || strings.HasSuffix(resolved, ".yml") {
		if err := parseYAML([]byte(req.Content), nil); err != nil {
			return err
		}
	}
	if err := a.backupFile(resolved, firstNonEmpty(req.Reason, "before-text-save")); err != nil {
		return err
	}
	return os.WriteFile(resolved, []byte(req.Content), 0644)
}

func (a *App) emit(event string, payload interface{}) {
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, event, payload)
	}
}
