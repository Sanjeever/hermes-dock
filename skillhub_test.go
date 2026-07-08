package main

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractSkillHubZipIgnoresUndeclaredPackageMeta(t *testing.T) {
	target := t.TempDir()
	skill := []byte("---\nname: article-polisher\n---\n\n# Article Polisher\n")
	zipPath := writeSkillHubTestZip(t, map[string][]byte{
		"SKILL.md":      skill,
		"_meta.json":    []byte(`{"source":"skillhub"}`),
		"references.md": []byte("extra"),
	})

	err := extractSkillHubZip(zipPath, target, []SkillHubFile{
		{Path: "SKILL.md", SHA256: sha256Hex(skill)},
		{Path: "references.md", SHA256: sha256Hex([]byte("extra"))},
	})
	if err != nil {
		t.Fatal(err)
	}
	if fileExists(filepath.Join(target, "_meta.json")) {
		t.Fatalf("package metadata should not be installed")
	}
	if !fileExists(filepath.Join(target, "SKILL.md")) {
		t.Fatalf("SKILL.md was not extracted")
	}
}

func TestExtractSkillHubZipRejectsOtherUndeclaredFiles(t *testing.T) {
	target := t.TempDir()
	skill := []byte("---\nname: article-polisher\n---\n\n# Article Polisher\n")
	zipPath := writeSkillHubTestZip(t, map[string][]byte{
		"SKILL.md": skill,
		"extra.md": []byte("extra"),
	})

	err := extractSkillHubZip(zipPath, target, []SkillHubFile{
		{Path: "SKILL.md", SHA256: sha256Hex(skill)},
	})
	if err == nil || !strings.Contains(err.Error(), "未声明文件：extra.md") {
		t.Fatalf("expected undeclared file error, got %v", err)
	}
}

func writeSkillHubTestZip(t *testing.T, files map[string][]byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "skill.zip")
	out, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(out)
	for name, data := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := out.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
