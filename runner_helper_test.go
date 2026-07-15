package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestProfileRunnerCandidatesIncludePackagedLinuxPaths(t *testing.T) {
	root := filepath.FromSlash("/home/user/.hermes-dock")
	got := profileRunnerCandidatesForExecutable(root, "amd64", filepath.FromSlash("/opt/hermes-dock/hermes-dock"))
	want := []string{
		filepath.FromSlash("/opt/hermes-dock/hermes-profile-runner-linux-amd64"),
		filepath.FromSlash("/opt/hermes-dock/hermes-profile-runner"),
		filepath.FromSlash("/home/user/.hermes-dock/build/profile-runner/hermes-profile-runner-linux-amd64"),
		filepath.FromSlash("/home/user/.hermes-dock/build/profile-runner/hermes-profile-runner"),
	}
	if len(got) != len(want) {
		t.Fatalf("candidate count = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("candidate %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestProfileRunnerCandidatesIncludeMacOSAppBundlePath(t *testing.T) {
	root := filepath.FromSlash("/Users/user/.hermes-dock")
	got := profileRunnerCandidatesForExecutable(root, "arm64", filepath.FromSlash("/Applications/hermes-dock.app/Contents/MacOS/hermes-dock"))
	want := filepath.FromSlash("/Applications/hermes-dock.app/Contents/MacOS/hermes-profile-runner-linux-arm64")
	if got[0] != want {
		t.Fatalf("first candidate = %q, want %q", got[0], want)
	}
}

func TestProfileRunnerCandidatesIncludeWindowsInstallPath(t *testing.T) {
	root := filepath.FromSlash("C:/Users/user/.hermes-dock")
	got := profileRunnerCandidatesForExecutable(root, "amd64", filepath.FromSlash("C:/Program Files/Hermes Dock/hermes-dock.exe"))
	want := filepath.FromSlash("C:/Program Files/Hermes Dock/hermes-profile-runner-linux-amd64")
	if got[0] != want {
		t.Fatalf("first candidate = %q, want %q", got[0], want)
	}
}

func TestSyncProfileRunnerReplacesStaleHelper(t *testing.T) {
	dir := t.TempDir()
	app := &App{instanceRoot: dir}
	source := filepath.Join(dir, "packaged-runner")
	target := filepath.Join(dir, "released-runner")
	if err := os.WriteFile(source, []byte("new-runner"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("old-runner"), 0755); err != nil {
		t.Fatal(err)
	}

	if err := app.syncProfileRunner(source, target, "linux"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "new-runner" {
		t.Fatalf("released runner = %q, want packaged runner", data)
	}
}

func TestSyncProfileRunnerStopsRunningContainerBeforeWindowsReplacement(t *testing.T) {
	installFakeDocker(t, true)
	dir := t.TempDir()
	app := &App{instanceRoot: dir}
	source := filepath.Join(dir, "packaged-runner")
	target := filepath.Join(dir, "launcher", "helpers", "hermes-profile-runner")
	mustWriteFile(t, app.composePath(), "services: {}\n", 0644)
	mustWriteFile(t, source, "new-runner", 0755)
	mustWriteFile(t, target, "old-runner", 0755)

	if err := app.syncProfileRunner(source, target, "windows"); err != nil {
		t.Fatal(err)
	}
	logData, err := os.ReadFile(fakeDockerLogPath(t))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(logData), "compose stop") {
		t.Fatalf("docker compose stop was not called: %s", logData)
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "new-runner" {
		t.Fatalf("released runner = %q, want packaged runner", data)
	}
}

func TestSyncProfileRunnerDoesNotStopWindowsContainerWhenUnchanged(t *testing.T) {
	installFakeDocker(t, true)
	dir := t.TempDir()
	app := &App{instanceRoot: dir}
	source := filepath.Join(dir, "packaged-runner")
	target := filepath.Join(dir, "launcher", "helpers", "hermes-profile-runner")
	mustWriteFile(t, app.composePath(), "services: {}\n", 0644)
	mustWriteFile(t, source, "same-runner", 0755)
	mustWriteFile(t, target, "same-runner", 0755)

	if err := app.syncProfileRunner(source, target, "windows"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(fakeDockerLogPath(t)); !os.IsNotExist(err) {
		t.Fatalf("docker should not be called for an unchanged runner: %v", err)
	}
}

func TestProfileRunnerSourceNeedsBuild(t *testing.T) {
	dir := t.TempDir()
	sourceDir := filepath.Join(dir, "cmd", "hermes-profile-runner")
	target := filepath.Join(dir, "launcher", "helpers", "hermes-profile-runner")
	mustWriteFile(t, filepath.Join(sourceDir, "main.go"), "package main\n", 0644)
	mustWriteFile(t, filepath.Join(sourceDir, "main_test.go"), "package main\n", 0644)
	mustWriteFile(t, target, "old-runner", 0755)

	oldTime := time.Now().Add(-2 * time.Hour)
	newTime := time.Now().Add(-time.Hour)
	if err := os.Chtimes(target, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(filepath.Join(sourceDir, "main.go"), newTime, newTime); err != nil {
		t.Fatal(err)
	}
	needsBuild, err := profileRunnerSourceNeedsBuild(sourceDir, target)
	if err != nil {
		t.Fatal(err)
	}
	if !needsBuild {
		t.Fatal("newer runner source should require a rebuild")
	}

	latestTime := time.Now()
	if err := os.Chtimes(target, latestTime, latestTime); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(filepath.Join(sourceDir, "main_test.go"), latestTime.Add(time.Hour), latestTime.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	needsBuild, err = profileRunnerSourceNeedsBuild(sourceDir, target)
	if err != nil {
		t.Fatal(err)
	}
	if needsBuild {
		t.Fatal("test-only source changes should not require a runner rebuild")
	}
}

func TestProfileRunnerSourceNeedsBuildWhenTargetMissing(t *testing.T) {
	dir := t.TempDir()
	sourceDir := filepath.Join(dir, "cmd", "hermes-profile-runner")
	mustWriteFile(t, filepath.Join(sourceDir, "main.go"), "package main\n", 0644)

	needsBuild, err := profileRunnerSourceNeedsBuild(sourceDir, filepath.Join(dir, "missing-runner"))
	if err != nil {
		t.Fatal(err)
	}
	if !needsBuild {
		t.Fatal("missing runner should require a build")
	}
}
