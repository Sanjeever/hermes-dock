package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	applyStateIdle       = "idle"
	applyStateValidating = "validating"
	applyStateApplying   = "applying"
	applyStateWaiting    = "waiting"
	applyStateSlow       = "slow"
	applyStateSucceeded  = "succeeded"
	applyStateFailed     = "failed"
)

func (a *App) beginExclusiveOperation(action string) (func(), error) {
	if !a.operationMu.TryLock() {
		if a.readApplyConfigStatus().Active {
			return nil, fmt.Errorf("正在应用配置，完成后才能%s", action)
		}
		return nil, fmt.Errorf("正在执行其他实例操作，暂时无法%s", action)
	}
	return a.operationMu.Unlock, nil
}

func (a *App) startApplyConfigTask() error {
	return a.startApplyConfigTaskWithOperationID(false, "", false)
}

func (a *App) startForceRebuildTask() error {
	return a.startApplyConfigTaskWithOperationID(false, "", true)
}

// startApplyConfigTaskWithOperationID starts the task and transfers ownership
// of operationMu to it. When operationLocked is true, the caller must hold the
// mutex; this function releases it on every setup error.
func (a *App) startApplyConfigTaskWithOperationID(operationLocked bool, id string, forceRecreate bool) error {
	if !operationLocked && !a.operationMu.TryLock() {
		if forceRecreate {
			return fmt.Errorf("正在执行其他实例操作，暂时无法强制重建 Hermes 容器")
		}
		return fmt.Errorf("正在执行其他实例操作，暂时无法应用配置")
	}
	a.applyMu.Lock()
	current, err := a.readApplyConfigStatusUnlocked()
	if err != nil && !os.IsNotExist(err) {
		a.applyMu.Unlock()
		a.operationMu.Unlock()
		return err
	}
	if current.Active {
		a.applyMu.Unlock()
		a.operationMu.Unlock()
		return fmt.Errorf("已有应用配置任务正在执行")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if id == "" {
		id = uuid.NewString()
	}
	status := ApplyConfigStatus{
		ID:        id,
		State:     applyStateValidating,
		Phase:     "validate",
		Message:   "正在检查配置",
		Active:    true,
		StartedAt: now,
		UpdatedAt: now,
	}
	if forceRecreate {
		status.Message = "正在准备强制重建 Hermes 容器"
	}
	if err := a.writeApplyConfigStatusUnlocked(status); err != nil {
		a.applyMu.Unlock()
		a.operationMu.Unlock()
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.applyCancel = cancel
	a.applyActiveID = status.ID
	a.applyOwnsOperation = true
	a.applyMu.Unlock()
	a.emit("apply:status", status)
	go a.runApplyConfigTask(ctx, status.ID, forceRecreate)
	return nil
}

func (a *App) runApplyConfigTask(ctx context.Context, id string, forceRecreate bool) {
	settings, err := a.readComposeSettings()
	if err != nil {
		a.failApplyConfigTask(id, err)
		return
	}
	if err := ensureWritableDirectory(settings.SharedDirectory); err != nil {
		a.failApplyConfigTask(id, err)
		return
	}
	composeHash, err := a.composeRuntimeHash()
	if err != nil {
		a.failApplyConfigTask(id, err)
		return
	}
	dufsHash, err := a.dufsRuntimeHash()
	if err != nil {
		a.failApplyConfigTask(id, err)
		return
	}
	state, err := a.readState()
	if err != nil {
		a.failApplyConfigTask(id, err)
		return
	}
	if state.PendingDufsOnly && !forceRecreate {
		inputHash, err := a.applyRuntimeInputHash(composeHash, dufsHash)
		if err != nil {
			a.failApplyConfigTask(id, err)
			return
		}
		dufsRecreate := state.LastAppliedDufsHash != dufsHash
		if err := a.updateApplyConfigStatus(id, func(status *ApplyConfigStatus) {
			status.State = applyStateApplying
			status.Phase = "dufs"
			status.Strategy = "dufs-only"
			status.Message = "正在应用文件管理配置"
			status.ComposeHash = composeHash
			status.DufsHash = dufsHash
			status.InputHash = inputHash
			status.HermesImage = state.HermesImage
			status.DufsRecreate = dufsRecreate
		}); err != nil {
			a.failApplyConfigTask(id, err)
			return
		}
		if err := a.applyDufsRuntimeContext(ctx, settings, dufsRecreate); err != nil {
			if ctx.Err() != nil {
				return
			}
			a.failApplyConfigTask(id, err)
			return
		}
		a.completeApplyConfigTask(id, "文件管理配置已应用", composeHash, dufsHash, inputHash, state.HermesImage, false)
		return
	}
	for _, prepare := range []func() error{
		a.ensureRuntimeDependencies,
		a.ensureContainerInitHelpers,
		a.ensureProfileRunnerHelper,
		a.syncSavedModelProviderEnv,
	} {
		if err := prepare(); err != nil {
			a.failApplyConfigTask(id, err)
			return
		}
	}
	manifest, err := a.writeRuntimeManifest()
	if err != nil {
		a.failApplyConfigTask(id, err)
		return
	}
	inputHash, err := a.applyRuntimeInputHash(composeHash, dufsHash)
	if err != nil {
		a.failApplyConfigTask(id, err)
		return
	}
	dufsRecreate := state.LastAppliedDufsHash != dufsHash
	recreate := forceRecreate || shouldRecreateComposeRuntime(state.LastAppliedComposeHash, composeHash, a.containerStatus(context.Background()))
	strategy := "restart"
	message := "正在快速重启 Hermes 服务"
	if recreate {
		strategy = "recreate"
		message = "正在重建 Hermes 容器"
	}
	if forceRecreate {
		message = "正在强制重建 Hermes 容器"
	}
	if err := a.updateApplyConfigStatus(id, func(status *ApplyConfigStatus) {
		status.State = applyStateApplying
		status.Phase = "docker"
		status.Strategy = strategy
		status.Message = message
		status.Generation = manifest.Generation
		status.ComposeHash = composeHash
		status.DufsHash = dufsHash
		status.InputHash = inputHash
		status.HermesImage = state.HermesImage
		status.DufsRecreate = dufsRecreate
		status.RunnableProfiles = runnableProfileCount(manifest)
	}); err != nil {
		a.failApplyConfigTask(id, err)
		return
	}
	if err := a.applyComposeRuntimeContext(ctx, recreate); err != nil {
		if ctx.Err() != nil {
			return
		}
		a.failApplyConfigTask(id, err)
		return
	}
	if err := a.applyDufsRuntimeContext(ctx, settings, dufsRecreate); err != nil {
		if ctx.Err() != nil {
			return
		}
		a.failApplyConfigTask(id, err)
		return
	}
	if err := a.updateApplyConfigStatus(id, func(status *ApplyConfigStatus) {
		status.State = applyStateWaiting
		status.Phase = "profiles"
		status.Message = waitingApplyMessage(0, status.RunnableProfiles, false)
	}); err != nil {
		a.failApplyConfigTask(id, err)
		return
	}
	a.monitorApplyConfigTask(ctx, id, manifest, composeHash, dufsHash, inputHash, state.HermesImage)
}

func (a *App) monitorApplyConfigTask(ctx context.Context, id string, manifest RuntimeManifest, composeHash string, dufsHash string, inputHash string, hermesImage string) {
	started := time.Now()
	if status := a.readApplyConfigStatus(); status.StartedAt != "" {
		if parsed, err := time.Parse(time.RFC3339, status.StartedAt); err == nil {
			started = parsed
		}
	}
	ticker := time.NewTicker(a.applyPollInterval)
	defer ticker.Stop()
	for {
		ready, running, err := a.currentGenerationProgress(manifest)
		if err != nil {
			a.failApplyConfigTask(id, err)
			return
		}
		if ready {
			a.completeApplyConfigTask(id, fmt.Sprintf("配置已应用，%d 个助手正在运行", running), composeHash, dufsHash, inputHash, hermesImage, true)
			return
		}
		slow := time.Since(started) >= a.applySlowAfter
		if err := a.updateApplyConfigStatus(id, func(status *ApplyConfigStatus) {
			status.RunningProfiles = running
			status.State = applyStateWaiting
			if slow {
				status.State = applyStateSlow
			}
			status.Message = waitingApplyMessage(running, status.RunnableProfiles, slow)
		}); err != nil {
			a.failApplyConfigTask(id, err)
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (a *App) completeApplyConfigTask(id string, message string, composeHash string, dufsHash string, inputHash string, hermesImage string, hermesApplied bool) {
	unchanged, err := a.finalizeRebuildAppliedSnapshot(composeHash, dufsHash, inputHash, hermesImage, hermesApplied)
	if err != nil {
		a.failApplyConfigTask(id, err)
		return
	}
	if !unchanged {
		message += "；应用期间检测到新的修改，请再次应用"
	}
	if err := a.succeedApplyConfigTask(id, message); err != nil {
		a.failApplyConfigTask(id, err)
		return
	}
}

func (a *App) applyRuntimeInputHash(composeHash string, dufsHash string) (string, error) {
	hash := sha256.New()
	_, _ = fmt.Fprintf(hash, "compose:%s\ndufs:%s\n", composeHash, dufsHash)
	paths := []string{a.profilesPath()}
	registry, err := a.readProfileRegistry()
	if err != nil {
		return "", err
	}
	for _, profile := range registry.Profiles {
		paths = append(paths,
			a.profileConfigPath(profile.ID),
			a.profileEnvPath(profile.ID),
			a.profileSoulPath(profile.ID),
			filepath.Join(a.profileDataDir(profile.ID), "skills"),
		)
	}
	for _, path := range paths {
		if err := hashRuntimeInputPath(hash, path); err != nil {
			return "", err
		}
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func hashRuntimeInputPath(hash io.Writer, path string) error {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		_, _ = fmt.Fprintf(hash, "missing:%s\n", path)
		return nil
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return hashRuntimeInputFile(hash, path, path, info)
	}
	return filepath.Walk(path, func(item string, itemInfo os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		return hashRuntimeInputFile(hash, path, item, itemInfo)
	})
}

func hashRuntimeInputFile(hash io.Writer, root string, path string, info os.FileInfo) error {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(hash, "path:%s:%s:%s\n", root, filepath.ToSlash(rel), info.Mode().String())
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(path)
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintf(hash, "link:%s\n", target)
		return nil
	}
	if !info.Mode().IsRegular() {
		return nil
	}
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.Copy(hash, file)
	return err
}

func (a *App) currentGenerationProgress(manifest RuntimeManifest) (bool, int, error) {
	data, err := os.ReadFile(a.runtimeStatusPath())
	if err != nil {
		if os.IsNotExist(err) {
			return false, 0, nil
		}
		return false, 0, fmt.Errorf("无法读取助手运行状态：%w", err)
	}
	var status RuntimeStatus
	if err := json.Unmarshal(data, &status); err != nil {
		return false, 0, fmt.Errorf("助手运行状态无效：%w", err)
	}
	if status.Generation != manifest.Generation {
		return false, 0, nil
	}
	running := 0
	for _, profile := range manifest.Profiles {
		if !profile.Runnable {
			continue
		}
		profileStatus, ok := status.Profiles[profile.ID]
		if !ok {
			continue
		}
		if profileStatus.State == "failed" {
			message := strings.TrimSpace(redact(profileStatus.Message))
			if message == "" {
				message = "启动失败"
			}
			return false, running, fmt.Errorf("助手 %s %s", firstNonEmpty(profile.Name, profile.ID), message)
		}
		if profileStatus.State == "running" {
			running++
		}
	}
	return running == runnableProfileCount(manifest), running, nil
}

func runnableProfileCount(manifest RuntimeManifest) int {
	count := 0
	for _, profile := range manifest.Profiles {
		if profile.Runnable {
			count++
		}
	}
	return count
}

func waitingApplyMessage(running int, total int, slow bool) string {
	if slow {
		return fmt.Sprintf("助手启动时间较长，仍在继续等待（%d/%d）", running, total)
	}
	return fmt.Sprintf("容器已启动，正在等待助手启动（%d/%d）", running, total)
}

func (a *App) succeedApplyConfigTask(id string, message string) error {
	if err := a.updateApplyConfigStatus(id, func(status *ApplyConfigStatus) {
		status.State = applyStateSucceeded
		status.Phase = "complete"
		status.Message = message
		status.Active = false
		status.Error = ""
		status.RunningProfiles = status.RunnableProfiles
		status.CompletedAt = time.Now().UTC().Format(time.RFC3339)
	}); err != nil {
		return fmt.Errorf("无法保存应用配置完成状态：%w", err)
	}
	a.clearApplyConfigTask(id)
	return nil
}

func (a *App) failApplyConfigTask(id string, taskErr error) {
	message := redact(taskErr.Error())
	if err := a.updateApplyConfigStatus(id, func(status *ApplyConfigStatus) {
		status.State = applyStateFailed
		status.Phase = "failed"
		status.Message = message
		status.Active = false
		status.Error = message
		status.CompletedAt = time.Now().UTC().Format(time.RFC3339)
	}); err != nil {
		persistMessage := redact(fmt.Sprintf("%s；保存任务失败状态失败：%v", message, err))
		a.emit("apply:status", ApplyConfigStatus{
			ID:          id,
			State:       applyStateFailed,
			Phase:       "failed",
			Message:     persistMessage,
			Error:       persistMessage,
			Active:      false,
			CompletedAt: time.Now().UTC().Format(time.RFC3339),
		})
		a.emit("docker:progress", StreamEvent{Line: persistMessage})
	}
	a.clearApplyConfigTask(id)
}

func (a *App) clearApplyConfigTask(id string) {
	a.applyMu.Lock()
	defer a.applyMu.Unlock()
	if a.applyActiveID != id {
		return
	}
	if a.applyCancel != nil {
		a.applyCancel()
	}
	a.applyCancel = nil
	a.applyActiveID = ""
	if a.applyOwnsOperation {
		a.applyOwnsOperation = false
		a.operationMu.Unlock()
	}
}

func (a *App) cancelApplyConfigTask() {
	a.applyMu.Lock()
	defer a.applyMu.Unlock()
	if a.applyCancel != nil {
		a.applyCancel()
	}
	a.applyCancel = nil
	a.applyActiveID = ""
	if a.applyOwnsOperation {
		a.applyOwnsOperation = false
		a.operationMu.Unlock()
	}
}

func (a *App) resumeApplyConfigTask() {
	status := a.readApplyConfigStatus()
	if !status.Active {
		return
	}
	if status.ComposeHash == "" || status.DufsHash == "" || status.InputHash == "" {
		a.failApplyConfigTask(status.ID, fmt.Errorf("上次应用配置缺少恢复信息，请重试"))
		return
	}
	var manifest RuntimeManifest
	if status.Strategy != "dufs-only" {
		if status.Generation == "" {
			a.failApplyConfigTask(status.ID, fmt.Errorf("上次应用配置在生成运行配置前中断，请重试"))
			return
		}
		data, err := os.ReadFile(a.runtimeManifestPath())
		if err != nil || json.Unmarshal(data, &manifest) != nil || manifest.Generation != status.Generation {
			a.failApplyConfigTask(status.ID, fmt.Errorf("上次应用配置的运行配置已失效，请重试"))
			return
		}
	}
	if (status.State == applyStateWaiting || status.State == applyStateSlow) && a.containerStatus(context.Background()) != "running" {
		a.failApplyConfigTask(status.ID, fmt.Errorf("上次应用配置中断且 Hermes 容器未运行，请重试"))
		return
	}
	if !a.operationMu.TryLock() {
		a.failApplyConfigTask(status.ID, fmt.Errorf("启动器正在执行其他实例操作，无法恢复应用任务"))
		return
	}
	a.applyMu.Lock()
	if a.applyActiveID != "" {
		a.applyMu.Unlock()
		a.operationMu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.applyCancel = cancel
	a.applyActiveID = status.ID
	a.applyOwnsOperation = true
	a.applyMu.Unlock()
	if status.State == applyStateApplying {
		go a.resumeApplyingConfigTask(ctx, status, manifest)
		return
	}
	if status.State != applyStateWaiting && status.State != applyStateSlow {
		a.failApplyConfigTask(status.ID, fmt.Errorf("上次应用配置停在不可恢复阶段，请重试"))
		return
	}
	if err := a.updateApplyConfigStatus(status.ID, func(current *ApplyConfigStatus) {
		current.State = applyStateWaiting
		current.Phase = "profiles"
		current.Message = "已恢复应用任务，正在等待助手启动"
	}); err != nil {
		a.failApplyConfigTask(status.ID, err)
		return
	}
	go a.monitorApplyConfigTask(ctx, status.ID, manifest, status.ComposeHash, status.DufsHash, status.InputHash, status.HermesImage)
}

func (a *App) resumeApplyingConfigTask(ctx context.Context, status ApplyConfigStatus, manifest RuntimeManifest) {
	currentComposeHash, err := a.composeRuntimeHash()
	if err != nil {
		a.failApplyConfigTask(status.ID, err)
		return
	}
	currentDufsHash, err := a.dufsRuntimeHash()
	if err != nil {
		a.failApplyConfigTask(status.ID, err)
		return
	}
	currentHash, err := a.applyRuntimeInputHash(currentComposeHash, currentDufsHash)
	if err != nil {
		a.failApplyConfigTask(status.ID, err)
		return
	}
	if currentComposeHash != status.ComposeHash || currentDufsHash != status.DufsHash || currentHash != status.InputHash {
		a.failApplyConfigTask(status.ID, fmt.Errorf("应用中断后配置已发生变化，请重新应用"))
		return
	}
	settings, err := a.readComposeSettings()
	if err != nil {
		a.failApplyConfigTask(status.ID, err)
		return
	}
	if status.Strategy == "dufs-only" {
		if err := a.applyDufsRuntimeContext(ctx, settings, status.DufsRecreate); err != nil {
			if ctx.Err() == nil {
				a.failApplyConfigTask(status.ID, err)
			}
			return
		}
		a.completeApplyConfigTask(status.ID, "文件管理配置已应用", status.ComposeHash, status.DufsHash, status.InputHash, status.HermesImage, false)
		return
	}
	if status.Strategy != "restart" && status.Strategy != "recreate" {
		a.failApplyConfigTask(status.ID, fmt.Errorf("上次应用配置的执行策略无效，请重试"))
		return
	}
	if err := a.applyComposeRuntimeContext(ctx, status.Strategy == "recreate"); err != nil {
		if ctx.Err() == nil {
			a.failApplyConfigTask(status.ID, err)
		}
		return
	}
	if err := a.applyDufsRuntimeContext(ctx, settings, status.DufsRecreate); err != nil {
		if ctx.Err() == nil {
			a.failApplyConfigTask(status.ID, err)
		}
		return
	}
	if err := a.updateApplyConfigStatus(status.ID, func(current *ApplyConfigStatus) {
		current.State = applyStateWaiting
		current.Phase = "profiles"
		current.Message = waitingApplyMessage(0, current.RunnableProfiles, false)
	}); err != nil {
		a.failApplyConfigTask(status.ID, err)
		return
	}
	a.monitorApplyConfigTask(ctx, status.ID, manifest, status.ComposeHash, status.DufsHash, status.InputHash, status.HermesImage)
}

func (a *App) readApplyConfigStatus() ApplyConfigStatus {
	a.applyMu.Lock()
	defer a.applyMu.Unlock()
	status, err := a.readApplyConfigStatusUnlocked()
	if err != nil {
		return ApplyConfigStatus{State: applyStateIdle}
	}
	return status
}

func (a *App) readApplyConfigStatusUnlocked() (ApplyConfigStatus, error) {
	var status ApplyConfigStatus
	data, err := os.ReadFile(a.applyStatusPath())
	if err != nil {
		return ApplyConfigStatus{State: applyStateIdle}, err
	}
	if err := json.Unmarshal(data, &status); err != nil {
		return ApplyConfigStatus{}, err
	}
	if status.State == "" {
		status.State = applyStateIdle
	}
	return status, nil
}

func (a *App) updateApplyConfigStatus(id string, update func(*ApplyConfigStatus)) error {
	a.applyMu.Lock()
	status, err := a.readApplyConfigStatusUnlocked()
	if err != nil {
		a.applyMu.Unlock()
		return err
	}
	if status.ID != id {
		a.applyMu.Unlock()
		return fmt.Errorf("应用配置任务已被替换")
	}
	before := status
	update(&status)
	status.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if status == before {
		a.applyMu.Unlock()
		return nil
	}
	if err := a.writeApplyConfigStatusUnlocked(status); err != nil {
		a.applyMu.Unlock()
		return err
	}
	a.applyMu.Unlock()
	a.emit("apply:status", status)
	return nil
}

func (a *App) writeApplyConfigStatusUnlocked(status ApplyConfigStatus) error {
	data, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return err
	}
	return atomicWriteFile(a.applyStatusPath(), append(data, '\n'), 0644)
}
