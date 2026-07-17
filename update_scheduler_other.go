//go:build !windows && !linux && !darwin

package main

import "errors"

func (a *App) registerUpdateTask() error {
	return errors.New("当前系统不支持自动更新任务")
}

func (a *App) unregisterUpdateTask() error {
	return nil
}

func (a *App) updateTaskRegistered() (bool, error) {
	return false, nil
}

func scheduledRelaunchAllowed() bool {
	return false
}
