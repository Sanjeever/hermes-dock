package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSafePathRejectsSymlinkComponents(t *testing.T) {
	app := NewApp()
	app.instanceRoot = t.TempDir()
	outside := t.TempDir()
	link := filepath.Join(app.instanceRoot, "linked")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := app.safePath(filepath.Join(link, "config.yaml")); err == nil {
		t.Fatal("safePath should reject symlink components")
	}
}

func TestSecureFileAccessRejectsSymlinkParent(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	outsideFile := filepath.Join(outside, "config.yaml")
	if err := os.WriteFile(outsideFile, []byte("outside"), 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "linked")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	target := filepath.Join(link, "config.yaml")
	if _, err := openFileBeneath(root, target); err == nil {
		t.Fatal("secure open should reject a symlink parent")
	}
	if err := atomicWriteFileBeneath(root, target, []byte("changed"), 0600); err == nil {
		t.Fatal("secure write should reject a symlink parent")
	}
	data, err := os.ReadFile(outsideFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "outside" {
		t.Fatalf("outside file changed: %q", data)
	}
}

func TestAtomicWriteFileBeneathReplacesRegularFile(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "data")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(target, []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := atomicWriteFileBeneath(root, target, []byte("new"), 0600); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "new" {
		t.Fatalf("content = %q, want new", data)
	}
}
