package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestValidateComposeSettingsRejectsInvalidValues(t *testing.T) {
	base := defaultComposeSettings()
	tests := []struct {
		name   string
		mutate func(*ComposeSettings)
	}{
		{name: "image newline", mutate: func(settings *ComposeSettings) { settings.Image = "valid/image:tag\nservices:" }},
		{name: "container name", mutate: func(settings *ComposeSettings) { settings.ContainerName = "bad name" }},
		{name: "gateway host", mutate: func(settings *ComposeSettings) { settings.GatewayHost = "localhost" }},
		{name: "dashboard port", mutate: func(settings *ComposeSettings) { settings.DashboardPort = "70000" }},
		{name: "memory", mutate: func(settings *ComposeSettings) { settings.MemoryLimit = "not-memory" }},
		{name: "cpu", mutate: func(settings *ComposeSettings) { settings.CPULimit = "0" }},
		{name: "shm", mutate: func(settings *ComposeSettings) { settings.ShmSize = "1g\nvolumes:" }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			settings := base
			tt.mutate(&settings)
			if err := validateComposeSettings(settings); err == nil {
				t.Fatal("invalid compose settings should be rejected")
			}
		})
	}
}

func TestRenderComposeQuotesUserControlledScalarValues(t *testing.T) {
	settings := defaultComposeSettings()
	content := renderCompose(settings, defaultProxySettings())
	var parsed map[string]interface{}
	if err := yaml.Unmarshal([]byte(content), &parsed); err != nil {
		t.Fatalf("rendered compose is invalid YAML: %v\n%s", err, content)
	}
	for _, want := range []string{
		`image: "` + settings.Image + `"`,
		`container_name: "` + settings.ContainerName + `"`,
		`shm_size: "` + settings.ShmSize + `"`,
		`memory: "` + settings.MemoryLimit + `"`,
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("compose missing quoted scalar %q", want)
		}
	}
}

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

func TestSaveComposeSettingsMountsSharedDirectory(t *testing.T) {
	app := newTestApp(t)
	sharedDirectory := filepath.Join(t.TempDir(), "team files")
	settings := app.readComposeSettings()
	settings.SharedDirectory = sharedDirectory

	if err := app.SaveComposeSettings(settings); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(sharedDirectory); err != nil || !info.IsDir() {
		t.Fatalf("shared directory was not created: %v", err)
	}
	data, err := os.ReadFile(app.composePath())
	if err != nil {
		t.Fatal(err)
	}
	compose := string(data)
	for _, want := range []string{
		`HERMES_WRITE_SAFE_ROOT: "/opt/data"`,
		`- "` + sharedDirectory + `:/opt/data/.dock/shared"`,
	} {
		if !strings.Contains(compose, want) {
			t.Fatalf("compose missing %q:\n%s", want, compose)
		}
	}
	state, err := app.readState()
	if err != nil {
		t.Fatal(err)
	}
	if state.ComposeSettings.SharedDirectory != sharedDirectory || !state.NeedsRebuild {
		t.Fatalf("shared directory settings were not saved: %+v", state.ComposeSettings)
	}
}

func TestSaveComposeSettingsLocksHermesImage(t *testing.T) {
	app := newTestApp(t)
	settings := app.readComposeSettings()
	settings.Image = "example/hermes:custom"

	if err := app.SaveComposeSettings(settings); err != nil {
		t.Fatal(err)
	}
	state, err := app.readState()
	if err != nil {
		t.Fatal(err)
	}
	if state.HermesImage != defaultImage || state.ComposeSettings.Image != defaultImage {
		t.Fatalf("saved image = state=%q settings=%q, want %q", state.HermesImage, state.ComposeSettings.Image, defaultImage)
	}
	content, err := os.ReadFile(app.composePath())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), `image: "`+defaultImage+`"`) {
		t.Fatalf("compose does not use locked image: %s", content)
	}
}

func TestMigrateFixedHermesImageRewritesOnlyHermesService(t *testing.T) {
	app := newTestApp(t)
	state, err := app.readState()
	if err != nil {
		t.Fatal(err)
	}
	migrations := state.Migrations[:0]
	for _, migration := range state.Migrations {
		if migration.ID != fixedImageMigrationID {
			migrations = append(migrations, migration)
		}
	}
	state.Migrations = migrations
	state.HermesImage = "example/hermes:custom"
	state.ComposeSettings.Image = "example/hermes:custom"
	state.NeedsRebuild = false
	state.PendingDufsOnly = true
	if err := app.writeState(state); err != nil {
		t.Fatal(err)
	}
	content := "services:\n  helper:\n    image: \"" + defaultImage + "\"\n  hermes:\n    image: \"example/hermes:custom\"\n"
	if err := atomicWriteFile(app.composePath(), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	if err := app.migrateFixedHermesImageIfNeeded(app.readComposeSettings()); err != nil {
		t.Fatal(err)
	}
	state, err = app.readState()
	if err != nil {
		t.Fatal(err)
	}
	if !migrationApplied(state.Migrations, fixedImageMigrationID) || !state.NeedsRebuild || state.PendingDufsOnly {
		t.Fatalf("fixed-image migration state = %+v", state)
	}
	if state.HermesImage != defaultImage || state.ComposeSettings.Image != defaultImage {
		t.Fatalf("migrated image = state=%q settings=%q, want %q", state.HermesImage, state.ComposeSettings.Image, defaultImage)
	}
	data, err := os.ReadFile(app.composePath())
	if err != nil {
		t.Fatal(err)
	}
	image, hasImage, err := composeServiceImage(data, "hermes")
	if err != nil {
		t.Fatal(err)
	}
	if !hasImage || image != defaultImage {
		t.Fatalf("migrated Hermes image = %q, want %q", image, defaultImage)
	}
}

func TestSaveTextFileRejectsHermesImageOverride(t *testing.T) {
	app := newTestApp(t)
	err := app.SaveTextFile(TextFileRequest{
		Path:    filepath.Base(app.overridePath()),
		Content: "services:\n  hermes:\n    image: example/hermes:custom\n",
	})
	if err == nil || !strings.Contains(err.Error(), "镜像已固定") {
		t.Fatalf("expected locked image error, got %v", err)
	}
}

func TestSaveComposeSettingsRejectsRelativeSharedDirectory(t *testing.T) {
	app := newTestApp(t)
	settings := app.readComposeSettings()
	settings.SharedDirectory = "relative/shared"
	if err := app.SaveComposeSettings(settings); err == nil || !strings.Contains(err.Error(), "绝对路径") {
		t.Fatalf("expected absolute path error, got %v", err)
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

func TestSaveEnvironmentMarksRebuildRequired(t *testing.T) {
	app := newTestApp(t)
	state, err := app.readState()
	if err != nil {
		t.Fatal(err)
	}
	state.NeedsRebuild = false
	if err := app.writeState(state); err != nil {
		t.Fatal(err)
	}

	if err := app.SaveEnvironment([]EnvVar{{Key: "TEST_SETTING", Value: "changed"}}); err != nil {
		t.Fatal(err)
	}
	state, err = app.readState()
	if err != nil {
		t.Fatal(err)
	}
	if !state.NeedsRebuild {
		t.Fatal("saved environment should require applying configuration")
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
