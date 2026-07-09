package main

import (
	"archive/tar"
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
	mustWriteFile(t, filepath.Join(app.instanceRoot, "data", ".dock", "profile-status.json"), "{}\n", 0644)
	mustWriteFile(t, filepath.Join(app.instanceRoot, "launcher", "web-sessions.json"), "{}\n", 0600)
	mustWriteFile(t, filepath.Join(app.instanceRoot, "launcher", "logs", "web-server.log"), "secret log\n", 0600)
	mustWriteFile(t, filepath.Join(app.instanceRoot, "launcher", "backups", "old.hdbackup"), "old\n", 0600)

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
	for _, want := range []string{"data/.dock", "launcher/backups", "launcher/logs", "launcher/web-sessions.json"} {
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
	} {
		if names[forbidden] {
			t.Fatalf("backup included forbidden path %s", forbidden)
		}
	}
	for _, want := range []string{"manifest.json", "checksums.txt", "data/.env", "launcher/profiles.json", "launcher/web-server.json", "docker-compose.yaml"} {
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

func TestImportInstanceBackupRestoresFilesAfterPreImportBackup(t *testing.T) {
	app := newTestApp(t)
	installFakeDocker(t, false)
	mustWriteFile(t, filepath.Join(app.instanceRoot, "data", ".env"), "API_KEY=before\n", 0600)
	target := filepath.Join(t.TempDir(), "source.hdbackup")
	if _, err := app.ExportInstanceBackup(target); err != nil {
		t.Fatal(err)
	}

	mustWriteFile(t, filepath.Join(app.instanceRoot, "data", ".env"), "API_KEY=after\n", 0600)
	mustWriteFile(t, filepath.Join(app.instanceRoot, "data", ".dock", "profile-status.json"), "{}\n", 0644)

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
