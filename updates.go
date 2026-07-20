package main

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	updateRepoSlug = "sqyl2026/hermes-dock-releases"
	updateCheckURL = "https://api.github.com/repos/" + updateRepoSlug + "/releases/latest"
	updateRepoURL  = "https://github.com/" + updateRepoSlug
	updateCooldown = 24 * time.Hour
	maxUpdateSize  = int64(1024 * 1024 * 1024)
)

var (
	scheduledUpdateMode   bool
	instanceRootOverride  string
	installedUpdateToken  string
	launchAfterUpdateMode bool
)

var updateCheckURLs = []string{
	updateCheckURL,
	"https://gh-proxy.com/" + updateCheckURL,
	"https://ghfast.top/" + updateCheckURL,
}

var updateMirrorPrefixes = []UpdateMirrorLink{
	{Label: "gh-proxy", URL: "https://gh-proxy.com/"},
	{Label: "ghfast", URL: "https://ghfast.top/"},
}

type updateState struct {
	SchemaVersion                 int    `json:"schemaVersion"`
	LastCheckedAt                 string `json:"lastCheckedAt"`
	LatestVersion                 string `json:"latestVersion"`
	ReleaseURL                    string `json:"releaseUrl"`
	AssetURL                      string `json:"assetUrl"`
	AssetName                     string `json:"assetName"`
	AssetSize                     int64  `json:"assetSize"`
	ChecksumURL                   string `json:"checksumUrl"`
	DismissedVersion              string `json:"dismissedVersion"`
	AutoUpdateEnabled             bool   `json:"autoUpdateEnabled"`
	LastError                     string `json:"lastError"`
	PostUpdateVersion             string `json:"postUpdateVersion"`
	PostUpdateTemplateVersion     string `json:"postUpdateTemplateVersion"`
	PostUpdateState               string `json:"postUpdateState"`
	PostUpdateMessage             string `json:"postUpdateMessage"`
	PostUpdateError               string `json:"postUpdateError"`
	PostUpdateUpdatedAt           string `json:"postUpdateUpdatedAt"`
	PostUpdateContainerWasRunning bool   `json:"postUpdateContainerWasRunning"`
	PostUpdateApplyID             string `json:"postUpdateApplyId"`
	PostUpdateSyncFailures        int    `json:"postUpdateSyncFailures"`
	PostUpdateContentChanged      bool   `json:"postUpdateContentChanged"`
	PostUpdateWritePending        bool   `json:"postUpdateWritePending"`
}

type updateRequest struct {
	Version     string `json:"version"`
	PackagePath string `json:"packagePath"`
	AssetName   string `json:"assetName"`
	SHA256      string `json:"sha256"`
	TargetPath  string `json:"targetPath"`
	Token       string `json:"token"`
	Relaunch    bool   `json:"relaunch"`
}

type githubReleaseResponse struct {
	TagName    string               `json:"tag_name"`
	HTMLURL    string               `json:"html_url"`
	Prerelease bool                 `json:"prerelease"`
	Draft      bool                 `json:"draft"`
	Assets     []githubReleaseAsset `json:"assets"`
}

type githubReleaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

func (a *App) CheckForUpdates(force bool) (UpdateInfo, error) {
	a.updateMu.Lock()
	defer a.updateMu.Unlock()
	return a.checkForUpdates(force, true)
}

func (a *App) checkForUpdates(force bool, persist bool) (UpdateInfo, error) {
	state, err := a.readUpdateState()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return UpdateInfo{}, fmt.Errorf("读取更新状态失败：%w", err)
	}
	if !force && state.LastCheckedAt != "" {
		checkedAt, err := time.Parse(time.RFC3339, state.LastCheckedAt)
		if err == nil && time.Since(checkedAt) < updateCooldown && state.LatestVersion != "" {
			return a.cachedUpdateInfo(state), nil
		}
	}

	release, err := fetchLatestRelease()
	if err != nil {
		return UpdateInfo{}, err
	}
	if release.Draft || release.Prerelease {
		return UpdateInfo{}, errors.New("最新发布不是稳定版本")
	}
	latest := normalizeVersion(release.TagName)
	if latest == "" {
		return UpdateInfo{}, errors.New("最新发布缺少版本号")
	}

	asset := selectReleaseAsset(release.TagName, release.Assets)
	checksum := findReleaseAsset("SHA256SUMS.txt", release.Assets)
	checkedAt := time.Now().UTC().Format(time.RFC3339)
	state.SchemaVersion = 1
	state.LastCheckedAt = checkedAt
	state.LatestVersion = latest
	state.ReleaseURL = firstNonEmpty(release.HTMLURL, updateRepoURL+"/releases/tag/"+release.TagName)
	state.AssetURL = asset.BrowserDownloadURL
	state.AssetName = asset.Name
	state.AssetSize = asset.Size
	state.ChecksumURL = checksum.BrowserDownloadURL
	if persist {
		if err := a.writeUpdateState(state); err != nil {
			return UpdateInfo{}, err
		}
	}

	info := UpdateInfo{
		CurrentVersion: appVersion,
		LatestVersion:  latest,
		Available:      compareVersions(latest, appVersion) > 0,
		Dismissed:      state.DismissedVersion == latest,
		ReleaseURL:     state.ReleaseURL,
		AssetURL:       asset.BrowserDownloadURL,
		AssetName:      asset.Name,
		AssetSize:      asset.Size,
		ChecksumURL:    checksum.BrowserDownloadURL,
		Mirrors:        mirrorLinks(asset.BrowserDownloadURL),
		CheckedAt:      checkedAt,
	}
	return info, nil
}

func (a *App) DismissUpdate(version string) error {
	a.updateMu.Lock()
	defer a.updateMu.Unlock()
	version = normalizeVersion(version)
	if version == "" {
		return errors.New("版本号不能为空")
	}
	state, err := a.readUpdateState()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("读取更新状态失败：%w", err)
	}
	state.SchemaVersion = 1
	state.DismissedVersion = version
	return a.writeUpdateState(state)
}

func (a *App) OpenUpdateURL(raw string) error {
	if !isAllowedUpdateURL(raw) {
		return errors.New("不允许打开该更新链接")
	}
	wailsRuntime.BrowserOpenURL(a.ctx, raw)
	return nil
}

func fetchLatestRelease() (githubReleaseResponse, error) {
	var lastErr error
	for _, endpoint := range updateCheckURLs {
		release, err := fetchLatestReleaseFrom(endpoint)
		if err == nil {
			return release, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = errors.New("没有可用的更新检查地址")
	}
	return githubReleaseResponse{}, lastErr
}

func fetchLatestReleaseFrom(endpoint string) (githubReleaseResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return githubReleaseResponse{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "hermes-dock/"+appVersion)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return githubReleaseResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return githubReleaseResponse{}, fmt.Errorf("检查更新失败：%s 返回 %s", endpoint, resp.Status)
	}
	var release githubReleaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return githubReleaseResponse{}, err
	}
	return release, nil
}

func (a *App) cachedUpdateInfo(state updateState) UpdateInfo {
	latest := normalizeVersion(state.LatestVersion)
	return UpdateInfo{
		CurrentVersion: appVersion,
		LatestVersion:  latest,
		Available:      compareVersions(latest, appVersion) > 0,
		Dismissed:      state.DismissedVersion == latest,
		ReleaseURL:     firstNonEmpty(state.ReleaseURL, updateRepoURL+"/releases/tag/v"+latest),
		AssetURL:       state.AssetURL,
		AssetName:      state.AssetName,
		AssetSize:      state.AssetSize,
		ChecksumURL:    state.ChecksumURL,
		Mirrors:        mirrorLinks(state.AssetURL),
		CheckedAt:      state.LastCheckedAt,
	}
}

func (a *App) readUpdateState() (updateState, error) {
	var state updateState
	data, err := os.ReadFile(a.updateStatePath())
	if err != nil {
		return updateState{SchemaVersion: 1}, err
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return updateState{SchemaVersion: 1}, err
	}
	if state.SchemaVersion == 0 {
		state.SchemaVersion = 1
	}
	return state, nil
}

func (a *App) writeUpdateState(state updateState) error {
	if err := ensureDir(filepath.Dir(a.updateStatePath())); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return atomicWriteFile(a.updateStatePath(), append(data, '\n'), 0644)
}

func selectReleaseAsset(tagName string, assets []githubReleaseAsset) githubReleaseAsset {
	for _, expected := range expectedReleaseAssetNames(tagName) {
		for _, asset := range assets {
			if asset.Name == expected {
				return asset
			}
		}
	}
	return githubReleaseAsset{}
}

func findReleaseAsset(name string, assets []githubReleaseAsset) githubReleaseAsset {
	for _, asset := range assets {
		if asset.Name == name {
			return asset
		}
	}
	return githubReleaseAsset{}
}

func expectedReleaseAssetNames(tagName string) []string {
	prefix := "hermes-dock-" + tagName
	switch runtime.GOOS + "/" + runtime.GOARCH {
	case "linux/amd64":
		if executable, err := os.Executable(); err == nil && filepath.Clean(executable) != "/opt/hermes-dock/hermes-dock" {
			return []string{prefix + "-linux-amd64.tar.gz", prefix + "-linux-amd64.deb"}
		}
		return []string{prefix + "-linux-amd64.deb", prefix + "-linux-amd64.tar.gz"}
	case "windows/amd64":
		if executable, err := os.Executable(); err == nil && !fileExists(filepath.Join(filepath.Dir(executable), "uninstall.exe")) {
			return []string{prefix + "-windows-amd64-portable.zip", prefix + "-windows-amd64-installer.exe"}
		}
		return []string{prefix + "-windows-amd64-installer.exe", prefix + "-windows-amd64-portable.zip"}
	case "darwin/arm64":
		return []string{prefix + "-darwin-arm64.zip"}
	default:
		return nil
	}
}

func mirrorLinks(downloadURL string) []UpdateMirrorLink {
	if downloadURL == "" {
		return nil
	}
	links := make([]UpdateMirrorLink, 0, len(updateMirrorPrefixes))
	for _, mirror := range updateMirrorPrefixes {
		links = append(links, UpdateMirrorLink{
			Label: mirror.Label,
			URL:   mirror.URL + downloadURL,
		})
	}
	return links
}

func isAllowedUpdateURL(raw string) bool {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "https" {
		return false
	}
	if parsed.Host == "github.com" {
		releasePath := "/" + updateRepoSlug + "/releases"
		return parsed.Path == releasePath || strings.HasPrefix(parsed.Path, releasePath+"/")
	}
	if parsed.Host == "gh-proxy.com" || parsed.Host == "ghfast.top" {
		return strings.Contains(raw, updateRepoURL+"/releases/download/")
	}
	return false
}

func normalizeVersion(version string) string {
	version = strings.TrimSpace(version)
	version = strings.TrimPrefix(version, "v")
	if index := strings.IndexAny(version, "-+"); index >= 0 {
		version = version[:index]
	}
	return version
}

func compareVersions(left string, right string) int {
	leftParts := versionParts(left)
	rightParts := versionParts(right)
	for i := 0; i < 3; i++ {
		if leftParts[i] > rightParts[i] {
			return 1
		}
		if leftParts[i] < rightParts[i] {
			return -1
		}
	}
	return 0
}

func versionParts(version string) [3]int {
	var parts [3]int
	fields := strings.Split(normalizeVersion(version), ".")
	for i := 0; i < len(fields) && i < 3; i++ {
		value, err := strconv.Atoi(fields[i])
		if err == nil {
			parts[i] = value
		}
	}
	return parts
}

func (a *App) SetAutoUpdateEnabled(enabled bool) (UpdateStatus, error) {
	a.updateMu.Lock()
	defer a.updateMu.Unlock()
	if !a.mu.TryLock() {
		return a.updateStatus(), errors.New("启动器正在执行其他操作，请稍后再修改自动更新设置")
	}
	defer a.mu.Unlock()

	state, err := a.readUpdateState()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return a.updateStatus(), fmt.Errorf("读取更新状态失败：%w", err)
	}
	if enabled {
		if err := a.registerUpdateTask(); err != nil {
			state.LastError = err.Error()
			_ = a.writeUpdateState(state)
			return a.updateStatus(), err
		}
	} else if err := a.unregisterUpdateTask(); err != nil {
		state.LastError = err.Error()
		_ = a.writeUpdateState(state)
		return a.updateStatus(), err
	}
	state.SchemaVersion = 1
	state.AutoUpdateEnabled = enabled
	state.LastError = ""
	if err := a.writeUpdateState(state); err != nil {
		if enabled {
			_ = a.unregisterUpdateTask()
		} else {
			_ = a.registerUpdateTask()
		}
		return a.updateStatus(), err
	}
	return a.updateStatus(), nil
}

func (a *App) updateStatus() UpdateStatus {
	state, stateErr := a.readUpdateState()
	registered, err := a.updateTaskRegistered()
	lastError := state.LastError
	if data, readErr := os.ReadFile(a.updaterErrorPath()); readErr == nil && strings.TrimSpace(string(data)) != "" {
		lastError = redact(strings.TrimSpace(string(data)))
	}
	if stateErr != nil && !errors.Is(stateErr, os.ErrNotExist) {
		lastError = "读取更新状态失败：" + stateErr.Error()
	}
	if err != nil {
		lastError = err.Error()
	}
	return UpdateStatus{
		AutoUpdateEnabled: state.AutoUpdateEnabled,
		TaskRegistered:    registered,
		LastError:         lastError,
		PostUpdateVersion: state.PostUpdateVersion,
		PostUpdateState:   state.PostUpdateState,
		PostUpdateMessage: state.PostUpdateMessage,
		PostUpdateError:   state.PostUpdateError,
	}
}

func (a *App) InstallUpdate(version string) error {
	a.updateMu.Lock()
	defer a.updateMu.Unlock()
	if !a.mu.TryLock() {
		return errors.New("启动器正在执行其他操作，请稍后再更新")
	}
	defer a.mu.Unlock()
	releaseLock, err := a.acquireUpdateFileLock()
	if err != nil {
		return err
	}
	handoffLock := false
	defer func() {
		if !handoffLock {
			releaseLock()
		}
	}()

	info, err := a.checkForUpdates(true, true)
	if err != nil {
		return err
	}
	if !info.Available {
		return errors.New("当前已是最新版本")
	}
	if requested := normalizeVersion(version); requested != "" && requested != info.LatestVersion {
		return errors.New("可用版本已变化，请重新检查更新")
	}
	request, err := a.downloadAndStageUpdate(info)
	if err != nil {
		a.recordUpdateErrorUnlocked(err)
		return err
	}
	request.Relaunch = true
	if err := a.launchUpdateHelper(request, os.Getpid()); err != nil {
		a.recordUpdateErrorUnlocked(err)
		return err
	}
	handoffLock = true
	a.recordUpdateErrorUnlocked(nil)
	go func() {
		time.Sleep(300 * time.Millisecond)
		if a.ctx != nil {
			wailsRuntime.Quit(a.ctx)
		}
	}()
	return nil
}

func (a *App) downloadAndStageUpdate(info UpdateInfo) (updateRequest, error) {
	if info.AssetURL == "" || info.AssetName == "" {
		return updateRequest{}, errors.New("当前平台没有可用的更新安装包")
	}
	if info.ChecksumURL == "" {
		return updateRequest{}, errors.New("发布缺少 SHA256SUMS.txt，已拒绝安装")
	}
	checksumData, err := downloadUpdateBytes(info.ChecksumURL, 2*1024*1024)
	if err != nil {
		return updateRequest{}, fmt.Errorf("下载更新校验文件失败：%w", err)
	}
	expectedHash, err := checksumForAsset(checksumData, info.AssetName)
	if err != nil {
		return updateRequest{}, err
	}
	stageDir := filepath.Join(a.updateDir(), "v"+info.LatestVersion)
	if err := os.MkdirAll(stageDir, 0700); err != nil {
		return updateRequest{}, err
	}
	packagePath := filepath.Join(stageDir, info.AssetName)
	a.emitUpdateProgress("正在下载安装包", 0)
	if err := a.downloadUpdateFile(info.AssetURL, packagePath, info.AssetSize); err != nil {
		return updateRequest{}, fmt.Errorf("下载安装包失败：%w", err)
	}
	actualHash, err := updateFileSHA256(packagePath)
	if err != nil {
		return updateRequest{}, err
	}
	if !strings.EqualFold(actualHash, expectedHash) {
		_ = os.Remove(packagePath)
		return updateRequest{}, errors.New("安装包 SHA-256 校验失败，已删除下载文件")
	}
	a.emitUpdateProgress("安装包校验完成", 100)
	target, err := os.Executable()
	if err != nil {
		return updateRequest{}, fmt.Errorf("无法定位当前程序：%w", err)
	}
	return updateRequest{
		Version:     info.LatestVersion,
		PackagePath: packagePath,
		AssetName:   info.AssetName,
		SHA256:      expectedHash,
		TargetPath:  target,
		Token:       uuid.NewString(),
	}, nil
}

func downloadUpdateBytes(rawURL string, limit int64) ([]byte, error) {
	if !isAllowedUpdateURL(rawURL) {
		return nil, errors.New("更新校验文件地址不受信任")
	}
	var lastErr error
	for _, candidate := range updateDownloadCandidates(rawURL) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, candidate, nil)
		if err != nil {
			cancel()
			return nil, err
		}
		req.Header.Set("User-Agent", "hermes-dock/"+appVersion)
		resp, err := http.DefaultClient.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			data, readErr := io.ReadAll(io.LimitReader(resp.Body, limit+1))
			resp.Body.Close()
			cancel()
			if readErr == nil && int64(len(data)) <= limit {
				return data, nil
			}
			if readErr != nil {
				lastErr = readErr
			} else {
				lastErr = errors.New("下载内容超过大小限制")
			}
			continue
		}
		if resp != nil {
			lastErr = fmt.Errorf("%s 返回 %s", candidate, resp.Status)
			resp.Body.Close()
		} else {
			lastErr = err
		}
		cancel()
	}
	return nil, lastErr
}

func (a *App) downloadUpdateFile(rawURL string, target string, expectedSize int64) error {
	if !isAllowedUpdateURL(rawURL) {
		return errors.New("更新安装包地址不受信任")
	}
	var lastErr error
	for _, candidate := range updateDownloadCandidates(rawURL) {
		if err := a.downloadUpdateFileFrom(candidate, target, expectedSize); err == nil {
			return nil
		} else {
			lastErr = err
		}
	}
	return lastErr
}

func (a *App) downloadUpdateFileFrom(rawURL string, target string, expectedSize int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "hermes-dock/"+appVersion)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s 返回 %s", rawURL, resp.Status)
	}
	size := resp.ContentLength
	if expectedSize > 0 {
		size = expectedSize
	}
	if size > maxUpdateSize {
		return errors.New("安装包超过大小限制")
	}
	file, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer file.Close()
	var downloaded int64
	var lastPercent int32 = -1
	reader := io.TeeReader(io.LimitReader(resp.Body, maxUpdateSize+1), writerFunc(func(p []byte) (int, error) {
		total := atomic.AddInt64(&downloaded, int64(len(p)))
		if size > 0 {
			percent := int32(total * 100 / size)
			if percent > 100 {
				percent = 100
			}
			if atomic.SwapInt32(&lastPercent, percent) != percent {
				a.emitUpdateProgress("正在下载安装包", int(percent))
			}
		}
		return len(p), nil
	}))
	written, err := io.Copy(file, reader)
	if err != nil {
		return err
	}
	if written > maxUpdateSize {
		return errors.New("安装包超过大小限制")
	}
	if expectedSize > 0 && written != expectedSize {
		return fmt.Errorf("安装包大小不匹配：期望 %d 字节，实际 %d 字节", expectedSize, written)
	}
	return file.Sync()
}

type writerFunc func([]byte) (int, error)

func (fn writerFunc) Write(p []byte) (int, error) {
	return fn(p)
}

func checksumForAsset(data []byte, assetName string) (string, error) {
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) != 2 || strings.TrimPrefix(fields[1], "*") != assetName {
			continue
		}
		if len(fields[0]) != sha256.Size*2 {
			break
		}
		if _, err := hex.DecodeString(fields[0]); err != nil {
			break
		}
		return strings.ToLower(fields[0]), nil
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("SHA256SUMS.txt 中缺少 %s", assetName)
}

func updateFileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func updateDownloadCandidates(rawURL string) []string {
	candidates := []string{rawURL}
	if strings.HasPrefix(rawURL, updateRepoURL+"/releases/download/") {
		for _, mirror := range updateMirrorPrefixes {
			candidates = append(candidates, mirror.URL+rawURL)
		}
	}
	return candidates
}

func (a *App) emitUpdateProgress(message string, percent int) {
	a.emit("update:progress", map[string]interface{}{"message": message, "percent": percent})
}

func (a *App) recordUpdateError(err error) {
	a.updateMu.Lock()
	defer a.updateMu.Unlock()
	a.recordUpdateErrorUnlocked(err)
}

func (a *App) recordUpdateErrorUnlocked(err error) {
	state, stateErr := a.readUpdateState()
	if stateErr != nil && !errors.Is(stateErr, os.ErrNotExist) {
		return
	}
	if err == nil {
		state.LastError = ""
		_ = os.Remove(a.updaterErrorPath())
	} else {
		state.LastError = err.Error()
	}
	_ = a.writeUpdateState(state)
}

func (a *App) recordExternalUpdateError(err error) {
	if err == nil {
		_ = os.Remove(a.updaterErrorPath())
		return
	}
	if ensureErr := ensureDir(a.updateDir()); ensureErr != nil {
		return
	}
	_ = atomicWriteFile(a.updaterErrorPath(), []byte(redact(err.Error())+"\n"), 0600)
}

func parseUpdateArguments(args []string) {
	for index := 0; index < len(args); index++ {
		switch args[index] {
		case "--scheduled-update":
			scheduledUpdateMode = true
		case "--instance-root":
			if index+1 < len(args) {
				index++
				instanceRootOverride = filepath.Clean(args[index])
			}
		case "--update-token":
			if index+1 < len(args) {
				index++
				installedUpdateToken = strings.TrimSpace(args[index])
			}
		case "--launch-after-update":
			launchAfterUpdateMode = true
		}
	}
	if !scheduledUpdateMode && !launchAfterUpdateMode && installedUpdateToken == "" {
		instanceRootOverride = ""
	}
}

func runScheduledUpdate() error {
	app := NewApp()
	app.instanceRoot = detectInstanceRoot()
	state, err := app.readUpdateState()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("读取更新状态失败：%w", err)
	}
	if !state.AutoUpdateEnabled {
		return nil
	}
	releaseLock, err := app.acquireUpdateFileLock()
	if err != nil {
		return err
	}
	handoffLock := false
	defer func() {
		if !handoffLock {
			releaseLock()
		}
	}()
	info, err := app.checkForUpdates(true, false)
	if err != nil {
		app.recordExternalUpdateError(err)
		return err
	}
	if !info.Available {
		app.recordExternalUpdateError(nil)
		return nil
	}
	request, err := app.downloadAndStageUpdate(info)
	if err != nil {
		app.recordExternalUpdateError(err)
		return err
	}
	request.Relaunch = scheduledRelaunchAllowed()
	if err := app.queueUpdateRequest(request); err != nil {
		app.recordExternalUpdateError(err)
		return err
	}
	handoffLock = true
	app.recordExternalUpdateError(nil)
	return nil
}

func (a *App) queueUpdateRequest(request updateRequest) error {
	if err := os.MkdirAll(a.updateDir(), 0700); err != nil {
		return err
	}
	data, err := json.Marshal(request)
	if err != nil {
		return err
	}
	if err := atomicWriteFile(a.updateRequestPath(), append(data, '\n'), 0600); err != nil {
		return err
	}
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if !fileExists(a.updateRequestPath()) {
			return nil
		}
		time.Sleep(time.Second)
	}
	if fileExists(a.updatePIDPath()) {
		return nil
	}
	if err := os.Remove(a.updateRequestPath()); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return a.launchUpdateHelper(request, 0)
}

func (a *App) startUpdateWatcher() {
	if err := os.MkdirAll(a.updateDir(), 0700); err != nil {
		return
	}
	_ = atomicWriteFile(a.updatePIDPath(), []byte(strconv.Itoa(os.Getpid())+"\n"), 0600)
	ctx, cancel := context.WithCancel(context.Background())
	a.updateWatcherCancel = cancel
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				a.consumeUpdateRequest()
			}
		}
	}()
}

func (a *App) stopUpdateWatcher() {
	if a.updateWatcherCancel != nil {
		a.updateWatcherCancel()
	}
	data, err := os.ReadFile(a.updatePIDPath())
	if err == nil && strings.TrimSpace(string(data)) == strconv.Itoa(os.Getpid()) {
		_ = os.Remove(a.updatePIDPath())
	}
}

func (a *App) consumeUpdateRequest() {
	processingPath := a.updateRequestPath() + ".processing"
	if err := os.Rename(a.updateRequestPath(), processingPath); err != nil {
		return
	}
	if !a.mu.TryLock() {
		_ = os.Rename(processingPath, a.updateRequestPath())
		return
	}
	defer a.mu.Unlock()
	defer os.Remove(processingPath)
	handoffLock := false
	defer func() {
		if !handoffLock {
			_ = os.Remove(a.updateLockPath())
		}
	}()
	data, err := os.ReadFile(processingPath)
	if err != nil {
		a.recordUpdateError(err)
		return
	}
	var request updateRequest
	if err := json.Unmarshal(data, &request); err != nil {
		a.recordUpdateError(err)
		return
	}
	packagePath := filepath.Clean(request.PackagePath)
	updateRoot := filepath.Clean(a.updateDir())
	if packagePath == updateRoot || !strings.HasPrefix(packagePath, updateRoot+string(os.PathSeparator)) {
		a.recordUpdateError(errors.New("定时更新安装包路径无效"))
		return
	}
	currentExecutable, err := os.Executable()
	if err != nil || filepath.Clean(request.TargetPath) != filepath.Clean(currentExecutable) {
		a.recordUpdateError(errors.New("定时更新目标程序路径无效"))
		return
	}
	actualHash, err := updateFileSHA256(request.PackagePath)
	if err != nil || !strings.EqualFold(actualHash, request.SHA256) {
		if err == nil {
			err = errors.New("定时更新安装包校验失败")
		}
		a.recordUpdateError(err)
		return
	}
	request.Relaunch = true
	if err := a.launchUpdateHelper(request, os.Getpid()); err != nil {
		a.recordUpdateError(err)
		return
	}
	handoffLock = true
	a.recordUpdateError(nil)
	if a.ctx != nil {
		wailsRuntime.Quit(a.ctx)
	}
}

func (a *App) launchUpdateHelper(request updateRequest, waitPID int) error {
	helper, err := installedUpdaterPath()
	if err != nil {
		return err
	}
	if !fileExists(helper) {
		return fmt.Errorf("更新组件不存在：%s", helper)
	}
	stageDir := filepath.Dir(request.PackagePath)
	stagedHelper := filepath.Join(stageDir, "hermes-dock-updater-"+request.Token)
	if runtime.GOOS == "windows" {
		stagedHelper += ".exe"
	}
	if err := copyExecutable(helper, stagedHelper); err != nil {
		return fmt.Errorf("准备更新组件失败：%w", err)
	}
	args := []string{
		"--package", request.PackagePath,
		"--asset-name", request.AssetName,
		"--target", request.TargetPath,
		"--instance-root", a.instanceRoot,
		"--token", request.Token,
		"--wait-pid", strconv.Itoa(waitPID),
	}
	if request.Relaunch {
		args = append(args, "--relaunch")
	}
	cmd := backgroundCommand(stagedHelper, args...)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动更新组件失败：%w", err)
	}
	return nil
}

func installedUpdaterPath() (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", err
	}
	name := "hermes-dock-updater"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return filepath.Join(filepath.Dir(executable), name), nil
}

func copyExecutable(source string, target string) error {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	output, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0700)
	if err != nil {
		return err
	}
	if _, err := io.Copy(output, input); err != nil {
		output.Close()
		return err
	}
	if err := output.Sync(); err != nil {
		output.Close()
		return err
	}
	return output.Close()
}

func (a *App) acknowledgeInstalledUpdate() {
	_ = os.Remove(filepath.Join(a.updateDir(), "restart-pending"))
	if installedUpdateToken == "" {
		return
	}
	path := filepath.Join(a.updateDir(), "health-"+installedUpdateToken)
	_ = atomicWriteFile(path, []byte("ok\n"), 0600)
}

func (a *App) acquireUpdateFileLock() (func(), error) {
	if err := os.MkdirAll(a.updateDir(), 0700); err != nil {
		return nil, err
	}
	path := a.updateLockPath()
	if info, err := os.Stat(path); err == nil && time.Since(info.ModTime()) > 2*time.Hour {
		_ = os.Remove(path)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if errors.Is(err, os.ErrExist) {
		return nil, errors.New("另一个更新任务正在执行")
	}
	if err != nil {
		return nil, err
	}
	_, writeErr := file.WriteString(strconv.Itoa(os.Getpid()) + "\n")
	closeErr := file.Close()
	if writeErr != nil {
		_ = os.Remove(path)
		return nil, writeErr
	}
	if closeErr != nil {
		_ = os.Remove(path)
		return nil, closeErr
	}
	return func() { _ = os.Remove(path) }, nil
}

func pendingUpdateRestart() bool {
	return fileExists(filepath.Join(detectInstanceRoot(), "launcher", "updates", "restart-pending"))
}
