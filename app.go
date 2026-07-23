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
	appVersion      = "1.11.17"
	templateVersion = "2026.07.22"
	defaultImage    = "nousresearch/hermes-agent:v2026.6.19"
)

type App struct {
	ctx                 context.Context
	instanceRoot        string
	mu                  sync.Mutex
	stateMu             sync.Mutex
	operationMu         sync.Mutex
	webSessionMu        sync.RWMutex
	loginMu             sync.Mutex
	logMu               sync.Mutex
	logCancel           context.CancelFunc
	loginSession        *loginSessionState
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
	postUpdateMu        sync.Mutex
	applyMu             sync.Mutex
	applyCancel         context.CancelFunc
	applyActiveID       string
	applyOwnsOperation  bool
	applySlowAfter      time.Duration
	applyPollInterval   time.Duration
}

func NewApp() *App {
	return &App{
		hostBridgeAddr:    hostBridgeAddress,
		applySlowAfter:    2 * time.Minute,
		applyPollInterval: 500 * time.Millisecond,
	}
}

func (a *App) ChooseSharedDirectory(currentPath string) (string, error) {
	if a.ctx == nil {
		return "", errors.New("应用尚未初始化")
	}
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title:                "选择共享文件目录",
		DefaultDirectory:     strings.TrimSpace(currentPath),
		CanCreateDirectories: true,
	})
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.instanceRoot = detectInstanceRoot()
	a.startUpdateWatcher()
	a.startupErr = a.ensureInstanceReady()
	if a.startupErr == nil {
		a.startupErr = a.preparePostUpdateTask()
	}
	if a.startupErr == nil {
		a.acknowledgeInstalledUpdate()
		a.resumeApplyConfigTask()
		a.startPostUpdateTask()
		a.startHostBridge()
		a.startWebServer()
		a.startTray()
	}
}

func (a *App) shutdown(ctx context.Context) {
	a.cancelApplyConfigTask()
	a.stopUpdateWatcher()
	a.stopTray()
	a.stopHostBridge(ctx)
	a.cleanupHostNotifications()
	a.stopWebServer(ctx)
	if err := a.cancelLoginSessionAndWait(""); err != nil {
		a.emit("docker:progress", StreamEvent{Line: redact(fmt.Sprintf("停止扫码绑定失败：%v", err))})
	}
	a.StopTailLogs()
}

func (a *App) GetAppState() (AppState, error) {
	return a.GetAppStateForProfile(a.currentProfileID())
}

func (a *App) GetAppStateForProfile(profileID string) (AppState, error) {
	if a.startupErr != nil {
		if err := a.ensureInstanceReady(); err != nil {
			return AppState{}, err
		}
		a.startupErr = nil
	} else if err := a.ensureInstanceReady(); err != nil {
		return AppState{}, err
	}
	profileID, err := a.resolveProfileID(profileID)
	if err != nil {
		return AppState{}, err
	}
	state, err := a.readState()
	if err != nil {
		return AppState{}, err
	}
	profiles, err := a.readProfileRegistry()
	if err != nil {
		return AppState{}, err
	}
	compose := a.readComposeSettings()
	proxy := a.readProxySettings()
	env, err := readEnvFile(a.profileEnvPath(profileID))
	if err != nil {
		return AppState{}, err
	}
	model, err := a.readModelConfigForProfile(profileID)
	if err != nil {
		return AppState{}, err
	}
	providers, err := a.readProviderConfigForProfile(profileID)
	if err != nil {
		return AppState{}, err
	}
	channels, err := a.GetChannelsForProfile(profileID)
	if err != nil {
		return AppState{}, err
	}
	containerStatus := a.containerStatus(context.Background())

	return AppState{
		AppVersion:       appVersion,
		InstanceRoot:     a.instanceRoot,
		NeedsRebuild:     state.NeedsRebuild,
		State:            state,
		Profiles:         profiles,
		ActiveProfile:    profileID,
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
		ApplyConfig:      a.readApplyConfigStatus(),
		BundledContent:   a.bundledContentAvailability(profiles),
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
		if err := a.migrateRuntimeDependencyComposeIfNeeded(settings); err != nil {
			return err
		}
		if err := a.migrateFixedHermesImageIfNeeded(settings); err != nil {
			return err
		}
		if err := a.migratePrivateHermesServicesIfNeeded(settings); err != nil {
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
	if err := a.migrateFixedHermesImageIfNeeded(settings); err != nil {
		return err
	}
	if err := a.migratePrivateHermesServicesIfNeeded(settings); err != nil {
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
		if err := a.migratePrivateHermesServicesIfNeeded(settings); err != nil {
			return err
		}
		if err := a.migrateDufsComposeIfNeeded(settings); err != nil {
			return err
		}
		if err := a.ensureProfileRegistry(); err != nil {
			return err
		}
		return a.ensureWebConfig()
	}
	if err := a.ensureProfileRegistry(); err != nil {
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
		RuntimeDependencyVersion:  existing.RuntimeDependencyVersion,
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
	if existing.InstanceID == "" {
		state.RuntimeDependencyVersion = runtimeDependencyBundleVersion
	}
	if err := a.writeState(state); err != nil {
		return LauncherState{}, err
	}
	if err := a.migratePrivateHermesServicesIfNeeded(settings); err != nil {
		return LauncherState{}, err
	}
	if err := a.migrateDufsComposeIfNeeded(settings); err != nil {
		return LauncherState{}, err
	}
	if err := a.ensureProfileRegistry(); err != nil {
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

func (a *App) FactoryResetInstance() (err error) {
	release, err := a.beginExclusiveOperation("恢复出厂设置")
	if err != nil {
		return err
	}
	defer release()
	a.mu.Lock()
	defer a.mu.Unlock()

	if err := a.validateResetRoot(); err != nil {
		return err
	}
	hostBridgeWasRunning := a.hostBridgeStatus().Running
	a.stopHostBridge(context.Background())
	defer func() {
		if err != nil && hostBridgeWasRunning {
			if restartErr := a.startHostBridge(); restartErr != nil {
				err = errors.Join(err, fmt.Errorf("恢复宿主机控制失败：%w", restartErr))
			}
		}
	}()
	a.StopTailLogs()
	if err := a.cancelLoginSessionAndWait(""); err != nil {
		return fmt.Errorf("停止扫码绑定失败：%w", err)
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

func (a *App) OpenFileManagement() error {
	settings := a.readComposeSettings()
	if !settings.DufsEnabled {
		return fmt.Errorf("Dufs 文件管理未开启")
	}
	runtime.BrowserOpenURL(a.ctx, a.dufsStatus().LocalURL)
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
	release, err := a.beginExclusiveOperation("保存文件")
	if err != nil {
		return err
	}
	defer release()
	resolved, err := a.safePath(req.Path)
	if err != nil {
		return err
	}
	if resolved == a.composePath() {
		return fmt.Errorf("标准 Docker Compose 由启动器管理，请使用覆盖文件")
	}
	if strings.HasSuffix(resolved, ".yaml") || strings.HasSuffix(resolved, ".yml") {
		if err := parseYAML([]byte(req.Content), nil); err != nil {
			return err
		}
	}
	if resolved == a.overridePath() {
		_, hasImage, err := composeServiceImage([]byte(req.Content), "hermes")
		if err != nil {
			return err
		}
		if hasImage {
			return fmt.Errorf("Hermes 镜像已固定，覆盖文件不能设置 services.hermes.image")
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
