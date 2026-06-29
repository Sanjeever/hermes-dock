package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func (a *App) profileRunnerPath() string {
	return filepath.Join(a.hermesDockDir(), "helpers", "hermes-profile-runner")
}

func (a *App) ensureProfileRunnerHelper() error {
	target := a.profileRunnerPath()
	if fileExists(target) {
		return os.Chmod(target, 0755)
	}
	if err := ensureDir(filepath.Dir(target)); err != nil {
		return err
	}
	if err := a.copyPrebuiltProfileRunner(target); err == nil {
		return os.Chmod(target, 0755)
	}
	if commandExists("go") {
		return a.buildProfileRunner(target)
	}
	return fmt.Errorf("缺少 Hermes profile runner，请在打包产物中提供 launcher/helpers/hermes-profile-runner")
}

func (a *App) copyPrebuiltProfileRunner(target string) error {
	candidates := profileRunnerCandidates(a.instanceRoot, runtime.GOARCH)
	for _, candidate := range candidates {
		if !fileExists(candidate) {
			continue
		}
		return copyFile(candidate, target, 0755)
	}
	return os.ErrNotExist
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
	cmd := exec.Command("go", "build", "-buildvcs=false", "-o", target, source)
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
	return os.Chmod(target, 0755)
}

func (a *App) projectRoot() string {
	cwd, err := os.Getwd()
	if err == nil && fileExists(filepath.Join(cwd, "go.mod")) {
		return cwd
	}
	return filepath.Dir(cwd)
}
