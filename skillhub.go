package main

import (
	"archive/zip"
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	skillHubAPIBase         = "https://api.skillhub.cn"
	skillHubDownloadLimit   = 20 * 1024 * 1024
	skillHubMaxFileCount    = 200
	skillHubHTTPTimeout     = 20 * time.Second
	skillHubInstallSubdir   = "skillhub"
	skillHubMetadataFile    = ".hermes-dock-skillhub.json"
	skillHubPackageMetaFile = "_meta.json"
	skillHubDefaultPage     = 1
	skillHubDefaultPageSize = 24
)

type skillHubListResponse struct {
	Code int `json:"code"`
	Data struct {
		Skills []skillHubListItem `json:"skills"`
		Total  int                `json:"total"`
	} `json:"data"`
	Message string `json:"message"`
}

type skillHubListItem struct {
	Slug          string                 `json:"slug"`
	Name          string                 `json:"name"`
	Description   string                 `json:"description"`
	DescriptionZH string                 `json:"description_zh"`
	Category      string                 `json:"category"`
	Source        string                 `json:"source"`
	Version       string                 `json:"version"`
	Downloads     int                    `json:"downloads"`
	Stars         int                    `json:"stars"`
	Installs      int                    `json:"installs"`
	Labels        map[string]string      `json:"labels"`
	Verified      bool                   `json:"verified"`
	Tags          []string               `json:"tags"`
	Homepage      string                 `json:"homepage"`
	OwnerName     string                 `json:"ownerName"`
	SubCategories []skillHubSubCategory  `json:"subCategories"`
	Publisher     map[string]interface{} `json:"publisher"`
}

type skillHubSubCategory struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

type skillHubCategoriesResponse struct {
	Items []struct {
		Key  string `json:"key"`
		Name string `json:"name"`
	} `json:"items"`
}

type skillHubDetailResponse struct {
	LatestVersion struct {
		Version string `json:"version"`
	} `json:"latestVersion"`
	Owner struct {
		DisplayName string `json:"displayName"`
		Handle      string `json:"handle"`
	} `json:"owner"`
	SecurityReports map[string]struct {
		Status     string `json:"status"`
		StatusText string `json:"statusText"`
		ReportURL  string `json:"reportUrl"`
	} `json:"securityReports"`
	Skill struct {
		Slug          string                 `json:"slug"`
		DisplayName   string                 `json:"displayName"`
		Summary       string                 `json:"summary"`
		SummaryZH     string                 `json:"summary_zh"`
		Category      string                 `json:"category"`
		Source        string                 `json:"source"`
		SourceURL     string                 `json:"sourceUrl"`
		Labels        map[string]string      `json:"labels"`
		Verified      bool                   `json:"verified"`
		IconURL       string                 `json:"iconUrl"`
		SubCategories []skillHubSubCategory  `json:"subCategories"`
		Stats         map[string]int         `json:"stats"`
		Tags          map[string]string      `json:"tags"`
		Publisher     map[string]interface{} `json:"publisher"`
	} `json:"skill"`
}

type skillHubFilesResponse struct {
	Count   int            `json:"count"`
	Files   []SkillHubFile `json:"files"`
	Version string         `json:"version"`
}

type skillHubSignatureResponse struct {
	Signed      bool   `json:"signed"`
	KeyID       string `json:"key_id"`
	ContentHash string `json:"content_hash"`
	Payload     string `json:"payload"`
}

type skillHubSignaturePayload struct {
	PackageMD5 string `json:"package_md5"`
}

type skillHubInstallMetadata struct {
	Source           string    `json:"source"`
	Slug             string    `json:"slug"`
	InstalledVersion string    `json:"installedVersion"`
	Name             string    `json:"name"`
	PackageMD5       string    `json:"packageMd5"`
	ContentHash      string    `json:"contentHash"`
	DownloadedAt     time.Time `json:"downloadedAt"`
	APIBase          string    `json:"apiBase"`
}

func (a *App) ListSkillHubSkills(query SkillHubQuery) (SkillHubState, error) {
	return a.ListSkillHubSkillsForProfile(a.currentProfileID(), query)
}

func (a *App) ListSkillHubSkillsForProfile(profileID string, query SkillHubQuery) (SkillHubState, error) {
	profileID, err := a.resolveProfileID(profileID)
	if err != nil {
		return SkillHubState{}, err
	}
	query = normalizeSkillHubQuery(query)
	categories, err := a.skillHubCategories()
	if err != nil {
		return SkillHubState{}, err
	}
	categoryNames := map[string]string{}
	for _, category := range categories {
		categoryNames[category.Key] = category.Name
	}
	params := url.Values{}
	params.Set("page", strconv.Itoa(query.Page))
	params.Set("pageSize", strconv.Itoa(query.PageSize))
	if strings.TrimSpace(query.Keyword) != "" {
		params.Set("keyword", strings.TrimSpace(query.Keyword))
	}
	if strings.TrimSpace(query.Category) != "" {
		params.Set("category", strings.TrimSpace(query.Category))
	}
	params.Set("sortBy", query.SortBy)
	params.Set("order", query.Order)
	var response skillHubListResponse
	if err := a.skillHubJSON("/api/skills?"+params.Encode(), &response); err != nil {
		return SkillHubState{}, err
	}
	if response.Code != 0 {
		return SkillHubState{}, fmt.Errorf("SkillHub 返回错误：%s", firstNonEmpty(response.Message, strconv.Itoa(response.Code)))
	}
	installed := a.installedSkillHubSlugsForProfile(profileID)
	state := SkillHubState{
		Categories: categories,
		Total:      response.Data.Total,
		Page:       query.Page,
		PageSize:   query.PageSize,
	}
	for _, item := range response.Data.Skills {
		state.Skills = append(state.Skills, skillHubListItemToSkill(item, categoryNames, installed))
	}
	return state, nil
}

func (a *App) GetSkillHubDetail(slug string) (SkillHubDetail, error) {
	return a.GetSkillHubDetailForProfile(a.currentProfileID(), slug)
}

func (a *App) GetSkillHubDetailForProfile(profileID string, slug string) (SkillHubDetail, error) {
	profileID, err := a.resolveProfileID(profileID)
	if err != nil {
		return SkillHubDetail{}, err
	}
	slug = strings.TrimSpace(slug)
	if !validSkillHubSlug(slug) {
		return SkillHubDetail{}, fmt.Errorf("SkillHub slug 不安全")
	}
	detailResponse, version, err := a.skillHubDetail(slug)
	if err != nil {
		return SkillHubDetail{}, err
	}
	categories, _ := a.skillHubCategories()
	categoryNames := map[string]string{}
	for _, category := range categories {
		categoryNames[category.Key] = category.Name
	}
	item := skillHubListItem{
		Slug:          detailResponse.Skill.Slug,
		Name:          detailResponse.Skill.DisplayName,
		Description:   detailResponse.Skill.Summary,
		DescriptionZH: detailResponse.Skill.SummaryZH,
		Category:      detailResponse.Skill.Category,
		Source:        detailResponse.Skill.Source,
		Version:       version,
		Labels:        detailResponse.Skill.Labels,
		Verified:      detailResponse.Skill.Verified,
		SubCategories: detailResponse.Skill.SubCategories,
		Publisher:     detailResponse.Skill.Publisher,
	}
	if detailResponse.Skill.Stats != nil {
		item.Downloads = detailResponse.Skill.Stats["downloads"]
		item.Stars = detailResponse.Skill.Stats["stars"]
		item.Installs = detailResponse.Skill.Stats["installs"]
	}
	var files skillHubFilesResponse
	if err := a.skillHubJSON(fmt.Sprintf("/api/v1/skills/%s/files?version=%s", url.PathEscape(slug), url.QueryEscape(version)), &files); err != nil {
		return SkillHubDetail{}, err
	}
	signature, err := a.skillHubSignature(slug, version)
	if err != nil {
		return SkillHubDetail{}, err
	}
	detail := SkillHubDetail{
		SkillHubSkill: skillHubListItemToSkill(item, categoryNames, a.installedSkillHubSlugsForProfile(profileID)),
		OwnerName:     firstNonEmpty(detailResponse.Owner.DisplayName, detailResponse.Owner.Handle),
		Homepage:      detailResponse.Skill.SourceURL,
		Files:         files.Files,
		FileCount:     files.Count,
		Signature:     signature,
	}
	for provider, report := range detailResponse.SecurityReports {
		detail.SecurityReports = append(detail.SecurityReports, SkillHubSecurity{
			Provider: provider,
			Status:   report.Status,
			Text:     report.StatusText,
			URL:      report.ReportURL,
		})
	}
	return detail, nil
}

func (a *App) InstallSkillHubSkill(slug string) error {
	return a.InstallSkillHubSkillForProfile(a.currentProfileID(), slug)
}

func (a *App) InstallSkillHubSkillForProfile(profileID string, slug string) error {
	release, err := a.beginExclusiveOperation("安装技能")
	if err != nil {
		return err
	}
	defer release()
	profileID, err = a.resolveProfileID(profileID)
	if err != nil {
		return err
	}
	slug = strings.TrimSpace(slug)
	if !validSkillHubSlug(slug) {
		return fmt.Errorf("SkillHub slug 不安全")
	}
	detailResponse, version, err := a.skillHubDetail(slug)
	if err != nil {
		return err
	}
	var files skillHubFilesResponse
	if err := a.skillHubJSON(fmt.Sprintf("/api/v1/skills/%s/files?version=%s", url.PathEscape(slug), url.QueryEscape(version)), &files); err != nil {
		return err
	}
	signature, err := a.skillHubSignature(slug, version)
	if err != nil {
		return err
	}
	zipPath, err := a.downloadSkillHubZip(slug)
	if err != nil {
		return err
	}
	defer os.Remove(zipPath)
	if signature.PackageMD5 != "" {
		sum, err := fileMD5(zipPath)
		if err != nil {
			return err
		}
		if !strings.EqualFold(sum, signature.PackageMD5) {
			return fmt.Errorf("SkillHub 技能包校验失败")
		}
	}
	target := filepath.Join(a.profileDataDir(profileID), "skills", skillHubInstallSubdir, slug)
	if err := ensureDir(filepath.Dir(target)); err != nil {
		return err
	}
	tmp, err := os.MkdirTemp(filepath.Dir(target), ".skillhub-"+slug+"-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	if err := extractSkillHubZip(zipPath, tmp, files.Files); err != nil {
		return err
	}
	metadata := skillHubInstallMetadata{
		Source:           "skillhub",
		Slug:             slug,
		InstalledVersion: version,
		Name:             detailResponse.Skill.DisplayName,
		PackageMD5:       signature.PackageMD5,
		ContentHash:      signature.ContentHash,
		DownloadedAt:     time.Now().UTC(),
		APIBase:          skillHubAPIBase,
	}
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(tmp, skillHubMetadataFile), data, 0600); err != nil {
		return err
	}
	if fileExists(target) {
		if err := a.backupDirectory(target, "before-skillhub-install-"+sanitizeName(slug)); err != nil {
			return err
		}
		if err := os.RemoveAll(target); err != nil {
			return err
		}
	}
	if err := os.Rename(tmp, target); err != nil {
		return err
	}
	return a.markRebuildRequired()
}

func normalizeSkillHubQuery(query SkillHubQuery) SkillHubQuery {
	if query.Page <= 0 {
		query.Page = skillHubDefaultPage
	}
	if query.PageSize <= 0 || query.PageSize > 48 {
		query.PageSize = skillHubDefaultPageSize
	}
	switch query.SortBy {
	case "score", "downloads", "stars", "updated_at":
	default:
		query.SortBy = "score"
	}
	if query.Order != "asc" {
		query.Order = "desc"
	}
	return query
}

func (a *App) skillHubCategories() ([]SkillHubCategory, error) {
	var response skillHubCategoriesResponse
	if err := a.skillHubJSON("/api/v1/categories", &response); err != nil {
		return nil, err
	}
	var categories []SkillHubCategory
	for _, item := range response.Items {
		categories = append(categories, SkillHubCategory{Key: item.Key, Name: item.Name})
	}
	return categories, nil
}

func (a *App) skillHubDetail(slug string) (skillHubDetailResponse, string, error) {
	var response skillHubDetailResponse
	if err := a.skillHubJSON("/api/v1/skills/"+url.PathEscape(slug), &response); err != nil {
		return response, "", err
	}
	version := strings.TrimSpace(response.LatestVersion.Version)
	if version == "" {
		return response, "", fmt.Errorf("SkillHub 技能缺少版本：%s", slug)
	}
	return response, version, nil
}

func (a *App) skillHubSignature(slug string, version string) (SkillHubSignature, error) {
	var response skillHubSignatureResponse
	if err := a.skillHubJSON(fmt.Sprintf("/api/v1/open/skills/%s/versions/%s/signature", url.PathEscape(slug), url.PathEscape(version)), &response); err != nil {
		return SkillHubSignature{}, err
	}
	signature := SkillHubSignature{
		Signed:      response.Signed,
		KeyID:       response.KeyID,
		ContentHash: response.ContentHash,
		Payload:     response.Payload,
	}
	var payload skillHubSignaturePayload
	if err := json.Unmarshal([]byte(response.Payload), &payload); err == nil {
		signature.PackageMD5 = payload.PackageMD5
	}
	return signature, nil
}

func (a *App) skillHubJSON(path string, out interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), skillHubHTTPTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, skillHubAPIBase+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("连接 SkillHub 失败：%w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("SkillHub 请求失败：HTTP %d", resp.StatusCode)
	}
	decoder := json.NewDecoder(io.LimitReader(resp.Body, skillHubDownloadLimit))
	if err := decoder.Decode(out); err != nil {
		return fmt.Errorf("解析 SkillHub 响应失败：%w", err)
	}
	return nil
}

func (a *App) downloadSkillHubZip(slug string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), skillHubHTTPTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, skillHubAPIBase+"/api/v1/download?slug="+url.QueryEscape(slug), nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("下载 SkillHub 技能失败：%w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("下载 SkillHub 技能失败：HTTP %d", resp.StatusCode)
	}
	tmp, err := os.CreateTemp("", "skillhub-*.zip")
	if err != nil {
		return "", err
	}
	defer tmp.Close()
	limited := io.LimitReader(resp.Body, skillHubDownloadLimit+1)
	written, err := io.Copy(tmp, limited)
	if err != nil {
		os.Remove(tmp.Name())
		return "", err
	}
	if written > skillHubDownloadLimit {
		os.Remove(tmp.Name())
		return "", fmt.Errorf("SkillHub 技能包超过 %d MB 限制", skillHubDownloadLimit/1024/1024)
	}
	return tmp.Name(), nil
}

func skillHubListItemToSkill(item skillHubListItem, categoryNames map[string]string, installed map[string]string) SkillHubSkill {
	name := firstNonEmpty(item.Name, item.Slug)
	description := firstNonEmpty(item.DescriptionZH, item.Description)
	var tags []string
	for _, sub := range item.SubCategories {
		if sub.Name != "" {
			tags = append(tags, sub.Name)
		}
	}
	tags = append(tags, item.Tags...)
	path, ok := installed[item.Slug]
	return SkillHubSkill{
		Slug:           item.Slug,
		Name:           name,
		Description:    description,
		Category:       item.Category,
		CategoryName:   firstNonEmpty(categoryNames[item.Category], item.Category),
		Source:         item.Source,
		Version:        item.Version,
		Downloads:      item.Downloads,
		Stars:          item.Stars,
		Installs:       item.Installs,
		RequiresAPIKey: item.Labels["requires_api_key"] == "true",
		Verified:       item.Verified || publisherVerified(item.Publisher),
		Installed:      ok,
		InstalledPath:  path,
		Tags:           tags,
	}
}

func (a *App) installedSkillHubSlugs() map[string]string {
	return a.installedSkillHubSlugsForProfile(a.currentProfileID())
}

func (a *App) installedSkillHubSlugsForProfile(profileID string) map[string]string {
	out := map[string]string{}
	root := filepath.Join(a.profileDataDir(profileID), "skills", skillHubInstallSubdir)
	entries, err := os.ReadDir(root)
	if err != nil {
		return out
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		slug := entry.Name()
		rel := filepath.ToSlash(filepath.Join("skills", skillHubInstallSubdir, slug))
		out[slug] = rel
		metadataPath := filepath.Join(root, slug, skillHubMetadataFile)
		data, err := os.ReadFile(metadataPath)
		if err != nil {
			continue
		}
		var metadata skillHubInstallMetadata
		if json.Unmarshal(data, &metadata) == nil && metadata.Slug != "" {
			out[metadata.Slug] = rel
		}
	}
	return out
}

func extractSkillHubZip(zipPath string, target string, expected []SkillHubFile) error {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer reader.Close()
	if len(reader.File) == 0 {
		return fmt.Errorf("SkillHub 技能包为空")
	}
	if len(reader.File) > skillHubMaxFileCount {
		return fmt.Errorf("SkillHub 技能包文件过多")
	}
	expectedHashes := map[string]string{}
	for _, file := range expected {
		expectedHashes[filepath.ToSlash(file.Path)] = strings.ToLower(file.SHA256)
	}
	hasSkill := false
	for _, file := range reader.File {
		name := filepath.ToSlash(file.Name)
		isDir := file.FileInfo().IsDir()
		if err := validateSkillHubZipPath(name, isDir); err != nil {
			return err
		}
		if strings.EqualFold(name, "SKILL.md") {
			hasSkill = true
		}
		if isDir {
			continue
		}
		if file.FileInfo().Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("SkillHub 技能包不能包含符号链接：%s", name)
		}
		if len(expectedHashes) > 0 && expectedHashes[name] == "" {
			if isSkillHubPackageMetaFile(name) {
				continue
			}
			return fmt.Errorf("SkillHub 技能包包含未声明文件：%s", name)
		}
		if err := extractSkillHubZipFile(file, target, expectedHashes[name]); err != nil {
			return err
		}
	}
	if !hasSkill {
		return fmt.Errorf("SkillHub 技能包缺少 SKILL.md")
	}
	return nil
}

func isSkillHubPackageMetaFile(name string) bool {
	return name == skillHubPackageMetaFile
}

func validateSkillHubZipPath(name string, isDir bool) error {
	if name == "" || strings.HasPrefix(name, "/") || strings.Contains(name, "\\") {
		return fmt.Errorf("SkillHub 技能包路径不安全：%s", name)
	}
	normalized := name
	if isDir {
		normalized = strings.TrimSuffix(name, "/")
	}
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(normalized)))
	if clean == "." || clean != normalized || strings.HasPrefix(clean, "../") || strings.Contains(clean, "/../") {
		return fmt.Errorf("SkillHub 技能包路径不安全：%s", name)
	}
	return nil
}

func extractSkillHubZipFile(file *zip.File, target string, expectedSHA string) error {
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()
	targetPath := filepath.Join(target, filepath.FromSlash(file.Name))
	if err := ensureDir(filepath.Dir(targetPath)); err != nil {
		return err
	}
	dst, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	hash := sha256.New()
	_, copyErr := io.Copy(dst, io.TeeReader(src, hash))
	closeErr := dst.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	if expectedSHA != "" {
		actual := hex.EncodeToString(hash.Sum(nil))
		if actual != expectedSHA {
			return fmt.Errorf("SkillHub 文件校验失败：%s", file.Name)
		}
	}
	return nil
}

func validSkillHubSlug(slug string) bool {
	if slug == "" || len(slug) > 120 {
		return false
	}
	for _, r := range slug {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			continue
		}
		return false
	}
	return true
}

func publisherVerified(publisher map[string]interface{}) bool {
	value, ok := publisher["verified"].(bool)
	return ok && value
}

func fileMD5(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
