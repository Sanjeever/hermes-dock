package main

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

//go:embed all:templates/seed-data
var seedData embed.FS

func (a *App) releaseSeedData() error {
	return a.releaseSeedDataTo(a.dataDir(), "default")
}

func (a *App) releaseSeedDataTo(targetRoot string, profileID string) error {
	initializeBundledState := !fileExists(filepath.Join(targetRoot, "SOUL.md")) && !fileExists(filepath.Join(targetRoot, "skills"))
	if err := ensureDir(targetRoot); err != nil {
		return err
	}
	if err := fs.WalkDir(seedData, "templates/seed-data", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel("templates/seed-data", path)
		if err != nil || rel == "." {
			return err
		}
		target := filepath.Join(targetRoot, rel)
		if d.IsDir() {
			return ensureDir(target)
		}
		if _, err := os.Stat(target); err == nil {
			return nil
		}
		data, err := seedData.ReadFile(path)
		if err != nil {
			return err
		}
		data = profileSeedData(data, rel, profileID)
		mode := os.FileMode(0644)
		if strings.HasSuffix(target, ".env") {
			mode = 0600
		}
		if err := ensureDir(filepath.Dir(target)); err != nil {
			return err
		}
		return os.WriteFile(target, data, mode)
	}); err != nil {
		return err
	}
	if !initializeBundledState || fileExists(a.bundledContentStatePath(profileID)) {
		return nil
	}
	return a.recordBundledContentState(profileID, targetRoot)
}

func profileSeedData(data []byte, rel string, profileID string) []byte {
	profileID = strings.TrimSpace(profileID)
	if profileID == "" || profileID == "default" {
		return data
	}
	home := "/opt/data/profiles/" + profileID
	switch filepath.ToSlash(rel) {
	case "config.yaml":
		text := strings.ReplaceAll(string(data), "cwd: /opt/data", "cwd: "+home)
		return []byte(text)
	case "SOUL.md":
		return []byte(rewriteProfileContainerHome(string(data), "default", profileID))
	default:
		return data
	}
}
