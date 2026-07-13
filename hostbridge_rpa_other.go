//go:build !darwin && !linux && !windows

package main

import "errors"

func hostRPAPlatformCapabilities() hostRPACapabilities {
	return hostRPACapabilities{Reason: "当前宿主机系统不支持桌面自动化", Backend: "none"}
}

func hostRPAListWindows() ([]hostRPAWindow, error) {
	return nil, errors.New("当前宿主机系统不支持桌面自动化")
}

func hostRPAActiveWindow() (hostRPAWindow, error) {
	return hostRPAWindow{}, errors.New("当前宿主机系统不支持桌面自动化")
}

func hostRPAActivateWindow(string) error {
	return errors.New("当前宿主机系统不支持桌面自动化")
}

func hostRPAMousePosition() (int, int, error) {
	return 0, 0, errors.New("当前宿主机系统不支持桌面自动化")
}

func hostRPAPerformMouse(hostRPAMouseRequest) error {
	return errors.New("当前宿主机系统不支持桌面自动化")
}

func hostRPAPerformKeyboard(hostRPAKeyboardRequest) error {
	return errors.New("当前宿主机系统不支持桌面自动化")
}
