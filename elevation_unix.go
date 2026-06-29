//go:build darwin || linux

package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

func ensureElevated() (bool, error) {
	if os.Geteuid() == 0 {
		return false, nil
	}
	exe, err := os.Executable()
	if err != nil {
		return false, fmt.Errorf("无法定位当前程序: %w", err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return false, fmt.Errorf("无法定位当前用户目录: %w", err)
	}
	switch runtime.GOOS {
	case "darwin":
		return relaunchWithAppleScript(exe, home, os.Args[1:])
	case "linux":
		return relaunchWithLinuxElevator(exe, home, os.Args[1:])
	default:
		return false, nil
	}
}

func relaunchWithAppleScript(exe string, home string, args []string) (bool, error) {
	command := "env " + shellJoin("HOME="+home) + " nohup " + shellJoin(append([]string{exe}, args...)...) + " >/dev/null 2>&1 &"
	script := fmt.Sprintf("do shell script %q with administrator privileges", command)
	if err := exec.Command("osascript", "-e", script).Run(); err != nil {
		return false, fmt.Errorf("需要管理员权限才能启动 Hermes Dock: %w", err)
	}
	return true, nil
}

func relaunchWithLinuxElevator(exe string, home string, args []string) (bool, error) {
	elevator, err := exec.LookPath("pkexec")
	if err != nil {
		elevator, err = exec.LookPath("sudo")
		if err != nil {
			return false, fmt.Errorf("需要 root 权限才能启动 Hermes Dock，请安装 pkexec 或从终端使用 sudo 启动")
		}
	}
	cmd := exec.Command(elevator, append(elevatedEnvArgs(home), append([]string{exe}, args...)...)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return false, fmt.Errorf("需要 root 权限才能启动 Hermes Dock: %w", err)
	}
	return true, nil
}

func elevatedEnvArgs(home string) []string {
	args := []string{"env", "HOME=" + home}
	for _, name := range []string{"DISPLAY", "WAYLAND_DISPLAY", "XAUTHORITY", "XDG_RUNTIME_DIR"} {
		if value := os.Getenv(name); value != "" {
			args = append(args, name+"="+value)
		}
	}
	return args
}

func shellJoin(parts ...string) string {
	quoted := make([]string, 0, len(parts))
	for _, part := range parts {
		quoted = append(quoted, shellQuote(part))
	}
	return strings.Join(quoted, " ")
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
