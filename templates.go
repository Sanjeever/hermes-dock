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
	if err := ensureDir(a.dataDir()); err != nil {
		return err
	}
	return fs.WalkDir(seedData, "templates/seed-data", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel("templates/seed-data", path)
		if err != nil || rel == "." {
			return err
		}
		target := filepath.Join(a.dataDir(), rel)
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
		mode := os.FileMode(0644)
		if strings.HasSuffix(target, ".env") {
			mode = 0600
		}
		if err := ensureDir(filepath.Dir(target)); err != nil {
			return err
		}
		return os.WriteFile(target, data, mode)
	})
}
