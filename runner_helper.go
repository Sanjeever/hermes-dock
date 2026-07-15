package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func (a *App) profileRunnerPath() string {
	return filepath.Join(a.hermesDockDir(), "helpers", "hermes-profile-runner")
}

func (a *App) ensureProfileRunnerHelper() error {
	target := a.profileRunnerPath()
	if err := ensureDir(filepath.Dir(target)); err != nil {
		return err
	}
	if err := a.copyPrebuiltProfileRunner(target); err == nil {
		return os.Chmod(target, 0755)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("更新 Hermes profile runner 失败：%w", err)
	}
	source := filepath.Join(a.projectRoot(), "cmd", "hermes-profile-runner")
	if commandExists("go") && fileExists(filepath.Join(source, "main.go")) {
		needsBuild, err := profileRunnerSourceNeedsBuild(source, target)
		if err != nil {
			return fmt.Errorf("检查 Hermes profile runner 源码失败：%w", err)
		}
		if needsBuild {
			return a.buildProfileRunner(target)
		}
	}
	if fileExists(target) {
		return os.Chmod(target, 0755)
	}
	return fmt.Errorf("缺少 Hermes profile runner，请在打包产物中提供 launcher/helpers/hermes-profile-runner")
}

func profileRunnerSourceNeedsBuild(sourceDir string, target string) (bool, error) {
	targetInfo, err := os.Stat(target)
	if errors.Is(err, os.ErrNotExist) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	newer := false
	err = filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		if info.ModTime().After(targetInfo.ModTime()) {
			newer = true
		}
		return nil
	})
	return newer, err
}

func (a *App) copyPrebuiltProfileRunner(target string) error {
	candidates := profileRunnerCandidates(a.instanceRoot, runtime.GOARCH)
	for _, candidate := range candidates {
		if !fileExists(candidate) {
			continue
		}
		return a.syncProfileRunner(candidate, target, runtime.GOOS)
	}
	return os.ErrNotExist
}

func (a *App) syncProfileRunner(source string, target string, goos string) error {
	data, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	if existing, err := os.ReadFile(target); err == nil && bytes.Equal(existing, data) {
		return nil
	}
	if goos == "windows" && fileExists(a.composePath()) && a.containerStatus(context.Background()) == "running" {
		if err := a.StopHermes(); err != nil {
			return fmt.Errorf("停止 Hermes 容器以更新 profile runner 失败：%w", err)
		}
	}
	return atomicWriteFile(target, data, 0755)
}

func profileRunnerCandidates(instanceRoot string, goarch string) []string {
	exe, _ := os.Executable()
	return profileRunnerCandidatesForExecutable(instanceRoot, goarch, exe)
}

func profileRunnerCandidatesForExecutable(instanceRoot string, goarch string, exe string) []string {
	candidates := make([]string, 0, 5)
	if exe != "" {
		exeDir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(exeDir, "hermes-profile-runner-linux-"+goarch),
			filepath.Join(exeDir, "hermes-profile-runner"),
		)
	}
	return append(candidates,
		filepath.Join(instanceRoot, "build", "profile-runner", "hermes-profile-runner-linux-"+goarch),
		filepath.Join(instanceRoot, "build", "profile-runner", "hermes-profile-runner"),
	)
}

func (a *App) buildProfileRunner(target string) error {
	source := filepath.Join(a.projectRoot(), "cmd", "hermes-profile-runner")
	if !fileExists(filepath.Join(source, "main.go")) {
		return fmt.Errorf("缺少 runner 源码：%s", source)
	}
	file, err := os.CreateTemp(filepath.Dir(target), ".hermes-profile-runner-*")
	if err != nil {
		return err
	}
	tmpPath := file.Name()
	if err := file.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	defer os.Remove(tmpPath)
	cmd := backgroundCommand("go", "build", "-buildvcs=false", "-o", tmpPath, source)
	cmd.Dir = a.projectRoot()
	cmd.Env = append(os.Environ(),
		"CGO_ENABLED=0",
		"GOOS=linux",
		"GOARCH="+runtime.GOARCH,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("构建 Hermes profile runner 失败：%w: %s", err, redact(string(out)))
	}
	if err := os.Chmod(tmpPath, 0755); err != nil {
		return err
	}
	return a.syncProfileRunner(tmpPath, target, runtime.GOOS)
}

func (a *App) projectRoot() string {
	cwd, err := os.Getwd()
	if err == nil && fileExists(filepath.Join(cwd, "go.mod")) {
		return cwd
	}
	return filepath.Dir(cwd)
}
