package main

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	skillPreviewLimit  = 30 * 1024
	skillFileListLimit = 100
)

type skillFrontmatter struct {
	Name        string                 `yaml:"name"`
	Description string                 `yaml:"description"`
	Version     string                 `yaml:"version"`
	Author      interface{}            `yaml:"author"`
	Platforms   interface{}            `yaml:"platforms"`
	Tags        interface{}            `yaml:"tags"`
	Metadata    map[string]interface{} `yaml:"metadata"`
}

func (a *App) ListProfileSkills() (SkillsState, error) {
	return a.ListProfileSkillsForProfile(a.currentProfileID())
}

func (a *App) ListProfileSkillsForProfile(profileID string) (SkillsState, error) {
	profileID, err := a.resolveProfileID(profileID)
	if err != nil {
		return SkillsState{}, err
	}
	bundled := a.bundledSkillNames()
	skillsDir := filepath.Join(a.profileDataDir(profileID), "skills")
	skills, err := a.scanInstalledSkills(profileID, skillsDir, bundled)
	if err != nil {
		return SkillsState{}, err
	}
	markSkillConflicts(skills)
	sortSkillSummaries(skills)
	state := SkillsState{
		ActiveProfile: profileID,
		Skills:        skills,
		Total:         len(skills),
	}
	conflictNames := map[string]bool{}
	for _, skill := range skills {
		if skill.Builtin {
			state.BuiltinCount++
		} else {
			state.CustomCount++
		}
		if skill.Conflict {
			conflictNames[skill.Name] = true
		}
	}
	state.ConflictCount = len(conflictNames)
	return state, nil
}

func (a *App) GetSkillDetail(path string) (SkillDetail, error) {
	return a.GetSkillDetailForProfile(a.currentProfileID(), path)
}

func (a *App) GetSkillDetailForProfile(profileID string, path string) (SkillDetail, error) {
	profileID, err := a.resolveProfileID(profileID)
	if err != nil {
		return SkillDetail{}, err
	}
	summary, dir, err := a.skillSummaryForPathForProfile(profileID, path)
	if err != nil {
		return SkillDetail{}, err
	}
	detail := SkillDetail{SkillSummary: summary, ConflictPaths: []string{}}
	preview, truncated, err := readSkillPreview(filepath.Join(dir, "SKILL.md"))
	if err != nil {
		return SkillDetail{}, err
	}
	detail.Preview = preview
	detail.PreviewTruncated = truncated
	files, count, truncated, err := listSkillFiles(dir)
	if err != nil {
		return SkillDetail{}, err
	}
	detail.Files = files
	detail.FileCount = count
	detail.FilesTruncated = truncated
	state, err := a.ListProfileSkillsForProfile(profileID)
	if err == nil {
		for _, item := range state.Skills {
			if item.Name == summary.Name && item.Path != summary.Path {
				detail.ConflictPaths = append(detail.ConflictPaths, item.Path)
			}
		}
	}
	return detail, nil
}

func (a *App) DeleteSkill(path string) error {
	return a.DeleteSkillForProfile(a.currentProfileID(), path)
}

func (a *App) DeleteSkillForProfile(profileID string, path string) error {
	profileID, err := a.resolveProfileID(profileID)
	if err != nil {
		return err
	}
	summary, dir, err := a.skillSummaryForPathForProfile(profileID, path)
	if err != nil {
		return err
	}
	if err := a.backupDirectory(dir, "before-skill-delete-"+sanitizeName(summary.Name)); err != nil {
		return err
	}
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	return a.markRebuildRequired()
}

func (a *App) SyncBundledSkills() (SyncBundledSkillsResult, error) {
	return a.SyncBundledSkillsForProfile(a.currentProfileID())
}

func (a *App) SyncBundledSkillsForProfile(profileID string) (SyncBundledSkillsResult, error) {
	profileID, err := a.resolveProfileID(profileID)
	if err != nil {
		return SyncBundledSkillsResult{}, err
	}
	targetRoot := a.profileDataDir(profileID)
	skillsDir := filepath.Join(targetRoot, "skills")
	result := SyncBundledSkillsResult{ActiveProfile: profileID, SyncedSkills: []string{}}
	files, err := bundledSkillSeedFiles(targetRoot)
	if err != nil {
		return result, err
	}
	if len(files) == 0 {
		return result, nil
	}
	if err := a.backupDirectory(skillsDir, "before-sync-bundled-skills"); err != nil {
		return result, err
	}
	result, err = writeBundledSkillSeedFiles(files, result)
	if err != nil || result.SyncedFiles == 0 {
		return result, err
	}
	return result, a.markRebuildRequired()
}

func (a *App) RestoreDefaultSkills() (SyncBundledSkillsResult, error) {
	return a.RestoreDefaultSkillsForProfile(a.currentProfileID())
}

func (a *App) RestoreDefaultSkillsForProfile(profileID string) (SyncBundledSkillsResult, error) {
	profileID, err := a.resolveProfileID(profileID)
	if err != nil {
		return SyncBundledSkillsResult{}, err
	}
	targetRoot := a.profileDataDir(profileID)
	skillsDir := filepath.Join(targetRoot, "skills")
	result := SyncBundledSkillsResult{ActiveProfile: profileID, SyncedSkills: []string{}}
	files, err := bundledSkillSeedFiles(targetRoot)
	if err != nil {
		return result, err
	}
	if len(files) == 0 {
		return result, nil
	}
	if err := a.backupDirectory(skillsDir, "before-restore-default-skills"); err != nil {
		return result, err
	}
	if err := os.RemoveAll(skillsDir); err != nil {
		return result, err
	}
	result, err = writeBundledSkillSeedFiles(files, result)
	if err != nil || result.SyncedFiles == 0 {
		return result, err
	}
	return result, a.markRebuildRequired()
}

func (a *App) RestoreDefaultSoul() error {
	return a.RestoreDefaultSoulForProfile(a.currentProfileID())
}

func (a *App) RestoreDefaultSoulForProfile(profileID string) error {
	profileID, err := a.resolveProfileID(profileID)
	if err != nil {
		return err
	}
	target := a.profileSoulPath(profileID)
	data, err := seedData.ReadFile("templates/seed-data/SOUL.md")
	if err != nil {
		return err
	}
	data = profileSeedData(data, "SOUL.md", profileID)
	if err := a.backupFile(target, "before-restore-default-soul"); err != nil {
		return err
	}
	if err := ensureDir(filepath.Dir(target)); err != nil {
		return err
	}
	if err := atomicWriteFile(target, data, 0644); err != nil {
		return err
	}
	return a.markRebuildRequired()
}

type bundledSkillSeedFile struct {
	source string
	rel    string
	target string
}

func bundledSkillSeedFiles(targetRoot string) ([]bundledSkillSeedFile, error) {
	var files []bundledSkillSeedFile
	err := fs.WalkDir(seedData, "templates/seed-data/skills", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if entry.Name() == ".DS_Store" {
			return nil
		}
		rel, err := filepath.Rel("templates/seed-data", path)
		if err != nil {
			return err
		}
		target := filepath.Join(targetRoot, rel)
		files = append(files, bundledSkillSeedFile{source: path, rel: rel, target: target})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

func writeBundledSkillSeedFiles(files []bundledSkillSeedFile, result SyncBundledSkillsResult) (SyncBundledSkillsResult, error) {
	for _, item := range files {
		data, err := seedData.ReadFile(item.source)
		if err != nil {
			return result, err
		}
		if err := ensureDir(filepath.Dir(item.target)); err != nil {
			return result, err
		}
		if err := atomicWriteFile(item.target, data, 0644); err != nil {
			return result, err
		}
		result.SyncedFiles++
		if filepath.Base(item.rel) == "SKILL.md" {
			frontmatter, err := parseSkillFrontmatter(data)
			name := strings.TrimSpace(frontmatter.Name)
			if err == nil && validSkillName(name) {
				result.SyncedSkills = append(result.SyncedSkills, name)
			}
		}
	}
	sort.Strings(result.SyncedSkills)
	return result, nil
}

func (a *App) OpenSkillDirectory(path string) error {
	return a.OpenSkillDirectoryForProfile(a.currentProfileID(), path)
}

func (a *App) OpenSkillDirectoryForProfile(profileID string, path string) error {
	profileID, err := a.resolveProfileID(profileID)
	if err != nil {
		return err
	}
	_, dir, err := a.skillSummaryForPathForProfile(profileID, path)
	if err != nil {
		return err
	}
	switch runtime.GOOS {
	case "darwin":
		return backgroundCommand("open", dir).Start()
	case "windows":
		return backgroundCommand("explorer", dir).Start()
	default:
		return backgroundCommand("xdg-open", dir).Start()
	}
}

func (a *App) scanInstalledSkills(profileID string, root string, bundled map[string]bool) ([]SkillSummary, error) {
	if _, err := os.Stat(root); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var skills []SkillSummary
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			name := entry.Name()
			if strings.HasPrefix(name, ".") && path != root {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Name() != "SKILL.md" {
			return nil
		}
		summary, err := a.skillSummaryFromDirForProfile(profileID, filepath.Dir(path), bundled)
		if err != nil {
			return err
		}
		skills = append(skills, summary)
		return nil
	})
	return skills, err
}

func (a *App) skillSummaryForPath(path string) (SkillSummary, string, error) {
	return a.skillSummaryForPathForProfile(a.currentProfileID(), path)
}

func (a *App) skillSummaryForPathForProfile(profileID string, path string) (SkillSummary, string, error) {
	rel, dir, err := a.resolveSkillDirForProfile(profileID, path)
	if err != nil {
		return SkillSummary{}, "", err
	}
	if !fileExists(filepath.Join(dir, "SKILL.md")) {
		return SkillSummary{}, "", fmt.Errorf("不是有效技能目录：%s", rel)
	}
	summary, err := a.skillSummaryFromDirForProfile(profileID, dir, a.bundledSkillNames())
	if err != nil {
		return SkillSummary{}, "", err
	}
	return summary, dir, nil
}

func (a *App) resolveSkillDir(path string) (string, string, error) {
	return a.resolveSkillDirForProfile(a.currentProfileID(), path)
}

func (a *App) resolveSkillDirForProfile(profileID string, path string) (string, string, error) {
	path = filepath.ToSlash(strings.TrimSpace(path))
	if path == "" {
		return "", "", fmt.Errorf("技能路径不能为空")
	}
	if strings.HasPrefix(path, "/") || strings.Contains(path, "..") {
		return "", "", fmt.Errorf("技能路径不安全")
	}
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(path)))
	if clean == "." || clean == "skills" || !strings.HasPrefix(clean, "skills/") {
		return "", "", fmt.Errorf("技能路径必须位于 skills/ 下")
	}
	profileRoot := a.profileDataDir(profileID)
	base := filepath.Join(profileRoot, "skills")
	dir := filepath.Clean(filepath.Join(profileRoot, filepath.FromSlash(clean)))
	root := filepath.Clean(base)
	if dir != root && !strings.HasPrefix(dir, root+string(os.PathSeparator)) {
		return "", "", fmt.Errorf("技能路径不能超出当前 profile")
	}
	info, err := os.Stat(dir)
	if err != nil {
		return "", "", err
	}
	if !info.IsDir() {
		return "", "", fmt.Errorf("技能路径不是目录：%s", clean)
	}
	return clean, dir, nil
}

func (a *App) skillSummaryFromDir(dir string, bundled map[string]bool) (SkillSummary, error) {
	return a.skillSummaryFromDirForProfile(a.currentProfileID(), dir, bundled)
}

func (a *App) skillSummaryFromDirForProfile(profileID string, dir string, bundled map[string]bool) (SkillSummary, error) {
	rel, err := filepath.Rel(a.profileDataDir(profileID), dir)
	if err != nil {
		return SkillSummary{}, err
	}
	rel = filepath.ToSlash(rel)
	data, readErr := os.ReadFile(filepath.Join(dir, "SKILL.md"))
	summary := SkillSummary{
		Name:     filepath.Base(dir),
		Path:     rel,
		Category: skillCategory(rel),
	}
	if readErr != nil {
		summary.Error = readErr.Error()
		return summary, nil
	}
	frontmatter, err := parseSkillFrontmatter(data)
	if err != nil {
		summary.Error = err.Error()
	} else {
		if strings.TrimSpace(frontmatter.Name) == "" {
			summary.Error = "缺少技能 name"
		} else {
			summary.Name = strings.TrimSpace(frontmatter.Name)
		}
		summary.Description = strings.TrimSpace(frontmatter.Description)
		summary.Version = strings.TrimSpace(frontmatter.Version)
		summary.Author = strings.TrimSpace(stringValue(frontmatter.Author))
		summary.Platforms = stringList(frontmatter.Platforms)
		summary.Tags = skillTags(frontmatter)
	}
	if !validSkillName(summary.Name) {
		summary.Error = firstNonEmpty(summary.Error, "技能 name 不符合规范")
	}
	summary.Builtin = bundled[summary.Name]
	size, updated := skillDirStats(dir)
	summary.SizeBytes = size
	if !updated.IsZero() {
		summary.UpdatedAt = updated.UTC().Format(time.RFC3339)
	}
	return summary, nil
}

func (a *App) bundledSkillNames() map[string]bool {
	names := map[string]bool{}
	_ = fs.WalkDir(seedData, "templates/seed-data/skills", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if entry.IsDir() || entry.Name() != "SKILL.md" {
			return nil
		}
		data, err := seedData.ReadFile(path)
		if err != nil {
			return nil
		}
		frontmatter, err := parseSkillFrontmatter(data)
		if err != nil {
			return nil
		}
		name := strings.TrimSpace(frontmatter.Name)
		if validSkillName(name) {
			names[name] = true
		}
		return nil
	})
	return names
}

func parseSkillFrontmatter(data []byte) (skillFrontmatter, error) {
	if !bytes.HasPrefix(data, []byte("---\n")) && !bytes.HasPrefix(data, []byte("---\r\n")) {
		return skillFrontmatter{}, fmt.Errorf("缺少 YAML frontmatter")
	}
	lines := bytes.SplitAfter(data, []byte("\n"))
	offset := len(lines[0])
	for _, line := range lines[1:] {
		trimmed := strings.TrimSpace(string(line))
		if trimmed == "---" {
			var out skillFrontmatter
			if err := yaml.Unmarshal(data[len(lines[0]):offset], &out); err != nil {
				return skillFrontmatter{}, err
			}
			return out, nil
		}
		offset += len(line)
	}
	return skillFrontmatter{}, fmt.Errorf("frontmatter 未闭合")
}

func readSkillPreview(path string) (string, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false, err
	}
	if len(data) <= skillPreviewLimit {
		return string(data), false, nil
	}
	return string(data[:skillPreviewLimit]), true, nil
}

func listSkillFiles(dir string) ([]SkillFileInfo, int, bool, error) {
	var files []SkillFileInfo
	count := 0
	err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		count++
		if len(files) >= skillFileListLimit {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		files = append(files, SkillFileInfo{
			Path:      filepath.ToSlash(rel),
			SizeBytes: info.Size(),
			UpdatedAt: info.ModTime().UTC().Format(time.RFC3339),
		})
		return nil
	})
	return files, count, count > len(files), err
}

func skillDirStats(dir string) (int64, time.Time) {
	var size int64
	var updated time.Time
	_ = filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return nil
		}
		size += info.Size()
		if info.ModTime().After(updated) {
			updated = info.ModTime()
		}
		return nil
	})
	return size, updated
}

func markSkillConflicts(skills []SkillSummary) {
	counts := map[string]int{}
	for _, skill := range skills {
		counts[skill.Name]++
	}
	for i := range skills {
		skills[i].Conflict = counts[skills[i].Name] > 1
	}
}

func sortSkillSummaries(skills []SkillSummary) {
	sort.SliceStable(skills, func(i, j int) bool {
		left, right := skills[i], skills[j]
		if left.Conflict != right.Conflict {
			return left.Conflict
		}
		if left.Builtin != right.Builtin {
			return !left.Builtin
		}
		if left.Name != right.Name {
			return left.Name < right.Name
		}
		return left.Path < right.Path
	})
}

func skillCategory(path string) string {
	parts := strings.Split(filepath.ToSlash(path), "/")
	if len(parts) < 3 {
		return "根目录"
	}
	return parts[1]
}

func validSkillName(name string) bool {
	if name == "" || len(name) > 64 {
		return false
	}
	for i, r := range name {
		valid := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_'
		if !valid {
			return false
		}
		if i == 0 && !(r >= 'a' && r <= 'z') {
			return false
		}
	}
	return true
}

func stringValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case []interface{}:
		items := stringList(v)
		return strings.Join(items, ", ")
	default:
		return ""
	}
}

func stringList(value interface{}) []string {
	var out []string
	switch v := value.(type) {
	case []string:
		out = append(out, v...)
	case []interface{}:
		for _, item := range v {
			if text := strings.TrimSpace(stringValue(item)); text != "" {
				out = append(out, text)
			}
		}
	case string:
		if strings.TrimSpace(v) != "" {
			out = append(out, strings.TrimSpace(v))
		}
	}
	sort.Strings(out)
	return out
}

func skillTags(frontmatter skillFrontmatter) []string {
	tags := stringList(frontmatter.Tags)
	metadata, ok := frontmatter.Metadata["hermes"].(map[string]interface{})
	if !ok {
		return tags
	}
	for _, tag := range stringList(metadata["tags"]) {
		if !containsString(tags, tag) {
			tags = append(tags, tag)
		}
	}
	sort.Strings(tags)
	return tags
}

func containsString(items []string, value string) bool {
	for _, item := range items {
		if item == value {
			return true
		}
	}
	return false
}
