//go:build linux

package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func hostRPAPlatformCapabilities() hostRPACapabilities {
	capabilities := hostRPACapabilities{Backend: "none"}
	if strings.EqualFold(strings.TrimSpace(os.Getenv("XDG_SESSION_TYPE")), "wayland") {
		capabilities.Reason = "当前 Wayland 会话不支持轻量桌面自动化"
		return capabilities
	}
	if strings.TrimSpace(os.Getenv("DISPLAY")) == "" {
		capabilities.Reason = "当前 Linux 环境没有可用的 X11 桌面会话"
		return capabilities
	}
	if _, err := exec.LookPath("xdotool"); err != nil {
		capabilities.Reason = "当前 Linux 主机未安装 xdotool，无法使用轻量桌面自动化"
		return capabilities
	}
	capabilities.Supported = true
	capabilities.Backend = "linux-xdotool"
	capabilities.Windows = true
	capabilities.Mouse = true
	capabilities.Keyboard = true
	capabilities.UnicodeText = true
	return capabilities
}

func hostRPAListWindows() ([]hostRPAWindow, error) {
	output, err := runXDoTool("search", "--onlyvisible", "--name", ".")
	if err != nil {
		if strings.TrimSpace(output) == "" {
			return []hostRPAWindow{}, nil
		}
		return nil, err
	}
	activeOutput, _ := runXDoTool("getactivewindow")
	activeID := strings.TrimSpace(activeOutput)
	windows := make([]hostRPAWindow, 0)
	for _, line := range strings.Fields(output) {
		window, windowErr := linuxWindowInfo(line)
		if windowErr != nil {
			continue
		}
		window.Active = line == activeID
		windows = append(windows, window)
	}
	return windows, nil
}

func hostRPAActiveWindow() (hostRPAWindow, error) {
	output, err := runXDoTool("getactivewindow")
	if err != nil {
		return hostRPAWindow{}, err
	}
	return linuxWindowInfo(strings.TrimSpace(output))
}

func hostRPAActivateWindow(id string) error {
	windowID, err := parseLinuxWindowID(id)
	if err != nil {
		return err
	}
	if _, err := runXDoTool("windowactivate", "--sync", windowID); err != nil {
		return err
	}
	active, err := hostRPAActiveWindow()
	if err != nil {
		return err
	}
	if active.ID != id {
		return errors.New("窗口未进入前台")
	}
	return nil
}

func hostRPAMousePosition() (int, int, error) {
	output, err := runXDoTool("getmouselocation", "--shell")
	if err != nil {
		return 0, 0, err
	}
	values := parseShellValues(output)
	x, xErr := strconv.Atoi(values["X"])
	y, yErr := strconv.Atoi(values["Y"])
	if xErr != nil || yErr != nil {
		return 0, 0, errors.New("无法解析鼠标位置")
	}
	return x, y, nil
}

func hostRPAPerformMouse(req hostRPAMouseRequest) error {
	switch req.Action {
	case "move":
		return linuxMoveMouse(req.X, req.Y, req.DurationMS)
	case "click":
		if err := linuxMoveMouse(req.X, req.Y, req.DurationMS); err != nil {
			return err
		}
		_, err := runXDoTool("click", "--repeat", strconv.Itoa(req.Count), "--delay", "80", linuxMouseButton(req.Button))
		return err
	case "drag":
		if err := linuxMoveMouse(req.FromX, req.FromY, 0); err != nil {
			return err
		}
		button := linuxMouseButton(req.Button)
		if _, err := runXDoTool("mousedown", button); err != nil {
			return err
		}
		moveErr := linuxMoveMouse(req.ToX, req.ToY, req.DurationMS)
		_, upErr := runXDoTool("mouseup", button)
		return errors.Join(moveErr, upErr)
	case "scroll":
		if err := linuxMoveMouse(req.X, req.Y, 0); err != nil {
			return err
		}
		if req.DY != 0 {
			button := "4"
			if req.DY < 0 {
				button = "5"
			}
			if _, err := runXDoTool("click", "--repeat", strconv.Itoa(absInt(req.DY)), button); err != nil {
				return err
			}
		}
		if req.DX != 0 {
			button := "7"
			if req.DX < 0 {
				button = "6"
			}
			_, err := runXDoTool("click", "--repeat", strconv.Itoa(absInt(req.DX)), button)
			return err
		}
		return nil
	default:
		return errors.New("不支持的鼠标动作")
	}
}

func hostRPAPerformKeyboard(req hostRPAKeyboardRequest) error {
	switch req.Action {
	case "press":
		key := linuxKey(req.Key)
		for index := 0; index < req.Count; index++ {
			if _, err := runXDoTool("key", "--clearmodifiers", key); err != nil {
				return err
			}
			if req.IntervalMS > 0 && index+1 < req.Count {
				time.Sleep(time.Duration(req.IntervalMS) * time.Millisecond)
			}
		}
		return nil
	case "hotkey":
		keys := make([]string, 0, len(req.Keys))
		for _, key := range req.Keys {
			keys = append(keys, linuxKey(key))
		}
		_, err := runXDoTool("key", "--clearmodifiers", strings.Join(keys, "+"))
		return err
	case "type":
		_, err := runXDoTool("type", "--clearmodifiers", "--delay", strconv.Itoa(req.IntervalMS), "--", req.Text)
		return err
	default:
		return errors.New("不支持的键盘动作")
	}
}

func linuxWindowInfo(windowID string) (hostRPAWindow, error) {
	if windowID == "" {
		return hostRPAWindow{}, errors.New("窗口 ID 为空")
	}
	title, err := runXDoTool("getwindowname", windowID)
	if err != nil {
		return hostRPAWindow{}, err
	}
	geometry, err := runXDoTool("getwindowgeometry", "--shell", windowID)
	if err != nil {
		return hostRPAWindow{}, err
	}
	values := parseShellValues(geometry)
	x, _ := strconv.Atoi(values["X"])
	y, _ := strconv.Atoi(values["Y"])
	width, _ := strconv.Atoi(values["WIDTH"])
	height, _ := strconv.Atoi(values["HEIGHT"])
	pid := 0
	if pidOutput, pidErr := runXDoTool("getwindowpid", windowID); pidErr == nil {
		pid, _ = strconv.Atoi(strings.TrimSpace(pidOutput))
	}
	return hostRPAWindow{
		ID:     "x11:" + windowID,
		PID:    pid,
		Title:  strings.TrimSpace(title),
		X:      x,
		Y:      y,
		Width:  width,
		Height: height,
	}, nil
}

func parseLinuxWindowID(id string) (string, error) {
	value := strings.TrimPrefix(strings.TrimSpace(id), "x11:")
	if value == id || value == "" {
		return "", errors.New("X11 窗口 ID 无效")
	}
	if _, err := strconv.ParseUint(value, 10, 64); err != nil {
		return "", errors.New("X11 窗口 ID 无效")
	}
	return value, nil
}

func linuxMoveMouse(x, y, durationMS int) error {
	if durationMS <= 0 {
		_, err := runXDoTool("mousemove", "--sync", strconv.Itoa(x), strconv.Itoa(y))
		return err
	}
	startX, startY, err := hostRPAMousePosition()
	if err != nil {
		return err
	}
	steps := durationMS / 16
	if steps < 1 {
		steps = 1
	}
	for step := 1; step <= steps; step++ {
		currentX := startX + (x-startX)*step/steps
		currentY := startY + (y-startY)*step/steps
		if _, err := runXDoTool("mousemove", strconv.Itoa(currentX), strconv.Itoa(currentY)); err != nil {
			return err
		}
		time.Sleep(time.Duration(durationMS/steps) * time.Millisecond)
	}
	return nil
}

func linuxMouseButton(button string) string {
	switch button {
	case "right":
		return "3"
	case "middle":
		return "2"
	default:
		return "1"
	}
}

func linuxKey(key string) string {
	switch key {
	case "cmd", "command", "win":
		return "super"
	case "control":
		return "ctrl"
	case "escape":
		return "Escape"
	case "enter":
		return "Return"
	case "tab":
		return "Tab"
	case "space":
		return "space"
	case "backspace":
		return "BackSpace"
	case "delete":
		return "Delete"
	case "pageup":
		return "Prior"
	case "pagedown":
		return "Next"
	case "left":
		return "Left"
	case "right":
		return "Right"
	case "up":
		return "Up"
	case "down":
		return "Down"
	case "home":
		return "Home"
	case "end":
		return "End"
	default:
		return key
	}
}

func parseShellValues(output string) map[string]string {
	values := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		key, value, ok := strings.Cut(scanner.Text(), "=")
		if ok {
			values[key] = value
		}
	}
	return values
}

func runXDoTool(args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "xdotool", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return string(output), errors.New("xdotool 执行超时")
		}
		detail := strings.TrimSpace(string(output))
		if detail == "" {
			detail = err.Error()
		}
		return string(output), fmt.Errorf("xdotool 执行失败：%s", detail)
	}
	return string(output), nil
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
