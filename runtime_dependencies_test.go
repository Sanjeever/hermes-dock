package main

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestEmbeddedRuntimeDependenciesMatchBuildArchitecture(t *testing.T) {
	wantPlatform := map[string]string{
		"amd64": "x86_64",
		"arm64": "aarch64",
	}[runtime.GOARCH]
	platform, err := runtimeDependencyFS.ReadFile(runtimeDependencySourceRoot + "/platform")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(platform)) != wantPlatform {
		t.Fatalf("embedded platform = %q, want %q", strings.TrimSpace(string(platform)), wantPlatform)
	}
	pythonVersion, err := runtimeDependencyFS.ReadFile(runtimeDependencySourceRoot + "/python-version")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(pythonVersion)) != "3.13" {
		t.Fatalf("embedded Python version = %q, want 3.13", strings.TrimSpace(string(pythonVersion)))
	}
	checksums, err := runtimeDependencyFS.ReadFile(runtimeDependencySourceRoot + "/SHA256SUMS")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(checksums), "paddlepaddle-3.1.1-cp313-cp313-linux_") {
		t.Fatal("embedded wheelhouse is missing the architecture-specific PaddlePaddle wheel")
	}
}

func TestEmbeddedRuntimeDependencyMetadataMatchesManifest(t *testing.T) {
	checksums, err := runtimeDependencyFS.ReadFile(runtimeDependencySourceRoot + "/SHA256SUMS")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(checksums), "\r") {
		t.Fatal("embedded SHA256SUMS must use Unix line endings")
	}
	expected := map[string]string{}
	for _, line := range strings.Split(string(checksums), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 2 {
			t.Fatalf("invalid checksum line %q", line)
		}
		expected[strings.TrimPrefix(fields[1], "./")] = strings.ToLower(fields[0])
	}
	entries, err := runtimeDependencyFS.ReadDir(runtimeDependencySourceRoot)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == "SHA256SUMS" {
			continue
		}
		content, err := runtimeDependencyFS.ReadFile(runtimeDependencySourceRoot + "/" + entry.Name())
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(content)
		got := hex.EncodeToString(sum[:])
		if got != expected[entry.Name()] {
			t.Fatalf("embedded metadata checksum for %s = %s, want %s", entry.Name(), got, expected[entry.Name()])
		}
	}
}

func TestVerifyRuntimeDependencyDirectoryRejectsUntrustedManifest(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "SHA256SUMS"), []byte("tampered\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := verifyRuntimeDependencyDirectory(root, false); err == nil || !strings.Contains(err.Error(), "内置清单不一致") {
		t.Fatalf("verifyRuntimeDependencyDirectory() error = %v", err)
	}
}

func TestCleanupRuntimeDependencyStaging(t *testing.T) {
	parent := t.TempDir()
	staging := filepath.Join(parent, ".runtime-deps-"+runtimeDependencyBundleVersion+"-interrupted")
	if err := os.Mkdir(staging, 0755); err != nil {
		t.Fatal(err)
	}
	if err := cleanupRuntimeDependencyStaging(parent); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(staging); !os.IsNotExist(err) {
		t.Fatalf("interrupted staging still exists: %v", err)
	}
}

func TestCleanupObsoleteRuntimeDependenciesPreservesUnknownDirectories(t *testing.T) {
	app := newTestApp(t)
	parent := filepath.Dir(app.runtimeDependencyBundlePath())
	for _, name := range []string{runtimeDependencyBundleVersion, "cp313-old"} {
		writeTestRuntimeDependencyBundleDirectory(t, filepath.Join(parent, name))
	}
	unknown := filepath.Join(parent, "user-directory")
	if err := os.MkdirAll(unknown, 0755); err != nil {
		t.Fatal(err)
	}

	if err := app.cleanupObsoleteRuntimeDependencies(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(parent, "cp313-old")); !os.IsNotExist(err) {
		t.Fatalf("obsolete bundle still exists: %v", err)
	}
	for _, path := range []string{app.runtimeDependencyBundlePath(), unknown} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected directory %s to be preserved: %v", path, err)
		}
	}
}

func writeTestRuntimeDependencyBundleDirectory(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(path, "wheels"), 0755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"SHA256SUMS", "platform", "python-version"} {
		if err := os.WriteFile(filepath.Join(path, name), []byte("test\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
}
