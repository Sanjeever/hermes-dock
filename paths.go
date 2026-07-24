package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func detectInstanceRoot() string {
	if instanceRootOverride != "" {
		return instanceRootOverride
	}
	home, err := os.UserHomeDir()
	if err != nil {
		cwd, _ := os.Getwd()
		return filepath.Join(cwd, ".hermes-dock")
	}
	return filepath.Join(home, ".hermes-dock")
}

func (a *App) hermesDockDir() string {
	return filepath.Join(a.instanceRoot, "launcher")
}

func (a *App) dataDir() string {
	return filepath.Join(a.instanceRoot, "data")
}

func (a *App) sharedDir() string {
	return filepath.Join(a.instanceRoot, "shared")
}

func (a *App) defaultDataDir() string {
	return a.dataDir()
}

func (a *App) profileDataDir(profileID string) string {
	profileID = strings.TrimSpace(profileID)
	if profileID == "" || profileID == "default" {
		return a.defaultDataDir()
	}
	return filepath.Join(a.defaultDataDir(), "profiles", profileID)
}

func (a *App) currentProfileID() string {
	registry, err := a.readProfileRegistry()
	if err != nil {
		return "default"
	}
	state, _ := a.readState()
	id := strings.TrimSpace(state.UI.LastProfile)
	if id == "" {
		return "default"
	}
	if profileExists(registry, id) {
		return id
	}
	return "default"
}

func (a *App) resolveProfileID(profileID string) (string, error) {
	profileID = strings.TrimSpace(profileID)
	if profileID == "" {
		profileID = a.currentProfileID()
	}
	registry, err := a.readProfileRegistry()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) && profileID == defaultProfileID && fileExists(a.defaultDataDir()) {
			return profileID, nil
		}
		return "", err
	}
	if !profileExists(registry, profileID) {
		return "", fmt.Errorf("profile 不存在：%s", profileID)
	}
	return profileID, nil
}

func (a *App) currentProfileDataDir() string {
	return a.profileDataDir(a.currentProfileID())
}

func (a *App) composePath() string {
	return filepath.Join(a.instanceRoot, "docker-compose.yaml")
}

func (a *App) overridePath() string {
	return filepath.Join(a.instanceRoot, "docker-compose.override.yaml")
}

func (a *App) envPath() string {
	return a.profileEnvPath(a.currentProfileID())
}

func (a *App) profileEnvPath(profileID string) string {
	return filepath.Join(a.profileDataDir(profileID), ".env")
}

func (a *App) defaultEnvPath() string {
	return filepath.Join(a.defaultDataDir(), ".env")
}

func (a *App) configPath() string {
	return a.profileConfigPath(a.currentProfileID())
}

func (a *App) profileConfigPath(profileID string) string {
	return filepath.Join(a.profileDataDir(profileID), "config.yaml")
}

func (a *App) defaultConfigPath() string {
	return filepath.Join(a.defaultDataDir(), "config.yaml")
}

func (a *App) soulPath() string {
	return a.profileSoulPath(a.currentProfileID())
}

func (a *App) profileSoulPath(profileID string) string {
	return filepath.Join(a.profileDataDir(profileID), "SOUL.md")
}

func (a *App) statePath() string {
	return filepath.Join(a.hermesDockDir(), "state.json")
}

func (a *App) profilesPath() string {
	return filepath.Join(a.hermesDockDir(), "profiles.json")
}

func (a *App) applyStatusPath() string {
	return filepath.Join(a.hermesDockDir(), "apply-status.json")
}

func (a *App) bundledContentStatePath(profileID string) string {
	return filepath.Join(a.hermesDockDir(), "profile-content", profileID+".json")
}

func (a *App) hostBridgeTokenPath() string {
	return filepath.Join(a.hermesDockDir(), "host-bridge.token")
}

func (a *App) dufsConfigPath() string {
	return filepath.Join(a.hermesDockDir(), "dufs", "config.yaml")
}

func (a *App) updateStatePath() string {
	return filepath.Join(a.hermesDockDir(), "update.json")
}

func (a *App) updateDir() string {
	return filepath.Join(a.hermesDockDir(), "updates")
}

func (a *App) updateRequestPath() string {
	return filepath.Join(a.updateDir(), "request.json")
}

func (a *App) updatePIDPath() string {
	return filepath.Join(a.updateDir(), "app.pid")
}

func (a *App) updateLockPath() string {
	return filepath.Join(a.updateDir(), "update.lock")
}

func (a *App) updaterErrorPath() string {
	return filepath.Join(a.updateDir(), "last-error")
}

func (a *App) dockDataDir() string {
	return filepath.Join(a.defaultDataDir(), ".dock")
}

func (a *App) runtimeManifestPath() string {
	return filepath.Join(a.dockDataDir(), "profiles-runtime.json")
}

func (a *App) runtimeStatusPath() string {
	return filepath.Join(a.dockDataDir(), "profile-status.json")
}

func (a *App) channelDirectoryPath() string {
	return a.profileChannelDirectoryPath(a.currentProfileID())
}

func (a *App) profileChannelDirectoryPath(profileID string) string {
	return filepath.Join(a.profileDataDir(profileID), "channel_directory.json")
}

func (a *App) safePath(path string) (string, error) {
	if path == "" {
		return "", errors.New("路径不能为空")
	}
	var resolved string
	if filepath.IsAbs(path) {
		resolved = filepath.Clean(path)
	} else {
		resolved = filepath.Join(a.instanceRoot, path)
	}
	root := filepath.Clean(a.instanceRoot)
	resolved = filepath.Clean(resolved)
	if resolved != root && !strings.HasPrefix(resolved, root+string(os.PathSeparator)) {
		return "", errors.New("路径不能超出实例目录")
	}
	if err := ensureNoSymlinkComponents(root, resolved); err != nil {
		return "", err
	}
	return resolved, nil
}

func ensureNoSymlinkComponents(root string, path string) error {
	root = filepath.Clean(root)
	path = filepath.Clean(path)
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return fmt.Errorf("路径超出允许范围")
	}
	current := root
	components := []string{}
	if rel != "." {
		components = strings.Split(rel, string(os.PathSeparator))
	}
	for _, component := range append([]string{""}, components...) {
		if component != "" {
			current = filepath.Join(current, component)
		}
		info, err := os.Lstat(current)
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("路径包含符号链接：%s", current)
		}
	}
	return nil
}

func (a *App) readState() (LauncherState, error) {
	var state LauncherState
	data, err := os.ReadFile(a.statePath())
	if err != nil {
		return defaultState(), err
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return defaultState(), err
	}
	return state, nil
}

func (a *App) readStateAllowMissing() (LauncherState, error) {
	state, err := a.readState()
	if errors.Is(err, os.ErrNotExist) {
		return defaultState(), nil
	}
	return state, err
}

func (a *App) writeState(state LauncherState) error {
	a.stateMu.Lock()
	defer a.stateMu.Unlock()
	return a.writeStateUnlocked(state)
}

func (a *App) updateState(update func(*LauncherState) error) error {
	return a.updateStateLocked(false, update)
}

func (a *App) updateStateAllowMissing(update func(*LauncherState) error) error {
	return a.updateStateLocked(true, update)
}

func (a *App) updateStateLocked(allowMissing bool, update func(*LauncherState) error) error {
	a.stateMu.Lock()
	defer a.stateMu.Unlock()
	state, err := a.readState()
	if err != nil {
		if !allowMissing || !errors.Is(err, os.ErrNotExist) {
			return err
		}
		state = defaultState()
	}
	if err := update(&state); err != nil {
		return err
	}
	return a.writeStateUnlocked(state)
}

func (a *App) writeStateUnlocked(state LauncherState) error {
	if err := ensureDir(filepath.Dir(a.statePath())); err != nil {
		return err
	}
	state.ComposeSettings.DufsPassword = ""
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return atomicWriteFile(a.statePath(), append(data, '\n'), 0644)
}

func (a *App) markRebuildRequired() error {
	return a.updateState(func(state *LauncherState) error {
		if state.NeedsRebuild && !state.PendingDufsOnly {
			return nil
		}
		state.NeedsRebuild = true
		state.PendingDufsOnly = false
		state.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		return nil
	})
}

func defaultState() LauncherState {
	return LauncherState{
		SchemaVersion:            1,
		AppVersion:               appVersion,
		ManagedCompose:           true,
		TemplateVersion:          templateVersion,
		RuntimeDependencyVersion: runtimeDependencyBundleVersion,
		SkillsSnapshotImage:      defaultImage,
		HermesImage:              defaultImage,
		ComposeSettings:          defaultComposeSettings(),
		UI:                       UIState{LastPage: "dashboard", LastProfile: "default"},
		ModelAuxiliaryMode:       "auto",
	}
}

func ensureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

func ensureWritableDirectory(path string) error {
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		if err := ensureDir(path); err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else if !info.IsDir() {
		return errors.New("共享文件路径不是目录")
	}

	file, err := os.CreateTemp(path, ".hermes-dock-write-check-*")
	if err != nil {
		return errors.New("共享文件目录不可写：" + err.Error())
	}
	tempPath := file.Name()
	if err := file.Close(); err != nil {
		_ = os.Remove(tempPath)
		return err
	}
	return os.Remove(tempPath)
}

func (a *App) ensureDockDataDir() error {
	if err := ensureDir(a.dockDataDir()); err != nil {
		return err
	}
	if runtime.GOOS == "windows" {
		return nil
	}
	return os.Chmod(a.dockDataDir(), 0777|os.ModeSticky)
}

func fileSHA256(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func commandExists(name string) bool {
	_, err := execLookPath(name)
	return err == nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

var execLookPath = func(name string) (string, error) {
	return exec.LookPath(name)
}
