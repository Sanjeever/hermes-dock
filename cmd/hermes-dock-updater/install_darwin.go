//go:build darwin

package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func installPackage(config updateConfig) (func() error, func(), error) {
	if !strings.HasSuffix(strings.ToLower(config.assetName), ".zip") {
		return nil, nil, errors.New("macOS 自动更新只支持 ZIP 应用包")
	}
	macOSDir := filepath.Dir(config.targetPath)
	contentsDir := filepath.Dir(macOSDir)
	appPath := filepath.Dir(contentsDir)
	if filepath.Ext(appPath) != ".app" {
		return nil, nil, errors.New("无法定位当前 macOS 应用包")
	}
	extractDir := filepath.Join(config.instanceRoot, "launcher", "updates", "extract-"+config.token)
	if err := os.RemoveAll(extractDir); err != nil {
		return nil, nil, err
	}
	if err := os.MkdirAll(extractDir, 0700); err != nil {
		return nil, nil, err
	}
	if err := runCommand("/usr/bin/ditto", "-x", "-k", config.packagePath, extractDir); err != nil {
		return nil, nil, err
	}
	items, err := filepath.Glob(filepath.Join(extractDir, "*.app"))
	if err != nil || len(items) != 1 {
		return nil, nil, errors.New("更新包中没有唯一的 macOS 应用")
	}
	backupPath := appPath + ".rollback-" + config.token
	if err := os.RemoveAll(backupPath); err != nil {
		return nil, nil, err
	}
	if err := os.Rename(appPath, backupPath); err != nil {
		return nil, nil, fmt.Errorf("备份当前应用失败：%w", err)
	}
	if err := os.Rename(items[0], appPath); err != nil {
		_ = os.Rename(backupPath, appPath)
		return nil, nil, fmt.Errorf("替换应用失败：%w", err)
	}
	rollback := func() error {
		if err := os.RemoveAll(appPath); err != nil {
			return err
		}
		return os.Rename(backupPath, appPath)
	}
	cleanup := func() {
		_ = os.RemoveAll(backupPath)
		_ = os.RemoveAll(extractDir)
	}
	return rollback, cleanup, nil
}
