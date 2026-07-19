//go:build windows

package main

import (
	"testing"

	"golang.org/x/sys/windows"
)

func TestDetachedCommandShowsApplicationWindow(t *testing.T) {
	cmd := detachedCommand(`C:\Program Files\Hermes Dock\hermes-dock.exe`)
	if cmd.SysProcAttr == nil {
		t.Fatal("detached command is missing Windows process attributes")
	}
	if cmd.SysProcAttr.HideWindow {
		t.Fatal("detached application window should be visible")
	}
	if cmd.SysProcAttr.CreationFlags&windows.CREATE_NEW_PROCESS_GROUP == 0 {
		t.Fatal("detached application should use a new process group")
	}
}

func TestBackgroundCommandHidesConsoleWindow(t *testing.T) {
	cmd := backgroundCommand("installer.exe")
	if cmd.SysProcAttr == nil {
		t.Fatal("background command is missing Windows process attributes")
	}
	if !cmd.SysProcAttr.HideWindow {
		t.Fatal("background command should hide its console window")
	}
	if cmd.SysProcAttr.CreationFlags&windows.CREATE_NO_WINDOW == 0 {
		t.Fatal("background command should not create a console window")
	}
}
