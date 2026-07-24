package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestExportInstanceBackupExcludesRuntimeState(t *testing.T) {
	app := newTestApp(t)
	mustWriteFile(t, filepath.Join(app.instanceRoot, "data", ".env"), "API_KEY=secret\n", 0600)
	mustWriteFile(t, filepath.Join(app.instanceRoot, "data", "state.db"), "database\n", 0600)
	mustWriteFile(t, filepath.Join(app.instanceRoot, "data", "skills", "custom", "SKILL.md"), "custom skill\n", 0600)
	mustWriteFile(t, filepath.Join(app.instanceRoot, "data", "customer-project", "src", "main.ts"), "export {}\n", 0600)
	mustWriteFile(t, filepath.Join(app.instanceRoot, "data", "customer-project", ".git", "config"), "[core]\n", 0600)
	mustWriteFile(t, filepath.Join(app.instanceRoot, "data", "customer-project", "node_modules", "package", "index.js"), "generated\n", 0644)
	mustWriteFile(t, filepath.Join(app.instanceRoot, "data", "tmp", "draft.txt"), "temporary\n", 0644)
	mustWriteFile(t, filepath.Join(app.instanceRoot, "data", "home", ".npm", "cache", "package"), "generated\n", 0644)
	mustWriteFile(t, filepath.Join(app.instanceRoot, "data", "profiles", "sales", ".env"), "API_KEY=profile-secret\n", 0600)
	mustWriteFile(t, filepath.Join(app.instanceRoot, "data", "profiles", "sales", "memories", "2026-07-21.md"), "memory\n", 0600)
	mustWriteFile(t, filepath.Join(app.instanceRoot, "data", "profiles", "sales", ".venv", "lib", "package.py"), "generated\n", 0644)
	mustWriteFile(t, filepath.Join(app.instanceRoot, "data", "profiles", "sales", "checkpoints", "snapshot.json"), "generated\n", 0644)
	mustWriteFile(t, filepath.Join(app.instanceRoot, "data", ".dock", "profile-status.json"), "{}\n", 0644)
	mustWriteFile(t, filepath.Join(app.instanceRoot, "launcher", "web-sessions.json"), "{}\n", 0600)
	mustWriteFile(t, filepath.Join(app.instanceRoot, "launcher", "logs", "web-server.log"), "secret log\n", 0600)
	mustWriteFile(t, filepath.Join(app.instanceRoot, "launcher", "backups", "old.hdbackup"), "old\n", 0600)
	mustWriteFile(t, app.applyStatusPath(), "{\"state\":\"succeeded\",\"active\":false}\n", 0644)
	mustWriteFile(t, filepath.Join(app.sharedDir(), "report.txt"), "shared\n", 0644)

	target := filepath.Join(t.TempDir(), "export.hdbackup")
	manifest, err := app.ExportInstanceBackup(target)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Path != target {
		t.Fatalf("manifest path = %q, want %q", manifest.Path, target)
	}
	if !manifest.IncludesSecrets {
		t.Fatalf("backup should include secrets")
	}
	for _, want := range []string{"data/.dock", "data/tmp", "data/home/.npm", "data/customer-project/node_modules", "data/profiles/sales/.venv", "data/profiles/sales/checkpoints", "launcher/backups", "launcher/logs", "launcher/web-sessions.json", "launcher/apply-status.json", "shared"} {
		if !containsPathPrefix(manifest.ExcludedPaths, want) {
			t.Fatalf("excluded paths missing %s: %#v", want, manifest.ExcludedPaths)
		}
	}

	names := readBackupTarNames(t, target)
	for _, forbidden := range []string{
		"data/.dock/profile-status.json",
		"launcher/web-sessions.json",
		"launcher/logs/web-server.log",
		"launcher/backups/old.hdbackup",
		"launcher/apply-status.json",
		"shared/report.txt",
		"data/tmp/draft.txt",
		"data/home/.npm/cache/package",
		"data/customer-project/node_modules/package/index.js",
		"data/profiles/sales/.venv/lib/package.py",
		"data/profiles/sales/checkpoints/snapshot.json",
	} {
		if names[forbidden] {
			t.Fatalf("backup included forbidden path %s", forbidden)
		}
	}
	for _, want := range []string{"manifest.json", "checksums.txt", "data/.env", "data/state.db", "data/skills/custom/SKILL.md", "data/customer-project/src/main.ts", "data/customer-project/.git/config", "data/profiles/sales/.env", "data/profiles/sales/memories/2026-07-21.md", "launcher/profiles.json", "launcher/profile-content/default.json", "launcher/web-server.json", "launcher/dufs/config.yaml", "docker-compose.yaml"} {
		if !names[want] {
			t.Fatalf("backup missing %s", want)
		}
	}

	inspected, err := app.InspectInstanceBackup(target)
	if err != nil {
		t.Fatal(err)
	}
	if inspected.FileCount != manifest.FileCount || inspected.TotalBytes != manifest.TotalBytes {
		t.Fatalf("inspected manifest mismatch: %+v vs %+v", inspected, manifest)
	}
}

func TestShouldExcludeInstanceBackupPathKeepsMigrationData(t *testing.T) {
	tests := []struct {
		path    string
		exclude bool
	}{
		{path: "data/tmp/report.xlsx", exclude: true},
		{path: "data/.cache/uv/archive", exclude: true},
		{path: "data/home/.npm/_cacache/package", exclude: true},
		{path: "data/home/.local/lib/package", exclude: true},
		{path: "data/home/.paddlex/models/model.bin", exclude: true},
		{path: "data/profiles/sales/cache/result", exclude: true},
		{path: "data/profiles/sales/checkpoints/snapshot.json", exclude: true},
		{path: "data/profiles/sales/models_dev_cache.json", exclude: true},
		{path: "data/customer-project/node_modules/package/index.js", exclude: true},
		{path: "data/customer-project/.venv/lib/package.py", exclude: true},
		{path: "data/customer-project/__pycache__/module.pyc", exclude: true},
		{path: "data/gateway.pid", exclude: true},
		{path: "data/.env", exclude: false},
		{path: "data/state.db", exclude: false},
		{path: "data/state.db-wal", exclude: false},
		{path: "data/skills/custom/SKILL.md", exclude: false},
		{path: "data/memories/2026-07-21.md", exclude: false},
		{path: "data/sessions/session.json", exclude: false},
		{path: "data/weixin/accounts/default.json", exclude: false},
		{path: "data/home/.lark-cli/config.json", exclude: false},
		{path: "data/customer-project/src/main.ts", exclude: false},
		{path: "data/customer-project/.git/config", exclude: false},
		{path: "data/profiles/sales/.env", exclude: false},
		{path: "data/profiles/sales/state.db-wal", exclude: false},
		{path: "data/profiles/sales/memories/2026-07-21.md", exclude: false},
	}
	for _, tt := range tests {
		t.Run(strings.ReplaceAll(tt.path, "/", "_"), func(t *testing.T) {
			if got := shouldExcludeInstanceBackupPath(tt.path); got != tt.exclude {
				t.Fatalf("shouldExcludeInstanceBackupPath(%q) = %v, want %v", tt.path, got, tt.exclude)
			}
		})
	}
}

func TestValidateRestorableBackupPathAcceptsLegacyBackupCaches(t *testing.T) {
	for _, path := range []string{
		"data/tmp/report.xlsx",
		"data/customer-project/node_modules/package/index.js",
		"data/profiles/sales/.venv/lib/package.py",
	} {
		if err := validateRestorableBackupPath(path); err != nil {
			t.Fatalf("legacy backup path %q should remain restorable: %v", path, err)
		}
	}
	if err := validateRestorableBackupPath("data/.dock/profile-status.json"); err == nil {
		t.Fatal("derived runtime state should not be restorable")
	}
}

func TestExtractInstanceBackupRejectsOversizedFileHeader(t *testing.T) {
	path := filepath.Join(t.TempDir(), "oversized.hdbackup")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(file)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{Name: "data/huge.bin", Mode: 0600, Size: instanceBackupMaxFileBytes + 1}); err != nil {
		t.Fatal(err)
	}
	_ = tw.Close()
	_ = gz.Close()
	_ = file.Close()

	if _, err := extractInstanceBackup(path, t.TempDir()); err == nil || !strings.Contains(err.Error(), "单个文件") {
		t.Fatalf("oversized archive error = %v", err)
	}
}

func TestValidateInstanceBackupManifestRejectsDeclaredSizeMismatch(t *testing.T) {
	manifest := InstanceBackupManifest{
		Format:        instanceBackupFormat,
		SchemaVersion: instanceBackupSchemaVersion,
		FileCount:     1,
		TotalBytes:    instanceBackupMaxTotalBytes + 1,
	}
	if err := validateInstanceBackupManifest(manifest); err == nil {
		t.Fatal("oversized manifest should be rejected")
	}
}

func TestExportInstanceBackupStopsRunningContainerAndRestarts(t *testing.T) {
	app := newTestApp(t)
	installFakeDocker(t, true)
	target := filepath.Join(t.TempDir(), "export.hdbackup")

	if _, err := app.ExportInstanceBackup(target); err != nil {
		t.Fatal(err)
	}
	logData, err := os.ReadFile(fakeDockerLogPath(t))
	if err != nil {
		t.Fatal(err)
	}
	log := string(logData)
	if !strings.Contains(log, "compose ps --status running --services") {
		t.Fatalf("docker running status was not checked: %s", log)
	}
	if !strings.Contains(log, "compose stop") {
		t.Fatalf("docker compose stop was not called: %s", log)
	}
	if !strings.Contains(log, "compose start") {
		t.Fatalf("docker compose start was not called: %s", log)
	}
}

func TestWriteBackupEntryRejectsFileSizeChange(t *testing.T) {
	path := filepath.Join(t.TempDir(), "growing-state.json")
	mustWriteFile(t, path, "{}\n", 0600)
	staleInfo, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, path, "{\"state\":\"updated after collection\"}\n", 0600)

	var archive bytes.Buffer
	tw := tar.NewWriter(&archive)
	checksums := map[string]string{}
	if err := writeBackupEntry(tw, instanceBackupEntry{AbsPath: path, RelPath: "launcher/state.json", Info: staleInfo}, checksums); err == nil {
		t.Fatal("file size changes should abort export")
	}
}

func TestCopyBackupToTempRejectsOversizedSource(t *testing.T) {
	path := filepath.Join(t.TempDir(), "oversized.hdbackup")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(instanceBackupMaxArchiveBytes + 1); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if _, cleanup, err := copyBackupToTemp(path); err == nil {
		cleanup()
		t.Fatal("oversized source should be rejected")
	}
}

func TestImportInstanceBackupRestoresFilesAfterPreImportBackup(t *testing.T) {
	app := newTestApp(t)
	installFakeDocker(t, false)
	mustWriteFile(t, filepath.Join(app.instanceRoot, "data", ".env"), "API_KEY=before\n", 0600)
	baselinePath := app.bundledContentStatePath(defaultProfileID)
	baselineBefore, err := os.ReadFile(baselinePath)
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "source.hdbackup")
	if _, err := app.ExportInstanceBackup(target); err != nil {
		t.Fatal(err)
	}

	mustWriteFile(t, filepath.Join(app.instanceRoot, "data", ".env"), "API_KEY=after\n", 0600)
	mustWriteFile(t, filepath.Join(app.instanceRoot, "data", ".dock", "profile-status.json"), "{}\n", 0644)
	mustWriteFile(t, baselinePath, "{\"templateVersion\":\"changed\"}\n", 0644)
	mustWriteFile(t, app.applyStatusPath(), "{\"state\":\"succeeded\",\"active\":false}\n", 0644)

	result, err := app.ImportInstanceBackup(InstanceBackupImportRequest{Path: target, Confirm: "导入"})
	if err != nil {
		t.Fatal(err)
	}
	if result.PreImportBackupPath == "" || !fileExists(result.PreImportBackupPath) {
		t.Fatalf("pre-import backup missing: %q", result.PreImportBackupPath)
	}
	restored, err := os.ReadFile(filepath.Join(app.instanceRoot, "data", ".env"))
	if err != nil {
		t.Fatal(err)
	}
	if string(restored) != "API_KEY=before\n" {
		t.Fatalf("env not restored: %s", restored)
	}
	if fileExists(filepath.Join(app.instanceRoot, "data", ".dock", "profile-status.json")) {
		t.Fatalf("runtime status should not be restored")
	}
	restoredBaseline, err := os.ReadFile(baselinePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(restoredBaseline) != string(baselineBefore) {
		t.Fatalf("bundled content baseline was not restored")
	}
	if fileExists(app.applyStatusPath()) {
		t.Fatal("device-local apply task status should be cleared during import")
	}
	logData, err := os.ReadFile(fakeDockerLogPath(t))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(logData), "compose down") {
		t.Fatalf("docker compose down was not called: %s", logData)
	}
}

func readBackupTarNames(t *testing.T, path string) map[string]bool {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	gz, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	out := map[string]bool{}
	for {
		header, err := tr.Next()
		if err == io.EOF {
			return out
		}
		if err != nil {
			t.Fatal(err)
		}
		out[header.Name] = true
	}
}

func containsPathPrefix(paths []string, want string) bool {
	for _, path := range paths {
		if path == want || strings.HasPrefix(path, want+"/") {
			return true
		}
	}
	return false
}

func mustWriteFile(t *testing.T, path string, content string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatal(err)
	}
}

func installFakeDocker(t *testing.T, running bool) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake docker script is unix-only")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "docker")
	runningBlock := ""
	if running {
		runningBlock = "if [ \"$*\" = \"compose ps --status running --services\" ]; then echo hermes; fi\n"
	}
	script := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> \"" + filepath.Join(dir, "docker.log") + "\"\n" + runningBlock + "exit 0\n"
	if err := os.WriteFile(path, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func fakeDockerLogPath(t *testing.T) string {
	t.Helper()
	path, err := execLookPath("docker")
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(filepath.Dir(path), "docker.log")
}
