package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBackupFileCreatesUniqueRecordsAndPaths(t *testing.T) {
	app := newTestApp(t)
	path := filepath.Join(app.dataDir(), "unique-backup.txt")
	if err := os.WriteFile(path, []byte("first"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := app.backupFile(path, "same-reason"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("second"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := app.backupFile(path, "same-reason"); err != nil {
		t.Fatal(err)
	}
	state, err := app.readState()
	if err != nil {
		t.Fatal(err)
	}
	last := state.Backups[len(state.Backups)-2:]
	if last[0].ID == last[1].ID || last[0].Path == last[1].Path {
		t.Fatalf("backup records collided: %+v", last)
	}
}
