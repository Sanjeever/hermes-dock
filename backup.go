package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

func newBackupID() string {
	return time.Now().UTC().Format("20060102T150405.000000000Z") + "-" + uuid.NewString()
}

func (a *App) createBackupRoot(id string, reason string) (string, error) {
	backupsRoot := filepath.Join(a.hermesDockDir(), "backups")
	if err := ensureDir(backupsRoot); err != nil {
		return "", err
	}
	root := filepath.Join(backupsRoot, id+"-"+reason)
	if err := os.Mkdir(root, 0755); err != nil {
		return "", err
	}
	return root, nil
}

func (a *App) backupFile(path string, reason string) (err error) {
	src, err := openFileBeneath(a.instanceRoot, path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer src.Close()
	openedInfo, err := src.Stat()
	if err != nil {
		return err
	}
	if !openedInfo.Mode().IsRegular() {
		return fmt.Errorf("只能备份普通文件：%s", path)
	}
	id := newBackupID()
	reason = sanitizeName(firstNonEmpty(reason, "backup"))
	rel, err := filepath.Rel(a.instanceRoot, path)
	if err != nil {
		rel = filepath.Base(path)
	}
	backupRoot, err := a.createBackupRoot(id, reason)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = os.RemoveAll(backupRoot)
		}
	}()
	target := filepath.Join(backupRoot, rel)
	if err := ensureDir(filepath.Dir(target)); err != nil {
		return err
	}
	dst, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer dst.Close()
	if _, err := io.Copy(dst, src); err != nil {
		return err
	}
	if err := a.updateStateAllowMissing(func(state *LauncherState) error {
		state.Backups = append(state.Backups, BackupRecord{
			ID:     id,
			Reason: reason,
			Path:   strings.TrimPrefix(target, a.instanceRoot+string(os.PathSeparator)),
		})
		state.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		return nil
	}); err != nil {
		return err
	}
	committed = true
	return nil
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
