package main

import (
	"os"
	"strings"
	"testing"
)

func TestCalculateRecommendedResourceLimits(t *testing.T) {
	const gib = int64(1024 * 1024 * 1024)
	tests := []struct {
		name       string
		memBytes   int64
		cpu        int
		wantMemory string
		wantCPU    string
		wantMemGB  int
	}{
		{name: "low memory keeps one gigabyte", memBytes: 2 * gib, cpu: 2, wantMemory: "1G", wantCPU: "2.0", wantMemGB: 2},
		{name: "reserves two gigabytes", memBytes: 16 * gib, cpu: 8, wantMemory: "14G", wantCPU: "8.0", wantMemGB: 16},
		{name: "floors docker memory", memBytes: 15*gib + gib/2, cpu: 6, wantMemory: "13G", wantCPU: "6.0", wantMemGB: 15},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := calculateRecommendedResourceLimits(tt.memBytes, tt.cpu)
			if err != nil {
				t.Fatalf("calculateRecommendedResourceLimits() error = %v", err)
			}
			if got.MemoryLimit != tt.wantMemory || got.CPULimit != tt.wantCPU || got.DockerMemoryGB != tt.wantMemGB || got.DockerCPU != tt.cpu {
				t.Fatalf("calculateRecommendedResourceLimits() = %+v, want memory=%s cpu=%s dockerMemoryGB=%d dockerCPU=%d", got, tt.wantMemory, tt.wantCPU, tt.wantMemGB, tt.cpu)
			}
		})
	}
}

func TestCalculateRecommendedResourceLimitsRejectsInvalidResources(t *testing.T) {
	if _, err := calculateRecommendedResourceLimits(0, 2); err == nil {
		t.Fatal("expected error for empty memory")
	}
	if _, err := calculateRecommendedResourceLimits(8*1024*1024*1024, 0); err == nil {
		t.Fatal("expected error for empty cpu")
	}
}

func TestMarkRebuildAppliedClearsNeedsRebuild(t *testing.T) {
	app := newTestApp(t)
	state, err := app.readState()
	if err != nil {
		t.Fatal(err)
	}
	state.HermesImage = "example/hermes:new"
	state.LastSuccessfulHermesImage = "example/hermes:old"
	state.NeedsRebuild = true
	if err := app.writeState(state); err != nil {
		t.Fatal(err)
	}

	if err := app.markRebuildApplied("compose-runtime-hash"); err != nil {
		t.Fatal(err)
	}
	state, err = app.readState()
	if err != nil {
		t.Fatal(err)
	}
	if state.NeedsRebuild {
		t.Fatal("successful rebuild should clear needsRebuild")
	}
	if state.LastSuccessfulHermesImage != state.HermesImage {
		t.Fatalf("last successful image = %q, want %q", state.LastSuccessfulHermesImage, state.HermesImage)
	}
	if state.LastAppliedComposeHash != "compose-runtime-hash" {
		t.Fatalf("last applied compose hash = %q", state.LastAppliedComposeHash)
	}
}

func TestShouldRecreateComposeRuntime(t *testing.T) {
	tests := []struct {
		name        string
		lastHash    string
		currentHash string
		status      string
		want        bool
	}{
		{name: "matching running container restarts", lastHash: "same", currentHash: "same", status: "running", want: false},
		{name: "matching stopped container restarts", lastHash: "same", currentHash: "same", status: "stopped", want: false},
		{name: "first optimized apply recreates", currentHash: "current", status: "running", want: true},
		{name: "compose change recreates", lastHash: "old", currentHash: "new", status: "running", want: true},
		{name: "missing container recreates", lastHash: "same", currentHash: "same", status: "missing", want: true},
		{name: "unknown container state recreates", lastHash: "same", currentHash: "same", status: "unknown", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldRecreateComposeRuntime(tt.lastHash, tt.currentHash, tt.status); got != tt.want {
				t.Fatalf("shouldRecreateComposeRuntime() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestComposeRuntimeHashIncludesOverride(t *testing.T) {
	app := newTestApp(t)
	before, err := app.composeRuntimeHash()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(app.overridePath(), []byte("services:\n  hermes:\n    environment:\n      TEST: changed\n"), 0644); err != nil {
		t.Fatal(err)
	}
	after, err := app.composeRuntimeHash()
	if err != nil {
		t.Fatal(err)
	}
	if before == after {
		t.Fatal("compose runtime hash did not change with override")
	}
}

func TestApplyComposeRuntimeSelectsDockerOperation(t *testing.T) {
	app := newTestApp(t)
	installFakeDocker(t, false)

	if err := app.applyComposeRuntime(false); err != nil {
		t.Fatal(err)
	}
	if err := app.applyComposeRuntime(true); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(fakeDockerLogPath(t))
	if err != nil {
		t.Fatal(err)
	}
	log := string(data)
	if !strings.Contains(log, "compose restart hermes") {
		t.Fatalf("warm apply did not restart the existing container: %s", log)
	}
	if !strings.Contains(log, "compose up -d --force-recreate --remove-orphans hermes") {
		t.Fatalf("compose change did not recreate the hermes service: %s", log)
	}
}
