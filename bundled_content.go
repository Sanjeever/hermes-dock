package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type bundledContentState struct {
	SchemaVersion   int               `json:"schemaVersion"`
	TemplateVersion string            `json:"templateVersion"`
	SoulVersion     string            `json:"soulVersion"`
	SkillsVersion   string            `json:"skillsVersion"`
	Files           map[string]string `json:"files"`
}

type bundledTemplateFile struct {
	rel  string
	data []byte
}

type bundledSyncPlan struct {
	file        bundledTemplateFile
	action      string
	existed     bool
	currentHash string
}

func (a *App) SyncBundledContent(req BundledContentSyncRequest) (BundledContentSyncResult, error) {
	release, err := a.beginExclusiveOperation("同步内置内容")
	if err != nil {
		return BundledContentSyncResult{}, err
	}
	defer release()
	return a.syncBundledContent(req, true, nil)
}

func (a *App) syncBundledContent(req BundledContentSyncRequest, forceSoul bool, beforeFirstWrite func() error) (BundledContentSyncResult, error) {
	if !req.SyncSoul && !req.SyncSkills {
		return BundledContentSyncResult{}, fmt.Errorf("请至少选择内置人格或内置技能")
	}
	registry, err := a.readProfileRegistry()
	if err != nil {
		return BundledContentSyncResult{}, err
	}
	if len(req.TargetProfileIDs) == 0 {
		return BundledContentSyncResult{}, fmt.Errorf("请至少选择一个目标 profile")
	}
	seen := map[string]bool{}
	result := BundledContentSyncResult{Results: []BundledContentProfileResult{}}
	changed := false
	writeStarted := false
	for _, rawID := range req.TargetProfileIDs {
		profileID := strings.TrimSpace(rawID)
		item := BundledContentProfileResult{ProfileID: profileID}
		switch {
		case seen[profileID]:
			item.Error = "目标 profile 重复"
		case !profileExists(registry, profileID):
			item.Error = "目标 profile 不存在"
		default:
			seen[profileID] = true
			item, err = a.syncBundledContentForProfile(profileID, req.SyncSoul, req.SyncSkills, forceSoul, func() error {
				if writeStarted || beforeFirstWrite == nil {
					return nil
				}
				if err := beforeFirstWrite(); err != nil {
					return err
				}
				writeStarted = true
				return nil
			})
			if err != nil {
				item.Success = false
				item.Error = redact(err.Error())
				if item.Added+item.Updated > 0 {
					item.Error = "已完成部分写入，后续步骤失败：" + item.Error
				}
			}
		}
		if item.Added+item.Updated > 0 {
			changed = true
		}
		if item.Success {
			result.Succeeded++
		} else {
			result.Failed++
		}
		result.Added += item.Added
		result.Updated += item.Updated
		result.Unchanged += item.Unchanged
		result.Skipped += item.Skipped
		result.Results = append(result.Results, item)
	}
	if changed {
		if err := a.markRebuildRequired(); err != nil {
			return result, err
		}
	}
	return result, nil
}

func (a *App) syncBundledContentForProfile(profileID string, syncSoul bool, syncSkills bool, forceSoul bool, beforeWrite func() error) (BundledContentProfileResult, error) {
	result := BundledContentProfileResult{ProfileID: profileID}
	targetRoot := a.profileDataDir(profileID)
	templates, err := bundledTemplateContent(profileID, targetRoot, syncSoul, syncSkills)
	if err != nil {
		return result, err
	}
	for _, file := range templates {
		target := filepath.Join(targetRoot, filepath.FromSlash(file.rel))
		if err := ensureNoSymlinkComponents(targetRoot, target); err != nil {
			return result, fmt.Errorf("内置内容目标路径不安全：%s：%w", file.rel, err)
		}
	}
	state, err := a.readBundledContentState(profileID)
	if err != nil && !os.IsNotExist(err) {
		return result, err
	}
	if state.Files == nil {
		state.Files = map[string]string{}
	}
	skippedSkillRoots := bundledSkillCollisionRoots(targetRoot, templates, state)
	skillRoots := bundledTemplateSkillRoots(templates)
	plans := make([]bundledSyncPlan, 0, len(templates))
	soulNeedsBackup := false
	skillsNeedBackup := false
	for _, file := range templates {
		target := filepath.Join(targetRoot, filepath.FromSlash(file.rel))
		current, readErr := os.ReadFile(target)
		if readErr != nil && !os.IsNotExist(readErr) {
			return result, readErr
		}
		templateHash := contentHash(file.data)
		currentHash := ""
		if readErr == nil {
			currentHash = contentHash(current)
		}
		action := ""
		skillRoot := skillRoots[file.rel]
		action = classifyBundledFile(!os.IsNotExist(readErr), currentHash, templateHash, state.Files[file.rel], skillRoot != "" && skippedSkillRoots[skillRoot])
		if forceSoul && file.rel == "SOUL.md" && readErr == nil && currentHash != templateHash {
			action = "updated"
		}
		plans = append(plans, bundledSyncPlan{file: file, action: action, existed: readErr == nil, currentHash: currentHash})
		if action == "added" || action == "updated" {
			if file.rel == "SOUL.md" && readErr == nil {
				soulNeedsBackup = true
			}
			if strings.HasPrefix(file.rel, "skills/") {
				skillsNeedBackup = true
			}
		}
	}
	if soulNeedsBackup {
		if err := a.backupFile(a.profileSoulPath(profileID), "before-sync-bundled-soul-"+profileID); err != nil {
			return result, err
		}
	}
	if skillsNeedBackup && fileExists(filepath.Join(targetRoot, "skills")) {
		if err := a.backupDirectory(filepath.Join(targetRoot, "skills"), "before-sync-bundled-skills-"+profileID); err != nil {
			return result, err
		}
	}
	for _, plan := range plans {
		switch plan.action {
		case "added", "updated":
			target := filepath.Join(targetRoot, filepath.FromSlash(plan.file.rel))
			if err := ensureDir(filepath.Dir(target)); err != nil {
				return result, err
			}
			if err := ensureNoSymlinkComponents(targetRoot, target); err != nil {
				return result, fmt.Errorf("内置内容目标路径不安全：%s：%w", plan.file.rel, err)
			}
			matches, err := bundledSyncTargetMatchesPlan(target, plan)
			if err != nil {
				return result, err
			}
			if !matches {
				result.Skipped++
				continue
			}
			if beforeWrite != nil {
				if err := beforeWrite(); err != nil {
					return result, err
				}
			}
			staged, err := stageBundledFile(target, plan.file.data)
			if err != nil {
				return result, err
			}
			defer os.Remove(staged)
			if err := ensureNoSymlinkComponents(targetRoot, target); err != nil {
				return result, fmt.Errorf("内置内容目标路径不安全：%s：%w", plan.file.rel, err)
			}
			matches, err = bundledSyncTargetMatchesPlan(target, plan)
			if err != nil {
				return result, err
			}
			if !matches {
				result.Skipped++
				continue
			}
			committed, err := commitBundledFile(target, staged, plan.existed)
			if err != nil {
				return result, err
			}
			if !committed {
				result.Skipped++
				continue
			}
			state.Files[plan.file.rel] = contentHash(plan.file.data)
			if plan.action == "added" {
				result.Added++
			} else {
				result.Updated++
			}
		case "unchanged":
			state.Files[plan.file.rel] = contentHash(plan.file.data)
			result.Unchanged++
		case "skipped":
			result.Skipped++
		}
	}
	state.SchemaVersion = 1
	if syncSoul {
		state.SoulVersion = templateVersion
	}
	if syncSkills {
		state.SkillsVersion = templateVersion
	}
	if state.SoulVersion == templateVersion && state.SkillsVersion == templateVersion {
		state.TemplateVersion = templateVersion
	}
	if err := a.writeBundledContentState(profileID, state); err != nil {
		return result, err
	}
	result.Success = true
	return result, nil
}

func bundledSyncTargetMatchesPlan(target string, plan bundledSyncPlan) (bool, error) {
	if !plan.existed {
		_, err := os.Lstat(target)
		if os.IsNotExist(err) {
			return true, nil
		}
		return false, err
	}
	opened, err := os.Open(target)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	defer opened.Close()
	return bundledSyncOpenedTargetMatchesPlan(target, opened, plan)
}

func bundledSyncOpenedTargetMatchesPlan(target string, opened *os.File, plan bundledSyncPlan) (bool, error) {
	before, err := opened.Stat()
	if err != nil {
		return false, err
	}
	current, err := io.ReadAll(opened)
	if err != nil {
		return false, err
	}
	after, err := opened.Stat()
	if err != nil {
		return false, err
	}
	pathInfo, err := os.Lstat(target)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !os.SameFile(before, after) || !os.SameFile(after, pathInfo) {
		return false, nil
	}
	if before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) {
		return false, nil
	}
	return contentHash(current) == plan.currentHash, nil
}

func stageBundledFile(target string, data []byte) (string, error) {
	file, err := os.CreateTemp(filepath.Dir(target), ".hermes-dock-bundled-*")
	if err != nil {
		return "", err
	}
	staged := file.Name()
	cleanup := func() {
		file.Close()
		os.Remove(staged)
	}
	if err := file.Chmod(0644); err != nil {
		cleanup()
		return "", err
	}
	if _, err := file.Write(data); err != nil {
		cleanup()
		return "", err
	}
	if err := file.Sync(); err != nil {
		cleanup()
		return "", err
	}
	if err := file.Close(); err != nil {
		os.Remove(staged)
		return "", err
	}
	return staged, nil
}

func commitBundledFile(target string, staged string, targetExisted bool) (bool, error) {
	if targetExisted {
		if err := os.Rename(staged, target); err != nil {
			return false, err
		}
		return true, nil
	}
	if err := os.Link(staged, target); err != nil {
		if errors.Is(err, os.ErrExist) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func bundledTemplateContent(profileID string, targetRoot string, includeSoul bool, includeSkills bool) ([]bundledTemplateFile, error) {
	var files []bundledTemplateFile
	if includeSoul {
		data, err := seedData.ReadFile("templates/seed-data/SOUL.md")
		if err != nil {
			return nil, err
		}
		files = append(files, bundledTemplateFile{rel: "SOUL.md", data: profileSeedData(data, "SOUL.md", profileID)})
	}
	if includeSkills {
		seeds, err := bundledSkillSeedFiles(targetRoot)
		if err != nil {
			return nil, err
		}
		for _, seed := range seeds {
			data, err := seedData.ReadFile(seed.source)
			if err != nil {
				return nil, err
			}
			files = append(files, bundledTemplateFile{rel: filepath.ToSlash(seed.rel), data: data})
		}
	}
	sort.Slice(files, func(i, j int) bool { return files[i].rel < files[j].rel })
	return files, nil
}

func bundledSkillCollisionRoots(targetRoot string, templates []bundledTemplateFile, state bundledContentState) map[string]bool {
	skipped := map[string]bool{}
	for _, file := range templates {
		if filepath.Base(file.rel) != "SKILL.md" || !strings.HasPrefix(file.rel, "skills/") {
			continue
		}
		root := filepath.ToSlash(filepath.Dir(file.rel))
		target := filepath.Join(targetRoot, filepath.FromSlash(file.rel))
		current, err := os.ReadFile(target)
		if os.IsNotExist(err) {
			if fileExists(filepath.Dir(target)) && !hasBundledStateForRoot(state, root) {
				skipped[root] = true
			}
			continue
		}
		if err != nil {
			continue
		}
		currentHash := contentHash(current)
		if currentHash == contentHash(file.data) {
			continue
		}
		previous := state.Files[file.rel]
		if previous == "" || currentHash != previous {
			skipped[root] = true
		}
	}
	return skipped
}

func hasBundledStateForRoot(state bundledContentState, root string) bool {
	prefix := strings.TrimSuffix(root, "/") + "/"
	for rel := range state.Files {
		if strings.HasPrefix(rel, prefix) {
			return true
		}
	}
	return false
}

func bundledTemplateSkillRoots(templates []bundledTemplateFile) map[string]string {
	rootSet := map[string]bool{}
	for _, file := range templates {
		if filepath.Base(file.rel) == "SKILL.md" && strings.HasPrefix(file.rel, "skills/") {
			rootSet[filepath.ToSlash(filepath.Dir(file.rel))] = true
		}
	}
	roots := make([]string, 0, len(rootSet))
	for root := range rootSet {
		roots = append(roots, root)
	}
	sort.Slice(roots, func(i, j int) bool { return len(roots[i]) > len(roots[j]) })
	indexed := map[string]string{}
	for _, file := range templates {
		for _, root := range roots {
			if file.rel == root+"/SKILL.md" || strings.HasPrefix(file.rel, root+"/") {
				indexed[file.rel] = root
				break
			}
		}
	}
	return indexed
}

func (a *App) recordBundledContentState(profileID string, targetRoot string) error {
	files, err := bundledTemplateContent(profileID, targetRoot, true, true)
	if err != nil {
		return err
	}
	state := bundledContentState{SchemaVersion: 1, TemplateVersion: templateVersion, SoulVersion: templateVersion, SkillsVersion: templateVersion, Files: map[string]string{}}
	for _, file := range files {
		path := filepath.Join(targetRoot, filepath.FromSlash(file.rel))
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		state.Files[file.rel] = contentHash(data)
	}
	return a.writeBundledContentState(profileID, state)
}

func (a *App) readBundledContentState(profileID string) (bundledContentState, error) {
	var state bundledContentState
	data, err := os.ReadFile(a.bundledContentStatePath(profileID))
	if err != nil {
		return state, err
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return bundledContentState{}, err
	}
	if state.Files == nil {
		state.Files = map[string]string{}
	}
	if state.TemplateVersion != "" {
		if state.SoulVersion == "" {
			state.SoulVersion = state.TemplateVersion
		}
		if state.SkillsVersion == "" {
			state.SkillsVersion = state.TemplateVersion
		}
	}
	return state, nil
}

func (a *App) writeBundledContentState(profileID string, state bundledContentState) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	if err := ensureDir(filepath.Dir(a.bundledContentStatePath(profileID))); err != nil {
		return err
	}
	return atomicWriteFile(a.bundledContentStatePath(profileID), append(data, '\n'), 0644)
}

func (a *App) bundledContentAvailability(registry ProfileRegistry) BundledContentAvailability {
	availability := BundledContentAvailability{}
	for _, profile := range registry.Profiles {
		state, err := a.readBundledContentState(profile.ID)
		if err != nil || state.SoulVersion != templateVersion || state.SkillsVersion != templateVersion {
			availability.PendingProfiles++
		}
	}
	availability.Available = availability.PendingProfiles > 0
	return availability
}

func contentHash(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func classifyBundledFile(exists bool, currentHash string, templateHash string, previousHash string, skillCollision bool) string {
	switch {
	case skillCollision:
		return "skipped"
	case !exists:
		return "added"
	case currentHash == templateHash:
		return "unchanged"
	case previousHash != "" && currentHash == previousHash:
		return "updated"
	default:
		return "skipped"
	}
}
