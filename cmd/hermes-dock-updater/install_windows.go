//go:build windows

package main

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func installPackage(config updateConfig) (func() error, func(), error) {
	lowerName := strings.ToLower(config.assetName)
	if strings.HasSuffix(lowerName, "-installer.exe") {
		return installWindowsInstaller(config)
	}
	if strings.HasSuffix(lowerName, "-portable.zip") {
		return installWindowsPortable(config)
	}
	return nil, nil, errors.New("Windows 更新包格式不受支持")
}

func installWindowsInstaller(config updateConfig) (func() error, func(), error) {
	rollback, cleanup, err := backupExecutable(config)
	if err != nil {
		return nil, nil, err
	}
	if err := runCommand(config.packagePath, "/S"); err != nil {
		_ = rollback()
		cleanup()
		return nil, nil, err
	}
	return rollback, cleanup, nil
}

func installWindowsPortable(config updateConfig) (func() error, func(), error) {
	rollback, backupCleanup, err := backupExecutable(config)
	if err != nil {
		return nil, nil, err
	}
	extractDir := filepath.Join(config.instanceRoot, "launcher", "updates", "extract-"+config.token)
	if err := os.RemoveAll(extractDir); err != nil {
		backupCleanup()
		return nil, nil, err
	}
	if err := extractPortableZip(config.packagePath, extractDir); err != nil {
		backupCleanup()
		return nil, nil, err
	}
	for _, name := range []string{"hermes-dock.exe", "hermes-dock-updater.exe", "hermes-profile-runner-linux-amd64"} {
		source := filepath.Join(extractDir, name)
		if _, err := os.Stat(source); err != nil {
			_ = rollback()
			backupCleanup()
			_ = os.RemoveAll(extractDir)
			return nil, nil, fmt.Errorf("便携版更新包缺少 %s", name)
		}
		if err := copyFile(source, filepath.Join(filepath.Dir(config.targetPath), name), 0755); err != nil {
			_ = rollback()
			backupCleanup()
			_ = os.RemoveAll(extractDir)
			return nil, nil, err
		}
	}
	cleanup := func() {
		backupCleanup()
		_ = os.RemoveAll(extractDir)
	}
	return rollback, cleanup, nil
}

func extractPortableZip(packagePath string, target string) error {
	archive, err := zip.OpenReader(packagePath)
	if err != nil {
		return err
	}
	defer archive.Close()
	if err := os.MkdirAll(target, 0700); err != nil {
		return err
	}
	for _, file := range archive.File {
		name := filepath.Base(file.Name)
		if name == "." || name == "" || file.FileInfo().IsDir() {
			continue
		}
		if file.UncompressedSize64 > 512*1024*1024 {
			return errors.New("压缩包文件超过大小限制")
		}
		input, err := file.Open()
		if err != nil {
			return err
		}
		output, err := os.OpenFile(filepath.Join(target, name), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0700)
		if err != nil {
			input.Close()
			return err
		}
		_, copyErr := io.Copy(output, input)
		closeErr := output.Close()
		input.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}
