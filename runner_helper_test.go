package main

import (
	"os"
	"path/filepath"
	"testing"
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
	source := filepath.Join(dir, "packaged-runner")
	target := filepath.Join(dir, "released-runner")
	if err := os.WriteFile(source, []byte("new-runner"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("old-runner"), 0755); err != nil {
		t.Fatal(err)
	}

	if err := syncProfileRunner(source, target); err != nil {
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
