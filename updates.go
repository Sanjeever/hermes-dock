package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	updateRepoSlug = "sqyl2026/hermes-dock-releases"
	updateCheckURL = "https://api.github.com/repos/" + updateRepoSlug + "/releases/latest"
	updateRepoURL  = "https://github.com/" + updateRepoSlug
	updateCooldown = 24 * time.Hour
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
	SchemaVersion    int    `json:"schemaVersion"`
	LastCheckedAt    string `json:"lastCheckedAt"`
	LatestVersion    string `json:"latestVersion"`
	ReleaseURL       string `json:"releaseUrl"`
	AssetURL         string `json:"assetUrl"`
	AssetName        string `json:"assetName"`
	DismissedVersion string `json:"dismissedVersion"`
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
}

func (a *App) CheckForUpdates(force bool) (UpdateInfo, error) {
	state, _ := a.readUpdateState()
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
	checkedAt := time.Now().UTC().Format(time.RFC3339)
	state.SchemaVersion = 1
	state.LastCheckedAt = checkedAt
	state.LatestVersion = latest
	state.ReleaseURL = firstNonEmpty(release.HTMLURL, updateRepoURL+"/releases/tag/"+release.TagName)
	state.AssetURL = asset.BrowserDownloadURL
	state.AssetName = asset.Name
	if err := a.writeUpdateState(state); err != nil {
		return UpdateInfo{}, err
	}

	info := UpdateInfo{
		CurrentVersion: appVersion,
		LatestVersion:  latest,
		Available:      compareVersions(latest, appVersion) > 0,
		Dismissed:      state.DismissedVersion == latest,
		ReleaseURL:     state.ReleaseURL,
		AssetURL:       asset.BrowserDownloadURL,
		AssetName:      asset.Name,
		Mirrors:        mirrorLinks(asset.BrowserDownloadURL),
		CheckedAt:      checkedAt,
	}
	return info, nil
}

func (a *App) DismissUpdate(version string) error {
	version = normalizeVersion(version)
	if version == "" {
		return errors.New("版本号不能为空")
	}
	state, _ := a.readUpdateState()
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

func expectedReleaseAssetNames(tagName string) []string {
	prefix := "hermes-dock-" + tagName
	switch runtime.GOOS + "/" + runtime.GOARCH {
	case "linux/amd64":
		return []string{prefix + "-linux-amd64.deb", prefix + "-linux-amd64.tar.gz"}
	case "windows/amd64":
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
