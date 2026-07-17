package main

import (
	"net"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestSaveWebSettingsRejectsOccupiedPortWithoutChangingConfig(t *testing.T) {
	app := newTestApp(t)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	port := strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)
	before, err := app.readWebConfig()
	if err != nil {
		t.Fatal(err)
	}
	if err := app.SaveWebSettings(WebSettingsRequest{Enabled: true, Host: "127.0.0.1", Port: port}); err == nil {
		t.Fatal("occupied port should be rejected")
	}
	after, err := app.readWebConfig()
	if err != nil {
		t.Fatal(err)
	}
	if after.Host != before.Host || after.Port != before.Port || after.Enabled != before.Enabled {
		t.Fatalf("web config changed after failed preflight: before=%+v after=%+v", before, after)
	}
}

func TestWebTextFilePathAllowsProfileEnv(t *testing.T) {
	app := newTestApp(t)
	if err := app.CreateProfile(CreateProfileRequest{ID: "sales", CopyMode: profileCopyClean}); err != nil {
		t.Fatal(err)
	}

	got, err := app.webTextFilePath("sales", "profile_env")
	if err != nil {
		t.Fatal(err)
	}

	want := filepath.Join("data", "profiles", "sales", ".env")
	if got != want {
		t.Fatalf("profile_env path = %q, want %q", got, want)
	}
}

func TestWebRPCProfileOperationsRequireExplicitProfile(t *testing.T) {
	app := NewApp()
	handlers := app.webRPCHandlers()
	for _, name := range []string{
		"GetAppStateForProfile",
		"SaveModelConfigForProfile",
		"SaveFeishuConfigForProfile",
		"ListProfileSkillsForProfile",
		"ReadWebTextFile",
		"BatchCopyProfileConfig",
		"SyncBundledContent",
	} {
		if handlers[name] == nil {
			t.Fatalf("missing profile-scoped Web RPC handler %s", name)
		}
	}
	for _, name := range []string{"GetAppState", "SaveModelConfig", "SaveFeishuConfig", "ListProfileSkills"} {
		if handlers[name] != nil {
			t.Fatalf("legacy current-profile Web RPC handler remains exposed: %s", name)
		}
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
