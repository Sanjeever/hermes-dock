package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const runtimeDependencyBundleVersion = "cp313-v1"

var runtimeDependencyMu sync.Mutex

func (a *App) runtimeDependencyBundlePath() string {
	return filepath.Join(a.hermesDockDir(), "runtime-deps", runtimeDependencyBundleVersion)
}

func (a *App) ensureRuntimeDependencies() error {
	runtimeDependencyMu.Lock()
	defer runtimeDependencyMu.Unlock()

	target := a.runtimeDependencyBundlePath()
	parent := filepath.Dir(target)
	if err := ensureDir(parent); err != nil {
		return err
	}
	if err := cleanupRuntimeDependencyStaging(parent); err != nil {
		return fmt.Errorf("清理中断的运行依赖释放目录失败：%w", err)
	}
	if _, err := os.Stat(target); err == nil {
		if err := verifyRuntimeDependencyDirectory(target, false); err != nil {
			return fmt.Errorf("内置运行依赖损坏：%w", err)
		}
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	staging, err := os.MkdirTemp(parent, ".runtime-deps-"+runtimeDependencyBundleVersion+"-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(staging)
	if err := extractRuntimeDependencies(staging); err != nil {
		return fmt.Errorf("释放内置运行依赖失败：%w", err)
	}
	if err := verifyRuntimeDependencyDirectory(staging, true); err != nil {
		return fmt.Errorf("校验内置运行依赖失败：%w", err)
	}
	if err := os.Rename(staging, target); err != nil {
		if _, statErr := os.Stat(target); statErr == nil {
			return verifyRuntimeDependencyDirectory(target, false)
		}
		return err
	}
	return nil
}

func cleanupRuntimeDependencyStaging(parent string) error {
	entries, err := os.ReadDir(parent)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), ".runtime-deps-") {
			continue
		}
		if err := os.RemoveAll(filepath.Join(parent, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}

func (a *App) cleanupObsoleteRuntimeDependencies() error {
	runtimeDependencyMu.Lock()
	defer runtimeDependencyMu.Unlock()

	parent := filepath.Dir(a.runtimeDependencyBundlePath())
	entries, err := os.ReadDir(parent)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == runtimeDependencyBundleVersion || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		path := filepath.Join(parent, entry.Name())
		if !runtimeDependencyBundleDirectory(path) {
			continue
		}
		if err := os.RemoveAll(path); err != nil {
			return err
		}
	}
	return nil
}

func runtimeDependencyBundleDirectory(path string) bool {
	for _, rel := range []string{"SHA256SUMS", "platform", "python-version"} {
		info, err := os.Stat(filepath.Join(path, rel))
		if err != nil || !info.Mode().IsRegular() {
			return false
		}
	}
	info, err := os.Stat(filepath.Join(path, "wheels"))
	return err == nil && info.IsDir()
}

func extractRuntimeDependencies(target string) error {
	return fs.WalkDir(runtimeDependencyFS, runtimeDependencySourceRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel := strings.TrimPrefix(strings.TrimPrefix(path, runtimeDependencySourceRoot), "/")
		if rel == "" {
			return nil
		}
		destination := filepath.Join(target, filepath.FromSlash(rel))
		if entry.IsDir() {
			return ensureDir(destination)
		}
		input, err := runtimeDependencyFS.Open(path)
		if err != nil {
			return err
		}
		if err := ensureDir(filepath.Dir(destination)); err != nil {
			input.Close()
			return err
		}
		output, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if err != nil {
			input.Close()
			return err
		}
		_, copyErr := io.Copy(output, input)
		inputCloseErr := input.Close()
		if copyErr == nil {
			copyErr = inputCloseErr
		}
		if copyErr == nil {
			copyErr = output.Sync()
		}
		closeErr := output.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	})
}

func verifyRuntimeDependencyDirectory(root string, verifyHashes bool) error {
	checksumPath := filepath.Join(root, "SHA256SUMS")
	trustedChecksums, err := runtimeDependencyFS.ReadFile(runtimeDependencySourceRoot + "/SHA256SUMS")
	if err != nil {
		return err
	}
	actualChecksums, err := os.ReadFile(checksumPath)
	if err != nil {
		return err
	}
	if !bytes.Equal(actualChecksums, trustedChecksums) {
		return fmt.Errorf("SHA256SUMS 与内置清单不一致")
	}
	checksumFile, err := os.Open(checksumPath)
	if err != nil {
		return err
	}
	defer checksumFile.Close()
	expected := map[string]string{}
	scanner := bufio.NewScanner(checksumFile)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) != 2 || len(fields[0]) != sha256.Size*2 {
			return fmt.Errorf("SHA256SUMS 格式无效")
		}
		rel := strings.TrimPrefix(filepath.ToSlash(fields[1]), "./")
		if !fs.ValidPath(rel) || rel == "SHA256SUMS" {
			return fmt.Errorf("SHA256SUMS 路径无效：%s", fields[1])
		}
		if expected[rel] != "" {
			return fmt.Errorf("SHA256SUMS 路径重复：%s", rel)
		}
		expected[rel] = strings.ToLower(fields[0])
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	if len(expected) == 0 {
		return fmt.Errorf("SHA256SUMS 为空")
	}
	for rel, want := range expected {
		path := filepath.Join(root, filepath.FromSlash(rel))
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("依赖文件类型无效：%s", rel)
		}
		if verifyHashes {
			got, err := runtimeDependencyFileSHA256(path)
			if err != nil {
				return err
			}
			if got != want {
				return fmt.Errorf("依赖文件校验失败：%s", rel)
			}
		}
	}
	return filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel != "SHA256SUMS" && expected[rel] == "" {
			return fmt.Errorf("存在未登记的依赖文件：%s", rel)
		}
		return nil
	})
}

func runtimeDependencyFileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
