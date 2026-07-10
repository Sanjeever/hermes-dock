package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (a *App) backupFile(path string, reason string) error {
	if _, err := os.Stat(path); err != nil {
		return nil
	}
	id := time.Now().UTC().Format("20060102T150405Z")
	reason = sanitizeName(firstNonEmpty(reason, "backup"))
	rel, err := filepath.Rel(a.instanceRoot, path)
	if err != nil {
		rel = filepath.Base(path)
	}
	target := filepath.Join(a.hermesDockDir(), "backups", id+"-"+reason, rel)
	if err := ensureDir(filepath.Dir(target)); err != nil {
		return err
	}
	src, err := os.Open(path)
	if err != nil {
		return err
	}
	defer src.Close()
	dst, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer dst.Close()
	if _, err := io.Copy(dst, src); err != nil {
		return err
	}
	state, err := a.readState()
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	state.Backups = append(state.Backups, BackupRecord{
		ID:     id,
		Reason: reason,
		Path:   strings.TrimPrefix(target, a.instanceRoot+string(os.PathSeparator)),
	})
	state.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	return a.writeState(state)
}

func sanitizeName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "backup"
	}
	var b strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteRune('-')
		}
	}
	return b.String()
}
