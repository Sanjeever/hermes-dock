//go:build darwin

package main

/*
#cgo LDFLAGS: -framework Cocoa
void hermesDockSetupTray(void);
void hermesDockRemoveTray(void);
*/
import "C"

import "github.com/wailsapp/wails/v2/pkg/runtime"

var trayApp *App

func (a *App) startTray() {
	trayApp = a
	C.hermesDockSetupTray()
}

func (a *App) stopTray() {
	C.hermesDockRemoveTray()
}

//export hermesDockTrayOpenApp
func hermesDockTrayOpenApp() {
	if trayApp != nil && trayApp.ctx != nil {
		runtime.Show(trayApp.ctx)
	}
}

//export hermesDockTrayOpenWeb
func hermesDockTrayOpenWeb() {
	if trayApp != nil {
		_ = trayApp.OpenWebManagement()
	}
}

//export hermesDockTrayCopyURL
func hermesDockTrayCopyURL() {
	if trayApp != nil && trayApp.ctx != nil {
		_ = runtime.ClipboardSetText(trayApp.ctx, trayApp.webStatus().PrimaryURL)
	}
}

//export hermesDockTrayQuitApp
func hermesDockTrayQuitApp() {
	if trayApp != nil && trayApp.ctx != nil {
		runtime.Quit(trayApp.ctx)
	}
}
