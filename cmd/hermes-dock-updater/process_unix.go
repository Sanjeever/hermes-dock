//go:build !windows

package main

import (
	"errors"
	"os/exec"
	"syscall"
)

func processRunning(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}

func detachedCommand(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil
	return cmd
}

func backgroundCommand(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}
