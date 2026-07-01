//go:build (darwin || linux) && !dev && !bindings

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
	if runtime.GOOS == "linux" {
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
	default:
		return false, nil
	}
}

func relaunchWithAppleScript(exe string, home string, args []string) (bool, error) {
	command := "env " + shellJoin("HOME="+home) + " nohup " + shellJoin(append([]string{exe}, args...)...) + " >/dev/null 2>&1 &"
	script := fmt.Sprintf("do shell script %q with administrator privileges", command)
	if err := exec.Command("osascript", "-e", script).Run(); err != nil {
		return false, fmt.Errorf("需要管理员权限才能启动企智盒: %w", err)
	}
	return true, nil
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
