package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
)

func TestApplyConfigTaskRejectsDuplicateSubmission(t *testing.T) {
	app := newTestApp(t)
	installFakeDocker(t, true)
	t.Cleanup(app.cancelApplyConfigTask)
	if err := app.RebuildHermes(); err != nil {
		t.Fatal(err)
	}
	if err := app.RebuildHermes(); err == nil || !strings.Contains(err.Error(), "正在执行") {
		t.Fatalf("expected duplicate task error, got %v", err)
	}
	waiting := waitForApplyState(t, app, applyStateWaiting)
	writeRuntimeStatusForApply(t, app, waiting.Generation)
	waitForApplyState(t, app, applyStateSucceeded)
}

func TestApplyConfigSlowThresholdIsTwoMinutes(t *testing.T) {
	if got := NewApp().applySlowAfter; got != 2*time.Minute {
		t.Fatalf("slow threshold = %s, want 2m", got)
	}
}

func TestApplyConfigTaskWaitsPastSlowThresholdAndClearsNeedsRebuild(t *testing.T) {
	app := newTestApp(t)
	installFakeDocker(t, true)
	app.applySlowAfter = 20 * time.Millisecond
	app.applyPollInterval = 5 * time.Millisecond
	t.Cleanup(app.cancelApplyConfigTask)
	state, err := app.readState()
	if err != nil {
		t.Fatal(err)
	}
	state.NeedsRebuild = true
	if err := app.writeState(state); err != nil {
		t.Fatal(err)
	}
	if err := app.RebuildHermes(); err != nil {
		t.Fatal(err)
	}
	slow := waitForApplyState(t, app, applyStateSlow)
	if !slow.Active || slow.Generation == "" {
		t.Fatalf("slow task became terminal: %+v", slow)
	}
	writeRuntimeStatusForApply(t, app, slow.Generation)
	done := waitForApplyState(t, app, applyStateSucceeded)
	if done.Active {
		t.Fatalf("completed task is still active: %+v", done)
	}
	state, err = app.readState()
	if err != nil {
		t.Fatal(err)
	}
	if state.NeedsRebuild {
		t.Fatal("delayed successful gateway did not clear NeedsRebuild")
	}
}

func TestApplyConfigTaskIgnoresOldGeneration(t *testing.T) {
	app := newTestApp(t)
	installFakeDocker(t, true)
	app.applySlowAfter = time.Second
	app.applyPollInterval = 5 * time.Millisecond
	t.Cleanup(app.cancelApplyConfigTask)
	if err := app.RebuildHermes(); err != nil {
		t.Fatal(err)
	}
	waiting := waitForApplyState(t, app, applyStateWaiting)
	writeRuntimeStatusForApply(t, app, "old-generation")
	time.Sleep(30 * time.Millisecond)
	if current := app.readApplyConfigStatus(); !current.Active || current.State == applyStateSucceeded {
		t.Fatalf("old generation completed task: %+v", current)
	}
	writeRuntimeStatusForApply(t, app, waiting.Generation)
	waitForApplyState(t, app, applyStateSucceeded)
}

func TestResumeApplyConfigTaskCompletesPersistedGeneration(t *testing.T) {
	app := newTestApp(t)
	installFakeDocker(t, true)
	registry, err := app.readProfileRegistry()
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := app.buildRuntimeManifest(registry)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(manifest)
	if err := atomicWriteFile(app.runtimeManifestPath(), data, 0644); err != nil {
		t.Fatal(err)
	}
	composeHash, err := app.composeRuntimeHash()
	if err != nil {
		t.Fatal(err)
	}
	dufsHash, err := app.dufsRuntimeHash()
	if err != nil {
		t.Fatal(err)
	}
	inputHash, err := app.applyRuntimeInputHash(composeHash, dufsHash)
	if err != nil {
		t.Fatal(err)
	}
	state, err := app.readState()
	if err != nil {
		t.Fatal(err)
	}
	status := ApplyConfigStatus{
		ID: "persisted-task", Generation: manifest.Generation, State: applyStateWaiting,
		Phase: "gateway", Strategy: "restart", Active: true, StartedAt: time.Now().UTC().Format(time.RFC3339),
		ComposeHash: composeHash, DufsHash: dufsHash, InputHash: inputHash, HermesImage: state.HermesImage,
	}
	if err := app.writeApplyConfigStatusUnlocked(status); err != nil {
		t.Fatal(err)
	}
	writeRuntimeStatusForApply(t, app, manifest.Generation)

	reopened := NewApp()
	reopened.instanceRoot = app.instanceRoot
	reopened.applyPollInterval = 5 * time.Millisecond
	t.Cleanup(reopened.cancelApplyConfigTask)
	reopened.resumeApplyConfigTask()
	waitForApplyState(t, reopened, applyStateSucceeded)
}

func TestApplyConfigTaskPreservesNewChangesMadeWhileWaiting(t *testing.T) {
	app := newTestApp(t)
	installFakeDocker(t, true)
	app.applyPollInterval = 5 * time.Millisecond
	t.Cleanup(app.cancelApplyConfigTask)
	if err := app.RebuildHermes(); err != nil {
		t.Fatal(err)
	}
	waiting := waitForApplyState(t, app, applyStateWaiting)
	if err := atomicWriteFile(app.profileSoulPath(defaultProfileID), []byte("changed while applying\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := app.markRebuildRequired(); err != nil {
		t.Fatal(err)
	}
	writeRuntimeStatusForApply(t, app, waiting.Generation)
	done := waitForApplyState(t, app, applyStateSucceeded)
	if !strings.Contains(done.Message, "新的修改") {
		t.Fatalf("completion message did not report concurrent changes: %+v", done)
	}
	state, err := app.readState()
	if err != nil {
		t.Fatal(err)
	}
	if !state.NeedsRebuild {
		t.Fatal("late success cleared a configuration change made while waiting")
	}
}

func TestContainerLifecycleIsBlockedDuringApplyConfig(t *testing.T) {
	app := newTestApp(t)
	installFakeDocker(t, true)
	t.Cleanup(app.cancelApplyConfigTask)
	if err := app.RebuildHermes(); err != nil {
		t.Fatal(err)
	}
	if err := app.StopHermes(); err == nil || !strings.Contains(err.Error(), "正在应用配置") {
		t.Fatalf("stop was not blocked during apply: %v", err)
	}
	waiting := waitForApplyState(t, app, applyStateWaiting)
	writeRuntimeStatusForApply(t, app, waiting.Generation)
	waitForApplyState(t, app, applyStateSucceeded)
}

func TestApplyConfigIsBlockedDuringExclusiveInstanceOperation(t *testing.T) {
	app := newTestApp(t)
	release, err := app.beginExclusiveOperation("测试操作")
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	if err := app.RebuildHermes(); err == nil || !strings.Contains(err.Error(), "其他实例操作") {
		t.Fatalf("apply task was not blocked by exclusive operation: %v", err)
	}
}

func TestResumeApplyingConfigTaskRerunsDockerStrategy(t *testing.T) {
	app := newTestApp(t)
	installFakeDocker(t, true)
	registry, err := app.readProfileRegistry()
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := app.buildRuntimeManifest(registry)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(manifest)
	if err := atomicWriteFile(app.runtimeManifestPath(), data, 0644); err != nil {
		t.Fatal(err)
	}
	composeHash, _ := app.composeRuntimeHash()
	dufsHash, _ := app.dufsRuntimeHash()
	inputHash, _ := app.applyRuntimeInputHash(composeHash, dufsHash)
	state, _ := app.readState()
	status := ApplyConfigStatus{
		ID: "resume-applying", Generation: manifest.Generation, State: applyStateApplying, Phase: "docker",
		Strategy: "restart", Active: true, StartedAt: time.Now().UTC().Format(time.RFC3339),
		ComposeHash: composeHash, DufsHash: dufsHash, InputHash: inputHash, HermesImage: state.HermesImage,
	}
	if err := app.writeApplyConfigStatusUnlocked(status); err != nil {
		t.Fatal(err)
	}
	app.applyPollInterval = 5 * time.Millisecond
	t.Cleanup(app.cancelApplyConfigTask)
	app.resumeApplyConfigTask()
	waiting := waitForApplyState(t, app, applyStateWaiting)
	writeRuntimeStatusForApply(t, app, waiting.Generation)
	waitForApplyState(t, app, applyStateSucceeded)
	log, _ := os.ReadFile(fakeDockerLogPath(t))
	if !strings.Contains(string(log), "compose restart hermes") {
		t.Fatalf("resume did not rerun the persisted strategy: %s", log)
	}
}

func TestResumeApplyingDufsOnlyTask(t *testing.T) {
	app := newTestApp(t)
	installFakeDocker(t, true)
	composeHash, _ := app.composeRuntimeHash()
	dufsHash, _ := app.dufsRuntimeHash()
	inputHash, _ := app.applyRuntimeInputHash(composeHash, dufsHash)
	state, _ := app.readState()
	status := ApplyConfigStatus{
		ID: "resume-dufs", State: applyStateApplying, Phase: "dufs", Strategy: "dufs-only",
		Active: true, StartedAt: time.Now().UTC().Format(time.RFC3339),
		ComposeHash: composeHash, DufsHash: dufsHash, InputHash: inputHash, HermesImage: state.HermesImage,
	}
	if err := app.writeApplyConfigStatusUnlocked(status); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.cancelApplyConfigTask)
	app.resumeApplyConfigTask()
	waitForApplyState(t, app, applyStateSucceeded)
	log, _ := os.ReadFile(fakeDockerLogPath(t))
	if !strings.Contains(string(log), "compose up -d dufs") {
		t.Fatalf("dufs-only resume did not rerun Dufs: %s", log)
	}
}

func TestApplyConfigStrategies(t *testing.T) {
	t.Run("restart", func(t *testing.T) {
		app := newTestApp(t)
		installFakeDocker(t, true)
		state, _ := app.readState()
		hash, _ := app.composeRuntimeHash()
		state.LastAppliedComposeHash = hash
		state.NeedsRebuild = true
		if err := app.writeState(state); err != nil {
			t.Fatal(err)
		}
		runApplyToSuccess(t, app)
		if status := app.readApplyConfigStatus(); status.Strategy != "restart" {
			t.Fatalf("strategy = %q", status.Strategy)
		}
		log, _ := os.ReadFile(fakeDockerLogPath(t))
		if !strings.Contains(string(log), "compose restart hermes") {
			t.Fatalf("restart command missing: %s", log)
		}
	})
	t.Run("force recreate", func(t *testing.T) {
		app := newTestApp(t)
		installFakeDocker(t, true)
		runApplyToSuccess(t, app)
		if status := app.readApplyConfigStatus(); status.Strategy != "recreate" {
			t.Fatalf("strategy = %q", status.Strategy)
		}
		log, _ := os.ReadFile(fakeDockerLogPath(t))
		if !strings.Contains(string(log), "compose up -d --force-recreate --remove-orphans hermes") {
			t.Fatalf("force recreate command missing: %s", log)
		}
	})
	t.Run("dufs only", func(t *testing.T) {
		app := newTestApp(t)
		installFakeDocker(t, true)
		state, _ := app.readState()
		state.NeedsRebuild = true
		state.PendingDufsOnly = true
		state.LastAppliedDufsHash = "old"
		if err := app.writeState(state); err != nil {
			t.Fatal(err)
		}
		if err := app.RebuildHermes(); err != nil {
			t.Fatal(err)
		}
		status := waitForApplyState(t, app, applyStateSucceeded)
		if status.Strategy != "dufs-only" {
			t.Fatalf("strategy = %q", status.Strategy)
		}
	})
}

func runApplyToSuccess(t *testing.T, app *App) {
	t.Helper()
	app.applyPollInterval = 5 * time.Millisecond
	t.Cleanup(app.cancelApplyConfigTask)
	if err := app.RebuildHermes(); err != nil {
		t.Fatal(err)
	}
	waiting := waitForApplyState(t, app, applyStateWaiting)
	writeRuntimeStatusForApply(t, app, waiting.Generation)
	waitForApplyState(t, app, applyStateSucceeded)
}

func waitForApplyState(t *testing.T, app *App, want string) ApplyConfigStatus {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		status := app.readApplyConfigStatus()
		if status.State == want || (want == applyStateWaiting && status.State == applyStateSlow) {
			return status
		}
		if status.State == applyStateFailed {
			t.Fatalf("apply task failed while waiting for %s: %+v", want, status)
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for apply state %s; current=%+v", want, app.readApplyConfigStatus())
	return ApplyConfigStatus{}
}

func writeRuntimeStatusForApply(t *testing.T, app *App, generation string) {
	t.Helper()
	status := RuntimeStatus{Generation: generation, UpdatedAt: time.Now().UTC().Format(time.RFC3339), Profiles: map[string]RuntimeProfileStatus{}}
	data, err := json.Marshal(status)
	if err != nil {
		t.Fatal(err)
	}
	if err := atomicWriteFile(app.runtimeStatusPath(), data, 0644); err != nil {
		t.Fatal(err)
	}
}
