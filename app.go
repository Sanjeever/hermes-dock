package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	appVersion      = "1.0.0"
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
	containerStatus := a.containerStatus(context.Background())

	return AppState{
		AppVersion:       appVersion,
		InstanceRoot:     a.instanceRoot,
		State:            state,
		Compose:          compose,
		Environment:      env,
		Model:            model,
		Channels:         channels,
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

func (a *App) OpenEndpoint(endpoint string) error {
	settings := a.readComposeSettings()
	var host, port string
	switch endpoint {
	case "dashboard":
		host = settings.DashboardHost
		port = settings.DashboardPort
	case "gateway":
		host = settings.GatewayHost
		port = settings.GatewayPort
	default:
		return fmt.Errorf("未知入口：%s", endpoint)
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	runtime.BrowserOpenURL(a.ctx, "http://"+host+":"+port)
	return nil
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
