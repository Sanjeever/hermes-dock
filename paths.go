package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func detectInstanceRoot() string {
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

func (a *App) composePath() string {
	return filepath.Join(a.instanceRoot, "docker-compose.yaml")
}

func (a *App) overridePath() string {
	return filepath.Join(a.instanceRoot, "docker-compose.override.yaml")
}

func (a *App) envPath() string {
	return filepath.Join(a.dataDir(), ".env")
}

func (a *App) configPath() string {
	return filepath.Join(a.dataDir(), "config.yaml")
}

func (a *App) statePath() string {
	return filepath.Join(a.hermesDockDir(), "state.json")
}

func (a *App) channelDirectoryPath() string {
	return filepath.Join(a.dataDir(), "channel_directory.json")
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
	return resolved, nil
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

func (a *App) writeState(state LauncherState) error {
	if err := ensureDir(filepath.Dir(a.statePath())); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(a.statePath(), append(data, '\n'), 0644)
}

func defaultState() LauncherState {
	return LauncherState{
		SchemaVersion:       1,
		AppVersion:          appVersion,
		ManagedCompose:      true,
		TemplateVersion:     templateVersion,
		SkillsSnapshotImage: defaultImage,
		HermesImage:         defaultImage,
		ComposeSettings:     defaultComposeSettings(),
		UI:                  UIState{LastPage: "dashboard"},
		ModelAuxiliaryMode:  "auto",
	}
}

func ensureDir(path string) error {
	return os.MkdirAll(path, 0755)
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
