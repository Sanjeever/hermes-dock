package main

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	instanceBackupFormat        = "hermes-dock.instance-backup"
	instanceBackupSchemaVersion = 1
	instanceBackupConfirmPhrase = "导入"
	instanceBackupManifestName  = "manifest.json"
	instanceBackupChecksumsName = "checksums.txt"
)

var instanceBackupRestorableRoots = []string{
	"data",
	"launcher/state.json",
	"launcher/profiles.json",
	"launcher/profile-content",
	"launcher/web-server.json",
	"launcher/dufs/config.yaml",
	"docker-compose.override.yaml",
	"docker-compose.yaml",
}

type instanceBackupEntry struct {
	AbsPath string
	RelPath string
	Info    fs.FileInfo
}

func (a *App) ExportInstanceBackup(targetPath string) (InstanceBackupManifest, error) {
	release, err := a.beginExclusiveOperation("导出实例备份")
	if err != nil {
		return InstanceBackupManifest{}, err
	}
	defer release()
	targetPath = strings.TrimSpace(targetPath)
	if targetPath == "" {
		if a.ctx == nil {
			return InstanceBackupManifest{}, fmt.Errorf("导出路径不能为空")
		}
		path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
			Title:                "导出 Hermes Dock 备份",
			DefaultFilename:      defaultInstanceBackupFilename(),
			CanCreateDirectories: true,
			Filters: []runtime.FileFilter{{
				DisplayName: "Hermes Dock 备份 (*.hdbackup)",
				Pattern:     "*.hdbackup",
			}},
		})
		if err != nil {
			return InstanceBackupManifest{}, err
		}
		if strings.TrimSpace(path) == "" {
			return InstanceBackupManifest{}, fmt.Errorf("已取消导出")
		}
		targetPath = path
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.exportInstanceBackupWithStoppedContainer(targetPath)
}

func (a *App) InspectInstanceBackup(path string) (InstanceBackupManifest, error) {
	path, err := a.chooseInstanceBackupPath(path, "选择 Hermes Dock 备份")
	if err != nil {
		return InstanceBackupManifest{}, err
	}
	manifest, err := readInstanceBackupManifest(path)
	if err != nil {
		return InstanceBackupManifest{}, err
	}
	manifest.Path = path
	return manifest, nil
}

func (a *App) ImportInstanceBackup(req InstanceBackupImportRequest) (InstanceBackupImportResult, error) {
	release, err := a.beginExclusiveOperation("导入实例备份")
	if err != nil {
		return InstanceBackupImportResult{}, err
	}
	defer release()
	if strings.TrimSpace(req.Confirm) != instanceBackupConfirmPhrase {
		return InstanceBackupImportResult{}, fmt.Errorf("请输入“%s”确认导入", instanceBackupConfirmPhrase)
	}
	sourcePath, err := a.chooseInstanceBackupPath(req.Path, "导入 Hermes Dock 备份")
	if err != nil {
		return InstanceBackupImportResult{}, err
	}
	sourcePath, err = filepath.Abs(sourcePath)
	if err != nil {
		return InstanceBackupImportResult{}, err
	}

	stableSource, cleanupSource, err := copyBackupToTemp(sourcePath)
	if err != nil {
		return InstanceBackupImportResult{}, err
	}
	defer cleanupSource()

	extractRoot, err := os.MkdirTemp("", "hermes-dock-import-*")
	if err != nil {
		return InstanceBackupImportResult{}, err
	}
	defer os.RemoveAll(extractRoot)

	manifest, err := extractInstanceBackup(stableSource, extractRoot)
	if err != nil {
		return InstanceBackupImportResult{}, err
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if err := a.validateResetRoot(); err != nil {
		return InstanceBackupImportResult{}, err
	}
	a.StopTailLogs()
	if err := a.cancelLoginSessionAndWait(""); err != nil {
		return InstanceBackupImportResult{}, fmt.Errorf("停止扫码绑定失败：%w", err)
	}
	if fileExists(a.composePath()) {
		a.emit("backup:progress", StreamEvent{Line: "正在停止当前容器"})
		if err := a.runComposeBlocking(context.Background(), "down"); err != nil {
			return InstanceBackupImportResult{}, err
		}
	}

	preImportPath := filepath.Join(a.hermesDockDir(), "backups", "pre-import-"+time.Now().UTC().Format("20060102T150405Z")+".hdbackup")
	a.emit("backup:progress", StreamEvent{Line: "正在创建导入前备份"})
	if _, err := a.exportInstanceBackupTo(preImportPath); err != nil {
		return InstanceBackupImportResult{}, fmt.Errorf("创建导入前备份失败：%w", err)
	}

	a.emit("backup:progress", StreamEvent{Line: "正在恢复备份文件"})
	if err := a.replaceInstanceFromBackup(extractRoot); err != nil {
		return InstanceBackupImportResult{}, err
	}
	if err := a.ensureContainerInitHelpers(); err != nil {
		return InstanceBackupImportResult{}, err
	}
	if err := a.ensureDockDataDir(); err != nil {
		return InstanceBackupImportResult{}, err
	}
	if err := a.ensureProfileRegistry(); err != nil {
		return InstanceBackupImportResult{}, err
	}
	if err := a.ensureWebConfig(); err != nil {
		return InstanceBackupImportResult{}, err
	}
	settings := a.readComposeSettings()
	settings, err = a.ensureDufsConfig(settings, "", "before-dufs-config-import")
	if err != nil {
		return InstanceBackupImportResult{}, err
	}
	if err := atomicWriteFile(a.composePath(), []byte(renderCompose(settings, a.readProxySettings())), 0644); err != nil {
		return InstanceBackupImportResult{}, err
	}
	now := time.Now().UTC()
	state, err := a.readState()
	if err != nil {
		return InstanceBackupImportResult{}, err
	}
	state.AppVersion = appVersion
	state.ComposeSettings = settings
	state.ComposeHash = fileSHA256(a.composePath())
	state.NeedsRebuild = true
	state.PendingDufsOnly = false
	state.Backups = append(state.Backups, BackupRecord{
		ID:     now.Format("20060102T150405Z"),
		Reason: "pre-import-instance",
		Path:   strings.TrimPrefix(preImportPath, a.instanceRoot+string(os.PathSeparator)),
	})
	state.UpdatedAt = now.Format(time.RFC3339)
	if err := a.writeState(state); err != nil {
		return InstanceBackupImportResult{}, err
	}
	if err := a.clearWebSessions(); err != nil {
		return InstanceBackupImportResult{}, err
	}
	if err := a.syncHostBridge(settings.HostControlEnabled == "true"); err != nil {
		return InstanceBackupImportResult{}, err
	}
	go func() {
		time.Sleep(200 * time.Millisecond)
		a.stopWebServer(context.Background())
		a.startWebServer()
	}()

	manifest.Path = sourcePath
	return InstanceBackupImportResult{Manifest: manifest, PreImportBackupPath: preImportPath}, nil
}

func (a *App) chooseInstanceBackupPath(path string, title string) (string, error) {
	path = strings.TrimSpace(path)
	if path != "" {
		return path, nil
	}
	if a.ctx == nil {
		return "", fmt.Errorf("备份路径不能为空")
	}
	selected, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: title,
		Filters: []runtime.FileFilter{{
			DisplayName: "Hermes Dock 备份 (*.hdbackup)",
			Pattern:     "*.hdbackup",
		}},
	})
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(selected) == "" {
		return "", fmt.Errorf("已取消选择备份")
	}
	return selected, nil
}

func (a *App) exportInstanceBackupTo(targetPath string) (InstanceBackupManifest, error) {
	targetPath = strings.TrimSpace(targetPath)
	if targetPath == "" {
		return InstanceBackupManifest{}, fmt.Errorf("导出路径不能为空")
	}
	if filepath.Ext(targetPath) == "" {
		targetPath += ".hdbackup"
	}
	if err := ensureDir(filepath.Dir(targetPath)); err != nil {
		return InstanceBackupManifest{}, err
	}
	entries, excluded, err := a.collectInstanceBackupEntries()
	if err != nil {
		return InstanceBackupManifest{}, err
	}
	manifest := a.buildInstanceBackupManifest(entries, excluded)
	manifest.Path = targetPath

	tmp, err := os.CreateTemp(filepath.Dir(targetPath), "."+filepath.Base(targetPath)+".tmp-*")
	if err != nil {
		return InstanceBackupManifest{}, err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	defer tmp.Close()

	gz, err := gzip.NewWriterLevel(tmp, gzip.BestSpeed)
	if err != nil {
		return InstanceBackupManifest{}, err
	}
	tw := tar.NewWriter(gz)
	checksums := map[string]string{}
	if err := writeBackupJSON(tw, instanceBackupManifestName, manifest); err != nil {
		_ = tw.Close()
		_ = gz.Close()
		return InstanceBackupManifest{}, err
	}
	for _, entry := range entries {
		if err := writeBackupEntry(tw, entry, checksums); err != nil {
			_ = tw.Close()
			_ = gz.Close()
			return InstanceBackupManifest{}, err
		}
	}
	if err := writeBackupChecksums(tw, checksums); err != nil {
		_ = tw.Close()
		_ = gz.Close()
		return InstanceBackupManifest{}, err
	}
	if err := tw.Close(); err != nil {
		_ = gz.Close()
		return InstanceBackupManifest{}, err
	}
	if err := gz.Close(); err != nil {
		return InstanceBackupManifest{}, err
	}
	if err := tmp.Close(); err != nil {
		return InstanceBackupManifest{}, err
	}
	_ = os.Remove(targetPath)
	if err := os.Rename(tmpPath, targetPath); err != nil {
		return InstanceBackupManifest{}, err
	}
	return manifest, nil
}

func (a *App) exportInstanceBackupWithStoppedContainer(targetPath string) (manifest InstanceBackupManifest, err error) {
	wasRunning := fileExists(a.composePath()) && a.containerStatus(context.Background()) == "running"
	if !wasRunning {
		return a.exportInstanceBackupTo(targetPath)
	}
	a.emit("backup:progress", StreamEvent{Line: "正在临时停止容器以创建一致备份"})
	if err := a.runComposeBlocking(context.Background(), "stop"); err != nil {
		return InstanceBackupManifest{}, err
	}
	defer func() {
		a.emit("backup:progress", StreamEvent{Line: "正在恢复导出前的容器运行状态"})
		if startErr := a.runComposeBlocking(context.Background(), "start"); startErr != nil && err == nil {
			err = startErr
		}
	}()
	return a.exportInstanceBackupTo(targetPath)
}

func (a *App) collectInstanceBackupEntries() ([]instanceBackupEntry, []string, error) {
	var entries []instanceBackupEntry
	excludedSet := map[string]bool{}
	addExcluded := func(path string) {
		path = filepath.ToSlash(path)
		if path != "" {
			excludedSet[path] = true
		}
	}
	for _, relRoot := range instanceBackupRestorableRoots {
		absRoot := filepath.Join(a.instanceRoot, filepath.FromSlash(relRoot))
		info, err := os.Lstat(absRoot)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, nil, err
		}
		if !info.IsDir() {
			if info.Mode().Type() != 0 {
				addExcluded(relRoot)
				continue
			}
			entries = append(entries, instanceBackupEntry{AbsPath: absRoot, RelPath: relRoot, Info: info})
			continue
		}
		err = filepath.WalkDir(absRoot, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(a.instanceRoot, path)
			if err != nil {
				return err
			}
			rel = filepath.ToSlash(rel)
			if rel == relRoot {
				return nil
			}
			if shouldExcludeInstanceBackupPath(rel) {
				addExcluded(rel)
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			info, err := d.Info()
			if err != nil {
				return err
			}
			if info.Mode().Type() != 0 && !info.IsDir() {
				addExcluded(rel)
				return nil
			}
			entries = append(entries, instanceBackupEntry{AbsPath: path, RelPath: rel, Info: info})
			return nil
		})
		if err != nil {
			return nil, nil, err
		}
	}
	for _, rel := range []string{"launcher/backups", "launcher/logs", "launcher/web-sessions.json", "launcher/apply-status.json", "data/.dock", "shared"} {
		if fileExists(filepath.Join(a.instanceRoot, filepath.FromSlash(rel))) {
			addExcluded(rel)
		}
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].RelPath < entries[j].RelPath })
	var excluded []string
	for path := range excludedSet {
		excluded = append(excluded, path)
	}
	sort.Strings(excluded)
	return entries, excluded, nil
}

func shouldExcludeInstanceBackupPath(rel string) bool {
	rel = filepath.ToSlash(filepath.Clean(rel))
	if rel == "." {
		return false
	}
	if rel == "launcher/backups" ||
		strings.HasPrefix(rel, "launcher/backups/") ||
		rel == "launcher/logs" ||
		strings.HasPrefix(rel, "launcher/logs/") ||
		rel == "launcher/web-sessions.json" ||
		rel == "launcher/apply-status.json" ||
		rel == "data/.dock" ||
		strings.HasPrefix(rel, "data/.dock/") {
		return true
	}
	if !strings.HasPrefix(rel, "data/") {
		return false
	}

	parts := strings.Split(strings.TrimPrefix(rel, "data/"), "/")
	for _, part := range parts {
		switch part {
		case "node_modules", ".venv", "venv", "__pycache__", ".pytest_cache", ".mypy_cache", ".ruff_cache":
			return true
		}
	}
	if parts[len(parts)-1] == ".DS_Store" {
		return true
	}

	switch parts[0] {
	case "tmp", ".cache", ".npm", "npm-global", "lsp", "logs", "cache", "audio_cache", "image_cache", "bin", "checkpoints":
		return true
	case "home":
		if len(parts) >= 2 {
			switch parts[1] {
			case ".cache", ".npm", ".local", ".paddlex":
				return true
			}
		}
	case "profiles":
		if len(parts) >= 3 {
			switch parts[2] {
			case "tmp", "cache", "audio_cache", "image_cache", "logs", "bin", "checkpoints":
				return true
			}
			if len(parts) == 3 && isInstanceBackupRuntimeFile(parts[2]) {
				return true
			}
		}
	}
	return len(parts) == 1 && isInstanceBackupRuntimeFile(parts[0])
}

func isInstanceBackupRuntimeFile(name string) bool {
	switch name {
	case "models_dev_cache.json", "provider_models_cache.json", "ollama_cloud_models_cache.json", "gateway_state.json", "gateway.pid", "gateway.lock", "auth.lock", "kanban.db.init.lock", ".scratch_tip_shown":
		return true
	default:
		return false
	}
}

func (a *App) buildInstanceBackupManifest(entries []instanceBackupEntry, excluded []string) InstanceBackupManifest {
	registry, _ := a.readProfileRegistry()
	profiles := make([]InstanceBackupProfile, 0, len(registry.Profiles))
	for _, profile := range registry.Profiles {
		profiles = append(profiles, InstanceBackupProfile{
			ID:        profile.ID,
			Name:      profile.Name,
			Enabled:   profile.Enabled,
			IsDefault: profile.ID == "default",
		})
	}
	var fileCount int
	var totalBytes int64
	for _, entry := range entries {
		if entry.Info.Mode().IsRegular() {
			fileCount++
			totalBytes += entry.Info.Size()
		}
	}
	return InstanceBackupManifest{
		Format:              instanceBackupFormat,
		SchemaVersion:       instanceBackupSchemaVersion,
		AppVersion:          appVersion,
		TemplateVersion:     templateVersion,
		CreatedAt:           time.Now().UTC().Format(time.RFC3339),
		SourceInstanceRoot:  a.instanceRoot,
		IncludesSecrets:     true,
		IncludesWebSettings: fileExists(a.webServerPath()),
		Profiles:            profiles,
		FileCount:           fileCount,
		TotalBytes:          totalBytes,
		ExcludedPaths:       excluded,
	}
}

func writeBackupJSON(tw *tar.Writer, name string, value interface{}) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	header := &tar.Header{Name: name, Mode: 0600, Size: int64(len(data)), ModTime: time.Now()}
	if err := tw.WriteHeader(header); err != nil {
		return err
	}
	_, err = tw.Write(data)
	return err
}

func writeBackupEntry(tw *tar.Writer, entry instanceBackupEntry, checksums map[string]string) error {
	info := entry.Info
	var src *os.File
	if info.Mode().IsRegular() {
		var err error
		src, err = os.Open(entry.AbsPath)
		if err != nil {
			return err
		}
		defer src.Close()
		info, err = src.Stat()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("备份文件类型已变化：%s", entry.RelPath)
		}
	}
	header, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return err
	}
	header.Name = entry.RelPath
	if info.IsDir() {
		header.Name = strings.TrimSuffix(header.Name, "/") + "/"
	}
	if err := tw.WriteHeader(header); err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return nil
	}
	sum := sha256.New()
	if _, err := io.CopyN(tw, io.TeeReader(src, sum), info.Size()); err != nil {
		return err
	}
	checksums[entry.RelPath] = hex.EncodeToString(sum.Sum(nil))
	return nil
}

func writeBackupChecksums(tw *tar.Writer, checksums map[string]string) error {
	paths := make([]string, 0, len(checksums))
	for path := range checksums {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	var b strings.Builder
	for _, path := range paths {
		b.WriteString(checksums[path])
		b.WriteString("  ")
		b.WriteString(path)
		b.WriteByte('\n')
	}
	data := []byte(b.String())
	header := &tar.Header{Name: instanceBackupChecksumsName, Mode: 0600, Size: int64(len(data)), ModTime: time.Now()}
	if err := tw.WriteHeader(header); err != nil {
		return err
	}
	_, err := tw.Write(data)
	return err
}

func readInstanceBackupManifest(path string) (InstanceBackupManifest, error) {
	file, err := os.Open(path)
	if err != nil {
		return InstanceBackupManifest{}, err
	}
	defer file.Close()
	gz, err := gzip.NewReader(file)
	if err != nil {
		return InstanceBackupManifest{}, err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return InstanceBackupManifest{}, err
		}
		if header.Name != instanceBackupManifestName {
			continue
		}
		var manifest InstanceBackupManifest
		if err := json.NewDecoder(tr).Decode(&manifest); err != nil {
			return InstanceBackupManifest{}, err
		}
		if err := validateInstanceBackupManifest(manifest); err != nil {
			return InstanceBackupManifest{}, err
		}
		return manifest, nil
	}
	return InstanceBackupManifest{}, fmt.Errorf("备份缺少 manifest.json")
}

func extractInstanceBackup(path string, targetRoot string) (InstanceBackupManifest, error) {
	file, err := os.Open(path)
	if err != nil {
		return InstanceBackupManifest{}, err
	}
	defer file.Close()
	gz, err := gzip.NewReader(file)
	if err != nil {
		return InstanceBackupManifest{}, err
	}
	defer gz.Close()

	var manifest InstanceBackupManifest
	var checksumText string
	computed := map[string]string{}
	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return InstanceBackupManifest{}, err
		}
		name, err := cleanBackupArchiveName(header.Name)
		if err != nil {
			return InstanceBackupManifest{}, err
		}
		switch name {
		case instanceBackupManifestName:
			if err := json.NewDecoder(tr).Decode(&manifest); err != nil {
				return InstanceBackupManifest{}, err
			}
			continue
		case instanceBackupChecksumsName:
			data, err := io.ReadAll(tr)
			if err != nil {
				return InstanceBackupManifest{}, err
			}
			checksumText = string(data)
			continue
		}
		if err := validateRestorableBackupPath(name); err != nil {
			return InstanceBackupManifest{}, err
		}
		dst := filepath.Join(targetRoot, filepath.FromSlash(name))
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(dst, os.FileMode(header.Mode)&0755); err != nil {
				return InstanceBackupManifest{}, err
			}
		case tar.TypeReg, tar.TypeRegA:
			if name == "data" {
				return InstanceBackupManifest{}, fmt.Errorf("备份中的 data 必须是目录")
			}
			if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
				return InstanceBackupManifest{}, err
			}
			sum := sha256.New()
			if err := writeExtractedFile(dst, tr, sum, os.FileMode(header.Mode)); err != nil {
				return InstanceBackupManifest{}, err
			}
			computed[name] = hex.EncodeToString(sum.Sum(nil))
		default:
			return InstanceBackupManifest{}, fmt.Errorf("备份包含不支持的文件类型：%s", name)
		}
	}
	if err := validateInstanceBackupManifest(manifest); err != nil {
		return InstanceBackupManifest{}, err
	}
	expected, err := parseBackupChecksums(checksumText)
	if err != nil {
		return InstanceBackupManifest{}, err
	}
	if err := verifyBackupChecksums(expected, computed); err != nil {
		return InstanceBackupManifest{}, err
	}
	return manifest, nil
}

func cleanBackupArchiveName(name string) (string, error) {
	name = strings.ReplaceAll(name, "\\", "/")
	name = strings.TrimPrefix(name, "./")
	clean := filepath.ToSlash(filepath.Clean(name))
	if clean == "." || strings.HasPrefix(clean, "../") || strings.HasPrefix(clean, "/") || strings.Contains(clean, "/../") {
		return "", fmt.Errorf("备份包含不安全路径：%s", name)
	}
	return clean, nil
}

func validateRestorableBackupPath(path string) error {
	if path == "data" || strings.HasPrefix(path, "data/") {
		if path == "data/.dock" || strings.HasPrefix(path, "data/.dock/") {
			return fmt.Errorf("备份包含运行态路径：%s", path)
		}
		return nil
	}
	if path == "launcher/profile-content" || strings.HasPrefix(path, "launcher/profile-content/") {
		return nil
	}
	for _, root := range instanceBackupRestorableRoots {
		if path == root {
			return nil
		}
	}
	return fmt.Errorf("备份包含不允许恢复的路径：%s", path)
}

func validateInstanceBackupManifest(manifest InstanceBackupManifest) error {
	if manifest.Format != instanceBackupFormat {
		return fmt.Errorf("不是 Hermes Dock 实例备份")
	}
	if manifest.SchemaVersion != instanceBackupSchemaVersion {
		return fmt.Errorf("不支持的备份版本：%d", manifest.SchemaVersion)
	}
	return nil
}

func writeExtractedFile(path string, src io.Reader, sum hash.Hash, mode os.FileMode) error {
	perm := mode.Perm()
	if perm == 0 {
		perm = 0600
	}
	dst, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	defer dst.Close()
	_, err = io.Copy(dst, io.TeeReader(src, sum))
	return err
}

func parseBackupChecksums(text string) (map[string]string, error) {
	if strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("备份缺少校验和")
	}
	out := map[string]string{}
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "  ", 2)
		if len(parts) != 2 || len(parts[0]) != sha256.Size*2 {
			return nil, fmt.Errorf("备份校验和格式错误")
		}
		path, err := cleanBackupArchiveName(parts[1])
		if err != nil {
			return nil, err
		}
		out[path] = parts[0]
	}
	return out, nil
}

func verifyBackupChecksums(expected map[string]string, computed map[string]string) error {
	if len(expected) != len(computed) {
		return fmt.Errorf("备份文件数量校验失败")
	}
	for path, want := range expected {
		got := computed[path]
		if got == "" {
			return fmt.Errorf("备份缺少文件：%s", path)
		}
		if !strings.EqualFold(got, want) {
			return fmt.Errorf("备份文件校验失败：%s", path)
		}
	}
	return nil
}

func copyBackupToTemp(sourcePath string) (string, func(), error) {
	src, err := os.Open(sourcePath)
	if err != nil {
		return "", func() {}, err
	}
	defer src.Close()
	tmp, err := os.CreateTemp("", "hermes-dock-source-*.hdbackup")
	if err != nil {
		return "", func() {}, err
	}
	tmpPath := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpPath) }
	if _, err := io.Copy(tmp, src); err != nil {
		_ = tmp.Close()
		cleanup()
		return "", func() {}, err
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return "", func() {}, err
	}
	return tmpPath, cleanup, nil
}

func (a *App) replaceInstanceFromBackup(extractRoot string) error {
	for _, path := range []string{
		a.dataDir(),
		a.composePath(),
		a.overridePath(),
		a.statePath(),
		a.profilesPath(),
		filepath.Dir(a.bundledContentStatePath(defaultProfileID)),
		a.applyStatusPath(),
		a.webServerPath(),
		a.dufsConfigPath(),
		a.webSessionsPath(),
	} {
		if err := os.RemoveAll(path); err != nil {
			return err
		}
	}
	for _, rel := range instanceBackupRestorableRoots {
		src := filepath.Join(extractRoot, filepath.FromSlash(rel))
		if !fileExists(src) {
			continue
		}
		dst := filepath.Join(a.instanceRoot, filepath.FromSlash(rel))
		if err := copyRestoredPath(src, dst); err != nil {
			return err
		}
	}
	return nil
}

func copyRestoredPath(src string, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(src, path)
			if err != nil {
				return err
			}
			target := filepath.Join(dst, rel)
			info, err := d.Info()
			if err != nil {
				return err
			}
			if info.IsDir() {
				return os.MkdirAll(target, info.Mode()&0755)
			}
			return copyRestoredFile(path, target, info.Mode())
		})
	}
	return copyRestoredFile(src, dst, info.Mode())
}

func copyRestoredFile(src string, dst string, mode os.FileMode) error {
	if err := ensureDir(filepath.Dir(dst)); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func defaultInstanceBackupFilename() string {
	return "hermes-dock-backup-" + time.Now().Format("20060102-150405") + ".hdbackup"
}
