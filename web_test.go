package main

import (
	"path/filepath"
	"testing"
)

func TestWebTextFilePathAllowsProfileEnv(t *testing.T) {
	app := NewApp()
	app.instanceRoot = t.TempDir()

	got, err := app.webTextFilePath("profile_env")
	if err != nil {
		t.Fatal(err)
	}

	want := filepath.Join("data", ".env")
	if got != want {
		t.Fatalf("profile_env path = %q, want %q", got, want)
	}
}
