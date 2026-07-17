package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type updateConfig struct {
	packagePath  string
	assetName    string
	targetPath   string
	instanceRoot string
	token        string
	waitPID      int
	relaunch     bool
}

func main() {
	config := parseFlags()
	if config.instanceRoot != "" {
		defer os.Remove(filepath.Join(config.instanceRoot, "launcher", "updates", "update.lock"))
	}
	if err := run(config); err != nil {
		logUpdate(config.instanceRoot, "更新失败："+err.Error())
		saveUpdateError(config.instanceRoot, err.Error())
		os.Exit(1)
	}
	saveUpdateError(config.instanceRoot, "")
}

func parseFlags() updateConfig {
	var config updateConfig
	flag.StringVar(&config.packagePath, "package", "", "update package path")
	flag.StringVar(&config.assetName, "asset-name", "", "release asset name")
	flag.StringVar(&config.targetPath, "target", "", "application executable path")
	flag.StringVar(&config.instanceRoot, "instance-root", "", "Hermes Dock instance root")
	flag.StringVar(&config.token, "token", "", "update health token")
	flag.IntVar(&config.waitPID, "wait-pid", 0, "application pid to wait for")
	flag.BoolVar(&config.relaunch, "relaunch", false, "relaunch application after install")
	flag.Parse()
	return config
}

func run(config updateConfig) error {
	if strings.TrimSpace(config.packagePath) == "" || strings.TrimSpace(config.targetPath) == "" || strings.TrimSpace(config.instanceRoot) == "" || strings.TrimSpace(config.token) == "" {
		return errors.New("更新参数不完整")
	}
	if filepath.Base(config.packagePath) != config.assetName {
		return errors.New("安装包名称不匹配")
	}
	if _, err := os.Stat(config.packagePath); err != nil {
		return fmt.Errorf("安装包不可用：%w", err)
	}
	if config.waitPID > 0 {
		logUpdate(config.instanceRoot, "等待企智盒退出")
		if err := waitForProcessExit(config.waitPID, 60*time.Second); err != nil {
			return err
		}
	}
	logUpdate(config.instanceRoot, "开始安装更新")
	rollback, cleanup, err := installPackage(config)
	if err != nil {
		return err
	}
	if cleanup != nil {
		defer cleanup()
	}
	if !config.relaunch {
		pendingPath := filepath.Join(config.instanceRoot, "launcher", "updates", "restart-pending")
		if err := os.WriteFile(pendingPath, []byte("pending\n"), 0600); err != nil {
			return fmt.Errorf("记录待启动状态失败：%w", err)
		}
		logUpdate(config.instanceRoot, "更新安装完成，将在下次登录时启动")
		return nil
	}
	process, err := launchApplication(config.targetPath, config.instanceRoot, config.token)
	if err != nil {
		if rollback != nil {
			_ = rollback()
		}
		return fmt.Errorf("启动新版本失败：%w", err)
	}
	healthPath := filepath.Join(config.instanceRoot, "launcher", "updates", "health-"+config.token)
	if waitForHealthFile(healthPath, 3*time.Minute) {
		_ = os.Remove(healthPath)
		logUpdate(config.instanceRoot, "更新完成，新版本启动正常")
		return nil
	}
	_ = process.Kill()
	_, _ = process.Wait()
	if rollback != nil {
		if err := rollback(); err != nil {
			return fmt.Errorf("新版本启动超时，回滚失败：%w", err)
		}
		_, _ = launchApplication(config.targetPath, config.instanceRoot, "")
	}
	return errors.New("新版本未通过启动检查，已回滚")
}

func waitForProcessExit(pid int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !processRunning(pid) {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return errors.New("等待企智盒退出超时")
}

func waitForHealthFile(path string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(path)
		if err == nil && strings.TrimSpace(string(data)) == "ok" {
			return true
		}
		time.Sleep(time.Second)
	}
	return false
}

func launchApplication(target string, instanceRoot string, token string) (*os.Process, error) {
	args := []string{"--instance-root", instanceRoot}
	if token != "" {
		args = append(args, "--update-token", token)
	}
	cmd := detachedCommand(target, args...)
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return cmd.Process, nil
}

func backupExecutable(config updateConfig) (func() error, func(), error) {
	backupPath := filepath.Join(config.instanceRoot, "launcher", "updates", "rollback-"+config.token+filepath.Ext(config.targetPath))
	if err := copyFile(config.targetPath, backupPath, 0700); err != nil {
		return nil, nil, fmt.Errorf("备份当前版本失败：%w", err)
	}
	rollback := func() error {
		return copyFile(backupPath, config.targetPath, 0755)
	}
	cleanup := func() {
		_ = os.Remove(backupPath)
	}
	return rollback, cleanup, nil
}

func copyFile(source string, target string, mode os.FileMode) error {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
		return err
	}
	output, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(output, input); err != nil {
		output.Close()
		return err
	}
	if err := output.Sync(); err != nil {
		output.Close()
		return err
	}
	return output.Close()
}

func logUpdate(instanceRoot string, message string) {
	if instanceRoot == "" {
		return
	}
	path := filepath.Join(instanceRoot, "launcher", "logs", "update.log")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return
	}
	defer file.Close()
	_, _ = file.WriteString(time.Now().Format(time.RFC3339) + " " + message + "\n")
}

func saveUpdateError(instanceRoot string, message string) {
	path := filepath.Join(instanceRoot, "launcher", "update.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var state map[string]interface{}
	if json.Unmarshal(data, &state) != nil {
		return
	}
	state["lastError"] = message
	data, err = json.MarshalIndent(state, "", "  ")
	if err != nil {
		return
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".hermes-dock-update-state-*")
	if err != nil {
		return
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	_ = temp.Chmod(0644)
	if _, err := temp.Write(append(data, '\n')); err != nil {
		temp.Close()
		return
	}
	if temp.Sync() != nil {
		temp.Close()
		return
	}
	if temp.Close() != nil {
		return
	}
	_ = os.Rename(tempPath, path)
}

func runCommand(name string, args ...string) error {
	cmd := backgroundCommand(name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s：%s", err, strings.TrimSpace(string(output)))
	}
	return nil
}
