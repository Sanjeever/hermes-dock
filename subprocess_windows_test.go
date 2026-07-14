//go:build windows

package main

import (
	"testing"

	"golang.org/x/sys/windows"
)

func TestBackgroundCommandHidesWindowsConsole(t *testing.T) {
	cmd := backgroundCommand("cmd.exe", "/c", "exit", "0")
	if cmd.SysProcAttr == nil || !cmd.SysProcAttr.HideWindow {
		t.Fatal("background command does not hide its Windows console")
	}
	if cmd.SysProcAttr.CreationFlags&windows.CREATE_NO_WINDOW == 0 {
		t.Fatal("background command does not use CREATE_NO_WINDOW")
	}
}
