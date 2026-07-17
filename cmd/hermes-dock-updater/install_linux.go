//go:build linux

package main

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func installPackage(config updateConfig) (func() error, func(), error) {
	if strings.HasSuffix(config.assetName, ".deb") {
		return installLinuxDeb(config)
	}
	if strings.HasSuffix(config.assetName, ".tar.gz") {
		return installLinuxArchive(config)
	}
	return nil, nil, errors.New("Linux 更新包格式不受支持")
}

func installLinuxDeb(config updateConfig) (func() error, func(), error) {
	rollback, cleanup, err := backupExecutable(config)
	if err != nil {
		return nil, nil, err
	}
	if err := runCommand("dpkg", "-i", config.packagePath); err != nil {
		_ = rollback()
		cleanup()
		return nil, nil, err
	}
	return rollback, cleanup, nil
}

func installLinuxArchive(config updateConfig) (func() error, func(), error) {
	rollback, backupCleanup, err := backupExecutable(config)
	if err != nil {
		return nil, nil, err
	}
	extractDir := filepath.Join(config.instanceRoot, "launcher", "updates", "extract-"+config.token)
	if err := os.RemoveAll(extractDir); err != nil {
		backupCleanup()
		return nil, nil, err
	}
	if err := extractLinuxArchive(config.packagePath, extractDir); err != nil {
		backupCleanup()
		return nil, nil, err
	}
	for _, name := range []string{"hermes-dock", "hermes-dock-updater", "hermes-profile-runner"} {
		source := filepath.Join(extractDir, name)
		if _, err := os.Stat(source); err != nil {
			_ = rollback()
			backupCleanup()
			_ = os.RemoveAll(extractDir)
			return nil, nil, fmt.Errorf("压缩包更新缺少 %s", name)
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

func extractLinuxArchive(packagePath string, target string) error {
	file, err := os.Open(packagePath)
	if err != nil {
		return err
	}
	defer file.Close()
	compressed, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer compressed.Close()
	if err := os.MkdirAll(target, 0700); err != nil {
		return err
	}
	archive := tar.NewReader(compressed)
	for {
		header, err := archive.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		if header.Typeflag != tar.TypeReg {
			continue
		}
		if header.Size > 512*1024*1024 {
			return errors.New("压缩包文件超过大小限制")
		}
		name := filepath.Base(header.Name)
		if name == "." || name == "" {
			continue
		}
		output, err := os.OpenFile(filepath.Join(target, name), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0700)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(output, archive)
		closeErr := output.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
}
