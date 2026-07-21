package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	postUpdateStatePending   = "pending"
	postUpdateStateWaiting   = "waiting"
	postUpdateStateSyncing   = "syncing"
	postUpdateStateApplying  = "applying"
	postUpdateStateSucceeded = "succeeded"
	postUpdateStateFailed    = "failed"
)

func (a *App) preparePostUpdateTask() error {
	if installedUpdateToken == "" && !launchAfterUpdateMode {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	wasRunning := a.containerStatus(ctx) == "running"
	cancel()

	a.updateMu.Lock()
	defer a.updateMu.Unlock()
	state, err := a.readUpdateState()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("读取升级后处理状态失败：%w", err)
	}
	if state.PostUpdateVersion == appVersion && state.PostUpdateState != "" {
		return nil
	}
	launcherState, err := a.readState()
	if err != nil {
		return fmt.Errorf("读取运行依赖版本失败：%w", err)
	}
	state.SchemaVersion = 1
	state.PostUpdateVersion = appVersion
	state.PostUpdateTemplateVersion = templateVersion
	state.PostUpdateState = postUpdateStatePending
	state.PostUpdateMessage = "等待同步升级后的内置内容"
	state.PostUpdateError = ""
	state.PostUpdateUpdatedAt = time.Now().UTC().Format(time.RFC3339)
	state.PostUpdateContainerWasRunning = wasRunning
	state.PostUpdateApplyID = ""
	state.PostUpdateSyncFailures = 0
	state.PostUpdateContentChanged = false
	state.PostUpdateRuntimeDepsChanged = launcherState.RuntimeDependencyVersion != runtimeDependencyBundleVersion
	state.PostUpdateWritePending = false
	if err := a.writeUpdateState(state); err != nil {
		return fmt.Errorf("保存升级后处理状态失败：%w", err)
	}
	return nil
}

func (a *App) startPostUpdateTask() {
	state, err := a.readUpdateState()
	if err != nil || state.PostUpdateVersion != appVersion || !postUpdateTaskActive(state.PostUpdateState) {
		return
	}
	if !a.postUpdateMu.TryLock() {
		return
	}
	go func() {
		defer a.postUpdateMu.Unlock()
		a.runPostUpdateTask(state)
	}()
}

func (a *App) queuePostUpdateTask() {
	go func() {
		a.postUpdateMu.Lock()
		defer a.postUpdateMu.Unlock()
		state, err := a.readUpdateState()
		if err != nil || state.PostUpdateVersion != appVersion || !postUpdateTaskActive(state.PostUpdateState) {
			return
		}
		a.runPostUpdateTask(state)
	}()
}

func postUpdateTaskActive(state string) bool {
	switch state {
	case postUpdateStatePending, postUpdateStateWaiting, postUpdateStateSyncing, postUpdateStateApplying:
		return true
	default:
		return false
	}
}

func (a *App) runPostUpdateTask(initial updateState) {
	if initial.PostUpdateState == postUpdateStateApplying && initial.PostUpdateApplyID != "" {
		status, err := a.waitForApplyConfigTask(initial.PostUpdateApplyID)
		if err != nil {
			a.failPostUpdateTask(err)
			return
		}
		if status.State == applyStateSucceeded && initial.PostUpdateSyncFailures > 0 {
			_ = a.updatePostUpdateState(func(state *updateState) {
				state.PostUpdateContentChanged = false
				state.PostUpdateRuntimeDepsChanged = false
			})
		}
		a.finishPostUpdateAfterApply(status, initial.PostUpdateSyncFailures)
		return
	}

	if status := a.readApplyConfigStatus(); status.Active {
		_ = a.updatePostUpdateState(func(state *updateState) {
			state.PostUpdateState = postUpdateStateWaiting
			state.PostUpdateMessage = "等待当前应用配置任务完成"
		})
		if _, err := a.waitForApplyConfigTask(status.ID); err != nil {
			a.failPostUpdateTask(err)
			return
		}
	}

	a.operationMu.Lock()
	operationHeld := true
	defer func() {
		if operationHeld {
			a.operationMu.Unlock()
		}
	}()
	registry, err := a.readProfileRegistry()
	if err != nil {
		a.failPostUpdateTask(err)
		return
	}
	profileIDs := make([]string, 0, len(registry.Profiles))
	for _, profile := range registry.Profiles {
		profileIDs = append(profileIDs, profile.ID)
	}
	if err := a.updatePostUpdateState(func(state *updateState) {
		state.PostUpdateState = postUpdateStateSyncing
		state.PostUpdateMessage = "正在安全同步所有助手的内置人格和技能"
		state.PostUpdateError = ""
		state.PostUpdateApplyID = ""
	}); err != nil {
		a.failPostUpdateTask(err)
		return
	}
	result, syncErr := a.syncBundledContent(BundledContentSyncRequest{
		TargetProfileIDs: profileIDs,
		SyncSoul:         true,
		SyncSkills:       true,
	}, false, func() error {
		return a.updatePostUpdateState(func(state *updateState) {
			state.PostUpdateWritePending = true
		})
	})
	changed := initial.PostUpdateContentChanged || initial.PostUpdateRuntimeDepsChanged || result.Added+result.Updated > 0
	syncFailure := bundledSyncFailureMessage(result)
	if err := a.updatePostUpdateState(func(state *updateState) {
		state.PostUpdateSyncFailures = result.Failed
		state.PostUpdateContentChanged = state.PostUpdateContentChanged || changed
		state.PostUpdateWritePending = false
	}); err != nil {
		a.failPostUpdateTask(err)
		return
	}
	if syncErr != nil {
		a.failPostUpdateTask(syncErr)
		return
	}
	if !changed && initial.PostUpdateWritePending {
		if err := a.markRebuildRequired(); err != nil {
			a.failPostUpdateTask(err)
			return
		}
		a.operationMu.Unlock()
		operationHeld = false
		if syncFailure != "" {
			a.failPostUpdateTask(errors.New(syncFailure))
			return
		}
		a.succeedPostUpdateTask("内置内容已检查；上次同步曾中断，已保留待应用状态，请手动应用配置")
		return
	}

	if !changed {
		a.operationMu.Unlock()
		operationHeld = false
		if syncFailure != "" {
			a.failPostUpdateTask(errors.New(syncFailure))
			return
		}
		a.succeedPostUpdateTask("所有助手的内置内容已检查，无需应用配置")
		return
	}
	if !initial.PostUpdateContainerWasRunning {
		a.operationMu.Unlock()
		operationHeld = false
		if syncFailure != "" {
			a.failPostUpdateTask(errors.New(syncFailure))
			return
		}
		a.succeedPostUpdateTask("内置内容已同步；升级前未确认 Hermes 正在运行，将在下次启动时应用")
		return
	}
	statusCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	currentContainerStatus := a.containerStatus(statusCtx)
	cancel()
	if !shouldAutoApplyPostUpdate(changed, initial.PostUpdateContainerWasRunning, currentContainerStatus) {
		a.operationMu.Unlock()
		operationHeld = false
		if syncFailure != "" {
			a.failPostUpdateTask(errors.New(syncFailure))
			return
		}
		a.succeedPostUpdateTask("内置内容已同步；Hermes 当前未运行，将在下次启动时应用")
		return
	}

	applyID := uuid.NewString()
	if err := a.updatePostUpdateState(func(state *updateState) {
		state.PostUpdateState = postUpdateStateApplying
		state.PostUpdateMessage = "正在应用升级后的配置"
		state.PostUpdateApplyID = applyID
	}); err != nil {
		a.failPostUpdateTask(err)
		return
	}
	if err := a.startApplyConfigTaskWithOperationID(true, applyID); err != nil {
		operationHeld = false
		a.failPostUpdateTask(err)
		return
	}
	operationHeld = false
	applyStatus := a.readApplyConfigStatus()
	if applyStatus.ID != applyID {
		a.failPostUpdateTask(errors.New("升级后应用配置任务未正常启动"))
		return
	}
	applyStatus, err = a.waitForApplyConfigTask(applyStatus.ID)
	if err != nil {
		a.failPostUpdateTask(err)
		return
	}
	if syncFailure != "" && applyStatus.State == applyStateSucceeded {
		_ = a.updatePostUpdateState(func(state *updateState) {
			state.PostUpdateContentChanged = false
			state.PostUpdateRuntimeDepsChanged = false
		})
		a.failPostUpdateTask(errors.New(syncFailure))
		return
	}
	a.finishPostUpdateAfterApply(applyStatus, result.Failed)
}

func shouldAutoApplyPostUpdate(changed bool, wasRunning bool, currentStatus string) bool {
	return changed && wasRunning && currentStatus == "running"
}

func (a *App) waitForApplyConfigTask(id string) (ApplyConfigStatus, error) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		status := a.readApplyConfigStatus()
		if status.ID != id {
			return status, fmt.Errorf("应用配置任务状态已变更，请重试")
		}
		if !status.Active {
			return status, nil
		}
		<-ticker.C
	}
}

func (a *App) finishPostUpdateAfterApply(status ApplyConfigStatus, syncFailures int) {
	if status.State != applyStateSucceeded {
		message := firstNonEmpty(strings.TrimSpace(status.Error), strings.TrimSpace(status.Message), "应用配置失败")
		a.failPostUpdateTask(errors.New(message))
		return
	}
	if syncFailures > 0 {
		a.failPostUpdateTask(fmt.Errorf("%d 个助手的内置内容同步失败", syncFailures))
		return
	}
	a.succeedPostUpdateTask("内置内容已同步，升级后配置已应用")
}

func bundledSyncFailureMessage(result BundledContentSyncResult) string {
	if result.Failed == 0 {
		return ""
	}
	details := make([]string, 0, 3)
	for _, item := range result.Results {
		if item.Success || strings.TrimSpace(item.Error) == "" {
			continue
		}
		details = append(details, fmt.Sprintf("%s：%s", item.ProfileID, item.Error))
		if len(details) == 3 {
			break
		}
	}
	message := fmt.Sprintf("%d 个助手的内置内容同步失败", result.Failed)
	if len(details) > 0 {
		message += "；" + strings.Join(details, "；")
	}
	return redact(message)
}

func (a *App) updatePostUpdateState(update func(*updateState)) error {
	a.updateMu.Lock()
	defer a.updateMu.Unlock()
	state, err := a.readUpdateState()
	if err != nil {
		return err
	}
	if state.PostUpdateVersion != appVersion {
		return errors.New("升级后处理任务已过期")
	}
	update(&state)
	state.PostUpdateUpdatedAt = time.Now().UTC().Format(time.RFC3339)
	return a.writeUpdateState(state)
}

func (a *App) succeedPostUpdateTask(message string) {
	if err := a.updatePostUpdateState(func(state *updateState) {
		state.PostUpdateState = postUpdateStateSucceeded
		state.PostUpdateMessage = message
		state.PostUpdateError = ""
	}); err != nil {
		a.emit("docker:progress", StreamEvent{Line: redact(fmt.Sprintf("保存升级后处理结果失败：%v", err))})
	}
}

func (a *App) failPostUpdateTask(taskErr error) {
	message := redact(taskErr.Error())
	if err := a.updatePostUpdateState(func(state *updateState) {
		state.PostUpdateState = postUpdateStateFailed
		state.PostUpdateMessage = "升级后处理未完成"
		state.PostUpdateError = message
	}); err != nil {
		a.emit("docker:progress", StreamEvent{Line: redact(fmt.Sprintf("保存升级后处理失败状态失败：%v", err))})
	}
}

func (a *App) RetryPostUpdate() (UpdateStatus, error) {
	a.updateMu.Lock()
	state, err := a.readUpdateState()
	if err != nil {
		a.updateMu.Unlock()
		return a.updateStatus(), err
	}
	if state.PostUpdateVersion != appVersion || state.PostUpdateState != postUpdateStateFailed {
		a.updateMu.Unlock()
		return a.updateStatus(), errors.New("当前没有可重试的升级后处理任务")
	}
	state.PostUpdateState = postUpdateStatePending
	state.PostUpdateMessage = "等待重新同步升级后的内置内容"
	state.PostUpdateError = ""
	state.PostUpdateUpdatedAt = time.Now().UTC().Format(time.RFC3339)
	state.PostUpdateApplyID = ""
	state.PostUpdateSyncFailures = 0
	err = a.writeUpdateState(state)
	a.updateMu.Unlock()
	if err != nil {
		return a.updateStatus(), err
	}
	a.queuePostUpdateTask()
	return a.updateStatus(), nil
}
