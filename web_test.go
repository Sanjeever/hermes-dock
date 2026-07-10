package main

import (
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"
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

func TestAddWebSessionReturnsReadErrorWithoutOverwritingFile(t *testing.T) {
	app := NewApp()
	app.instanceRoot = t.TempDir()
	if err := ensureDir(app.hermesDockDir()); err != nil {
		t.Fatal(err)
	}
	path := app.webSessionsPath()
	if err := os.WriteFile(path, []byte("not-json"), 0600); err != nil {
		t.Fatal(err)
	}

	if err := app.addWebSession(webSession{IDHash: "new"}); err == nil {
		t.Fatal("addWebSession should reject a corrupted session file")
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "not-json" {
		t.Fatalf("corrupted session file was overwritten: %q", got)
	}
}

func TestAddWebSessionSerializesConcurrentUpdates(t *testing.T) {
	app := NewApp()
	app.instanceRoot = t.TempDir()
	if err := ensureDir(app.hermesDockDir()); err != nil {
		t.Fatal(err)
	}

	const count = 20
	expires := time.Now().UTC().Add(time.Hour).Format(time.RFC3339)
	errs := make(chan error, count)
	var wg sync.WaitGroup
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			errs <- app.addWebSession(webSession{IDHash: strconv.Itoa(index), ExpiresAt: expires})
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}

	file, err := app.readWebSessions()
	if err != nil {
		t.Fatal(err)
	}
	if len(file.Sessions) != count {
		t.Fatalf("session count = %d, want %d", len(file.Sessions), count)
	}
}
