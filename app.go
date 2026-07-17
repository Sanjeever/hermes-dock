package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	appVersion      = "1.9.12"
	templateVersion = "2026.07.16"
	defaultImage    = "nousresearch/hermes-agent:v2026.6.19"
)

type App struct {
	ctx                 context.Context
	instanceRoot        string
	mu                  sync.Mutex
	webSessionMu        sync.RWMutex
	loginMu             sync.Mutex
	logCancel           context.CancelFunc
	loginCancel         context.CancelFunc
	loginPlatform       string
	startupErr          error
	web                 *webRuntime
	hostBridge          *hostBridgeRuntime
	hostBridgeMu        sync.RWMutex
	hostBridgeAddr      string
	hostRPAMu           sync.Mutex
	hostRPAOwner        string
	hostRPAExpiresAt    time.Time
	notificationMu      sync.Mutex
	notificationReady   bool
	updateMu            sync.Mutex
	updateWatcherCancel context.CancelFunc
}

func NewApp() *App {
	return &App{hostBridgeAddr: hostBridgeAddress}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.instanceRoot = detectInstanceRoot()
	a.startUpdateWatcher()
	a.startupErr = a.ensureInstanceReady()
	if a.startupErr == nil {
		a.acknowledgeInstalledUpdate()
		a.startHostBridge()
		a.startWebServer()
		a.startTray()
	}
}

func (a *App) shutdown(ctx context.Context) {
	a.stopUpdateWatcher()
	a.stopTray()
	a.stopHostBridge(ctx)
	a.cleanupHostNotifications()
	a.stopWebServer(ctx)
	a.cancelLoginSession("")
	a.StopTailLogs()
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
	profiles, _ := a.readProfileRegistry()
	compose := a.readComposeSettings()
	proxy := a.readProxySettings()
	env, _ := readEnvFile(a.envPath())
	model, _ := a.readModelConfig()
	providers, _ := a.readProviderConfig()
	channels, _ := a.GetChannels()
	containerStatus := a.containerStatus(context.Background())

	return AppState{
		AppVersion:       appVersion,
		InstanceRoot:     a.instanceRoot,
		NeedsRebuild:     state.NeedsRebuild,
		State:            state,
		Profiles:         profiles,
		ActiveProfile:    a.currentProfileID(),
		ProfileStatus:    a.readRuntimeStatus(containerStatus),
		Compose:          compose,
		Proxy:            proxy,
		Environment:      env,
		Model:            model,
		Providers:        providers,
		Channels:         channels,
		DockerAvailable:  commandExists("docker"),
		ComposeAvailable: a.composeAvailable(context.Background()),
		ContainerStatus:  containerStatus,
		Web:              a.webStatus(),
		Dufs:             a.dufsStatus(),
		HostBridge:       a.hostBridgeStatus(),
		Update:           a.updateStatus(),
	}, nil
}

func (a *App) ensureInstanceReady() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.ensureInstanceReadyLocked()
}

func (a *App) ensureInstanceReadyLocked() error {
	if fileExists(a.statePath()) && fileExists(a.composePath()) && fileExists(a.defaultEnvPath()) && fileExists(a.defaultConfigPath()) {
		settings := a.readComposeSettings()
		var err error
		settings, err = a.ensureDufsConfig(settings, "", "before-dufs-config-repair")
		if err != nil {
			return err
		}
		if err := a.ensureContainerInitHelpers(); err != nil {
			return err
		}
		if err := a.migrateComposeIfNeeded(settings); err != nil {
			return err
		}
		if err := a.migrateDufsComposeIfNeeded(settings); err != nil {
			return err
		}
		if err := a.ensureComposeResourceDefaults(); err != nil {
			return err
		}
		if err := a.writeOverrideIfMissing(); err != nil {
			return err
		}
		if err := a.ensureDockDataDir(); err != nil {
			return err
		}
		if err := a.ensureProfileRegistry(); err != nil {
			return err
		}
		if err := a.migrateProfileStreamingDefaults(); err != nil {
			return err
		}
		return a.ensureWebConfig()
	}
	if err := ensureDir(a.instanceRoot); err != nil {
		return err
	}
	if err := ensureDir(a.hermesDockDir()); err != nil {
		return err
	}
	if err := a.ensureContainerInitHelpers(); err != nil {
		return err
	}
	if err := a.releaseSeedData(); err != nil {
		return err
	}
	if err := a.ensureDockDataDir(); err != nil {
		return err
	}
	settings := a.readComposeSettings()
	if !fileExists(a.statePath()) && !fileExists(a.composePath()) {
		settings = a.withInitialResourceDefaults(settings)
	} else if settings.Image == "" {
		settings = defaultComposeSettings()
	}
	var err error
	settings, err = a.ensureDufsConfig(settings, "", "before-dufs-config-initialize")
	if err != nil {
		return err
	}
	if !fileExists(a.composePath()) {
		if err := a.writeCompose(settings, "initialize-compose"); err != nil {
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
		state.HermesImage = settings.Image
		state.ComposeSettings = settings
		state.ComposeHash = fileSHA256(a.composePath())
		state.InitializedAt = now
		state.UpdatedAt = now
		state.Migrations = appendIfMissingMigration(state.Migrations, MigrationRecord{ID: "seed-data-v1", AppliedAt: now})
		if err := a.writeState(state); err != nil {
			return err
		}
		if err := a.migrateDufsComposeIfNeeded(settings); err != nil {
			return err
		}
		if err := a.ensureProfileRegistry(); err != nil {
			return err
		}
		if err := a.migrateProfileStreamingDefaults(); err != nil {
			return err
		}
		return a.ensureWebConfig()
	}
	if err := a.ensureProfileRegistry(); err != nil {
		return err
	}
	if err := a.migrateProfileStreamingDefaults(); err != nil {
		return err
	}
	if err := a.migrateDufsComposeIfNeeded(settings); err != nil {
		return err
	}
	return a.ensureWebConfig()
}

func (a *App) InitializeInstance(settings ComposeSettings) (LauncherState, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	return a.initializeInstanceLocked(settings)
}

func (a *App) initializeInstanceLocked(settings ComposeSettings) (LauncherState, error) {
	settings = a.withRecommendedResourceDefaults(settings)
	settings = withComposeDefaults(settings)
	if err := ensureDir(a.instanceRoot); err != nil {
		return LauncherState{}, err
	}
	if err := ensureDir(a.hermesDockDir()); err != nil {
		return LauncherState{}, err
	}
	if err := a.ensureContainerInitHelpers(); err != nil {
		return LauncherState{}, err
	}
	if err := a.releaseSeedData(); err != nil {
		return LauncherState{}, err
	}
	if err := a.ensureDockDataDir(); err != nil {
		return LauncherState{}, err
	}
	var err error
	settings, err = a.ensureDufsConfig(settings, settings.DufsPassword, "before-dufs-config-initialize")
	if err != nil {
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
		LastAppliedComposeHash:    existing.LastAppliedComposeHash,
		TemplateVersion:           templateVersion,
		SkillsSnapshotImage:       defaultImage,
		HermesImage:               settings.Image,
		ComposeSettings:           settings,
		PreviousHermesImage:       existing.PreviousHermesImage,
		LastSuccessfulHermesImage: settings.Image,
		InitializedAt:             firstNonEmpty(existing.InitializedAt, now),
		UpdatedAt:                 now,
		NeedsRebuild:              existing.NeedsRebuild,
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
	if err := a.migrateDufsComposeIfNeeded(settings); err != nil {
		return LauncherState{}, err
	}
	if err := a.ensureProfileRegistry(); err != nil {
		return LauncherState{}, err
	}
	if err := a.migrateProfileStreamingDefaults(); err != nil {
		return LauncherState{}, err
	}
	return a.readState()
}

func (a *App) writeOverrideIfMissing() error {
	if _, err := os.Stat(a.overridePath()); errors.Is(err, os.ErrNotExist) {
		return atomicWriteFile(a.overridePath(), []byte("# 在这里添加高级 Docker Compose 覆盖配置。\n"), 0644)
	}
	return nil
}

func (a *App) FactoryResetInstance() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if err := a.validateResetRoot(); err != nil {
		return err
	}
	a.stopHostBridge(context.Background())
	a.StopTailLogs()
	if a.loginCancel != nil {
		a.loginCancel()
		a.loginCancel = nil
	}

	if fileExists(a.composePath()) {
		if err := a.runComposeBlocking(context.Background(), "down"); err != nil {
			return err
		}
	}
	updateState, _ := a.readUpdateState()
	updateTaskRegistered, _ := a.updateTaskRegistered()
	if updateState.AutoUpdateEnabled || updateTaskRegistered {
		if err := a.unregisterUpdateTask(); err != nil {
			return fmt.Errorf("删除自动更新任务失败：%w", err)
		}
	}
	a.stopUpdateWatcher()
	if err := a.removeInstanceExceptShared(); err != nil {
		a.startUpdateWatcher()
		return err
	}
	a.startupErr = nil
	if err := a.ensureInstanceReadyLocked(); err != nil {
		a.startUpdateWatcher()
		return err
	}
	a.startUpdateWatcher()
	return a.startHostBridge()
}

func (a *App) removeInstanceExceptShared() error {
	entries, err := os.ReadDir(a.instanceRoot)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.Name() == filepath.Base(a.sharedDir()) {
			continue
		}
		if err := os.RemoveAll(filepath.Join(a.instanceRoot, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}

func (a *App) validateResetRoot() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	expected := filepath.Clean(filepath.Join(home, ".hermes-dock"))
	actual := filepath.Clean(a.instanceRoot)
	if actual != expected {
		return fmt.Errorf("拒绝重置：实例目录不是 ~/.hermes-dock")
	}
	if actual == filepath.Clean(home) || actual == string(os.PathSeparator) {
		return fmt.Errorf("拒绝重置：实例目录不安全")
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
	case "dufs":
		if !settings.DufsEnabled {
			return fmt.Errorf("Dufs 文件管理未开启")
		}
		host = "127.0.0.1"
		port = settings.DufsPort
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
	mode := os.FileMode(0644)
	if filepath.Base(resolved) == ".env" {
		mode = 0600
	}
	if err := atomicWriteFile(resolved, []byte(req.Content), mode); err != nil {
		return err
	}
	return a.markRebuildRequired()
}

func (a *App) emit(event string, payload interface{}) {
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, event, payload)
	}
	a.emitWeb(event, payload)
}
