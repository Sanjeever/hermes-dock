package main

import (
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
